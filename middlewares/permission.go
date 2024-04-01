package middlewares

import (
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

const KeyAffectedRoles = "affected_groups"

type PermissionData struct {
	AffectedRoles []string `json:"affected_roles,omitempty"`
}

// RequirePermissionGroup checks if the authenticated user is an admin or has access to the given action.
// Furthermore, it loads all roles the user can affect with the given action into the context using KeyAffectedGroups as key.
func RequirePermissionGroup(app core.App, listName string, action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			apiKey := c.Request().Header.Get("api-key")

			if record == nil && apiKey == "" && admin == nil {
				return apis.NewForbiddenError("Authentication is required for this endpoint", nil)
			}

			if apiKey != "" {
				users, err := app.Dao().FindRecordsByExpr(names.TableUsers, dbx.HashExp{"api_key": apiKey})
				if err != nil || len(users) != 1 {
					return apis.NewForbiddenError("Api Key not found", nil)
				}
				record = users[0]
				c.Set(apis.ContextAuthRecordKey, record)
			}
			hasPermission, permissionData, err := GetPermission(app.Dao(), record.Id, listName, action)
			if err != nil {
				return apis.NewForbiddenError("Permissions could not be loaded", nil)
			}
			if !hasPermission {
				return apis.NewForbiddenError("You are not allowed to access this endpoint", nil)
			}
			c.Set(KeyAffectedRoles, permissionData.AffectedRoles)

			return next(c)
		}
	}
}

func LoadApiKey(app core.App) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("api-key")

			if apiKey != "" {
				users, err := app.Dao().FindRecordsByExpr(names.TableUsers, dbx.HashExp{"api_key": apiKey})
				if err != nil || len(users) != 1 {
					return apis.NewForbiddenError("Api Key not found", nil)
				}
				record := users[0]
				c.Set(apis.ContextAuthRecordKey, record)
			}
			return next(c)
		}
	}
}

func GetUserRoles(dao *daos.Dao, userId string) ([]string, error) {
	type RoleData struct {
		Role string `db:"role"`
	}
	var roleData []RoleData
	err := dao.DB().Select("role").From(names.TableRoles).Where(dbx.HashExp{"user": userId}).All(&roleData)
	if err != nil {
		return nil, err
	}
	roles := util.MapSlice(roleData, func(v RoleData) string { return v.Role })
	roles = append(roles, "default")
	return roles, nil
}

func GetPermission(dao *daos.Dao, userId string, list string, action string) (bool, PermissionData, error) {
	roles, err := GetUserRoles(dao, userId)
	if err != nil {
		return false, PermissionData{}, err
	}
	if list == "" {
		list = "global"
	}
	permissionRecords, err := dao.FindRecordsByExpr(names.TablePermissions, dbx.And(dbx.OrLike("role", roles...)), dbx.HashExp{"list": list, "action": action})
	if err != nil {
		return false, PermissionData{}, err
	}
	permissionDataMap := mergePermissions(permissionRecords)
	key := fmt.Sprintf("%v.%v", list, action)
	permissionData, exist := permissionDataMap[key]
	return exist, permissionData, nil
}

func GetAllPermissions(dao *daos.Dao, userId string) (map[string]PermissionData, error) {
	roles, err := GetUserRoles(dao, userId)
	if err != nil {
		return nil, err
	}
	permissionRecords, err := dao.FindRecordsByExpr(names.TablePermissions, dbx.OrLike("role", roles...))
	if err != nil {
		return nil, err
	}
	return mergePermissions(permissionRecords), nil
}

func mergePermissions(permissionRecords []*models.Record) map[string]PermissionData {
	result := map[string]PermissionData{}
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
	return result
}

func CanAffectUser(c echo.Context, dao *daos.Dao, userId string) (bool, error) {
	roles, err := GetUserRoles(dao, userId)
	if err != nil {
		return false, err
	}
	return CanAffectRole(c, roles), nil
}

func CanAffectRole(c echo.Context, roles []string) bool {
	if c.Get(KeyAffectedRoles) != nil {
		affectedRoles := c.Get(KeyAffectedRoles).([]string)
		return util.IsSubset(affectedRoles, roles)
	}
	return false
}
