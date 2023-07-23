package util

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

func existCommonElementInSlices[T comparable](left []T, right []T) bool {
	for _, a := range left {
		for _, b := range right {
			if a == b {
				return true
			}
		}
	}
	return false
}

// RequirePermission checks if the authenticated user is an admin or has at least one of the given permissions
func RequirePermission(permissions ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)

			if admin != nil {
				return next(c)
			}

			if record == nil || (len(permissions) > 0 && !existCommonElementInSlices(permissions, record.GetStringSlice("permissions"))) {
				return apis.NewForbiddenError("The authorized user is not allowed to perform this action", nil)
			}

			return next(c)
		}
	}
}
