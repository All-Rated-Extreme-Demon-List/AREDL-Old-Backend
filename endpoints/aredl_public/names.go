package aredl_public

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

type NameUser struct {
	Id         string `db:"id" json:"id,omitempty"`
	GlobalName string `db:"global_name" json:"global_name,omitempty"`
	Role       string `db:"role" json:"role,omitempty"`
	AredlPlus  bool   `db:"aredl_plus" json:"aredl_plus,omitempty"`
}

// registerNamesEndpoint godoc
//
//	@Summary		Important users
//	@Description	Gives a map of important users grouped by their role. This also includes aredl plus members
//	@Tags			aredl_public
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	map[string]NameUser
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/names [get]
func registerNamesEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/names",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{}),
		},
		Handler: func(c echo.Context) error {
			var users []NameUser
			tableNames := map[string]string{
				"base": names.TableUsers,
			}
			err := util.LoadFromDb(app.Dao().DB(), &users, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
				query.Where(dbx.Or(dbx.NotIn(prefixResolver("role"), "member"), dbx.HashExp{"aredl_plus": true}))
			})
			if err != nil {
				return util.NewErrorResponse(err, "failed to query data")
			}
			result := make(map[string][]NameUser)
			result["aredl_plus"] = make([]NameUser, 0)
			for _, user := range users {
				if user.AredlPlus {
					result["aredl_plus"] = append(result["aredl_plus"], user)
				}
				if user.Role == "member" {
					continue
				}
				list, exists := result[user.Role]
				if !exists {
					list = make([]NameUser, 0)
				}
				result[user.Role] = append(list, user)
			}
			return c.JSON(200, result)
		},
	})
	return err
}
