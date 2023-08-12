package user

import (
	"AREDL/names"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
	"regexp"
)

func registerNameChangeRequestEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/name-change",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_request_name_change"),
			util.LoadParam(util.LoadData{
				"new_name": util.LoadString(true, validation.Match(regexp.MustCompile("^([a-zA-Z0-9 ._]{4,20}$)"))),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				sameAsOld := userRecord.GetString("global_name") == c.Get("new_name")
				requestRecord, _ := txDao.FindFirstRecordByData(names.TableNameChangeRequests, "user", userRecord.Id)
				if requestRecord == nil {
					requestCollection, err := txDao.FindCollectionByNameOrId(names.TableNameChangeRequests)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load collection")
					}
					requestRecord = models.NewRecord(requestCollection)
				} else if sameAsOld {
					if err := txDao.DeleteRecord(requestRecord); err != nil {
						return util.NewErrorResponse(err, "Failed to delete request")
					}
					return nil
				}
				if sameAsOld {
					return util.NewErrorResponse(nil, "New name is the same as the old one")
				}
				requestForm := forms.NewRecordUpsert(app, requestRecord)
				requestForm.SetDao(txDao)
				err := requestForm.LoadData(map[string]any{
					"user":     userRecord.Id,
					"new_name": c.Get("new_name"),
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load data")
				}
				if err = requestForm.Submit(); err != nil {
					return util.NewErrorResponse(err, "Invalid data")
				}
				return nil
			})
			return err
		},
	})
	return err
}
