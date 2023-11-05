package middlewares

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
)

const KeyAffectedRoles = "affected_groups"

// RequirePermissionGroup checks if the authenticated user is an admin or has access to the given action.
// Furthermore, it loads all roles the user can affect with the given action into the context using KeyAffectedGroups as key.
func RequirePermissionGroup(app core.App, listName string, action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			if admin != nil {
				return next(c)
			}
			if record == nil {
				return apis.NewForbiddenError("Authentication is required for this endpoint", nil)
			}
			type RoleData struct {
				Role string `db:"role"`
			}
			var roleData []RoleData
			err := app.Dao().DB().Select("role").From(names.TableRoles).Where(dbx.HashExp{"user": record.Id}).All(&roleData)
			if err != nil {
				return apis.NewForbiddenError("Failed to load role", nil)
			}
			roles := util.MapSlice(roleData, func(v RoleData) string { return v.Role })
			roles = append(roles, "default")

			if listName == "" {
				listName = "global"
			}
			permissions, err := app.Dao().FindRecordsByExpr(names.TablePermissions, dbx.HashExp{"action": action, "list": listName})
			if err != nil {
				return apis.NewForbiddenError("Permissions could not be loaded", nil)
			}
			foundRole := false
			allAffectedRoles := make([]string, 0)
			for _, permission := range permissions {
				if !util.AnyMatch(roles, permission.GetStringSlice("role")) {
					continue
				}
				foundRole = true
				allAffectedRoles = append(allAffectedRoles, permission.GetStringSlice("affected_roles")...)
			}

			if !foundRole {
				return apis.NewForbiddenError("You are not allowed to access this endpoint", nil)
			}
			c.Set(KeyAffectedRoles, allAffectedRoles)

			return next(c)
		}
	}
}

func CanAffectRole(c echo.Context, role string) bool {
	if c.Get(KeyAffectedRoles) != nil {
		affectedRoles := c.Get(KeyAffectedRoles).([]string)
		return list.ExistInSlice(role, affectedRoles)
	}
	return false
}
