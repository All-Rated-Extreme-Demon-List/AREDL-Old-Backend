package global

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/tools/list"
	"net/http"
)

// registerChangeRoleEndpoint godoc
//
//	@Summary		Change user role
//	@Description	Promote or demote a user
//	@Description	Requires user permission: user_change_role
//	@Description	Additionally the user needs to be able to affect the user with their permission and give the user the new role
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			id		path	string		true	"internal user id"
//	@Param			roles	query	[]string	true	"new roles"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/users/{id}/role [patch]
func registerChangeRoleEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPatch,
		Path:   "/users/:id/role",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_change_role"),
			middlewares.LoadParam(middlewares.LoadData{
				"id":    middlewares.LoadString(true),
				"roles": middlewares.LoadStringArray(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, err := txDao.FindRecordById(names.TableUsers, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find given user")
				}

				currentRoles, err := middlewares.GetUserRoles(txDao, userRecord.Id)
				if err != nil {
					return util.NewErrorResponse(err, "Could not load user roles")
				}
				if !middlewares.CanAffectRole(c, currentRoles) {
					return util.NewErrorResponse(nil, "Not allowed to change the rank of the given user")
				}

				newRoles := c.Get("roles").([]string)
				if !middlewares.CanAffectRole(c, newRoles) {
					return util.NewErrorResponse(nil, "Not allowed to change to given rank")
				}
				rolesToRemove := list.SubtractSlice(currentRoles, newRoles)
				rolesToAdd := list.SubtractSlice(newRoles, currentRoles)

				_, err = txDao.DB().Delete(names.TableRoles, dbx.And(dbx.In("role", list.ToInterfaceSlice(rolesToRemove)...), dbx.HashExp{"user": userRecord.Id})).Execute()
				if err != nil {
					return util.NewErrorResponse(err, "Failed to remove roles")
				}
				for _, role := range rolesToAdd {
					_, err = txDao.DB().Insert(names.TableRoles, dbx.Params{"role": role, "user": userRecord.Id}).Execute()
					if err != nil {
						return util.NewErrorResponse(err, "Failed to add role")
					}
				}
				return nil
			})
			return err
		},
	})
	return err
}
