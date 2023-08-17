package public

import (
	"AREDL/queryhelper"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func registerLeaderboardEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/leaderboard",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{
				"page":        util.AddDefault(1, util.LoadInt(false, validation.Min(1))),
				"per_page":    util.AddDefault(40, util.LoadInt(false, validation.Min(1), validation.Max(200))),
				"name_filter": util.LoadString(false),
			}),
		},
		Handler: func(c echo.Context) error {
			page := int64(c.Get("page").(int))
			perPage := int64(c.Get("per_page").(int))
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var result []queryhelper.LeaderboardEntry
				fields := []interface{}{
					"points", "rank",
					queryhelper.Extend{FieldName: "User", Fields: []interface{}{"id", "global_name", "country"}},
				}
				query, prefixTable, err := queryhelper.Build(txDao.DB(), result, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				if c.Get("name_filter") != nil {
					query.Where(dbx.Like(prefixTable["user."]+".global_name", c.Get("name_filter").(string)))
				}
				err = query.Offset((page - 1) * perPage).Limit(perPage).OrderBy(prefixTable[""] + ".rank").All(&result)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}
				return c.JSON(http.StatusOK, result)
			})
			return err
		},
	})
	return err
}
