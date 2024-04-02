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

type NameChangeRequest struct {
	Id   string `db:"id" json:"id"`
	User struct {
		Id         string `db:"id" json:"id"`
		GlobalName string `db:"global_name" json:"name"`
	} `db:"user" json:"user" extend:"user,users,id"`
	NewName string `db:"new_name" json:"new_name"`
}

// registerNameChangeListEndpoint godoc
//
//	@Summary		List name change requests
//	@Description	Lists all open name change requests
//	@Description	Requires user permission: name_change_review
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	[]NameChangeRequest
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/name-change-requests [get]
func registerNameChangeListEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/name-change-requests",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "name_change_review"),
		},
		Handler: func(c echo.Context) error {
			var result []NameChangeRequest
			tableNames := map[string]string{
				"base":  names.TableNameChangeRequests,
				"users": names.TableUsers,
			}
			err := util.LoadFromDb(app.Dao().DB(), &result, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
				query.OrderBy(prefixResolver("created"))
			})
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load request data")
			}
			return c.JSON(http.StatusOK, result)
		},
	})
	return err
}
