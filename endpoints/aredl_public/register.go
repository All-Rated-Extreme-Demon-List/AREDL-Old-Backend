package aredl_public

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/aredl"

// RegisterEndpoints registers all routes that are under pathPrefix
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app,
		registerListEndpoint,
		registerLevelEndpoint,
		registerLevelHistoryEndpoint,
		registerLeaderboardEndpoint,
		registerUserEndpoint,
		registerPackEndpoint,
		registerNamesEndpoint,
	)
}
