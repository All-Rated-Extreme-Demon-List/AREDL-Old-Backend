package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
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
	Page  int `json:"page"`
	Pages int `json:"pages"`
}

// registerLeaderboardEndpoint godoc
//
//	@Summary		Aredl leaderboard
//	@Description	Gives leaderboard as a paged list ordered by rank. Players with zero list points are omitted
//	@Tags			aredl
//	@Param			page		query	int		false	"select page"	default(1)	minimum(1)
//	@Param			user_id		query	string	false	"get the page the given user is on instead of the given page, does not work with name filter active"
//	@Param			per_page	query	int		false	"number of results per page"	default(40)	minimum(1) maximum(200)
//	@Param			name_filter	query	string	false	"filters names to only contain the given substring"
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	Leaderboard
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/leaderboard [get]
func registerLeaderboardEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/leaderboard",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"page":        middlewares.AddDefault(1, middlewares.LoadInt(false, validation.Min(1))),
				"user_id":     middlewares.LoadString(false),
				"per_page":    middlewares.AddDefault(40, middlewares.LoadInt(false, validation.Min(1), validation.Max(200))),
				"name_filter": middlewares.LoadString(false),
			}),
		},
		Handler: func(c echo.Context) error {
			page := c.Get("page").(int)
			perPage := c.Get("per_page").(int)
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				if c.Get("user_id") != nil {
					if c.Get("name_filter") != nil {
						return util.NewErrorResponse(nil, "Cannot use name_filter with user_id")
					}
					userId := c.Get("user_id").(string)
					err := txDao.DB().Select(fmt.Sprintf("((rank - 1) / %v) + 1 AS rank", perPage)).From(aredl.LeaderboardTableName).Where(dbx.HashExp{"user": userId}).Row(&page)
					if util.IsNotNoResultError(err) {
						return util.NewErrorResponse(err, "Failed to request user page")
					}
				}
				var result Leaderboard
				result.Page = page
				tableNames := map[string]string{
					"base":  aredl.LeaderboardTableName,
					"users": names.TableUsers,
				}
				err := util.LoadFromDb(txDao.DB(), &result.List, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if c.Get("name_filter") != nil {
						query.Where(dbx.Like(prefixResolver("user.global_name"), c.Get("name_filter").(string)))
					}
					query.Offset(int64((page - 1) * perPage)).Limit(int64(perPage)).OrderBy(prefixResolver("rank"))
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

				c.Response().Header().Set("Cache-Control", "public, max-age=1800")

				return c.JSON(http.StatusOK, result)
			})
			return err
		},
	})
	return err
}
