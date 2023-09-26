package moderation

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

type UserEntry struct {
	Id         string `db:"id" json:"id"`
	GlobalName string `db:"global_name" json:"name"`
}

// registerUserListEndpoint godoc
//
//	@Summary		List users
//	@Description	Paged list of all users filtered by name. Userd to get user ids and select a user for other actions
//	@Description	Requires user permission: user_list
//	@Tags			moderation
//	@Param			page		query	int		false	"select page"					default(1)	minimum(1)
//	@Param			per_page	query	int		false	"number of results per page"	default(40)	minimum(1)	maximum(200)
//	@Param			name_filter	query	string	false	"filters names to only contain the given substring"
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	[]UserEntry
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/list [get]
func registerUserListEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/user/list",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_list"),
			middlewares.LoadParam(middlewares.LoadData{
				"page":        middlewares.AddDefault(1, middlewares.LoadInt(false, validation.Min(1))),
				"per_page":    middlewares.AddDefault(40, middlewares.LoadInt(false, validation.Min(1), validation.Max(200))),
				"name_filter": middlewares.LoadString(false),
			}),
		},
		Handler: func(c echo.Context) error {
			page := int64(c.Get("page").(int))
			perPage := int64(c.Get("per_page").(int))
			var result []UserEntry
			tableNames := map[string]string{
				"base": names.TableUsers,
			}
			err := util.LoadFromDb(app.Dao().DB(), &result, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
				if c.Get("name_filter") != nil {
					query.Where(dbx.Like(prefixResolver("global_name"), c.Get("name_filter").(string)))
				}
				query.Offset((page - 1) * perPage).Limit(perPage).OrderBy(prefixResolver("global_name"))
			})
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load request data")
			}
			return c.JSON(http.StatusOK, result)
		},
	})
	return err
}
