package aredl

import (
	"AREDL/demonlist"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

type Pack struct {
	Id     string  `db:"id" json:"id,omitempty"`
	Name   string  `db:"name" json:"name,omitempty"`
	Color  string  `db:"color" json:"color,omitempty"`
	Points float64 `db:"points" json:"points,omitempty"`
	Levels []struct {
		Id       string  `db:"id" json:"id,omitempty"`
		Position int     `db:"position" json:"position,omitempty"`
		Name     string  `db:"name" json:"name,omitempty"`
		Points   float64 `db:"points" json:"points,omitempty"`
		Legacy   bool    `db:"legacy" json:"legacy,omitempty"`
		LevelId  int     `db:"level_id" json:"level_id,omitempty"`
	} `json:"levels"`
}

// registerPackEndpoint godoc
//
//	@Summary		Aredl packs
//	@Description	Gives a list of all packs
//	@Tags			aredl
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]Pack
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/packs [get]
func registerPackEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/packs",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var result []Pack
				tableNames := map[string]string{
					"base": aredl.Packs.PackTableName,
				}
				err := util.LoadFromDb(txDao.DB(), &result, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.OrderBy(prefixResolver("placement_order"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user data")
				}

				output := result[:0]
				tableNames["base"] = aredl.LevelTableName
				for _, pack := range result {
					err = util.LoadFromDb(txDao.DB(), &pack.Levels, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
						query.InnerJoin(demonlist.Aredl().Packs.PackLevelTableName+" pl", dbx.NewExp(prefixResolver("id")+" = pl.level"))
						query.Where(dbx.HashExp{"pl.pack": pack.Id})
					})
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
