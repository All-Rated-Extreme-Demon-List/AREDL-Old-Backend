package global

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

type MergeRequest struct {
	Id      string `db:"id" json:"id"`
	Primary struct {
		Id         string `db:"id" json:"id"`
		GlobalName string `db:"global_name" json:"name"`
	} `db:"user" json:"primary" extend:"user,users,id"`
	Secondary struct {
		Id         string `db:"id" json:"id"`
		GlobalName string `db:"global_name" json:"name"`
	} `db:"to_merge" json:"secondary" extend:"to_merge,users,id"`
}

// registerMergeRequestListEndpoint godoc
//
//	@Summary		List name merge requests
//	@Description	Lists all open merge requests
//	@Description	Requires user permission: user_merge_review
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	[]NameChangeRequest
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/merge-requests [get]
func registerMergeRequestListEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/merge-requests",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_merge_review"),
		},
		Handler: func(c echo.Context) error {
			var result []MergeRequest
			tableNames := map[string]string{
				"base":  names.TableMergeRequests,
				"users": names.TableUsers,
			}
			err := util.LoadFromDb(app.Dao().DB(), &result, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
				query.OrderBy(prefixResolver("created"))
			})
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load request data")
			}
			c.Response().Header().Set("Cache-Control", "no-store")
			return c.JSON(http.StatusOK, result)
		},
	})
	return err
}
