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

type LevelUser struct {
	Id         string `db:"id" json:"id,omitempty"`
	GlobalName string `db:"global_name" json:"global_name,omitempty"`
}

type LevelRecord struct {
	Id          string    `db:"id" json:"id,omitempty"`
	VideoUrl    string    `db:"video_url" json:"video_url,omitempty"`
	Fps         int       `db:"fps" json:"fps,omitempty"`
	Mobile      bool      `db:"mobile" json:"mobile,omitempty"`
	SubmittedBy LevelUser `db:"submitted_by" json:"submitted_by,omitempty" extend:"submitted_by,users,id"`
}

type LevelPack struct {
	Id     string  `db:"id" json:"id,omitempty"`
	Name   string  `db:"name" json:"name,omitempty"`
	Color  string  `db:"color" json:"color,omitempty"`
	Points float64 `db:"points" json:"points,omitempty"`
}

type Level struct {
	Id            string         `db:"id" json:"id,omitempty"`
	Position      int            `db:"position" json:"position,omitempty"`
	Name          string         `db:"name" json:"name,omitempty"`
	Points        float64        `db:"points" json:"points,omitempty"`
	Legacy        bool           `db:"legacy" json:"legacy,omitempty"`
	LevelId       int            `db:"level_id" json:"level_id,omitempty"`
	TwoPlayer     bool           `db:"two_player" json:"two_player"`
	LevelPassword string         `db:"level_password" json:"level_password,omitempty"`
	CustomSong    string         `db:"custom_song" json:"custom_song,omitempty"`
	Publisher     LevelUser      `db:"publisher" json:"publisher,omitempty" extend:"publisher,users,id"`
	Verification  *LevelRecord   `json:"verification,omitempty" extend:"id,records,submitted_by"`
	Creators      *[]LevelUser   `json:"creators,omitempty"`
	Records       *[]LevelRecord `json:"records,omitempty"`
	Packs         *[]LevelPack   `json:"packs,omitempty"`
}

// registerLevelEndpoint godoc
//
//	@Summary		Level details
//	@Id				aredl.level
//	@Description	Detailed information on a level. I naddition optional data such as records, creators, verification and packs can be requested.
//	@Tags			aredl_public
//	@Param			id				query	string	false	"internal level id"
//	@Param			level_id		query	int		false	"gd level id"																							minimum(1)
//	@Param			two_player		query	bool	false	"if level was requested using level_id this specifies whether it should load the two player version"	default(false)
//	@Param			records			query	bool	false	"include records"																						default(false)
//	@Param			creators		query	bool	false	"include creators"																						default(false)
//	@Param			verification	query	bool	false	"include verification"																					default(false)
//	@Param			packs			query	bool	false	"include packs"																							default(false)
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	Level
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/level [get]
func registerLevelEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/level",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"id":           middlewares.LoadString(false),
				"level_id":     middlewares.LoadInt(false, validation.Min(1)),
				"records":      middlewares.AddDefault(false, middlewares.LoadBool(false)),
				"creators":     middlewares.AddDefault(false, middlewares.LoadBool(false)),
				"verification": middlewares.AddDefault(false, middlewares.LoadBool(false)),
				"two_player":   middlewares.AddDefault(false, middlewares.LoadBool(false)),
				"packs":        middlewares.AddDefault(false, middlewares.LoadBool(false)),
			}),
		},
		Handler: func(c echo.Context) error {
			hasLevelId := c.Get("level_id") != nil
			hasId := c.Get("id") != nil
			aredl := demonlist.Aredl()
			if !hasLevelId && !hasId {
				return util.NewErrorResponse(nil, "level_id or id required")
			}
			if hasLevelId && hasId {
				return util.NewErrorResponse(nil, "Can't query for level_id and id at the same time")
			}
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var level Level
				tables := map[string]string{
					"base":    aredl.LevelTableName,
					"records": aredl.RecordsTableName,
					"users":   names.TableUsers,
				}
				err := util.LoadFromDb(txDao.DB(), &level, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if hasLevelId {
						query.Where(dbx.HashExp{prefixResolver("level_id"): c.Get("level_id"), prefixResolver("two_player"): c.Get("two_player")})
					}
					if hasId {
						query.Where(dbx.HashExp{prefixResolver("id"): c.Get("id")})
					}
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}
				if c.Get("verification").(bool) {
					tables["base"] = tables["records"]
					err = util.LoadFromDb(txDao.DB(), &level.Verification, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
						query.Where(dbx.HashExp{prefixResolver("level"): level.Id, prefixResolver("placement_order"): 1})
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("creators").(bool) {
					tables["base"] = tables["users"]
					err = util.LoadFromDb(txDao.DB(), &level.Creators, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
						query.InnerJoin(aredl.CreatorTableName+" c", dbx.NewExp(fmt.Sprintf("%v=c.creator", prefixResolver("id"))))
						query.Where(dbx.HashExp{"c.level": level.Id})
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("records").(bool) {
					tables["base"] = tables["records"]
					err = util.LoadFromDb(txDao.DB(), &level.Records, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
						query.Where(dbx.HashExp{prefixResolver("level"): level.Id})
						query.AndWhere(dbx.NewExp(prefixResolver("placement_order") + " <> 1"))
						query.OrderBy(prefixResolver("placement_order"))
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("packs").(bool) {
					tables["base"] = aredl.Packs.PackTableName
					err = util.LoadFromDb(txDao.DB(), &level.Packs, tables, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
						query.Where(dbx.Exists(dbx.NewExp(fmt.Sprintf(
							`SELECT NULL FROM %v pl WHERE pl.level = {:levelId} AND pl.pack = %v`,
							demonlist.Aredl().Packs.PackLevelTableName,
							prefixResolver("id")), dbx.Params{"levelId": level.Id})))
						query.OrderBy(prefixResolver("placement_order"))
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				return c.JSON(http.StatusOK, level)
			})
			return err
		},
	})
	return err
}
