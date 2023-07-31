package user

import (
	"AREDL/names"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"net/http"
)

func registerBasicListEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/list",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
		},
		Handler: func(c echo.Context) error {
			type ListEntry struct {
				ID       int `db:"level_id" json:"id"`
				Position int `db:"position" json:"position"`
			}
			var list []ListEntry
			err := app.Dao().DB().Select("level_id", "position").From(names.TableLevels).OrderBy("position").All(&list)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Failed to load list data", nil)
			}
			return c.JSON(http.StatusOK, list)
		},
	})
	return err
}
