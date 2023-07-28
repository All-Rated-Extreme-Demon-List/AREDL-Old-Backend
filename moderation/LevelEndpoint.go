package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

const pathLevelPrefix = pathPrefix + "/level"

func registerLevelPlaceEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/place",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"level_id":                 {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"position":                 {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"name":                     {util.LoadString, true, nil, util.PackRules()},
				"creator_ids":              {util.LoadStringArray, true, nil, util.PackRules()},
				"verifier":                 {util.LoadString, true, nil, util.PackRules()},
				"publisher":                {util.LoadString, true, nil, util.PackRules()},
				"level_password":           {util.LoadString, false, nil, util.PackRules()},
				"custom_song":              {util.LoadString, false, nil, util.PackRules()},
				"two_player":               {util.LoadBool, false, false, util.PackRules()},
				"qualifying_percent":       {util.LoadInt, false, 100, util.PackRules(validation.Min(1), validation.Max(100))},
				"verification_video":       {util.LoadString, true, nil, util.PackRules(is.URL)},
				"verification_fps":         {util.LoadInt, true, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"verification_device":      {util.LoadString, true, nil, util.PackRules(validation.In("pc", "mobile"))},
				"verification_raw_footage": {util.LoadString, false, "", util.PackRules(is.URL)},
			}),
		},
		Handler: func(c echo.Context) error {
			position := c.Get("position").(int)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				highestPosition, err := queryMaxPosition(txDao)
				if position > highestPosition+1 {
					return apis.NewBadRequestError("New position is outside the list", nil)
				}
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				// Move all levels down from the placement position
				_, err = txDao.DB().Update(names.TableLevels, dbx.Params{"position": dbx.NewExp("position+1")}, dbx.NewExp("position>={:position}", dbx.Params{"position": position})).Execute()
				if err != nil {
					return apis.NewApiError(500, "Error placing level", nil)
				}
				collection, err := txDao.FindCollectionByNameOrId(names.TableLevels)
				if err != nil {
					return apis.NewApiError(500, "Error placing level", nil)
				}

				// Write new level record into db
				levelRecord := models.NewRecord(collection)
				levelForm := forms.NewRecordUpsert(app, levelRecord)
				levelForm.SetDao(txDao)

				err = levelForm.LoadData(map[string]any{
					"level_id":           c.Get("level_id"),
					"position":           position,
					"name":               c.Get("name"),
					"verifier":           c.Get("verifier"),
					"publisher":          c.Get("publisher"),
					"verification":       c.Get("verification_video"),
					"level_password":     c.Get("level_password"),
					"custom_song":        c.Get("custom_song"),
					"2_player":           c.Get("two_player"),
					"qualifying_percent": c.Get("qualifying_percent"),
				})
				if err != nil {
					return apis.NewApiError(500, "Error placing level", nil)
				}
				err = levelForm.Submit()
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(500, "Error placing level", nil)
					}
				}

				creatorIds := c.Get("creator_ids").([]string)
				for _, creatorId := range creatorIds {
					_, err := util.AddRecordByCollectionName(txDao, app, names.TableCreators, map[string]any{
						"creator": creatorId,
						"level":   levelRecord.Id,
					})
					if err != nil {
						switch err.(type) {
						case validation.Errors:
							return apis.NewBadRequestError(err.Error(), nil)
						default:
							return apis.NewApiError(500, "Error placing level", nil)
						}
					}
				}

				_, err = util.AddRecordByCollectionName(txDao, app, names.TableSubmissions, map[string]any{
					"fps":             c.Get("verification_fps"),
					"video_url":       c.Get("verification_video"),
					"level":           levelRecord.Id,
					"status":          "accepted",
					"device":          c.Get("verification_device"),
					"percentage":      100,
					"placement_order": 1,
					"submitted_by":    c.Get("verifier"),
					"raw_footage":     c.Get("verification_raw_footage"),
				})
				if err != nil {
					return apis.NewApiError(500, "Failed to add verification submission"+err.Error(), nil)
				}

				err = points.UpdateListPointsByLevelRange(txDao, position, highestPosition+1)
				if err != nil {
					return apis.NewApiError(500, "Failed to update list points", nil)
				}

				_, err = util.AddRecordByCollectionName(txDao, app, names.TableLevelHistory, map[string]any{
					"level":        levelRecord.Id,
					"action":       "placed",
					"new_position": position,
					"cause":        levelRecord.Id,
					"action_by":    userRecord.Id,
				})
				if err != nil {
					return apis.NewApiError(500, "Failed to write place into the position history", nil)
				}

				_, err = txDao.DB().NewQuery(`
				INSERT INTO ` + names.TableLevelHistory + ` (level, action, new_position, cause, action_by)
				SELECT l.id AS level, 'placedAbove' AS action, l.position AS new_position, {:cause} AS cause, {:action_by} AS action_by
				FROM ` + names.TableLevels + ` l
				WHERE l.position BETWEEN {:minPos} + 1 AND {:maxPos} + 1
				`).Bind(dbx.Params{
					"minPos":    position,
					"maxPos":    highestPosition,
					"cause":     levelRecord.Id,
					"action_by": userRecord.Id,
				}).Execute()
				if err != nil {
					return apis.NewApiError(500, "Failed to write to position history", nil)
				}
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		},
	})
	return err
}

func registerLevelMoveEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/move",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id":           {util.LoadString, true, nil, util.PackRules()},
				"new_position": {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
			}),
		},
		Handler: func(c echo.Context) error {
			recordId := c.Get("id")
			newPos := c.Get("new_position").(int)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				// Get current position of the level
				record, err := txDao.FindRecordById(names.TableLevels, recordId.(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find level", nil)
				}
				oldPos := record.GetInt("position")

				if oldPos == newPos {
					return apis.NewBadRequestError("Level already is at that position", nil)
				}

				highestPosition, err := queryMaxPosition(txDao)
				if newPos > highestPosition {
					return apis.NewBadRequestError("New position is outside the list", nil)
				}

				// Determine in what direction the level was moved.
				// Move down
				minPos := oldPos
				maxPos := newPos
				moveInc := -1
				movedStatus := "movedDown"
				otherStatus := "movedPastDown"
				if newPos < oldPos {
					// Move up
					minPos = newPos
					maxPos = oldPos
					moveInc = 1
					movedStatus = "movedUp"
					otherStatus = "movedPastUp"
				}

				query := txDao.DB().Update(
					names.TableLevels,
					dbx.Params{"position": dbx.NewExp("CASE WHEN position = {:old} THEN {:new} ELSE position + {:inc} END",
						dbx.Params{"old": oldPos, "new": newPos, "inc": moveInc})},
					dbx.Between("position",
						minPos,
						maxPos,
					))
				if _, err = query.Execute(); err != nil {
					return apis.NewApiError(500, "Failed to update", nil)
				}
				// update list points for the new positions
				err = points.UpdateListPointsByLevelRange(txDao, minPos, maxPos)
				if err != nil {
					return apis.NewApiError(500, "Failed to update", nil)
				}

				_, err = util.AddRecordByCollectionName(txDao, app, names.TableLevelHistory, map[string]any{
					"level":        record.Id,
					"action":       movedStatus,
					"new_position": newPos,
					"cause":        record.Id,
					"action_by":    userRecord.Id,
				})
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(500, "Failed to write place into the position history", nil)
					}
				}

				_, err = txDao.DB().NewQuery(`
				INSERT INTO ` + names.TableLevelHistory + ` (level, action, new_position, cause, action_by)
				SELECT l.id AS level, {:status} AS action, l.position AS new_position, {:cause} AS cause, {:action_by} AS action_by
				FROM ` + names.TableLevels + ` l
				WHERE l.position BETWEEN {:minPos} AND {:maxPos} AND l.position <> {:newPos}
				`).Bind(dbx.Params{
					"status":    otherStatus,
					"minPos":    minPos,
					"maxPos":    maxPos,
					"cause":     record.Id,
					"action_by": userRecord.Id,
					"newPos":    newPos,
				}).Execute()
				if err != nil {
					return apis.NewApiError(500, "Failed to write to position history", nil)
				}
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		},
	})
	return err
}

func registerLevelUpdateEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id":                 {util.LoadString, true, nil, util.PackRules()},
				"level_id":           {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"name":               {util.LoadString, false, nil, util.PackRules()},
				"verification":       {util.LoadString, true, nil, util.PackRules()},
				"creator_ids":        {util.LoadStringArray, false, nil, util.PackRules()},
				"publisher":          {util.LoadString, false, nil, util.PackRules()},
				"level_password":     {util.LoadString, false, nil, util.PackRules()},
				"custom_song":        {util.LoadString, false, nil, util.PackRules()},
				"2_player":           {util.LoadBool, false, nil, util.PackRules()},
				"qualifying_percent": {util.LoadInt, false, nil, util.PackRules(validation.Min(1), validation.Max(100))},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				levelRecord, err := txDao.FindRecordById(names.TableLevels, c.Get("id").(string))
				if err != nil {
					return apis.NewBadRequestError("Level not found", nil)
				}
				levelForm := forms.NewRecordUpsert(app, levelRecord)
				levelForm.SetDao(txDao)
				err = levelForm.LoadData(map[string]any{
					"level_id":           util.UseOtherIfNil(c.Get("level_id"), levelRecord.GetString("level_id")),
					"name":               util.UseOtherIfNil(c.Get("name"), levelRecord.GetString("name")),
					"verification":       util.UseOtherIfNil(c.Get("verification"), levelRecord.GetString("verification")),
					"publisher":          util.UseOtherIfNil(c.Get("publisher"), levelRecord.GetString("publisher")),
					"level_password":     util.UseOtherIfNil(c.Get("level_password"), levelRecord.GetString("level_password")),
					"custom_song":        util.UseOtherIfNil(c.Get("custom_song"), levelRecord.GetString("custom_song")),
					"2_player":           util.UseOtherIfNil(c.Get("2_player"), levelRecord.GetString("2_player")),
					"qualifying_percent": util.UseOtherIfNil(c.Get("qualifying_percent"), levelRecord.GetString("qualifying_percent")),
				})
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(500, "Failed to update levels", nil)
					}
				}
				err = levelForm.Submit()
				if err != nil {
					return apis.NewApiError(500, "Failed to save level record", nil)
				}
				// Delete old creators
				type Creator struct {
					ID string `db:"creator"`
				}
				var currentCreatorsData []Creator
				err = txDao.DB().Select("creator").From(names.TableCreators).Where(dbx.HashExp{"level": levelRecord.Id}).All(&currentCreatorsData)
				if err != nil {
					return apis.NewApiError(500, "Failed to fetch current creators"+err.Error(), nil)
				}
				currentCreators := util.MapSlice(currentCreatorsData, func(t Creator) string {
					return t.ID
				})
				var newCreators []string
				if c.Get("creator_ids") != nil {
					newCreators = c.Get("creator_ids").([]string)
				}
				creatorsToRemove := util.SliceDifference(currentCreators, newCreators)
				creatorsToAdd := util.SliceDifference(newCreators, currentCreators)
				for _, creator := range creatorsToRemove {
					creatorRecord, err := txDao.FindFirstRecordByData(names.TableCreators, "creator", creator)
					if err != nil {
						return apis.NewApiError(500, "Failed to load to be removed creator", nil)
					}
					err = txDao.DeleteRecord(creatorRecord)
					if err != nil {
						return apis.NewApiError(500, "Failed to remove to be removed creator", nil)
					}
				}
				for _, creator := range creatorsToAdd {
					_, err = util.AddRecordByCollectionName(txDao, app, names.TableCreators, map[string]any{
						"creator": creator,
						"level":   levelRecord.Id,
					})
					if err != nil {
						return apis.NewApiError(500, "Failed to add creator", nil)
					}
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerUpdatePointsEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/update-points",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			// high requirement, because this is used in very rare occasions i.e. when the point curve changes.
			util.RequirePermission("listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"min_position": {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"max_position": {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
			}),
		},
		Handler: func(c echo.Context) error {
			err := points.UpdateListPointsByLevelRange(app.Dao(), c.Get("min_position").(int), c.Get("max_position").(int))
			if err != nil {
				return apis.NewApiError(500, "Failed to update", nil)
			}
			return c.String(200, "Updated points")
		},
	})
	return err
}

func queryMaxPosition(dao *daos.Dao) (int, error) {
	var position int
	err := dao.DB().Select("max(position)").From(names.TableLevels).Row(&position)
	if err != nil {
		return 0, err
	}
	return position, nil
}
