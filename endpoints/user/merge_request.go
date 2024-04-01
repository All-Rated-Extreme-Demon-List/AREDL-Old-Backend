package user

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
	"regexp"
)

// registerMergeRequestEndpoint godoc
//
//	@Summary		Merge Request
//	@Description	Creates a merge request for the user with a placeholder user. Needs to be reviewed by a moderator.
//	@Description	Requires user permission: user_request_merge
//	@Tags			user
//	@Param			placeholder_name	query	string	true	"name of the placeholder user to be merged with"
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/user/merge-request [post]
func registerMergeRequestEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge-request",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_request_merge"),
			middlewares.LoadParam(middlewares.LoadData{
				"placeholder_name": middlewares.LoadString(true, validation.Match(regexp.MustCompile("^([a-zA-Z0-9 ._]{0,30}$)"))),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				if record, _ := txDao.FindFirstRecordByData(names.TableMergeRequests, "user", userRecord.Id); record != nil {
					return util.NewErrorResponse(nil, "Merge request already exists")
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return util.NewErrorResponse(err, "Could not load collection")
				}
				legacyRecord := &models.Record{}
				err = txDao.RecordQuery(userCollection).
					AndWhere(dbx.HashExp{
						"global_name": c.Get("placeholder_name"),
						"placeholder": true,
					}).Limit(1).One(legacyRecord)
				if err != nil {
					return util.NewErrorResponse(err, "Unknown legacy user")
				}
				_, err = util.AddRecordByCollectionName(txDao, app, names.TableMergeRequests, map[string]any{
					"user":     userRecord.Id,
					"to_merge": legacyRecord.Id,
				})
				if err != nil {
					return util.NewErrorResponse(err, "Failed to create request")
				}
				return nil
			})
			return err
		},
	})
	return err
}
