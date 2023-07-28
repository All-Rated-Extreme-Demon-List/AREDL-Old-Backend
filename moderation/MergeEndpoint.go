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

func mergeAccounts(dao *daos.Dao, userId string, toMergeId string) error {
	userRecord, err := dao.FindRecordById(names.TableUsers, userId)
	if err != nil {
		return apis.NewBadRequestError("Could not find user", nil)
	}
	otherRecord, err := dao.FindRecordById(names.TableUsers, toMergeId)
	if err != nil {
		return apis.NewBadRequestError("Could not find user to merge", nil)
	}
	submissions, err := dao.FindRecordsByExpr(names.TableSubmissions, dbx.In("submitted_by", otherRecord.Id))
	for _, submission := range submissions {
		submission.Set("submitted_by", userRecord.Id)
		err = dao.SaveRecord(submission)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed updating submissions: "+err.Error(), nil)
		}
	}
	createdLevels, err := dao.FindRecordsByExpr(names.TableCreators, dbx.HashExp{"creator": otherRecord.Id})
	for _, createdLevel := range createdLevels {
		createdLevel.Set("creator", userRecord.Id)
		err = dao.SaveRecord(createdLevel)
		if err != nil {
			return apis.NewApiError(500, "Failed updating created levels: "+err.Error(), nil)
		}
	}
	completedPacks, err := dao.FindRecordsByExpr(names.TableCompletedPacks, dbx.In("user", otherRecord.Id))
	for _, completedPack := range completedPacks {
		err = dao.DeleteRecord(completedPack)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed deleting packs", nil)
		}
	}

	err = dao.DeleteRecord(otherRecord)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to delete legacy user", nil)
	}
	err = points.UpdateCompletedPacksByUser(dao, userRecord.Id)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Failed to update packs", nil)
	}
	err = points.UpdateUserPointsByUserId(dao, userRecord.Id)
	return nil
}

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
					return apis.NewApiError(http.StatusInternalServerError, "Could not find merge request", nil)
				}
				err = mergeAccounts(txDao, requestRecord.GetString("user"), requestRecord.GetString("to_merge"))
				if err != nil {
					return err
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to delete request", nil)
				}
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

func registerMergeDirectEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge/direct",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"user_id":     {util.LoadString, true, nil, util.PackRules()},
				"to_merge_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				err := mergeAccounts(txDao, c.Get("user_id").(string), c.Get("to_merge_id").(string))
				return err
			})
			return err
		},
	})
	return err
}
