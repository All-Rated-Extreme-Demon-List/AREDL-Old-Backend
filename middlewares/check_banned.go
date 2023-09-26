package middlewares

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

// CheckBanned checks if the authenticated user is banned and rejects request if so
func CheckBanned() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			record, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if record != nil && record.GetBool("banned_from_list") {
				return apis.NewForbiddenError("Banned from demonlist", nil)
			}
			return next(c)
		}
	}
}
