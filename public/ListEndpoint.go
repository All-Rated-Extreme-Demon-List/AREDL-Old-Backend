package public

import (
	"AREDL/demonlist"
	"AREDL/queryhelper"
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

func registerBasicListEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/list",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{
				"level_id":            util.LoadInt(false, validation.Min(1)),
				"id":                  util.LoadString(false),
				"includeRecords":      util.AddDefault(false, util.LoadBool(false)),
				"includeCreators":     util.AddDefault(false, util.LoadBool(false)),
				"includeVerification": util.AddDefault(false, util.LoadBool(false)),
				"includePacks":        util.AddDefault(false, util.LoadBool(false)),
			}),
		},
		Handler: func(c echo.Context) error {
			hasLevelId := c.Get("level_id") != nil
			hasId := c.Get("id") != nil
			if !hasLevelId && !hasId {
				// return entire list
				var list []queryhelper.AredlLevel
				fields := []interface{}{"id", "position", "name", "level_id", "legacy"}
				err := queryhelper.Build(app.Dao().DB(), &list, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
					query.OrderBy(prefixResolver("position"))
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}
				return c.JSON(http.StatusOK, list)
			}
			if hasLevelId && hasId {
				return util.NewErrorResponse(nil, "Can't query for level_id and id at the same time")
			}
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				type ResultData struct {
					queryhelper.AredlLevel
					Verification *queryhelper.AredlSubmission   `json:"verification,omitempty"`
					Creators     *[]queryhelper.User            `json:"creators,omitempty"`
					Records      *[]queryhelper.AredlSubmission `json:"records,omitempty"`
					Pack         *[]queryhelper.Pack            `json:"packs,omitempty"`
				}
				var result ResultData
				fields := []interface{}{
					"id", "position", "name", "points", "legacy", "level_id", "level_password", "custom_song",
					queryhelper.Extend{FieldName: "Publisher", Fields: []interface{}{"id", "global_name"}},
				}
				err := queryhelper.Build(txDao.DB(), &result.AredlLevel, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
					if hasLevelId {
						query.Where(dbx.HashExp{prefixResolver("level_id"): c.Get("level_id")})
					}
					if hasId {
						query.Where(dbx.HashExp{prefixResolver("id"): c.Get("id")})
					}
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}

				if c.Get("includeVerification").(bool) {
					fields = []interface{}{"id", "video_url", "fps", "mobile", queryhelper.Extend{
						FieldName: "SubmittedBy", Fields: []interface{}{"id", "global_name"},
					}}
					err = queryhelper.Build(txDao.DB(), &result.Verification, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
						query.Where(dbx.HashExp{prefixResolver("level"): result.Id, prefixResolver("placement_order"): 1})
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("includeCreators").(bool) {
					fields = []interface{}{"id", "global_name"}
					err = queryhelper.Build(txDao.DB(), &result.Creators, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
						query.InnerJoin(demonlist.Aredl().CreatorTableName+" c", dbx.NewExp(fmt.Sprintf("%v=c.creator", prefixResolver("id"))))
						query.Where(dbx.HashExp{"c.level": result.Id})
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("includeRecords").(bool) {
					fields = []interface{}{"id", "video_url", "fps", "mobile", queryhelper.Extend{
						FieldName: "SubmittedBy", Fields: []interface{}{"id", "global_name"},
					}}
					err = queryhelper.Build(txDao.DB(), &result.Records, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
						query.Where(dbx.HashExp{prefixResolver("level"): result.Id, prefixResolver("status"): demonlist.StatusAccepted})
						query.AndWhere(dbx.NewExp(prefixResolver("placement_order") + " <> 1"))
						query.OrderBy(prefixResolver("placement_order"))
					})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("includePacks").(bool) {
					fields = []interface{}{"id", "name", "color", "points"}
					err = queryhelper.Build(txDao.DB(), &result.Pack, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
						query.Where(dbx.Exists(dbx.NewExp(fmt.Sprintf(
							`SELECT NULL FROM %v pl WHERE pl.level = {:levelId} AND pl.pack = %v`,
							demonlist.Aredl().Packs.PackLevelTableName,
							prefixResolver("id")), dbx.Params{"levelId": result.Id})))
						query.OrderBy(prefixResolver("placement_order"))
					})
				}
				return c.JSON(http.StatusOK, result)
			})
			return err
		},
	})
	return err
}

func registerLevelHistoryEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/level-history",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{
				"level_id": util.LoadInt(false, validation.Min(1)),
				"id":       util.LoadString(false),
			}),
		},
		Handler: func(c echo.Context) error {
			hasLevelId := c.Get("level_id") != nil
			hasId := c.Get("id") != nil
			if !hasLevelId && !hasId {
				return util.NewErrorResponse(nil, "level_id or id has to be set")
			}
			if hasLevelId && hasId {
				return util.NewErrorResponse(nil, "Can't query for level_id and id at the same time")
			}
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				var result []queryhelper.HistoryEntry
				fields := []interface{}{
					"action", "new_position",
					queryhelper.Extend{FieldName: "Level", Fields: []interface{}{}},
					queryhelper.Extend{FieldName: "Cause", Fields: []interface{}{"id", "name", "level_id"}},
				}
				err := queryhelper.Build(txDao.DB(), &result, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
					if hasLevelId {
						query.Where(dbx.HashExp{prefixResolver("level.level_id"): c.Get("level_id")})
					}
					if hasId {
						query.Where(dbx.HashExp{prefixResolver("level.id"): c.Get("id")})
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

func registerListDisplayNamesEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/display-names",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{}),
		},
		Handler: func(c echo.Context) error {
			var users []queryhelper.User
			fields := []interface{}{
				"id", "global_name", "role", "aredl_plus",
			}
			err := queryhelper.Build(app.Dao().DB(), &users, fields, func(query *dbx.SelectQuery, prefixResolver queryhelper.PrefixResolver) {
				query.Where(dbx.Or(dbx.NotIn(prefixResolver("role"), "member"), dbx.HashExp{"aredl_plus": true}))
			})
			if err != nil {
				return util.NewErrorResponse(err, "failed to query data")
			}
			result := make(map[string][]queryhelper.User)
			result["aredl_plus"] = make([]queryhelper.User, 0)
			for _, user := range users {
				if user.AredlPlus {
					result["aredl_plus"] = append(result["aredl_plus"], user)
				}
				if user.Role == "member" && !user.AredlPlus {
					continue
				}
				list, exists := result[user.Role]
				if !exists {
					list = make([]queryhelper.User, 0)
				}
				result[user.Role] = append(list, user)
			}
			return c.JSON(200, result)
		},
	})
	return err
}
