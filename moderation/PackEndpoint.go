package moderation

import (
	"AREDL/demonlist"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

func registerPackCreate(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/create",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.LoadParam(util.LoadData{
				"packData": util.LoadMap("", util.LoadData{
					"name":            util.LoadString(true),
					"color":           util.LoadString(true),
					"placement_order": util.LoadInt(false),
					"levels":          util.LoadStringArray(true),
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

func registerPackUpdate(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.LoadParam(util.LoadData{
				"packData": util.LoadMap("", util.LoadData{
					"id":              util.LoadString(true),
					"name":            util.LoadString(false),
					"color":           util.LoadString(false),
					"placement_order": util.LoadInt(false),
					"levels":          util.LoadStringArray(false),
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

func registerPackDelete(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/delete",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.LoadParam(util.LoadData{
				"record_id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := demonlist.DeletePack(app.Dao(), aredl, c.Get("record_id").(string))
			return err
		},
	})
	return err
}
