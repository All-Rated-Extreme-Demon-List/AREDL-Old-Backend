package moderation

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerBanAccountEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/ban",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listHelper", "listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"discord_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				userRecord, err := txDao.FindFirstRecordByData(names.TableUsers, "discord_id", c.Get("discord_id"))
				if err != nil {
					return apis.NewBadRequestError("Could not fin user by discord id", nil)
				}
				if util.IsPrivileged(authUserRecord.GetStringSlice("permissions"), userRecord.GetStringSlice("permissions")) {
					return apis.NewBadRequestError("You are not privileged to ban that user", nil)
				}
				userRecord.Set("banned_from_list", true)
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to ban user", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerUnbanAccountEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/unban",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listHelper", "listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"discord_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return apis.NewApiError(500, "User not found", nil)
				}
				userRecord, err := txDao.FindFirstRecordByData(names.TableUsers, "discord_id", c.Get("discord_id"))
				if err != nil {
					return apis.NewBadRequestError("Could not fin user by discord id", nil)
				}
				if util.IsPrivileged(authUserRecord.GetStringSlice("permissions"), userRecord.GetStringSlice("permissions")) {
					return apis.NewBadRequestError("You are not privileged to unban that user", nil)
				}
				userRecord.Set("banned_from_list", false)
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to unban user", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
