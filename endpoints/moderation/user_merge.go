package moderation

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
//	@Tags			moderation
//	@Param			primaryId	query	string	true	"primary user that the data gets merged into"
//	@Param			secondaryId	query	string	true	"secondary user that gets deleted"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/merge [post]
func registerUserMergeEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/merge",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_merge"),
			middlewares.LoadParam(middlewares.LoadData{
				"primaryId":   middlewares.LoadString(true),
				"secondaryId": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := demonlist.MergeUsers(app.Dao(), c.Get("primaryId").(string), c.Get("secondaryId").(string))
			if err != nil {
				return util.NewErrorResponse(err, "Failed to merge")
			}
			return c.String(200, "Merged")
		},
	})
	return err
}
