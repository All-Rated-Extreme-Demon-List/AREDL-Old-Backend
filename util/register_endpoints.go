package util

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterEndpoints(app core.App, pathPrefix string, toRegister ...func(e *echo.Group, app core.App) error) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		for _, registerFunc := range toRegister {
			err := registerFunc(e.Router.Group(pathPrefix), app)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
