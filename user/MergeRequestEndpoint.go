package user

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerMergeRequestEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge-request",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_request_merge"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"placeholder_name": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				if record, _ := txDao.FindFirstRecordByData(names.TableMergeRequests, "user", userRecord.Id); record != nil {
					return apis.NewBadRequestError("Merge request already exists", nil)
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Could not load collection", nil)
				}
				legacyRecord := &models.Record{}
				err = txDao.RecordQuery(userCollection).
					AndWhere(dbx.HashExp{
						"global_name": c.Get("placeholder_name"),
						"placeholder": true,
					}).Limit(1).One(legacyRecord)
				if err != nil {
					return apis.NewBadRequestError("Unknown legacy user", nil)
				}
				_, err = util.AddRecordByCollectionName(txDao, app, names.TableMergeRequests, map[string]any{
					"user":     userRecord.Id,
					"to_merge": legacyRecord.Id,
				})
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to create request", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
