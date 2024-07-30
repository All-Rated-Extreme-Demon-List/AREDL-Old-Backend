package aredl

import (
	"AREDL/demonlist"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

type ListEntry struct {
	Id            string  `db:"id" json:"id,omitempty"`
	Position      int     `db:"position" json:"position,omitempty"`
	Name          string  `db:"name" json:"name,omitempty"`
	Points        float64 `db:"points" json:"points,omitempty"`
	LevelId       int     `db:"level_id" json:"level_id,omitempty"`
	TwoPlayer     bool    `db:"two_player" json:"two_player"`
	Legacy        bool    `db:"legacy" json:"legacy,omitempty"`
	Enjoyment     float64 `db:"enjoyment" json:"enjoyment,omitempty"`
	IsEdelPending bool    `db:"is_edel_pending" json:"is_edel_pending,omitempty"`
}

// registerLevelsEndpoint godoc
//
//	@Summary		Full simple list
//	@Description	Gives a list of every placed level ordered by position. To get more details on a level use /aredl/levels/:id
//	@Tags			aredl
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]ListEntry
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/levels [get]
func registerLevelsEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/levels",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
		},
		Handler: levelsHandler(app),
	})
	return err
}

// registerListEndpoint godoc
//
//	@Summary		(DEPRECATED) Full simple list
//	@Description	Use /aredl/levels instead
//	@Tags			aredl
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]ListEntry
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/list [get]
func registerListEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/list",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
		},
		Handler: levelsHandler(app),
	})
	return err
}

func levelsHandler(app core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		aredl := demonlist.Aredl()
		var list []ListEntry
		tableNames := map[string]string{
			"base": aredl.LevelTableName,
		}
		err := util.LoadFromDb(app.Dao().DB(), &list, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
			query.OrderBy(prefixResolver("position"))
		})
		if err != nil {
			return util.NewErrorResponse(err, "Failed to load demonlist data")
		}

		// Set Cache-Control header
		//c.Response().Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

		return c.JSON(http.StatusOK, list)
	}
}
