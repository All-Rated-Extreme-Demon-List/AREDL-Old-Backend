package moderation

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase"
)

const pathPrefix = "/api/mod"

// RegisterEndpoints registers all routes that are used for moderation
func RegisterEndpoints(app *pocketbase.PocketBase) {
	util.RegisterEndpoints(app,
		registerLevelPlaceEndpoint,
		registerLevelUpdateEndpoint,
		registerUpdateListEndpoint,
		registerMergeAcceptEndpoint,
		registerMergeRejectEndpoint,
		registerMergeDirectEndpoint,
		registerNameChangeAcceptEndpoint,
		registerNameChangeRejectEndpoint,
		registerBanAccountEndpoint,
		registerUnbanAccountEndpoint,
		registerSubmissionUpdateEndpoint,
		registerCreatePlaceholderUser,
		registerPackCreate,
		registerPackDelete,
		registerPackUpdate,
		registerChangeRoleEndpoint,
	)
}
