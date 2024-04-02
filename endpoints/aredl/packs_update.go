package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

// registerPackUpdate godoc
//
//	@Summary		Update a AREDL pack
//	@Description	Updates a pack and updates all user points that now have or lost the new pack.
//	@Description	Requires user permission: aredl.manage_packs
//	@Security		ApiKeyAuth
//	@Tags			aredl
//	@Param			id				path	string		true	"internal pack id"
//	@Param			name			query	string		false	"display name"
//	@Param			color			query	string		false	"display color"
//	@Param			placement_order	query	int			false	"position of pack"
//	@Param			levels			query	[]string	false	"new list of internal level ids. Pack has to have at least two levels"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/packs/{id} [patch]
func registerPackUpdate(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPatch,
		Path:   "/packs/:id",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "manage_packs"),
			middlewares.LoadParam(middlewares.LoadData{
				"packData": middlewares.LoadMap("", middlewares.LoadData{
					"id":              middlewares.LoadString(true),
					"name":            middlewares.LoadString(false),
					"color":           middlewares.LoadString(false),
					"placement_order": middlewares.LoadInt(false),
					"levels":          middlewares.LoadStringArray(false),
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
