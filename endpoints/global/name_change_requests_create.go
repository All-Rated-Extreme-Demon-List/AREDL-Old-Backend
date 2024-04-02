package global

import (
	"AREDL/middlewares"
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

// registerNameChangeRequestEndpoint godoc
//
//	@Summary		Name Change Request
//	@Description	Creates a name change request for the user. Needs to be reviewed by a moderator.
//	@Description	Requires user permission: user_request_name_change
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			new_name	query	string	true	"name to change to"
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/name-change-requests [put]
func registerNameChangeRequestEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPut,
		Path:   "/name-change-requests",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_request_name_change"),
			middlewares.LoadParam(middlewares.LoadData{
				"new_name": middlewares.LoadString(true, validation.Match(regexp.MustCompile("^([a-zA-Z0-9 ._]{4,20}$)"))),
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
