package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

func registerMergeEndpoint(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/merge",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.RequirePermission("listMod", "listAdmin", "developer"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"discord_id":  {util.LoadString, true, nil, util.PackRules()},
				"legacy_name": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				userRecord, err := txDao.FindFirstRecordByData(names.TableUsers, "discord_id", c.Get("discord_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find user by discord id", nil)
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return apis.NewApiError(500, "Could not load collection", nil)
				}
				legacyRecord := &models.Record{}
				err = txDao.RecordQuery(userCollection).
					AndWhere(dbx.HashExp{
						"global_name": c.Get("legacy_name"),
						"legacy":      true,
					}).Limit(1).One(legacyRecord)
				if err != nil {
					return apis.NewBadRequestError("Unknown legacy user", nil)
				}
				submissions, err := txDao.FindRecordsByExpr(names.TableSubmissions, dbx.In("submitted_by", legacyRecord.Id))
				for _, submission := range submissions {
					submission.Set("submitted_by", userRecord.Id)
					err = txDao.SaveRecord(submission)
					if err != nil {
						return apis.NewApiError(500, "Failed updating submissions: "+err.Error(), nil)
					}
				}
				completedPacks, err := txDao.FindRecordsByExpr(names.TableCompletedPacks, dbx.In("user", legacyRecord.Id))
				for _, completedPack := range completedPacks {
					err = txDao.DeleteRecord(completedPack)
					if err != nil {
						return apis.NewApiError(500, "Failed deleting packs", nil)
					}
				}
				err = txDao.DeleteRecord(legacyRecord)
				if err != nil {
					return apis.NewApiError(500, "Failed to delete legacy user", nil)
				}
				err = points.UpdateCompletedPacksByUser(txDao, userRecord.Id)
				if err != nil {
					return apis.NewApiError(500, "Failed to update packs", nil)
				}
				err = points.UpdateUserPointsByUserId(txDao, userRecord.Id)
				return nil
			})
			return err
		},
	})
	return err
}
