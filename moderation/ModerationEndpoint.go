package moderation

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
)

const pathPrefix = "/api/mod"

// RegisterEndpoints registers all routes that are used for moderation
func RegisterEndpoints(e *echo.Echo, app *pocketbase.PocketBase) error {
	err := registerLevelEndpoints(e, app)

	return err
}
