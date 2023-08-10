package demonlist

import (
	"AREDL/util"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"modernc.org/mathutil"
)

type SubmissionStatus string

const (
	StatusAccepted          SubmissionStatus = "accepted"
	StatusPending           SubmissionStatus = "pending"
	StatusRejected          SubmissionStatus = "rejected"
	StatusRejectedRetryable SubmissionStatus = "rejected_retryable"
)

func UpsertSubmission(dao *daos.Dao, app core.App, listData ListData, submissionData map[string]any, allowedOriginalStatus []SubmissionStatus) (*models.Record, error) {
	submissionRecord := &models.Record{}
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		submissionRecords, err := txDao.FindRecordsByExpr(listData.SubmissionTableName,
			dbx.Or(
				dbx.HashExp{"id": submissionData["id"]},
				dbx.HashExp{"submitted_by": submissionData["submitted_by"], "level": submissionData["level"]}))
		if err != nil {
			return util.NewErrorResponse(err, "Failed to query for submissions")
		}
		if len(submissionRecords) == 0 {
			if submissionData["id"] != nil {
				// should have found at least one entry
				return util.NewErrorResponse(nil, "Could not find submission")
			}
			submissionCollection, err := txDao.FindCollectionByNameOrId(listData.SubmissionTableName)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load collection")
			}
			submissionRecord = models.NewRecord(submissionCollection)
			var maxPlacementOrder int
			err = txDao.DB().Select("COALESCE(max(placement_order),0)").From(listData.SubmissionTableName).Where(dbx.HashExp{
				"level": submissionData["level"],
			}).Row(&maxPlacementOrder)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to query order")
			}
			submissionData["placement_order"] = maxPlacementOrder + 1
		} else if len(submissionRecords) == 1 {
			submissionRecord = submissionRecords[0]
			// check if status change is valid
			if !list.ExistInSlice(submissionRecord.GetString("status"), util.MapSlice(allowedOriginalStatus, func(s SubmissionStatus) string { return string(s) })) {
				return util.NewErrorResponse(err, "Not allowed to change submission status")
			}
			// order changes
			placement, existsPlacement := submissionData["placement_order"]
			if existsPlacement {
				var maxPlacementOrder int
				err = txDao.DB().Select("COALESCE(max(placement_order),0)").From(listData.SubmissionTableName).Where(dbx.HashExp{
					"level": submissionRecord.GetString("level"),
				}).Row(&maxPlacementOrder)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to query order")
				}
				// move other levels
				newPlacement := placement.(int)
				if newPlacement < 1 || newPlacement > maxPlacementOrder+1 {
					return util.NewErrorResponse(nil, "Placement position out of range")
				}
				oldPlacement := submissionRecord.GetInt("placement_order")
				increment := util.If(newPlacement > oldPlacement, -1, 1)
				_, err = txDao.DB().Update(
					listData.SubmissionTableName,
					dbx.Params{
						"placement_order": dbx.NewExp("placement_order + {:inc}",
							dbx.Params{"inc": increment})},
					dbx.And(
						dbx.Between("placement_order", mathutil.Min(newPlacement, oldPlacement), mathutil.Max(newPlacement, oldPlacement)),
						dbx.HashExp{"level": submissionRecord.GetString("level")})).Execute()
				if err != nil {
					return util.NewErrorResponse(err, "Failed to change order for other records")
				}
			}
		} else {
			return util.NewErrorResponse(nil, "Found too many records")
		}
		submissionForm := forms.NewRecordUpsert(app, submissionRecord)
		submissionForm.SetDao(txDao)
		err = submissionForm.LoadData(submissionData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to load data")
		}
		err = submissionForm.Submit()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to submit new submission data")
		}
		// update leaderboard for user if submission status has changed
		if submissionData["status"] != nil {
			err = UpdateLeaderboardAndPacksForUser(txDao, listData, submissionRecord.GetString("submitted_by"))
			if err != nil {
				return util.NewErrorResponse(err, "Failed to update leaderboard")
			}
		}
		return nil
	})
	return submissionRecord, err
}

func DeleteSubmission(dao *daos.Dao, listData ListData, submissionId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		submissionRecord, err := txDao.FindRecordById(listData.SubmissionTableName, submissionId)
		if err != nil {
			return util.NewErrorResponse(err, "Could not find submission")
		}
		err = txDao.DeleteRecord(submissionRecord)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete submission")
		}
		_, err = txDao.DB().Update(
			listData.SubmissionTableName,
			dbx.Params{"placement_order": dbx.NewExp("placement_order - 1")},
			dbx.And(
				dbx.NewExp("placement_order >= {:placement}", dbx.Params{"placement": submissionRecord.GetInt("placement_order")}),
				dbx.HashExp{"level": submissionRecord.GetString("level")})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update other placement positions")
		}
		// update leaderboard
		err = UpdateLeaderboardAndPacksForUser(txDao, listData, submissionRecord.GetString("submitted_by"))
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update leaderboard")
		}
		return nil
	})
	return err
}
