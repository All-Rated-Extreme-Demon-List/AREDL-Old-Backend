package global

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterEndpoints registers all routes that are used by users
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app, "/api",
		registerPermissionsEndpoint,
		registerGetApiKeyEndpoint,
		registerMergeRequestEndpoint,
		registerMergeRequestListEndpoint,
		registerMergeRequestAcceptEndpoint,
		registerMergeRequestRejectEndpoint,
		registerNameChangeRequestEndpoint,
		registerNameChangeListEndpoint,
		registerNameChangeRejectEndpoint,
		registerNameChangeAcceptEndpoint,
		registerUserListEndpoint,
		registerBanAccountEndpoint,
		registerUserMergeEndpoint,
		registerChangeRoleEndpoint,
		registerCreatePlaceholderUser,
		registerUnbanAccountEndpoint,
	)
}
