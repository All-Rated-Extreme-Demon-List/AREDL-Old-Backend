package aredl_public

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

type Leaderboard struct {
	List []struct {
		Rank   int     `db:"rank" json:"rank,omitempty"`
		Points float64 `db:"points" json:"points,omitempty"`
		User   struct {
			Id         string `db:"id" json:"id,omitempty"`
			GlobalName string `db:"global_name" json:"global_name,omitempty"`
			Country    string `db:"country" json:"country,omitempty"`
		} `db:"user" json:"user,omitempty" extend:"user,users,id"`
	} `json:"list"`
	Pages int `json:"pages"`
}

// registerLeaderboardEndpoint godoc
//
//	@Summary		Aredl leaderboard
//	@Description	Gives leaderboard as a paged list ordered by rank. Players with zero list points are omitted
//	@Tags			aredl_public
//	@Param			page		query	int		false	"select page"					default(1)	minimum(1)
//	@Param			per_page	query	int		false	"number of results per page"	default(40)	minimum(1)	maximum(200)
//	@Param			name_filter	query	string	false	"filters names to only contain the given substring"
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	Leaderboard
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/leaderboard [get]
func registerLeaderboardEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/leaderboard",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"page":        middlewares.AddDefault(1, middlewares.LoadInt(false, validation.Min(1))),
				"per_page":    middlewares.AddDefault(40, middlewares.LoadInt(false, validation.Min(1), validation.Max(200))),
				"name_filter": middlewares.LoadString(false),
			}),
		},
		Handler: func(c echo.Context) error {
			page := int64(c.Get("page").(int))
			perPage := int64(c.Get("per_page").(int))
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var result Leaderboard
				tableNames := map[string]string{
					"base":  aredl.LeaderboardTableName,
					"users": names.TableUsers,
				}
				err := util.LoadFromDb(txDao.DB(), &result.List, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if c.Get("name_filter") != nil {
						query.Where(dbx.Like(prefixResolver("user.global_name"), c.Get("name_filter").(string)))
					}
					query.Offset((page - 1) * perPage).Limit(perPage).OrderBy(prefixResolver("rank"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}
				query := txDao.DB().
					Select(fmt.Sprintf("(count(*) / %v + 1)", perPage)).
					From(fmt.Sprintf("%v %v", aredl.LeaderboardTableName, "lb"))
				if c.Get("name_filter") != nil {
					query.InnerJoin(fmt.Sprintf("%v %v", names.TableUsers, "user"), dbx.NewExp("lb.user = user.id")).
						Where(dbx.Like("user.global_name", c.Get("name_filter").(string)))
				}
				err = query.Row(&result.Pages)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to calculate page count")
				}
				return c.JSON(http.StatusOK, result)
			})
			return err
		},
	})
	return err
}
