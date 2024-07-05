package global

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

// registerUnbanAccountEndpoint godoc
//
//	@Summary		Unban user
//	@Description	Unbans a user
//	@Description	Requires user permission: user_ban
//	@Description	Additionally the user needs to be able to affect the user with their permission
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			id	path	string	true	"internal user id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/users/{id}/unban [post]
func registerUnbanAccountEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/users/:id/unban",
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
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, c.Get("id").(string))
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
			c.Response().Header().Set("Cache-Control", "no-store")
			return err
		},
	})
	return err
}
