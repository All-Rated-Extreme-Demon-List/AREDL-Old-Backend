package moderation

import (
	"AREDL/demonlist"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerNameChangeAcceptEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/name-change/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "name_change_review"),
			util.LoadParam(util.LoadData{
				"request_id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableNameChangeRequests, c.Get("request_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Request not found")
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, requestRecord.GetString("user"))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find user in request")
				}
				userRecord.Set("global_name", requestRecord.GetString("new_name"))
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to change username")
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete request")
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerNameChangeRejectEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/name-change/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "name_change_review"),
			util.LoadParam(util.LoadData{
				"request_id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableNameChangeRequests, c.Get("request_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Request not found")
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete request")
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerCreatePlaceholderUser(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/create-placeholder",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "create_placeholder"),
			util.LoadParam(util.LoadData{
				"username": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := txDao.FindFirstRecordByData(names.TableUsers, "global_name", c.Get("username").(string))
				if userRecord != nil && userRecord.GetBool("placeholder") {
					return util.NewErrorResponse(nil, "Placeholder user with that name already exists")
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user collection")
				}
				_, err = util.CreatePlaceholderUser(app, txDao, userCollection, c.Get("username").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to create placeholder user")
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerBanAccountEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/ban",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_bans"),
			util.LoadParam(util.LoadData{
				"discord_id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				userRecord, err := txDao.FindFirstRecordByData(names.TableUsers, "discord_id", c.Get("discord_id"))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find user by discord id")
				}
				if !util.CanAffectRole(c, userRecord.GetString("role")) {
					return util.NewErrorResponse(err, "Cannot perform action on given user")
				}
				userRecord.Set("banned_from_list", true)
				userRecord.Set("role", "member")
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to ban user")
				}
				aredl := demonlist.Aredl()
				err = demonlist.UpdateLeaderboardByUserIds(txDao, aredl, []interface{}{userRecord.Id})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update leaderboard")
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerUnbanAccountEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/unban",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_bans"),
			util.LoadParam(util.LoadData{
				"discord_id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				userRecord, err := txDao.FindFirstRecordByData(names.TableUsers, "discord_id", c.Get("discord_id"))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to unban user")
				}
				userRecord.Set("banned_from_list", false)
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to unban user")
				}
				aredl := demonlist.Aredl()
				err = demonlist.UpdateLeaderboardByUserIds(txDao, aredl, []interface{}{userRecord.Id})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update leaderboard")
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerChangeRoleEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/role",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "change_role"),
			util.LoadParam(util.LoadData{
				"user_id": util.LoadString(true),
				"role":    util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, c.Get("user_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find given user")
				}
				if !util.CanAffectRole(c, userRecord.GetString("role")) {
					return util.NewErrorResponse(nil, "Not allowed to change the rank of the given user")
				}
				if !util.CanAffectRole(c, c.Get("role").(string)) {
					return util.NewErrorResponse(nil, "Not allowed to change to given rank")
				}
				userRecord.Set("role", c.Get("role").(string))
				err = txDao.SaveRecord(userRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update role")
				}
				return nil
			})
			return err
		},
	})
	return err
}
