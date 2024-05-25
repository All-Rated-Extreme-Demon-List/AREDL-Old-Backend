package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/tools/types"
	"net/http"
)

type User struct {
	Id             string         `db:"id" json:"id,omitempty"`
	Created        types.DateTime `db:"created" json:"joined,omitempty"`
	GlobalName     string         `db:"global_name" json:"global_name,omitempty"`
	Description    string         `db:"description" json:"description,omitempty"`
	Country        string         `db:"country" json:"country,omitempty"`
	Badges         string         `db:"badges" json:"badges,omitempty"`
	AredlVerified  bool           `db:"aredl_verified" json:"aredl_verified,omitempty"`
	BannedFromList bool           `db:"banned_from_list" json:"banned_from_list,omitempty"`
	Placeholder    bool           `db:"placeholder" json:"placeholder,omitempty"`
	DiscordId      string         `db:"discord_id" json:"discord_id,omitempty"`
	AvatarUrl      string         `db:"avatar_url" json:"avatar_url,omitempty"`
	BannerColor    string         `db:"banner_color" json:"banner_color,omitempty"`
	LinkedYoutube  string         `db:"youtube_id" json:"linked_youtube,omitempty"`
	LinkedTwitch   string         `db:"twitch_id" json:"linked_twitch,omitempty"`
	LinkedTwitter  string         `db:"twitter_id" json:"linked_twitter,omitempty"`
	Roles          []string       `json:"roles"`
	Rank           *struct {
		Position int     `db:"rank" json:"position"`
		Points   float64 `db:"points" json:"points"`
	} `json:"rank,omitempty"`
	CompletedPacks []struct {
		Id     string  `db:"id" json:"id,omitempty"`
		Name   string  `db:"name" json:"name,omitempty"`
		Color  string  `db:"color" json:"color,omitempty"`
		Points float64 `db:"points" json:"points"`
	} `json:"packs,omitempty"`
	Records []struct {
		VideoUrl       string `db:"video_url" json:"video_url,omitempty"`
		Mobile         bool   `db:"mobile" json:"mobile,omitempty"`
		PlacementOrder int    `db:"placement_order" json:"placement_order"`
		Level          struct {
			Id        string  `db:"id" json:"id,omitempty"`
			Position  int     `db:"position" json:"position,omitempty"`
			Name      string  `db:"name" json:"name,omitempty"`
			TwoPlayer bool    `db:"two_player" json:"two_player"`
			Points    float64 `db:"points" json:"points"`
			Legacy    bool    `db:"legacy" json:"legacy"`
			LevelId   int     `db:"level_id" json:"level_id,omitempty"`
		} `db:"level" json:"level,omitempty" extend:"level,levels,id"`
	} `json:"records,omitempty"`
	CreatedLevels []struct {
		Id        string  `db:"id" json:"id,omitempty"`
		Position  int     `db:"position" json:"position,omitempty"`
		Name      string  `db:"name" json:"name,omitempty"`
		TwoPlayer bool    `db:"two_player" json:"two_player"`
		Points    float64 `db:"points" json:"points"`
		Legacy    bool    `db:"legacy" json:"legacy"`
		LevelId   int     `db:"level_id" json:"level_id,omitempty"`
		Created   struct {
			Creator string `db:"creator"`
		} `json:"-" extend:"id,creators,level" db:"creators"`
	} `json:"created_levels"`
	PublishedLevels []struct {
		Id        string  `db:"id" json:"id,omitempty"`
		Position  int     `db:"position" json:"position,omitempty"`
		Name      string  `db:"name" json:"name,omitempty"`
		TwoPlayer bool    `db:"two_player" json:"two_player"`
		Points    float64 `db:"points" json:"points"`
		Legacy    bool    `db:"legacy" json:"legacy"`
		LevelId   int     `db:"level_id" json:"level_id,omitempty"`
	} `json:"published_levels"`
}

// registerUserEndpoint godoc
//
//	@Summary		User info
//	@Description	Gives detailed information about a user
//	@Tags			aredl
//	@Param			id				path	string	true	"user id"
//	@Param			is_discord_id	query	bool	false	"if the provided id is a discord id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	User
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/profiles/{id} [get]
func registerUserEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/profiles/:id",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"id":            middlewares.LoadString(true),
				"is_discord_id": middlewares.AddDefault(false, middlewares.LoadBool(false)),
			}),
		},
		Handler: func(c echo.Context) error {
			aredl := demonlist.Aredl()
			userId := c.Get("id").(string)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var user User
				tableNames := map[string]string{
					"base": names.TableUsers,
				}
				err := util.LoadFromDb(txDao.DB(), &user, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					if c.Get("is_discord_id").(bool) {
						query.Where(dbx.HashExp{prefixResolver("discord_id"): userId})
					} else {
						query.Where(dbx.HashExp{prefixResolver("id"): userId})
					}
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user data")
				}
				tableNames["base"] = aredl.Packs.PackTableName
				err = util.LoadFromDb(txDao.DB(), &user.CompletedPacks, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.InnerJoin(demonlist.Aredl().Packs.CompletedPacksTableName+" cp", dbx.NewExp(prefixResolver("id")+" = cp.pack"))
					query.Where(dbx.HashExp{"cp.user": user.Id})
					query.OrderBy(prefixResolver("placement_order"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user packs")
				}
				tableNames["base"] = aredl.RecordsTableName
				tableNames["levels"] = aredl.LevelTableName
				err = util.LoadFromDb(txDao.DB(), &user.Records, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("submitted_by"): user.Id})
					query.OrderBy(prefixResolver("level.position"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user levels")
				}
				tableNames["base"] = aredl.LevelTableName
				tableNames["creators"] = aredl.CreatorTableName
				err = util.LoadFromDb(txDao.DB(), &user.CreatedLevels, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("creators.creator"): userId})
					query.OrderBy(prefixResolver("position"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load created levels")
				}
				err = util.LoadFromDb(txDao.DB(), &user.PublishedLevels, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{"publisher": userId})
					query.OrderBy(prefixResolver("position"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load published levels")
				}
				tableNames["base"] = aredl.LeaderboardTableName
				err = util.LoadFromDb(txDao.DB(), &user.Rank, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("user"): user.Id})
				})
				// ignore error if it's because the user is not on the leaderboard
				if util.IsNotNoResultError(err) {
					return util.NewErrorResponse(err, "Failed to load user rank")
				}
				type RoleData struct {
					Role string `db:"role"`
				}
				var roleData []RoleData
				err = app.Dao().DB().Select("role").From(names.TableRoles).Where(dbx.HashExp{"user": userId}).All(&roleData)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load roles")
				}
				user.Roles = util.MapSlice(roleData, func(v RoleData) string { return v.Role })
				return c.JSON(http.StatusOK, user)
			})
			return err
		},
	})
	return err
}
