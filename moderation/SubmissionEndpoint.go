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
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"record_id":        {util.LoadString, true, nil, util.PackRules()},
				"fps":              {util.LoadInt, false, nil, util.PackRules(validation.Min(30), validation.Max(360))},
				"video_url":        {util.LoadString, false, nil, util.PackRules(is.URL)},
				"level":            {util.LoadString, false, nil, util.PackRules()},
				"mobile":           {util.LoadBool, false, nil, util.PackRules()},
				"percentage":       {util.LoadInt, false, nil, util.PackRules(validation.Min(1), validation.Max(100))},
				"ldm_id":           {util.LoadInt, false, nil, util.PackRules(validation.Min(1))},
				"raw_footage":      {util.LoadString, false, nil, util.PackRules(is.URL)},
				"placement":        {util.LoadInt, false, nil, util.PackRules()},
				"status":           {util.LoadString, false, nil, util.PackRules(validation.In(string(demonlist.StatusRejected), string(demonlist.StatusAccepted), string(demonlist.StatusRejectedRetryable), string(demonlist.StatusPending)))},
				"rejection_reason": {util.LoadString, false, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return util.NewErrorResponse(nil, "User not found")
			}
			aredl := demonlist.Aredl()
			submissionData := map[string]interface{}{
				"id": c.Get("record_id"),
			}
			util.AddToMapIfNotNil(submissionData, "status", c.Get("status"))
			util.AddToMapIfNotNil(submissionData, "fps", c.Get("fps"))
			util.AddToMapIfNotNil(submissionData, "video_url", c.Get("video_url"))
			util.AddToMapIfNotNil(submissionData, "mobile", c.Get("mobile"))
			util.AddToMapIfNotNil(submissionData, "level", c.Get("level"))
			util.AddToMapIfNotNil(submissionData, "percentage", c.Get("percentage"))
			util.AddToMapIfNotNil(submissionData, "ldm_id", c.Get("ldm_id"))
			util.AddToMapIfNotNil(submissionData, "raw_footage", c.Get("raw_footage"))
			util.AddToMapIfNotNil(submissionData, "placement_order", c.Get("placement"))
			util.AddToMapIfNotNil(submissionData, "rejection_reason", c.Get("rejection_reason"))
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
