package public

import (
	"AREDL/demonlist"
	"AREDL/queryhelper"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func registerUserEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/user",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.LoadParam(util.LoadData{
				"id": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			userId := c.Get("id").(string)
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				type ResultData struct {
					queryhelper.User
					CompletedPacks  *[]queryhelper.Pack            `json:"packs,omitempty"`
					CompletedLevels *[]queryhelper.AredlSubmission `json:"levels,omitempty"`
				}
				var result ResultData
				fields := []interface{}{
					"id", "created", "global_name", "role", "description", "country", "badges", "aredl_verified", "aredl_plus", "banned_from_list", "placeholder", "avatar_url", "banner_color",
				}
				query, prefixTable, err := queryhelper.Build(txDao.DB(), result, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				err = query.Where(dbx.HashExp{prefixTable[""] + ".id": userId}).One(&result)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user data")
				}
				fields = []interface{}{"name", "color", "points"}
				query, prefixTable, err = queryhelper.Build(txDao.DB(), result.CompletedPacks, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				query.InnerJoin(demonlist.Aredl().Packs.CompletedPacksTableName+" cp", dbx.NewExp(prefixTable[""]+".id = cp.pack"))
				err = query.Where(dbx.HashExp{"cp.user": result.Id}).OrderBy(prefixTable[""] + ".placement_order").All(&result.CompletedPacks)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user packs")
				}
				fields = []interface{}{
					"fps", "mobile", "video_url",
					queryhelper.Extend{FieldName: "Level", Fields: []interface{}{"name", "position", "points", "legacy", "level_id"}},
				}
				query, prefixTable, err = queryhelper.Build(txDao.DB(), result.CompletedLevels, fields)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to build query")
				}
				err = query.Where(dbx.HashExp{prefixTable[""] + ".submitted_by": result.Id, prefixTable[""] + ".status": demonlist.StatusAccepted}).
					OrderBy(prefixTable["level."] + ".position").All(&result.CompletedLevels)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user levels")
				}
				return c.JSON(http.StatusOK, result)
			})
			return err
		},
	})
	return err
}
