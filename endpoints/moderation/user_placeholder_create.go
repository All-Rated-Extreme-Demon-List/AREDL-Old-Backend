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

type CreatePlaceholderResponse struct {
	Id string `json:"id"`
}

// registerCreatePlaceholderUser godoc
//
//	@Summary		Create a placeholder user
//	@Description	Creates a placeholder user for users that are not registered on the list yet
//	@Description	Requires user permission: create_placeholder
//	@Tags			moderation
//	@Param			username	query	string	true	"display name"
//	@Schemes		http https
//	@Produce		json
//	@Success		200 {object}	CreatePlaceholderResponse
//	@Failure		400	{object}	util.ErrorResponse
//	@Failure		403	{object}	util.ErrorResponse
//	@Router			/mod/user/create-placeholder [post]
func registerCreatePlaceholderUser(e *echo.Echo, app core.App) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/user/create-placeholder",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			middlewares.CheckBanned(),
			middlewares.RequirePermissionGroup(app, "", "create_placeholder"),
			middlewares.LoadParam(middlewares.LoadData{
				"username": middlewares.LoadString(true),
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				response := CreatePlaceholderResponse{}
				userRecord, _ := txDao.FindFirstRecordByData(names.TableUsers, "global_name", c.Get("username").(string))
				if userRecord != nil && userRecord.GetBool("placeholder") {
					return util.NewErrorResponse(nil, "Placeholder user with that name already exists")
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return util.NewErrorResponse(err, "Failed to load user collection")
				}
				createdUser, err := util.CreatePlaceholderUser(app, txDao, userCollection, c.Get("username").(string))
				if err != nil {
					return util.NewErrorResponse(err, "Failed to create placeholder user")
				}
				response.Id = createdUser.Id
				return c.JSON(200, response)
			})
			return err
		},
	})
	return err
}
