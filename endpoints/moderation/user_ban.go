package moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerBanAccountEndpoint godoc
//
//	@Summary		Ban user
//	@Description	Bans a user and removes them from the leaderboard
//	@Description	Requires user permission: user_ban
//	@Description	Additionally the user needs to be able to affect the user with their permission
//	@Tags			moderation
//	@Param			id	query	string	true	"internal user id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/ban [post]
func registerBanAccountEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/ban",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_ban"),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				authUserRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if authUserRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find user by discord id")
				}
				if !middlewares.CanAffectRole(c, userRecord.GetString("role")) {
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
