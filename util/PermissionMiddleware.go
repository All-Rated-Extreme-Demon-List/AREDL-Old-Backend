package util

import (
	"AREDL/names"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
)

const KeyAffectedGroups = "affected_groups"

func existCommonElementInSlices[T comparable](left []T, right []T) bool {
	for _, a := range left {
		if list.ExistInSlice(a, right) {
			return true
		}
	}
	return false
}

// RequirePermissionGroup checks if the authenticated user is an admin or has access to the given action.
// Furthermore, it loads all roles the user can affect with the given action into the context using KeyAffectedGroups as key.
func RequirePermissionGroup(app core.App, action string) echo.MiddlewareFunc {
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

			role := record.GetString("role")

			permissions, err := app.Dao().FindRecordsByExpr(names.TablePermissions, dbx.HashExp{"action": action})
			if err != nil {
				return apis.NewForbiddenError("Permissions could not be loaded", nil)
			}
			foundRole := false
			allAffectedRoles := make([]string, 0)
			for _, permission := range permissions {
				if !list.ExistInSlice(role, permission.GetStringSlice("role")) {
					continue
				}
				foundRole = true
				allAffectedRoles = append(allAffectedRoles, permission.GetStringSlice("affected_roles")...)
			}

			if !foundRole {
				return apis.NewForbiddenError("You are not allowed to access this endpoint", nil)
			}
			c.Set(KeyAffectedGroups, allAffectedRoles)

			return next(c)
		}
	}
}
