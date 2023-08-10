package moderation

import (
	"AREDL/demonlist"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"net/http"
)

func registerPackCreate(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/create",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"name":      {util.LoadString, true, nil, util.PackRules()},
				"colour":    {util.LoadString, true, nil, util.PackRules()},
				"placement": {util.LoadInt, false, nil, util.PackRules()},
				"levels":    {util.LoadStringArray, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			packData := map[string]interface{}{
				"name":   c.Get("name"),
				"colour": c.Get("colour"),
				"levels": c.Get("levels"),
			}
			util.AddToMapIfNotNil(packData, "placement_order", c.Get("placement"))
			err := demonlist.UpsertPack(app.Dao(), app, aredl, packData)
			return err
		},
	})
	return err
}

func registerPackUpdate(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"record_id": {util.LoadString, true, nil, util.PackRules()},
				"name":      {util.LoadString, false, nil, util.PackRules()},
				"colour":    {util.LoadString, false, nil, util.PackRules()},
				"placement": {util.LoadInt, false, nil, util.PackRules()},
				"levels":    {util.LoadStringArray, false, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			packData := map[string]interface{}{
				"id": c.Get("record_id"),
			}
			util.AddToMapIfNotNil(packData, "name", c.Get("name"))
			util.AddToMapIfNotNil(packData, "colour", c.Get("colour"))
			util.AddToMapIfNotNil(packData, "placement_order", c.Get("placement"))
			util.AddToMapIfNotNil(packData, "levels", c.Get("levels"))
			err := demonlist.UpsertPack(app.Dao(), app, aredl, packData)
			return err
		},
	})
	return err
}

func registerPackDelete(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/delete",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"record_id": {util.LoadString, true, nil, util.PackRules()},
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
