package aredl

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
	"regexp"
)

// registerSubmissionEndpoint godoc
//
//	@Summary		Create or Update submission
//	@Description	Creates a submission. If a submission for a level already exists, it will be updated instead. Submissions can only be updated when its status is pending or rejected_retryable
//	@Description	Requires user permission: aredl.user_submit
//	@Description	If the user has the permission aredl.priority they will automatically be assigned to the priority queue
//	@Security		ApiKeyAuth
//	@Tags			aredl
//	@Param			level				query	string	true	"internal level id"
//	@Param			video_url			query	string	true	"display video url"	format(url)
//	@Param			mobile				query	bool	true	"whether submission was done on mobile"
//	@Param			ldm_id				query	int		false	"ldm gd level id if used"
//	@Param			raw_footage			query	string	false	"raw footage"	format(url)
//	@Param			additional_notes	query	string	false	"additional notes the user wants to add to a submission. Max 100 characters"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/me/submissions [post]
func registerSubmissionEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/me/submissions",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "user_submit"),
			middlewares.LoadParam(middlewares.LoadData{
				"submissionData": middlewares.LoadMap("", middlewares.LoadData{
					"level":            middlewares.LoadString(true),
					"video_url":        middlewares.LoadString(true, is.URL),
					"mobile":           middlewares.LoadBool(true),
					"ldm_id":           middlewares.LoadInt(false),
					"raw_footage":      middlewares.LoadString(false, is.URL),
					"additional_notes": middlewares.LoadString(false, validation.Match(regexp.MustCompile("^([a-zA-Z0-9 ._]{0,100}$)"))),
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
				hasPriority, _, err := middlewares.GetPermission(txDao, userRecord.Id, "aredl", "priority")
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load priority")
				}
				submissionData["priority"] = hasPriority
				err = demonlist.UpsertSubmission(txDao, app, aredl, submissionData)
				return err
			})
			return err
		},
	})
	return err
}
