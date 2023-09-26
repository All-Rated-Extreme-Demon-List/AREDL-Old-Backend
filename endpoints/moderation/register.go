package moderation

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

const pathPrefix = "/api/mod"

// RegisterEndpoints registers all routes that are under pathPrefix
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app,
		registerCreatePlaceholderUser,
		registerNameChangeAcceptEndpoint,
		registerNameChangeRejectEndpoint,
		registerNameChangeListEndpoint,
		registerUserListEndpoint,
		registerBanAccountEndpoint,
		registerUnbanAccountEndpoint,
		registerChangeRoleEndpoint,
	)
}
