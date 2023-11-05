package user

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/user"

// RegisterEndpoints registers all routes that are used by users
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app,
		registerMergeRequestEndpoint,
		registerNameChangeRequestEndpoint,
		registerPermissionsEndpoint,
	)
}
