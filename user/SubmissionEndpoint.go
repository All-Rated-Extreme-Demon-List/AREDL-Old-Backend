package user

import (
	"AREDL/names"
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
	"github.com/pocketbase/pocketbase/tools/inflector"
	"net/http"
)

func registerSubmissionEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submit",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("member"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id":          {util.LoadString, true, nil, util.PackRules()},
				"fps":         {util.LoadInt, true, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"video_url":   {util.LoadString, true, nil, util.PackRules(is.URL)},
				"device":      {util.LoadString, true, nil, util.PackRules(validation.In("pc", "mobile"))},
				"percentage":  {util.LoadInt, false, 100, util.PackRules(validation.Min(1), validation.Max(100))},
				"ldm_id":      {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"raw_footage": {util.LoadString, false, "", util.PackRules(is.URL)},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewApiError(http.StatusInternalServerError, "User not found", nil)
				}
				if userRecord.GetBool("banned_from_list") {
					return apis.NewBadRequestError("Banned from submissions", nil)
				}
				levelRecord, err := txDao.FindRecordById(names.TableLevels, c.Get("id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find level", nil)
				}
				// check if there already is a submission by that player
				submissionCollection, err := txDao.FindCollectionByNameOrId(names.TableSubmissions)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Error processing request", nil)
				}
				var placementOrder int
				submissionRecord := &models.Record{}
				err = txDao.RecordQuery(submissionCollection).
					AndWhere(dbx.HashExp{
						inflector.Columnify("level"):        levelRecord.Id,
						inflector.Columnify("submitted_by"): userRecord.Id,
					}).Limit(1).One(submissionRecord)
				if err == nil {
					// submission exists
					if submissionRecord.GetString("status") != "rejected_retryable" {
						return apis.NewBadRequestError("Already submitted", nil)
					}
					placementOrder = submissionRecord.GetInt("placement_order")
				} else {
					// create new submission
					submissionRecord = models.NewRecord(submissionCollection)
					err = txDao.DB().Select("max(placement_order)").From(names.TableSubmissions).Where(dbx.HashExp{
						inflector.Columnify("level"): levelRecord.Id,
					}).Row(&placementOrder)
					placementOrder++
				}
				submissionForm := forms.NewRecordUpsert(app, submissionRecord)

				err = submissionForm.LoadData(map[string]any{
					"status":          "pending",
					"fps":             c.Get("fps"),
					"video_url":       c.Get("video_url"),
					"device":          c.Get("device"),
					"percentage":      c.Get("percentage"),
					"ldm_id":          c.Get("ldm_id"),
					"raw_footage":     c.Get("raw_footage"),
					"submitted_by":    userRecord.Id,
					"level":           levelRecord.Id,
					"placement_order": placementOrder,
				})
				submissionForm.SetDao(txDao)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to submit", nil)
				}
				err = submissionForm.Submit()
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(http.StatusInternalServerError, "Error placing level", nil)
					}
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerSubmissionWithdrawEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/withdraw",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermission("member"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return apis.NewBadRequestError("User was not found", nil)
				}
				submissionRecord, err := txDao.FindRecordById(names.TableSubmissions, c.Get("id").(string))
				if err != nil {
					return apis.NewBadRequestError("Submission was not found", nil)
				}
				if submissionRecord.GetString("submitted_by") != userRecord.Id {
					return apis.NewBadRequestError("Submission was not by the requesting user", nil)
				}
				if submissionRecord.GetString("status") != "pending" {
					return apis.NewBadRequestError("Submission was already processed", nil)
				}
				err = txDao.DeleteRecord(submissionRecord)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to delete submission", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}
