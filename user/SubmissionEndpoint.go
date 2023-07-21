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
			util.RequirePermission("member"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"id":          {util.LoadInt, true, nil, util.PackRules(validation.Min(1))},
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
					return apis.NewApiError(500, "User not found", nil)
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
					return apis.NewApiError(500, "Error processing request", nil)
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
					"placement_order": placementOrder,
				})
				if err != nil {
					return apis.NewApiError(500, "Failed to submit", nil)
				}
				err = submissionForm.Submit()
				if err != nil {
					switch err.(type) {
					case validation.Errors:
						return apis.NewBadRequestError(err.Error(), nil)
					default:
						return apis.NewApiError(500, "Error placing level", nil)
					}
				}
				return nil
			})
			return err
		},
	})
	return err
}
