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
			}),
		},
		Handler: func(c echo.Context) error {
			hasLevelId := c.Get("level_id") != nil
			hasId := c.Get("id") != nil
			if !hasLevelId && !hasId {
				// return entire list
				var list []queryhelper.AredlLevel
				query, _, err := queryhelper.Build(app.Dao().DB(), list, []interface{}{"id", "position", "name", "level_id", "legacy"})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				err = query.OrderBy("position").All(&list)
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
				}
				var result ResultData
				fields := []interface{}{
					"id", "position", "name", "points", "legacy", "level_id", "level_password", "custom_song",
					queryhelper.Extend{FieldName: "Publisher", Fields: []interface{}{"id", "global_name"}},
				}
				query, prefixTable, err := queryhelper.Build(txDao.DB(), result.AredlLevel, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				whereExpression := dbx.HashExp{}
				if hasLevelId {
					whereExpression[prefixTable[""]+".level_id"] = c.Get("level_id")
				}
				if hasId {
					whereExpression[prefixTable[""]+".id"] = c.Get("id")
				}
				err = query.Where(whereExpression).One(&result)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load demonlist data")
				}
				if c.Get("includeVerification").(bool) {
					query, prefixTable, err = queryhelper.Build(txDao.DB(), result.Verification,
						[]interface{}{"id", "video_url", "fps", "mobile", queryhelper.Extend{
							FieldName: "SubmittedBy", Fields: []interface{}{"id", "global_name"},
						}})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to build query")
					}
					err = query.Where(dbx.HashExp{prefixTable[""] + ".level": result.Id, prefixTable[""] + ".placement_order": 1}).One(&result.Verification)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("includeCreators").(bool) {
					query, prefixTable, err = queryhelper.Build(txDao.DB(), result.Creators, []interface{}{"id", "global_name"})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to build query")
					}
					query.InnerJoin(demonlist.Aredl().CreatorTableName+" c", dbx.NewExp(fmt.Sprintf("%v.id=c.creator", prefixTable[""])))
					err = query.Where(dbx.HashExp{"c.level": result.Id}).All(&result.Creators)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
				}
				if c.Get("includeRecords").(bool) {
					query, prefixTable, err = queryhelper.Build(txDao.DB(), result.Verification,
						[]interface{}{"id", "video_url", "fps", "mobile", queryhelper.Extend{
							FieldName: "SubmittedBy", Fields: []interface{}{"id", "global_name"},
						}})
					if err != nil {
						return util.NewErrorResponse(err, "Failed to build query")
					}
					err = query.
						Where(dbx.HashExp{prefixTable[""] + ".level": result.Id, prefixTable[""] + ".status": demonlist.StatusAccepted}).
						AndWhere(dbx.NewExp(prefixTable[""] + ".placement_order <> 1")).
						OrderBy(prefixTable[""] + ".placement_order").
						All(&result.Records)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to load demonlist data")
					}
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
				query, prefixTable, err := queryhelper.Build(txDao.DB(), result, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				whereExpression := dbx.HashExp{}
				if hasLevelId {
					whereExpression[prefixTable["level."]+".level_id"] = c.Get("level_id")
				}
				if hasId {
					whereExpression[prefixTable["level."]+".id"] = c.Get("id")
				}
				err = query.Where(whereExpression).OrderBy(prefixTable[""] + ".created").All(&result)
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
