package public

import (
	"AREDL/demonlist"
	"AREDL/queryhelper"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func registerPackEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/packs",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				type ResultData struct {
					queryhelper.Pack
					Levels []queryhelper.AredlLevel
				}
				var result []ResultData
				fields := []interface{}{
					"id", "name", "color", "placement_order", "points",
				}
				query, _, err := queryhelper.Build(txDao.DB(), result, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				err = query.All(&result)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user data")
				}
				output := result[:0]
				for _, pack := range result {
					fields = []interface{}{
						"id", "level_id", "name", "position", "points", "legacy",
					}
					query, prefixTable, err := queryhelper.Build(txDao.DB(), pack.Levels, fields)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to build query")
					}
					query.InnerJoin(demonlist.Aredl().Packs.PackLevelTableName+" pl", dbx.NewExp(prefixTable[""]+".id = pl.level"))
					query.Where(dbx.HashExp{"pl.pack": pack.Id})
					err = query.All(&pack.Levels)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load user data")
					}
					output = append(output, pack)
				}
				return c.JSON(http.StatusOK, output)
			})
			return err
		},
	})
	return err
}
