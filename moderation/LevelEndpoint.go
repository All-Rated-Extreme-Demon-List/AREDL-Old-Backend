package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

func registerLevelEndpoints(e *echo.Echo, app *pocketbase.PocketBase) error {
	err := registerLevelPlace(e, app)
	if err != nil {
		return err
	}
	err = registerLevelMove(e, app)
	if err != nil {
		return err
	}
	err = registerUpdatePoints(e, app)
	return err
}

func registerLevelPlace(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/place",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"level_id": {util.LoadInt, true, util.PackRules(validation.Min(1))},
				"position": {util.LoadInt, true, util.PackRules(validation.Min(1))},
				"name":     {util.LoadString, true, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			position := c.Get("position").(int)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				// TODO check if new position is outside the highest position
				// Move all levels down from the placement position
				_, err := txDao.DB().Update(names.TableLevels, dbx.Params{"position": dbx.NewExp("position+1")}, dbx.NewExp("position>={:position}", dbx.Params{"position": position})).Execute()
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

				//TODO values
				err = form.LoadData(map[string]any{
					"level_id":           c.Get("level_id"),
					"position":           position,
					"name":               c.Get("name"),
					"qualifying_percent": 100,
					"verifier":           "test",
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
				// TODO upper bound
				err = points.UpdateListPoints(txDao, position, 1000)
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

func registerLevelMove(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/move",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"level_id":     {util.LoadInt, true, util.PackRules(validation.Min(1))},
				"new_position": {util.LoadInt, true, util.PackRules(validation.Min(1))},
			}),
		},
		Handler: func(c echo.Context) error {
			levelId := c.Get("level_id").(int)
			newPos := c.Get("new_position").(int)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				// Get current position of the level
				record, err := txDao.FindFirstRecordByData(names.TableLevels, "level_id", levelId)
				if err != nil {
					return apis.NewBadRequestError("Could not find level", nil)
				}
				oldPos := record.GetInt("position")

				if oldPos == newPos {
					return apis.NewBadRequestError("Level already is at that position", nil)
				}

				// TODO check if new position is outside of the highest position

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

func registerUpdatePoints(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathLevelPrefix + "/update-points",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			// high requirement, because this is used in very rare occasions i.e. when the point curve changes.
			util.RequirePermission("listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"min_position": {util.LoadInt, true, util.PackRules(validation.Min(1))},
				"max_position": {util.LoadInt, true, util.PackRules(validation.Min(1))},
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
