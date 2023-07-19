package moderation

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/mod"

// RegisterEndpoints registers all routes that are used for moderation
func RegisterEndpoints(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		err := registerLevelEndpoints(e.Router, app)
		if err != nil {
			return err
		}
		return nil
	})
}
