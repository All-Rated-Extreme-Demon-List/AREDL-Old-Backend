package moderation

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func registerNameChangeAcceptEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/name-change/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableNameChangeRequests, c.Get("request_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Request not found", nil)
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, requestRecord.GetString("user"))
				if err != nil {
					return apis.NewApiError(500, "Could not find user in request", nil)
				}
				userRecord.Set("global_name", requestRecord.GetString("new_name"))
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to change username", nil)
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete request", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerNameChangeRejectEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/name-change/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableNameChangeRequests, c.Get("request_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Request not found", nil)
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete request", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
