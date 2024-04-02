package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/tools/types"
	"net/http"
)

type HistoryEntry struct {
	Level       struct{}       `db:"level" json:"-" extend:"level,levels,id"`
	Action      string         `db:"action" json:"action,omitempty" enums:"placed,placedAbove,movedUp,movedDown,movedPastUp,movedPastDown,movedToLegacy,movedFromLegacy"`
	NewPosition int            `db:"new_position" json:"new_position,omitempty"`
	Created     types.DateTime `db:"created" json:"timestamp,omitempty"`
	Cause       struct {
		Id      string `db:"id" json:"id,omitempty"`
		Name    string `db:"name" json:"name,omitempty"`
		LevelId int    `db:"level_id" json:"level_id,omitempty"`
	} `db:"cause" json:"cause,omitempty" extend:"cause,levels,id"`
	//ActionBy *struct {
	//	Id         *string `db:"id" json:"id,omitempty"`
	//	GlobalName *string `db:"global_name" json:"global_name,omitempty"`
	//} `db:"action_by" json:"action_by,omitempty" extend:"action_by,users,id"`
}

// registerLevelHistoryEndpoint godoc
//
//	@Summary		History of a level
//	@Description	Lists the placement, move & legacy history of a level by either using its internal or gd id. Possible actions: placed, placedAbove, movedUp, movedDown, movedPastUp, movedPastDown, movedToLegacy, movedFromLegacy
//	@Tags			aredl
//	@Param			id			path	string	true	"internal level id or gd level id"
//	@Param			level_id	query	int		false	"gd level id"	minimum(1)
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	[]HistoryEntry
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/levels/{id}/history [get]
func registerLevelHistoryEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/levels/:id/history",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"id":       middlewares.LoadString(false),
				"level_id": middlewares.LoadInt(false, validation.Min(1)),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			id := c.Get("id").(string)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var result []HistoryEntry
				tables := map[string]string{
					"base":   aredl.HistoryTableName,
					"levels": aredl.LevelTableName,
					"users":  names.TableUsers,
				}
				err := util.LoadFromDb(txDao.DB(), &result, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if util.IsGDId(id) {
						query.Where(dbx.HashExp{prefixResolver("level.level_id"): id})
					} else {
						query.Where(dbx.HashExp{prefixResolver("level.id"): id})
					}
					query.OrderBy(prefixResolver("created"))
				})
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
