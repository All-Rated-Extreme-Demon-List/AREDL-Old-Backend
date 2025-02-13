package global

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

// registerNameChangeAcceptEndpoint godoc
//
//	@Summary		Accept name change request
//	@Description	Accepts a name change request from a user
//	@Description	Requires user permission: name_change_review
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			id	path	string	true	"request id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/name-change-requests/{id}/accept [post]
func registerNameChangeAcceptEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/name-change-requests/:id/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "name_change_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableNameChangeRequests, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Request not found")
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, requestRecord.GetString("user"))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find user in request")
				}
				userRecord.Set("global_name", requestRecord.GetString("new_name"))
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to change username")
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete request")
				}
				return nil
			})
			c.Response().Header().Set("Cache-Control", "no-store")
			return err
		},
	})
	return err
}
