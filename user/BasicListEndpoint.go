package user

import (
	"AREDL/demonlist"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"net/http"
)

func registerBasicListEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/demonlist",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			type ListEntry struct {
				ID       int `db:"level_id" json:"id"`
				Position int `db:"position" json:"position"`
			}
			var list []ListEntry
			err := app.Dao().DB().
				Select("level_id", "position").
				From(aredl.LevelTableName).
				Where(dbx.HashExp{"legacy": false}).
				OrderBy("position").
				All(&list)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load demonlist data")
			}
			return c.JSON(http.StatusOK, list)
		},
	})
	return err
}
