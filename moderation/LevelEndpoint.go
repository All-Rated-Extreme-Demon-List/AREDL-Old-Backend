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
				"level_id":           {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"position":           {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"name":               {util.LoadString, true, nil, util.PackRules()},
				"creators":           {util.LoadString, true, nil, util.PackRules(is.JSON)},
				"verifier":           {util.LoadString, true, nil, util.PackRules()},
				"publisher":          {util.LoadString, true, nil, util.PackRules()},
				"level_password":     {util.LoadString, false, nil, util.PackRules()},
				"custom_song":        {util.LoadString, false, nil, util.PackRules()},
				"two_player":         {util.LoadBool, false, false, util.PackRules()},
				"qualifying_percent": {util.LoadInt, false, 100, util.PackRules(validation.Min(1), validation.Max(100))},
			}),
		},
		Handler: func(c echo.Context) error {
			position := c.Get("position").(int)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				highestPosition, err := queryMaxPosition(txDao)
				if position > highestPosition+1 {
					return apis.NewBadRequestError("New position is outside the list", nil)
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
				record := models.NewRecord(collection)
				form := forms.NewRecordUpsert(app, record)
				form.SetDao(txDao)

				err = form.LoadData(map[string]any{
					"level_id":           c.Get("level_id"),
					"position":           position,
					"name":               c.Get("name"),
					"verifier":           c.Get("verifier"),
					"publisher":          c.Get("publisher"),
					"level_password":     c.Get("level_password"),
					"custom_song":        c.Get("custom_song"),
					"2_player":           c.Get("two_player"),
					"qualifying_percent": c.Get("qualifying_percent"),
				})
				if err != nil {
					return apis.NewApiError(500, "Error placing level", nil)
				}
				err = form.Submit()
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(500, "Error placing level", nil)
					}
				}
				err = points.UpdateListPoints(txDao, position, highestPosition+1)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
			return c.String(200, "added")
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
				if newPos < oldPos {
					// Move up
					minPos = newPos
					maxPos = oldPos
					moveInc = 1
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
				err = points.UpdateListPoints(txDao, minPos, maxPos)
				if err != nil {
					return apis.NewApiError(500, "Failed to update", nil)
				}

				return nil
			})
			if err != nil {
				return err
			}
			return c.String(200, "Moved level")
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
			err := points.UpdateListPoints(app.Dao(), c.Get("min_position").(int), c.Get("max_position").(int))
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
