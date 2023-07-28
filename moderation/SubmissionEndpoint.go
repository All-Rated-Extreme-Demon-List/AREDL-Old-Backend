package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"net/http"
)

func registerSubmissionAcceptEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submission/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listHelper", "listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"submission_id": {util.LoadString, true, nil, util.PackRules()},
				"fps":           {util.LoadInt, false, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"video_url":     {util.LoadString, false, nil, util.PackRules(is.URL)},
				"device":        {util.LoadString, false, nil, util.PackRules(validation.In("pc", "mobile"))},
				"percentage":    {util.LoadInt, false, nil, util.PackRules(validation.Min(1), validation.Max(100))},
				"ldm_id":        {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"raw_footage":   {util.LoadString, false, "", util.PackRules(is.URL)},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				submissionRecord, err := txDao.FindRecordById(names.TableSubmissions, c.Get("submission_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find submission by id", nil)
				}
				if submissionRecord.Get("status") != "pending" {
					return apis.NewBadRequestError("Submission is not pending", nil)
				}
				submissionForm := forms.NewRecordUpsert(app, submissionRecord)
				submissionForm.SetDao(txDao)
				err = submissionForm.LoadData(map[string]any{
					"status":      "accepted",
					"reviewer":    userRecord.Id,
					"fps":         util.UseOtherIfNil(c.Get("fps"), submissionRecord.GetString("fps")),
					"video_url":   util.UseOtherIfNil(c.Get("video_url"), submissionRecord.GetString("video_url")),
					"percentage":  util.UseOtherIfNil(c.Get("percentage"), submissionRecord.GetString("percentage")),
					"ldm_id":      util.UseOtherIfNil(c.Get("ldm_id"), submissionRecord.GetString("ldm_id")),
					"raw_footage": util.UseOtherIfNil(c.Get("raw_footage"), submissionRecord.GetString("raw_footage")),
				})
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to load record data", nil)
				}
				err = submissionForm.Submit()
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to submit changes", nil)
				}
				err = points.UpdateCompletedPacksByUser(txDao, submissionRecord.GetString("submitted_by"))
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update packs", nil)
				}
				err = points.UpdateUserPointsByUserId(txDao, submissionRecord.GetString("submitted_by"))
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed tu update user points", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerSubmissionRejectEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submission/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("listHelper", "listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"submission_id":      {util.LoadString, true, nil, util.PackRules()},
				"reason":             {util.LoadString, true, nil, util.PackRules()},
				"reject_all_pending": {util.LoadBool, false, false, util.PackRules()},
				"retryable":          {util.LoadBool, false, false, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				submissionRecord, err := txDao.FindRecordById(names.TableSubmissions, c.Get("submission_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find submission by id", nil)
				}
				if list.ExistInSlice(submissionRecord.Get("status"), []any{"rejected, rejected_retryable"}) {
					return apis.NewBadRequestError("Submission is already rejected", nil)
				}
				reason := c.Get("reason").(string)
				retryable := c.Get("retryable").(bool)
				if (submissionRecord.GetString("status") == "rejected" && !retryable) || (submissionRecord.GetString("status") == "rejected_retryable" && retryable) {
					return apis.NewBadRequestError("Submission already is in that state", nil)
				}
				if c.Get("reject_all_pending").(bool) {
					submissionRecords, err := txDao.FindRecordsByExpr(names.TableSubmissions, dbx.HashExp{
						"submitted_by": submissionRecord.GetString("submitted_by"),
						"status":       "pending",
					})
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to get all pending records", nil)
					}
					for _, submission := range submissionRecords {
						if submission.Id == submissionRecord.Id {
							continue
						}
						err = rejectSubmissionRecord(app, txDao, submission, userRecord.Id, reason, retryable)
						if err != nil {
							return apis.NewApiError(http.StatusInternalServerError, "Failed to update submission", nil)
						}
					}
				}
				err = rejectSubmissionRecord(app, txDao, submissionRecord, userRecord.Id, reason, retryable)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update submission", nil)
				}
				err = points.UpdateCompletedPacksByUser(txDao, submissionRecord.GetString("submitted_by"))
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update packs", nil)
				}
				err = points.UpdateUserPointsByUserId(txDao, submissionRecord.GetString("submitted_by"))
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed tu update user points", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func rejectSubmissionRecord(app *pocketbase.PocketBase, dao *daos.Dao, record *models.Record, reviewerId string, reason string, retryable bool) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		wasRejectedAlready := list.ExistInSlice(record.Get("status"), []any{"rejected", "rejected_retryable"})
		status := "rejected"
		if retryable {
			status = "rejected_retryable"
		}
		form := forms.NewRecordUpsert(app, record)
		form.SetDao(txDao)
		err := form.LoadData(map[string]any{
			"reviewer":         reviewerId,
			"status":           status,
			"rejection_reason": reason,
		})
		if err != nil {
			return err
		}
		err = form.Submit()
		if err != nil {
			return err
		}
		if wasRejectedAlready {
			// don't add reason to log if it already was rejected for a reason
			// maybe update reason?
			return nil
		}
		return err
	})
	return err
}
