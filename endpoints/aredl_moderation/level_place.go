package aredl_moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerLevelPlaceEndpoint godoc
//
//	@Summary		Place AREDL level
//	@Description	Places a new level into aredl. It automatically updates history and leaderboards
//	@Description	Requires user permission: aredl.manage_levels
//	@Security		ApiKeyAuth[authorization]
//	@Tags			aredl_moderation
//	@Param			creator_ids					query	[]string	true	"list of all creators using their internal user ids"
//	@Param			level_id					query	int			true	"gd level id"						minimum(1)
//	@Param			position					query	int			true	"position to place the level at"	minimum(1)
//	@Param			name						query	string		true	"displayed name of the level"
//	@Param			publisher					query	string		true	"publisher user id"
//	@Param			level_password				query	string		false	"gd level password"
//	@Param			custom_song					query	string		false	"reference to custom song"						default(100)
//	@Param			legacy						query	bool		false	"whether the level should be placed as legacy"	default(false)
//	@Param			verification_submitted_by	query	string		true	"user id of the verifier"
//	@Param			verification_video_url		query	string		true	"video url of the verification"	format(url)
//	@Param			verification_fps			query	int			true	"framerate of the verification"	minimum(30)	maximum(360)
//	@Param			verification_mobile			query	bool		true	"whether verification was done on mobile"
//	@Param			verification_raw_footage	query	string		false	"verification raw footage"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/mod/level/place [post]
func registerLevelPlaceEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/level/place",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "manage_levels"),
			middlewares.LoadParam(middlewares.LoadData{
				"creator_ids": middlewares.LoadStringArray(true),
				"levelData": middlewares.LoadMap("", middlewares.LoadData{
					"level_id":       middlewares.LoadInt(true, validation.Min(1)),
					"position":       middlewares.LoadInt(true, validation.Min(1)),
					"name":           middlewares.LoadString(true),
					"publisher":      middlewares.LoadString(true),
					"level_password": middlewares.LoadString(false),
					"custom_song":    middlewares.LoadString(false),
					"legacy":         middlewares.AddDefault(false, middlewares.LoadBool(false)),
				}),
				"verificationData": middlewares.LoadMap("verification_", middlewares.LoadData{
					"submitted_by": middlewares.LoadString(true),
					"video_url":    middlewares.LoadString(true, is.URL),
					"fps":          middlewares.LoadInt(true, validation.Min(30), validation.Max(360)),
					"mobile":       middlewares.LoadBool(true),
					"raw_footage":  middlewares.LoadString(false),
				}),
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return util.NewErrorResponse(nil, "User not found")
			}
			aredl := demonlist.Aredl()

			levelData := c.Get("levelData").(map[string]interface{})

			verificationData := c.Get("verificationData").(map[string]interface{})
			verificationData["percentage"] = 100

			creatorIds := c.Get("creator_ids").([]string)

			err := demonlist.PlaceLevel(app.Dao(), app, userRecord.Id, aredl, levelData, verificationData, creatorIds)

			return err
		},
	})
	return err
}
