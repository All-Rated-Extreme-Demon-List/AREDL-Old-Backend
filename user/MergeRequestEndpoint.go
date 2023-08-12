package user

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerMergeRequestEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge-request",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_request_merge"),
			util.LoadParam(util.LoadData{
				"placeholder_name": util.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				if record, _ := txDao.FindFirstRecordByData(names.TableMergeRequests, "user", userRecord.Id); record != nil {
					return util.NewErrorResponse(nil, "Merge request already exists")
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return util.NewErrorResponse(err, "Could not load collection")
				}
				legacyRecord := &models.Record{}
				err = txDao.RecordQuery(userCollection).
					AndWhere(dbx.HashExp{
						"global_name": c.Get("placeholder_name"),
						"placeholder": true,
					}).Limit(1).One(legacyRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Unknown legacy user")
				}
				_, err = util.AddRecordByCollectionName(txDao, app, names.TableMergeRequests, map[string]any{
					"user":     userRecord.Id,
					"to_merge": legacyRecord.Id,
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to create request")
				}
				return nil
			})
			return err
		},
	})
	return err
}
