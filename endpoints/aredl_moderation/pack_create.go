package aredl_moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

// registerPackCreate godoc
//
//	@Summary		Create a new AREDL pack
//	@Description	Creates a new pack and updates all user points that now have the new pack.
//	@Description	Requires user permission: aredl.manage_packs
//	@Security		ApiKeyAuth[authorization]
//	@Tags			aredl_moderation
//	@Param			name			query	string		true	"display name"
//	@Param			color			query	string		true	"display color"
//	@Param			placement_order	query	int			false	"position of pack"
//	@Param			levels			query	[]string	true	"list of internal level ids. Pack has to have at least two levels"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/mod/pack/create [post]
func registerPackCreate(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/create",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "manage_packs"),
			middlewares.LoadParam(middlewares.LoadData{
				"packData": middlewares.LoadMap("", middlewares.LoadData{
					"name":            middlewares.LoadString(true),
					"color":           middlewares.LoadString(true),
					"placement_order": middlewares.LoadInt(false),
					"levels":          middlewares.LoadStringArray(true),
				}),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			packData := c.Get("packData").(map[string]interface{})
			err := demonlist.UpsertPack(app.Dao(), app, aredl, packData)
			return err
		},
	})
	return err
}
