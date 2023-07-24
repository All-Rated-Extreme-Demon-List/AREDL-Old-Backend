package user

import (
	"AREDL/names"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
	"regexp"
)

func registerNameChangeRequestEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/name-change",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("member"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				// Limit username
				"new_name": {util.LoadString, true, nil, util.PackRules(validation.Match(regexp.MustCompile("^([a-zA-Z0-9 ._]{4,20}$)")))},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				sameAsOld := userRecord.GetString("global_name") == c.Get("new_name")
				requestRecord, _ := txDao.FindFirstRecordByData(names.TableNameChangeRequests, "user", userRecord.Id)
				if requestRecord == nil {
					requestCollection, err := txDao.FindCollectionByNameOrId(names.TableNameChangeRequests)
					if err != nil {
						return apis.NewApiError(500, "Failed to load collection", nil)
					}
					requestRecord = models.NewRecord(requestCollection)
				} else if sameAsOld {
					if err := txDao.DeleteRecord(requestRecord); err != nil {
						return apis.NewApiError(500, "Failed to delete request", nil)
					}
					return nil
				}
				if sameAsOld {
					return apis.NewBadRequestError("New name is the same as the old one", nil)
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
