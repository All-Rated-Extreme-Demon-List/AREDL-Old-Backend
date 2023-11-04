package aredl_moderation

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerSubmissionAcceptEndpoint godoc
//
//	@Summary		Accept AREDL submission.
//	@Description	Requires user permission: aredl.submission_review
//	@Security		ApiKeyAuth[authorization]
//	@Tags			aredl_moderation
//	@Param			id			query	string	true	"internal submission id"
//	@Param			fps			query	int		false	"framerate"	minimum(30)	maximum(360)
//	@Param			video_url	query	string	false	"video url"	format(url)
//	@Param			mobile		query	bool	false	"whether submisssion was one on mobile"
//	@Param			ldm_id		query	int		false	"gd id of used ldm"
//	@Param			raw_footage	query	string	false	"raw footage"	format(url)
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/aredl/mod/submission/accept [post]
func registerSubmissionAcceptEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/submission/accept",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "aredl", "submission_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"submissionData": middlewares.LoadMap("", middlewares.LoadData{
					"id":          middlewares.LoadString(true),
					"fps":         middlewares.LoadInt(false, validation.Min(30), validation.Max(360)),
					"video_url":   middlewares.LoadString(false, is.URL),
					"mobile":      middlewares.LoadBool(false),
					"ldm_id":      middlewares.LoadInt(false, validation.Min(1)),
					"raw_footage": middlewares.LoadString(false, is.URL),
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
			submissionData["reviewer"] = userRecord.Id
			return app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				submissionRecord, err := txDao.FindRecordById(aredl.SubmissionsTableName, submissionData["id"].(string))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to accept submission")
				}
				if submissionRecord.GetBool("rejected") {
					return util.NewErrorResponse(nil, "Submission has already been rejected")
				}
				var recordData map[string]any
				recordData["reviewer"] = userRecord.Id
				keys := []string{"level", "fps", "video_url", "mobile", "ldm_id", "raw_footage", "created"}
				for _, key := range keys {
					if value, ok := submissionData[key]; ok {
						recordData[key] = value
					} else {
						recordData[key] = submissionRecord.Get(key)
					}
				}
				recordCollection, err := txDao.FindCollectionByNameOrId(aredl.RecordsTableName)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load record collection")
				}
				record := models.NewRecord(recordCollection)
				if submissionRecord.GetBool("is_update") {
					records, err := txDao.FindRecordsByExpr(aredl.RecordsTableName, dbx.HashExp{"submitted_by": submissionRecord.GetString("submitted_by"), "level": submissionRecord.GetString("level")})
					if err != nil || len(records) != 1 {
						return util.NewErrorResponse(err, "Unable to find updated record")
					}
					record = records[0]
				} else {
					var maxRecordPlacement int
					err = txDao.DB().Select("COALESCE(max(placement_order),0)").From(aredl.RecordsTableName).Where(dbx.HashExp{"level": submissionData["level"]}).Row(&maxRecordPlacement)
					if err != nil {
						return util.NewErrorResponse(err, "Failed to query max placement pos")
					}
					recordData["placement_order"] = maxRecordPlacement + 1
				}
				recordForm := forms.NewRecordUpsert(app, record)
				recordForm.SetDao(txDao)
				err = recordForm.LoadData(recordData)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load new record data")
				}
				err = recordForm.Submit()
				if err != nil {
					return util.NewErrorResponse(err, "Failed to save record")
				}
				err = txDao.DeleteRecord(submissionRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to delete submission")
				}
				return demonlist.UpdateLeaderboardAndPacksForUser(txDao, aredl, submissionRecord.GetString("submitted_by"))
			})
		},
	})
	return err
}
