package public

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api"

// RegisterEndpoints registers all routes that are used by users
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app,
		registerBasicListEndpoint,
		registerLevelHistoryEndpoint,
		registerLeaderboardEndpoint,
		registerUserEndpoint,
		registerPackEndpoint,
	)
}
