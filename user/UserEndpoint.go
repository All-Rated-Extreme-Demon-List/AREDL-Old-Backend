package user

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/user"

// RegisterEndpoints registers all routes that are used by users
func RegisterEndpoints(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		err := registerSubmissionEndpoint(e.Router, app)
		if err != nil {
			return err
		}
		err = registerSubmissionWithdraw(e.Router, app)
		if err != nil {
			return err
		}
		return nil
	})
}
