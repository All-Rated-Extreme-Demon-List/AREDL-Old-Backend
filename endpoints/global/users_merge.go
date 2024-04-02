package global

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

// registerUserMergeEndpoint godoc
//
//	@Summary		Merge two users
//	@Description	Directly merges two users
//	@Description	Requires user permission: user_merge
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			primary_id		path	string	true	"primary user that the data gets merged into"
//	@Param			secondary_id	query	string	true	"secondary user that gets deleted"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/users/merge [post]
func registerUserMergeEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/users/merge",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_merge"),
			middlewares.LoadParam(middlewares.LoadData{
				"primary_id":   middlewares.LoadString(true),
				"secondary_id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := demonlist.MergeUsers(app.Dao(), c.Get("id").(string), c.Get("secondary_id").(string))
			if err != nil {
				return util.NewErrorResponse(err, "Failed to merge")
			}
			return c.String(200, "Merged")
		},
	})
	return err
}
