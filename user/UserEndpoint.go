package user

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase"
)

const pathPrefix = "/api/user"

// RegisterEndpoints registers all routes that are used by users
func RegisterEndpoints(app *pocketbase.PocketBase) {
	util.RegisterEndpoints(app,
		registerSubmissionEndpoint,
		registerSubmissionWithdrawEndpoint,
		registerMergeRequestEndpoint,
		registerNameChangeRequestEndpoint,
	)
}
