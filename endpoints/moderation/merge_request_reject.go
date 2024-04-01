package moderation

import (
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

// registerMergeRequestRejectEndpoint godoc
//
//	@Summary		Reject merge request
//	@Description	Rejects merge request
//	@Description	Requires user permission: user_merge_review
//	@Tags			moderation
//	@Param			requestId	query	string	true	"request id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/merge-request/reject [post]
func registerMergeRequestRejectEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/merge-request/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_merge_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"requestId": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				record, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("requestId").(string))
				if err != nil {
					return err
				}
				err = txDao.DeleteRecord(record)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return util.NewErrorResponse(err, "Failed to reject")
			}
			return c.String(200, "Rejected")
		},
	})
	return err
}
