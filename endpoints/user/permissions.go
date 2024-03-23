package user

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

type PermissionData struct {
	AffectedRoles []string `json:"affected_roles,omitempty"`
}

// registerPermissionsEndpoint godoc
//
//	@Summary		Get a list of Permissions
//	@Description	Returns all the available permissions to the authenticated user, if there is no authenticaiton provided, the permissions will be empty
//	@Tags			user
//	@Schemes		http https
//	@Security		ApiKeyAuth[authorization]
//	@Produce		json
//	@Success		200 {object}	map[string]PermissionData
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/user/permissions [get]
func registerPermissionsEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/permissions",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadApiKey(app),
			middlewares.CheckBanned(),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return c.JSON(200, map[string]PermissionData{})
				}
				type RoleData struct {
					Role string `db:"role"`
				}
				result := map[string]PermissionData{}
				var roleData []RoleData
				err := app.Dao().DB().Select("role").From(names.TableRoles).Where(dbx.HashExp{"user": userRecord.Id}).All(&roleData)
				if err != nil {
					return apis.NewForbiddenError("Failed to load role", nil)
				}
				roles := util.MapSlice(roleData, func(v RoleData) string { return v.Role })
				roles = append(roles, "default")
				permissionRecords, err := txDao.FindRecordsByExpr(names.TablePermissions, dbx.OrLike("role", roles...))
				if err != nil {
					return apis.NewForbiddenError("Failed to load permissions", nil)
				}
				for _, permission := range permissionRecords {
					fullAction := fmt.Sprintf("%v.%v", permission.GetString("list"), permission.GetString("action"))
					if permData, ok := result[fullAction]; ok {
						result[fullAction] = PermissionData{
							AffectedRoles: append(permData.AffectedRoles, permission.GetStringSlice("affected_roles")...),
						}
					} else {
						result[fullAction] = PermissionData{
							AffectedRoles: permission.GetStringSlice("affected_roles"),
						}
					}
				}
				return c.JSON(200, result)
			})
			return err
		},
	})
	return err
}
