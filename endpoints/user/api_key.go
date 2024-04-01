package user

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

type ApiKeyResponse struct {
	ApiKey         string `json:"api_key"`
	NewlyGenerated bool   `json:"newly_generated"`
}

// registerGetApiKeyEndpoint godoc
//
//	@Summary		Get Api Key
//	@Description	Gets the authenticated users api key. If the user does not have one it generates a new one.
//	@Description	Requires user permission: user_request_api_key
//	@Tags			user
//	@Security		ApiKeyAuth[authorization]
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	ApiKeyResponse
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/user/api-key [get]
func registerGetApiKeyEndpoint(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodGet,
		Path:   pathPrefix + "/api-key",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "user_request_api_key"),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				if userRecord == nil {
					return util.NewErrorResponse(nil, "User not found")
				}
				var response ApiKeyResponse
				response.ApiKey = userRecord.GetString("api_key")
				response.NewlyGenerated = false
				if response.ApiKey == "" {
					response.ApiKey = util.RandString(32)
					response.NewlyGenerated = true
					userRecord.Set("api_key", response.ApiKey)
					err := txDao.SaveRecord(userRecord)
					if err != nil {
						return util.NewErrorResponse(nil, "Failed to create api key")
					}
				}
				return c.JSON(200, response)
			})
			return err
		},
	})
	return err
}
