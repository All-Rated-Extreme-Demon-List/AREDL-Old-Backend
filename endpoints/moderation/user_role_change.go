package moderation

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerChangeRoleEndpoint godoc
//
//	@Summary		Change user role
//	@Description	Promote or demote a user
//	@Description	Requires user permission: user_change_role
//	@Description	Additionally the user needs to be able to affect the user with their permission and give the user the new role
//	@Tags			moderation
//	@Param			id		query	string	true	"internal user id"
//	@Param			role	query	string	true	"new role"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/role [post]
func registerChangeRoleEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/role",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_change_role"),
			middlewares.LoadParam(middlewares.LoadData{
				"id":   middlewares.LoadString(true),
				"role": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find given user")
				}
				if !middlewares.CanAffectRole(c, userRecord.GetString("role")) {
					return util.NewErrorResponse(nil, "Not allowed to change the rank of the given user")
				}
				if !middlewares.CanAffectRole(c, c.Get("role").(string)) {
					return util.NewErrorResponse(nil, "Not allowed to change to given rank")
				}
				userRecord.Set("role", c.Get("role").(string))
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update role")
				}
				return nil
			})
			return err
		},
	})
	return err
}
