package moderation

import (
	"AREDL/demonlist"
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

func mergeAccounts(dao *daos.Dao, userId string, toMergeId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		userRecord, err := txDao.FindRecordById(names.TableUsers, userId)
		if err != nil {
			return util.NewErrorResponse(err, "Could not find user")
		}
		otherRecord, err := txDao.FindRecordById(names.TableUsers, toMergeId)
		if err != nil {
			return util.NewErrorResponse(err, "Could not find user to merge")
		}
		aredl := demonlist.Aredl()
		type ColumnData struct {
			Table  string
			Column string
		}
		// remove conflicting records from old user
		_, err = txDao.DB().Delete(aredl.CreatorTableName,
			dbx.And(
				dbx.HashExp{"creator": otherRecord.Id},
				dbx.Exists(dbx.NewExp(fmt.Sprintf(`
					SELECT NULL FROM %s c
					WHERE c.creator = {:userId} AND c.level = %s.level`,
					aredl.CreatorTableName,
					aredl.CreatorTableName),
					dbx.Params{"userId": userRecord.Id})))).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete duplicate creators")
		}
		_, err = txDao.DB().Delete(aredl.SubmissionTableName,
			dbx.And(
				dbx.HashExp{"submitted_by": otherRecord.Id},
				dbx.Exists(dbx.NewExp(fmt.Sprintf(`
					SELECT NULL FROM %s rs
					WHERE rs.submitted_by = {:userId} AND rs.level = %s.level`,
					aredl.SubmissionTableName,
					aredl.SubmissionTableName),
					dbx.Params{"userId": userRecord.Id})))).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete duplicate submissions")
		}
		// merge
		columnsToChange := []ColumnData{
			{aredl.SubmissionTableName, "submitted_by"},
			{aredl.SubmissionTableName, "reviewer"},
			{aredl.CreatorTableName, "creator"},
			{aredl.HistoryTableName, "action_by"},
			{aredl.LevelTableName, "publisher"},
		}
		for _, data := range columnsToChange {
			_, err = txDao.DB().Update(data.Table, dbx.Params{data.Column: userRecord.Id}, dbx.HashExp{data.Column: otherRecord.Id}).Execute()
			if err != nil {
				return util.NewErrorResponse(err, "Failed to merge "+data.Table)
			}
		}
		_, err = txDao.DB().Delete(aredl.LeaderboardTableName, dbx.HashExp{"user": otherRecord.Id}).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete leaderboard entry")
		}
		_, err = txDao.DB().Delete(aredl.Packs.CompletedPacksTableName, dbx.HashExp{"user": otherRecord.Id}).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete completed packs")
		}
		err = txDao.DeleteRecord(otherRecord)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete merged user")
		}
		err = demonlist.UpdateLeaderboardAndPacksForUser(txDao, aredl, userRecord.Id)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update leaderboard")
		}
		return nil
	})
	return err
}

func registerMergeAcceptEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "merge_review"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("request_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find merge request")
				}
				err = mergeAccounts(txDao, requestRecord.GetString("user"), requestRecord.GetString("to_merge"))
				if err != nil {
					return err
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete request")
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
			util.RequirePermissionGroup(app, "merge_review"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"request_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				requestRecord, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("request_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not find merge request")
				}
				err = txDao.DeleteRecord(requestRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete request")
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
			util.RequirePermissionGroup(app, "direct_merge"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"user_id":     {util.LoadString, true, nil, util.PackRules()},
				"to_merge_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				// TODO check affected roles
				err := mergeAccounts(txDao, c.Get("user_id").(string), c.Get("to_merge_id").(string))
				return err
			})
			return err
		},
	})
	return err
}
