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

// registerSubmissionWithdrawEndpoint godoc
//
//	@Summary		Delete submission
//	@Description	Deletes a submission as long as it still is open for review.
//	@Description	Requires user permission: aredl.user_submission_delete
//	@Security		ApiKeyAuth
//	@Tags			aredl
//	@Param			id	path	string	true	"submission id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/me/submissions/{id} [delete]
func registerSubmissionWithdrawEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodDelete,
		Path:   "/me/submissions/:id",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "user_submission_delete"),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "Could not load user")
				}
				aredl := demonlist.Aredl()
				submissionRecord, err := txDao.FindRecordById(aredl.SubmissionsTableName, c.Get("id").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Submission was not found")
				}
				if submissionRecord.GetString("submitted_by") != userRecord.Id {
					return util.NewErrorResponse(err, "Submission does not belong to the user")
				}
				if submissionRecord.GetBool("rejected") {
					return util.NewErrorResponse(err, "Submission was already processed")
				}
				err = demonlist.DeleteSubmission(txDao, aredl, submissionRecord)
				if err != nil {
					return err
				}
				err = txDao.DeleteRecord(submissionRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete submission")
				}
				return nil
			})
			return err
		},
	})
	return err
}
