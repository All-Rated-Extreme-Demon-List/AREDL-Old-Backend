package moderation

import (
	"AREDL/demonlist"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"net/http"
)

func registerSubmissionUpdateEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submission/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "submission_review"),
			util.LoadParam(util.LoadData{
				"submissionData": util.LoadMap("", util.LoadData{
					"id":               util.LoadString(true),
					"level":            util.LoadString(false),
					"status":           util.LoadString(false, validation.In(string(demonlist.StatusRejected), string(demonlist.StatusAccepted), string(demonlist.StatusRejectedRetryable), string(demonlist.StatusPending))),
					"fps":              util.LoadInt(false, validation.Min(30), validation.Max(360)),
					"video_url":        util.LoadString(false, is.URL),
					"mobile":           util.LoadBool(false),
					"percentage":       util.LoadInt(false, validation.Min(1), validation.Max(100)),
					"ldm_id":           util.LoadInt(false, validation.Min(1)),
					"raw_footage":      util.LoadString(false, is.URL),
					"placement_order":  util.LoadInt(false, validation.Min(1)),
					"rejection_reason": util.LoadString(false),
				}),
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return util.NewErrorResponse(nil, "User not found")
			}
			aredl := demonlist.Aredl()
			submissionData := c.Get("submissionData").(map[string]interface{})
			if submissionData["status"] != nil {
				submissionData["reviewer"] = userRecord.Id
				if list.ExistInSlice(submissionData["status"].(string), []string{string(demonlist.StatusRejectedRetryable), string(demonlist.StatusRejected)}) {
					if submissionData["rejection_reason"] == nil {
						return util.NewErrorResponse(nil, "Rejection must include reason")
					}
				}
			}
			allowedOriginalStatus := []demonlist.SubmissionStatus{demonlist.StatusPending, demonlist.StatusAccepted, demonlist.StatusRejected, demonlist.StatusRejectedRetryable}
			_, err := demonlist.UpsertSubmission(app.Dao(), app, aredl, submissionData, allowedOriginalStatus)
			return err
		},
	})
	return err
}
