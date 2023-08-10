package user

import (
	"AREDL/demonlist"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerSubmissionEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submit",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_submissions"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"level":       {util.LoadString, true, nil, util.PackRules()},
				"fps":         {util.LoadInt, true, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"video_url":   {util.LoadString, true, nil, util.PackRules(is.URL)},
				"mobile":      {util.LoadBool, true, nil, util.PackRules()},
				"percentage":  {util.LoadInt, false, 100, util.PackRules(validation.Min(1), validation.Max(100))},
				"ldm_id":      {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"raw_footage": {util.LoadString, false, nil, util.PackRules(is.URL)},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				aredl := demonlist.Aredl()
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				levelRecord, err := txDao.FindRecordById(aredl.LevelTableName, c.Get("level").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find level", nil)
				}
				submissionData := map[string]interface{}{
					"level":        levelRecord.Id,
					"status":       demonlist.StatusPending,
					"fps":          c.Get("fps"),
					"video_url":    c.Get("video_url"),
					"mobile":       c.Get("mobile"),
					"percentage":   c.Get("percentage"),
					"submitted_by": userRecord.Id,
				}
				util.AddToMapIfNotNil(submissionData, "ldm_id", c.Get("ldm_id"))
				util.AddToMapIfNotNil(submissionData, "raw_footage", c.Get("raw_footage"))
				allowedOriginalStatus := []demonlist.SubmissionStatus{demonlist.StatusRejectedRetryable}
				_, err = demonlist.UpsertSubmission(txDao, app, aredl, submissionData, allowedOriginalStatus)
				return err
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
			util.RequirePermissionGroup(app, "user_submissions"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"record_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "Could not load user")
				}
				aredl := demonlist.Aredl()
				submissionRecord, err := txDao.FindRecordById(aredl.SubmissionTableName, c.Get("record_id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Submission was not found")
				}
				if submissionRecord.GetString("submitted_by") != userRecord.Id {
					return util.NewErrorResponse(err, "Submission does not belong to the user")
				}
				if submissionRecord.GetString("status") != "pending" {
					return util.NewErrorResponse(err, "Submission was already processed")
				}
				err = demonlist.DeleteSubmission(txDao, aredl, submissionRecord.Id)
				if err != nil {
					return err
				}
				return nil
			})
			return err
		},
	})
	return err
}
