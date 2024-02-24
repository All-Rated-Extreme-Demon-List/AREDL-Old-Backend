package aredl_user

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
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

// registerSubmissionEndpoint godoc
//
//	@Summary		Create or Update submission
//	@Description	Creates a submission. If a submission for a level already exists, it will be updated instead. Submissions can only be updated when its status is pending or rejected_retryable
//	@Description	Requires user permission: aredl.user_submit
//	@Tags			aredl_user
//	@Param			level		query	string	true	"internal level id"
//	@Param			fps			query	int		true	"framerate"			minimum(30)	maximum(360)
//	@Param			video_url	query	string	true	"display video url"	format(url)
//	@Param			mobile		query	bool	true	"whether submission was done on mobile"
//	@Param			ldm_id		query	int		false	"ldm gd level id if used"
//	@Param			raw_footage	query	string	false	"raw footage"	format(url)
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/user/submit [post]
func registerSubmissionEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submit",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "user_submit"),
			middlewares.LoadParam(middlewares.LoadData{
				"submissionData": middlewares.LoadMap("", middlewares.LoadData{
					"level":       middlewares.LoadString(true),
					"fps":         middlewares.LoadInt(true, validation.Min(30), validation.Max(360)),
					"video_url":   middlewares.LoadString(true, is.URL),
					"mobile":      middlewares.LoadBool(true),
					"ldm_id":      middlewares.LoadInt(false),
					"raw_footage": middlewares.LoadString(false, is.URL),
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
				submissionData["submitted_by"] = userRecord.Id

				// verify that submitted level exists
				_, err := txDao.FindRecordById(aredl.LevelTableName, submissionData["level"].(string))
				if err != nil {
					return apis.NewBadRequestError("Invalid level", nil)
				}
				err = demonlist.UpsertSubmission(txDao, app, aredl, submissionData)
				return err
			})
			return err
		},
	})
	return err
}
