package util

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterEndpoints(app core.App, toRegister ...func(e *echo.Echo, app core.App) error) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		for _, registerFunc := range toRegister {
			err := registerFunc(e.Router, app)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
