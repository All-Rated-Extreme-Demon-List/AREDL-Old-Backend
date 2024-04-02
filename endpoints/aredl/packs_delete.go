package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

// registerPackDelete godoc
//
//	@Summary		Delete an AREDL pack
//	@Description	Deletes a pack and updates all user points that now have the new pack.
//	@Description	Requires user permission: aredl.manage_packs
//	@Security		ApiKeyAuth[authorization]
//	@Tags			aredl
//	@Param			id	path	string	true	"internal pack id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/packs/{id} [delete]
func registerPackDelete(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodDelete,
		Path:   "/packs/:id",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "manage_packs"),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := demonlist.DeletePack(app.Dao(), aredl, c.Get("id").(string))
			return err
		},
	})
	return err
}
