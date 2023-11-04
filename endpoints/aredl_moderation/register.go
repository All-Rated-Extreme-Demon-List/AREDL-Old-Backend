package aredl_moderation

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/aredl/mod"

// RegisterEndpoints registers all routes that are used by mods
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app,
		registerLevelPlaceEndpoint,
		registerLevelUpdateEndpoint,
		registerUpdateListEndpoint,
		registerSubmissionAcceptEndpoint,
		registerSubmissionRejectEndpoint,
		registerPackCreate,
		registerPackUpdate,
		registerPackDelete,
	)
}
