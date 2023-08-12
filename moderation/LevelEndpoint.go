package moderation

import (
	"AREDL/demonlist"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerLevelPlaceEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/level/place",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_levels"),
			util.LoadParam(util.LoadData{
				"creator_ids": util.LoadStringArray(true),
				"levelData": util.LoadMap("", util.LoadData{
					"level_id":           util.LoadInt(true, validation.Min(1)),
					"position":           util.LoadInt(true, validation.Min(1)),
					"name":               util.LoadString(true),
					"publisher":          util.LoadString(true),
					"level_password":     util.LoadString(false),
					"custom_song":        util.LoadString(false),
					"qualifying_percent": util.AddDefault(100, util.LoadInt(false, validation.Min(1), validation.Max(100))),
					"legacy":             util.AddDefault(false, util.LoadBool(false)),
				}),
				"verificationData": util.LoadMap("verification_", util.LoadData{
					"submitted_by": util.LoadString(true),
					"video_url":    util.LoadString(true, is.URL),
					"fps":          util.LoadInt(true, validation.Min(30), validation.Max(360)),
					"mobile":       util.LoadBool(true),
					"raw_footage":  util.LoadString(false),
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

func registerLevelUpdateEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/level/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermissionGroup(app, "manage_levels"),
			util.LoadParam(util.LoadData{
				"id":          util.LoadString(true),
				"creator_ids": util.LoadStringArray(false),
				"levelData": util.LoadMap("", util.LoadData{
					"level_id":           util.LoadInt(false),
					"name":               util.LoadString(false),
					"verification":       util.LoadString(false),
					"publisher":          util.LoadString(false),
					"level_password":     util.LoadString(false),
					"custom_song":        util.LoadString(false),
					"legacy":             util.LoadBool(false),
					"position":           util.LoadInt(false, validation.Min(1)),
					"qualifying_percent": util.LoadInt(false, validation.Min(1), validation.Max(100)),
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

func registerUpdateListEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/level/update-demonlist",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermissionGroup(app, "update_listpoints"),
			util.LoadParam(util.LoadData{
				"min_position": util.LoadInt(true, validation.Min(1)),
				"max_position": util.LoadInt(true, validation.Min(1)),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				err := demonlist.UpdateAllCompletedPacks(txDao, aredl)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update completed packs")
				}
				err = demonlist.UpdateLevelListPointsByPositionRange(txDao, aredl, c.Get("min_position").(int), c.Get("max_position").(int))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update list points")
				}
				return nil
			})
			return err
		},
	})
	return err
}
