package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

// registerUpdateListEndpoint godoc
//
//	@Summary		Update AREDL points and leaderboard
//	@Description	Updates all points. Should be used if other automatic updates didn't work.
//	@Description	Requires user permission: aredl.update_listpoints
//	@Security		ApiKeyAuth
//	@Tags			aredl
//	@Param			min_position	query	int	true	"min list position from what to update"
//	@Param			max_position	query	int	true	"max list position from what to update"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/leaderboard/refresh [post]
func registerUpdateListEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/leaderboard/refresh",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.RequirePermissionGroup(app, "aredl", "update_listpoints"),
			middlewares.LoadParam(middlewares.LoadData{
				"min_position": middlewares.LoadInt(true, validation.Min(1)),
				"max_position": middlewares.LoadInt(true, validation.Min(1)),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				err := demonlist.UpdateAllCompletedPacks(txDao, aredl)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update completed packs")
				}
				err = demonlist.UpdatePointTable(txDao, aredl)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update point table")
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
