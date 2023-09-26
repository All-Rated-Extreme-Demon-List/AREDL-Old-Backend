package aredl_moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
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

// registerSubmissionUpdateEndpoint godoc
//
//	@Summary		Update AREDL submission
//	@Description	Update metadata and status of submission. Used to review submissions.
//	@Description	Requires user permission: aredl.submission_review
//	@Tags			aredl_moderation
//	@Param			id					query	string	true	"internal submission id"
//	@Param			level				query	string	false	"internal level id"
//	@Param			status				query	string	false	"submission status"	Enums(accepted, pending, rejected, rejected_retryable)
//	@Param			fps					query	int		false	"framerate"			minimum(30)	maximum(360)
//	@Param			video_url			query	string	false	"video url"			format(url)
//	@Param			mobile				query	bool	false	"whether submisssion was one on mobile"
//	@Param			ldm_id				query	int		false	"gd id of used ldm"
//	@Param			raw_footage			query	string	false	"raw footage"	format(url)
//	@Param			placement_order		query	int		false	"new position to move submission in viewed order. position 0 is verification"
//	@Param			rejection_reason	query	string	false	"when add reason new status is rejected or rejected_retryable"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/mod/submission/update [post]
func registerSubmissionUpdateEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submission/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "submission_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"submissionData": middlewares.LoadMap("", middlewares.LoadData{
					"id":               middlewares.LoadString(true),
					"level":            middlewares.LoadString(false),
					"status":           middlewares.LoadString(false, validation.In(string(demonlist.StatusRejected), string(demonlist.StatusAccepted), string(demonlist.StatusRejectedRetryable), string(demonlist.StatusPending))),
					"fps":              middlewares.LoadInt(false, validation.Min(30), validation.Max(360)),
					"video_url":        middlewares.LoadString(false, is.URL),
					"mobile":           middlewares.LoadBool(false),
					"ldm_id":           middlewares.LoadInt(false, validation.Min(1)),
					"raw_footage":      middlewares.LoadString(false, is.URL),
					"placement_order":  middlewares.LoadInt(false, validation.Min(1)),
					"rejection_reason": middlewares.LoadString(false),
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
