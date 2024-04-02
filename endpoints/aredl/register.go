package aredl

import (
	"AREDL/util"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterEndpoints registers all routes that are under pathPrefix
func RegisterEndpoints(app core.App) {
	util.RegisterEndpoints(app, "/api/aredl",
		registerListEndpoint,
		registerLevelsEndpoint,
		registerLevelEndpoint,
		registerLevelHistoryEndpoint,
		registerLeaderboardEndpoint,
		registerUserEndpoint,
		registerPackEndpoint,
		registerNamesEndpoint,
		registerMeSubmissionList,
		registerSubmissionWithdrawEndpoint,
		registerSubmissionEndpoint,
		registerLevelPlaceEndpoint,
		registerLevelUpdateEndpoint,
		registerPackCreate,
		registerPackDelete,
		registerPackUpdate,
		registerRecordList,
		registerSubmissionList,
		registerSubmissionAcceptEndpoint,
		registerSubmissionRejectEndpoint,
		registerUpdateListEndpoint,
	)
}
