package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func registerMergeAcceptEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("request_id").(string))
				if err != nil {
					return apis.NewApiError(500, "Could not find merge request", nil)
				}
				userRecord, err := txDao.FindRecordById(names.TableUsers, requestRecord.GetString("user"))
				if err != nil {
					return apis.NewBadRequestError("Could not find user", nil)
				}
				otherRecord, err := txDao.FindRecordById(names.TableUsers, requestRecord.GetString("to_merge"))
				if err != nil {
					return apis.NewBadRequestError("Could not find user to merge", nil)
				}
				submissions, err := txDao.FindRecordsByExpr(names.TableSubmissions, dbx.In("submitted_by", otherRecord.Id))
				for _, submission := range submissions {
					submission.Set("submitted_by", userRecord.Id)
					err = txDao.SaveRecord(submission)
					if err != nil {
						return apis.NewApiError(500, "Failed updating submissions: "+err.Error(), nil)
					}
				}
				completedPacks, err := txDao.FindRecordsByExpr(names.TableCompletedPacks, dbx.In("user", otherRecord.Id))
				for _, completedPack := range completedPacks {
					err = txDao.DeleteRecord(completedPack)
					if err != nil {
						return apis.NewApiError(500, "Failed deleting packs", nil)
					}
				}
				err = txDao.DeleteRecord(otherRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete legacy user", nil)
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete request", nil)
				}
				err = points.UpdateCompletedPacksByUser(txDao, userRecord.Id)
				if err != nil {
					return apis.NewApiError(500, "Failed to update packs", nil)
				}
				err = points.UpdateUserPointsByUserId(txDao, userRecord.Id)
				return nil
			})
			return err
		},
	})
	return err
}

func registerMergeRejectEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("request_id").(string))
				if err != nil {
					return apis.NewApiError(500, "Could not find merge request", nil)
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete request", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
