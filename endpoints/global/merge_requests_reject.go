package global

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
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Param			id	path	string	true	"request id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/merge-requests/{id}/reject [post]
func registerMergeRequestRejectEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/merge-requests/:id/reject",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_merge_review"),
			middlewares.LoadParam(middlewares.LoadData{
				"id": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				record, err := txDao.FindRecordById(names.TableMergeRequests, c.Get("id").(string))
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
			c.Response().Header().Set("Cache-Control", "no-store")
			return c.String(200, "Rejected")
		},
	})
	return err
}
