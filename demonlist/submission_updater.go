package demonlist

import (
	"AREDL/util"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"modernc.org/mathutil"
)

func UpsertSubmission(dao *daos.Dao, app core.App, listData ListData, submissionData map[string]any) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		submissions, err := txDao.FindRecordsByExpr(listData.SubmissionsTableName,
			dbx.Or(
				dbx.HashExp{"id": submissionData["id"]},
				dbx.HashExp{"submitted_by": submissionData["submitted_by"], "level": submissionData["level"]}))
		if err != nil {
			return util.NewErrorResponse(err, "Failed to query for submissions")
		}

		if len(submissions) == 1 {
			// update submission
			submissionForm := forms.NewRecordUpsert(app, submissions[0])
			submissionForm.SetDao(txDao)
			submissionData["is_update"] = false
			submissionData["rejected"] = false
			err = submissionForm.LoadData(submissionData)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load data")
			}
			err = submissionForm.Submit()
			if err != nil {
				return util.NewErrorResponse(err, "Failed to submit new submission data")
			}
		} else if len(submissions) == 0 {
			// create submission
			records, err := txDao.FindRecordsByExpr(listData.RecordsTableName,
				dbx.Or(
					dbx.HashExp{"id": submissionData["id"]},
					dbx.HashExp{"submitted_by": submissionData["submitted_by"], "level": submissionData["level"]}))
			if err != nil {
				return util.NewErrorResponse(err, "Failed to query for records")
			}
			submissionRecord := &models.Record{}
			submissionCollection, err := txDao.FindCollectionByNameOrId(listData.SubmissionsTableName)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load collection")
			}
			submissionRecord = models.NewRecord(submissionCollection)
			if len(records) == 0 {
				// new submission
				var maxSubmissionsPlacement int
				err = txDao.DB().Select("COALESCE(max(placement_order),0)").From(listData.SubmissionsTableName).Where(dbx.HashExp{
					"level": submissionData["level"],
				}).Row(&maxSubmissionsPlacement)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to max query submission order")
				}
				var maxRecordsPlacement int
				err = txDao.DB().Select("COALESCE(max(placement_order),0)").From(listData.RecordsTableName).Where(dbx.HashExp{
					"level": submissionData["level"],
				}).Row(&maxRecordsPlacement)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to max query record order")
				}
				submissionData["is_update"] = false
				submissionData["placement_order"] = mathutil.Max(maxRecordsPlacement, maxSubmissionsPlacement) + 1
			} else {
				// update record
				record := records[0]
				submissionData["is_update"] = true
				submissionData["placement_order"] = record.GetInt("placement_order")
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
		} else {
			return util.NewErrorResponse(nil, "Invalid state")
		}
		return nil
	})
	return err
}

func DeleteSubmission(dao *daos.Dao, listData ListData, submission *models.Record) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		err := txDao.DeleteRecord(submission)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete submission")
		}
		_, err = txDao.DB().Update(
			listData.SubmissionsTableName,
			dbx.Params{"placement_order": dbx.NewExp("placement_order - 1")},
			dbx.And(
				dbx.NewExp("placement_order >= {:placement}", dbx.Params{"placement": submission.GetInt("placement_order")}),
				dbx.HashExp{"level": submission.GetString("level")})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update other placement positions")
		}
		_, err = txDao.DB().Update(
			listData.RecordsTableName,
			dbx.Params{"placement_order": dbx.NewExp("placement_order - 1")},
			dbx.And(
				dbx.NewExp("placement_order >= {:placement}", dbx.Params{"placement": submission.GetInt("placement_order")}),
				dbx.HashExp{"level": submission.GetString("level")})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update other placement positions")
		}
		return nil
	})
	return err
}
