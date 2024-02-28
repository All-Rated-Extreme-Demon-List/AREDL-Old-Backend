package aredl_moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerLevelUpdateEndpoint godoc
//
//	@Summary		Update AREDL level
//	@Description	Updates level data. It automatically updates history and leaderboards.
//	@Description	Requires user permission: aredl.manage_levels
//	@Security		ApiKeyAuth[authorization]
//	@Tags			aredl_moderation
//	@Param			id				query	string		true	"internal level id"
//	@Param			creator_ids		query	[]string	false	"list of all creators using their internal user ids"
//	@Param			level_id		query	int			false	"gd level id"												minimum(1)
//	@Param			position		query	int			false	"position to move to if different form current position"	minimum(1)
//	@Param			name			query	string		false	"displayed name of the level"
//	@Param			publisher		query	string		false	"publisher user id"
//	@Param			level_password	query	string		false	"gd level password"
//	@Param			custom_song		query	string		false	"reference to custom song"
//	@Param			legacy			query	bool		false	"whether the level should be placed as legacy"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/mod/level/update [post]
func registerLevelUpdateEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/level/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.RequirePermissionGroup(app, "aredl", "manage_levels"),
			middlewares.LoadParam(middlewares.LoadData{
				"id":          middlewares.LoadString(true),
				"creator_ids": middlewares.LoadStringArray(false),
				"levelData": middlewares.LoadMap("", middlewares.LoadData{
					"level_id":       middlewares.LoadInt(false),
					"name":           middlewares.LoadString(false),
					"verification":   middlewares.LoadString(false),
					"publisher":      middlewares.LoadString(false),
					"level_password": middlewares.LoadString(false),
					"custom_song":    middlewares.LoadString(false),
					"legacy":         middlewares.LoadBool(false),
					"position":       middlewares.LoadInt(false, validation.Min(1)),
				}),
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
			}
			aredl := demonlist.Aredl()
			levelData := c.Get("levelData").(map[string]interface{})
			return demonlist.UpdateLevel(app.Dao(), app, c.Get("id").(string), userRecord.Id, aredl, levelData, c.Get("creator_ids"))
		},
	})
	return err
}
