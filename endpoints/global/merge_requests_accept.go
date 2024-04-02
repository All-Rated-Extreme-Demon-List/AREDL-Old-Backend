package global

import (
	"AREDL/demonlist"
	"AREDL/middlewares"
	"AREDL/names"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"net/http"
)

// registerMergeRequestAcceptEndpoint godoc
//
//	@Summary		Accept merge request
//	@Description	Accepts and merge request and merges the respective users
//	@Description	Requires user permission: user_merge_review
//	@Tags			global
//	@Param			id	path	string	true	"request id"
//	@Schemes		http https
//	@Produce		json
//	@Success		200
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/merge-requests/{id}/accept [post]
func registerMergeRequestAcceptEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   "/merge-requests/:id/accept",
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
				err = demonlist.MergeUsers(txDao, record.GetString("user"), record.GetString("to_merge"))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to merge")
				}
				err = txDao.DeleteRecord(record)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return util.NewErrorResponse(err, "Failed to merge")
			}
			return c.String(200, "Merged")
		},
	})
	return err
}
