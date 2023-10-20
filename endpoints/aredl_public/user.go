package aredl_public

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
	Role           string         `db:"role" json:"role,omitempty"`
	Description    string         `db:"description" json:"description,omitempty"`
	Country        string         `db:"country" json:"country,omitempty"`
	Badges         string         `db:"badges" json:"badges,omitempty"`
	AredlVerified  bool           `db:"aredl_verified" json:"aredl_verified,omitempty"`
	AredlPlus      bool           `db:"aredl_plus" json:"aredl_plus,omitempty"`
	BannedFromList bool           `db:"banned_from_list" json:"banned_from_list,omitempty"`
	Placeholder    bool           `db:"placeholder" json:"placeholder,omitempty"`
	DiscordId      string         `db:"discord_id" json:"discord_id,omitempty"`
	AvatarUrl      string         `db:"avatar_url" json:"avatar_url,omitempty"`
	BannerColor    string         `db:"banner_color" json:"banner_color,omitempty"`
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
		VideoUrl string `db:"video_url" json:"video_url,omitempty"`
		Fps      int    `db:"fps" json:"fps,omitempty"`
		Mobile   bool   `db:"mobile" json:"mobile,omitempty"`
		Level    struct {
			Id       string  `db:"id" json:"id,omitempty"`
			Position int     `db:"position" json:"position,omitempty"`
			Name     string  `db:"name" json:"name,omitempty"`
			Points   float64 `db:"points" json:"points"`
			Legacy   bool    `db:"legacy" json:"legacy"`
			LevelId  int     `db:"level_id" json:"level_id,omitempty"`
		} `db:"level" json:"level,omitempty" extend:"level,levels,id"`
	} `json:"records,omitempty"`
}

// registerUserEndpoint godoc
//
//	@Summary		User info
//	@Description	Gives detailed information about a user
//	@Tags			aredl_public
//	@Param			id	query	string	true	"user id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200	{object}	User
//	@Failure		400	{object}	util.ErrorResponse
//	@Router			/aredl/user [get]
func registerUserEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/user",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
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
					query.Where(dbx.HashExp{prefixResolver("id"): userId})
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
				tableNames["base"] = aredl.SubmissionTableName
				tableNames["levels"] = aredl.LevelTableName
				err = util.LoadFromDb(txDao.DB(), &user.Records, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("submitted_by"): user.Id, prefixResolver("status"): demonlist.StatusAccepted})
					query.OrderBy(prefixResolver("level.position"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user levels")
				}
				tableNames["base"] = aredl.LeaderboardTableName
				err = util.LoadFromDb(txDao.DB(), &user.Rank, tableNames, func(query *dbx.SelectQuery, prefixResolver util.PrefixResolver) {
					query.Where(dbx.HashExp{prefixResolver("user"): user.Id})
				})
				// ignore error if it's because the user is not on the leaderboard
				if util.IsNotNoResultError(err) {
					return util.NewErrorResponse(err, "Failed to load user rank")
				}
				return c.JSON(http.StatusOK, user)
			})
			return err
		},
	})
	return err
}
