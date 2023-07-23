package util

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterEndpoints(app *pocketbase.PocketBase, toRegister ...func(e *echo.Echo, app *pocketbase.PocketBase) error) {
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
