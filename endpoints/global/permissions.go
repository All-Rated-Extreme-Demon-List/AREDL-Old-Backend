package global

import (
	"AREDL/middlewares"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

// registerPermissionsEndpoint godoc
//
//	@Summary		Get a list of Permissions
//	@Description	Returns all the available permissions to the authenticated user, if there is no authenticaiton provided, the permissions will be empty
//	@Security		ApiKeyAuth
//	@Tags			global
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	map[string]middlewares.PermissionData
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/me/permissions [get]
func registerPermissionsEndpoint(e *echo.Group, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   "/me/permissions",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.LoadApiKey(app),
			middlewares.CheckBanned(),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return c.JSON(200, map[string]middlewares.PermissionData{})
				}
				result, err := middlewares.GetAllPermissions(txDao, userRecord.Id)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load permissions")
				}
				c.Response().Header().Set("Cache-Control", "no-store")
				return c.JSON(200, result)
			})
			return err
		},
	})
	return err
}
