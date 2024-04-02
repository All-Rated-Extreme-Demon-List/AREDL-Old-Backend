package aredl

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerSubmissionRejectEndpoint godoc
//
//	@Summary		Reject AREDL submission.
//	@Description	Requires user permission: aredl.submission_review
//	@Security		ApiKeyAuth
//	@Tags			aredl
//	@Param			id					path	string	true	"internal submission id"
//	@Param			rejection_reason	query	string	false	"rejection reason"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/submissions/{id}/reject [post]
func registerSubmissionRejectEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/submission/:id/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "submission_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"id":               middlewares.LoadString(true),
				"rejection_reason": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
			if userRecord == nil {
				return util.NewErrorResponse(nil, "User not found")
			}
			aredl := demonlist.Aredl()
			return app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				submissionRecord, err := txDao.FindRecordById(aredl.SubmissionsTableName, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Could not load submission")
				}
				if submissionRecord.GetBool("rejected") {
					return util.NewErrorResponse(nil, "Submission already has been rejected")
				}
				submissionRecord.Set("rejected", true)
				submissionRecord.Set("rejection_reason", c.Get("rejection_reason").(string))
				submissionRecord.Set("reviewer", userRecord.Id)
				err = txDao.SaveRecord(submissionRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update submission")
				}
				return nil
			})
		},
	})
	return err
}
