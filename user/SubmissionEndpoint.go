package user

import (
	"AREDL/demonlist"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerSubmissionEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submit",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_submissions"),
			util.LoadParam(util.LoadData{
				"submissionData": util.LoadMap("", util.LoadData{
					"level":       util.LoadString(true),
					"fps":         util.LoadInt(true, validation.Min(30), validation.Max(360)),
					"video_url":   util.LoadString(true, is.URL),
					"mobile":      util.LoadBool(true),
					"percentage":  util.AddDefault(100, util.LoadInt(false, validation.Min(1), validation.Max(100))),
					"ldm_id":      util.LoadInt(false),
					"raw_footage": util.LoadString(false, is.URL),
				}),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				aredl := demonlist.Aredl()
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				submissionData := c.Get("submissionData").(map[string]interface{})
				submissionData["status"] = demonlist.StatusPending
				submissionData["submitted_by"] = userRecord.Id

				// verify that submitted level exists
				_, err := txDao.FindRecordById(aredl.LevelTableName, submissionData["level"].(string))
				if err != nil {
					return apis.NewBadRequestError("Invalid level", nil)
				}
				allowedOriginalStatus := []demonlist.SubmissionStatus{demonlist.StatusRejectedRetryable}
				_, err = demonlist.UpsertSubmission(txDao, app, aredl, submissionData, allowedOriginalStatus)
				return err
			})
			return err
		},
	})
	return err
}

func registerSubmissionWithdrawEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/withdraw",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "user_submissions"),
			util.LoadParam(util.LoadData{
				"record_id": util.LoadString(true),
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
