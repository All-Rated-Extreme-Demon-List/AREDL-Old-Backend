package user

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerNameChangeRequestEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/change-name",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("member"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"new_name": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			// TODO validate name
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				requestRecord, _ := txDao.FindFirstRecordByData(names.TableNameChangeRequests, "user", userRecord.Id)
				if requestRecord == nil {
					requestCollection, err := txDao.FindCollectionByNameOrId(names.TableNameChangeRequests)
					if err != nil {
						return apis.NewApiError(500, "Failed to load collection", nil)
					}
					requestRecord = models.NewRecord(requestCollection)
				}
				requestForm := forms.NewRecordUpsert(app, requestRecord)
				requestForm.SetDao(txDao)
				err := requestForm.LoadData(map[string]any{
					"user":     userRecord.Id,
					"new_name": c.Get("new_name"),
				})
				if err != nil {
					return apis.NewApiError(500, "Failed to load data", nil)
				}
				if err = requestForm.Submit(); err != nil {
					return apis.NewBadRequestError("Invalid data", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
