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
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"level_id":                 {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"position":                 {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"name":                     {util.LoadString, true, nil, util.PackRules()},
				"creator_ids":              {util.LoadStringArray, true, nil, util.PackRules()},
				"verifier":                 {util.LoadString, true, nil, util.PackRules()},
				"publisher":                {util.LoadString, true, nil, util.PackRules()},
				"level_password":           {util.LoadString, false, nil, util.PackRules()},
				"custom_song":              {util.LoadString, false, nil, util.PackRules()},
				"qualifying_percent":       {util.LoadInt, false, 100, util.PackRules(validation.Min(1), validation.Max(100))},
				"legacy":                   {util.LoadBool, false, false, util.PackRules()},
				"verification_video":       {util.LoadString, true, nil, util.PackRules(is.URL)},
				"verification_fps":         {util.LoadInt, true, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"verification_device":      {util.LoadString, true, nil, util.PackRules(validation.In("pc", "mobile"))},
				"verification_raw_footage": {util.LoadString, false, nil, util.PackRules(is.URL)},
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
			}
			aredl := demonlist.Aredl()
			levelData := map[string]interface{}{
				"level_id":           c.Get("level_id"),
				"position":           c.Get("position"),
				"name":               c.Get("name"),
				"publisher":          c.Get("publisher"),
				"level_password":     c.Get("level_password"),
				"custom_song":        c.Get("custom_song"),
				"qualifying_percent": c.Get("qualifying_percent"),
				"legacy":             c.Get("legacy"),
			}
			verificationData := map[string]interface{}{
				"fps":          c.Get("verification_fps"),
				"video_url":    c.Get("verification_video"),
				"device":       c.Get("verification_device"),
				"percentage":   100,
				"submitted_by": c.Get("verifier"),
				"raw_footage":  c.Get("verification_raw_footage")}
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
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id":                 {util.LoadString, true, nil, util.PackRules()},
				"level_id":           {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"name":               {util.LoadString, false, nil, util.PackRules()},
				"verification":       {util.LoadString, false, nil, util.PackRules()},
				"creator_ids":        {util.LoadStringArray, false, nil, util.PackRules()},
				"publisher":          {util.LoadString, false, nil, util.PackRules()},
				"level_password":     {util.LoadString, false, nil, util.PackRules()},
				"custom_song":        {util.LoadString, false, nil, util.PackRules()},
				"legacy":             {util.LoadBool, false, nil, util.PackRules()},
				"position":           {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"qualifying_percent": {util.LoadInt, false, nil, util.PackRules(validation.Min(1), validation.Max(100))},
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
			}
			aredl := demonlist.Aredl()
			levelData := make(map[string]interface{})
			util.AddToMapIfNotNil(levelData, "level_id", c.Get("level_id"))
			util.AddToMapIfNotNil(levelData, "name", c.Get("name"))
			util.AddToMapIfNotNil(levelData, "publisher", c.Get("publisher"))
			util.AddToMapIfNotNil(levelData, "level_password", c.Get("level_password"))
			util.AddToMapIfNotNil(levelData, "custom_song", c.Get("custom_song"))
			util.AddToMapIfNotNil(levelData, "legacy", c.Get("legacy"))
			util.AddToMapIfNotNil(levelData, "qualifying_percent", c.Get("qualifying_percent"))
			util.AddToMapIfNotNil(levelData, "position", c.Get("position"))
			util.AddToMapIfNotNil(levelData, "verification", c.Get("verification"))
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
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"min_position": {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
				"max_position": {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
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
