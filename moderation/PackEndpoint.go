package moderation

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"modernc.org/mathutil"
	"net/http"
)

func registerPackCreate(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/create",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"name":      {util.LoadString, true, nil, util.PackRules()},
				"colour":    {util.LoadString, true, nil, util.PackRules()},
				"placement": {util.LoadInt, false, nil, util.PackRules()},
				"levels":    {util.LoadStringArray, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				highestPlacement, err := queryMaxPlacementPosition(txDao)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to query max placement position", nil)
				}
				placement := highestPlacement + 1
				if c.Get("placement") != nil {
					placement = c.Get("placement").(int)
					if placement > highestPlacement+1 {
						return apis.NewBadRequestError(fmt.Sprintf("Placement can't be higher than current highest placement of %d", highestPlacement+1), nil)
					}
				}
				// Move all levels down from the placement position
				_, err = txDao.DB().Update(names.TablePacks, dbx.Params{"placement_order": dbx.NewExp("placement_order+1")}, dbx.NewExp("placement_order>={:placement}", dbx.Params{"placement": placement})).Execute()
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to move down other packs", nil)
				}

				packRecord, err := util.AddRecordByCollectionName(txDao, app, names.TablePacks, map[string]any{
					"name":            c.Get("name"),
					"colour":          c.Get("colour"),
					"placement_order": placement,
				})
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to create pack", nil)
				}
				packLevelCollection, err := txDao.FindCollectionByNameOrId(names.TablePackLevels)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to load pack level collection", nil)
				}
				if len(c.Get("levels").([]string)) < 2 {
					return apis.NewBadRequestError("Levels must include at least two levels", nil)
				}
				for _, level := range c.Get("levels").([]string) {
					_, err := util.AddRecord(txDao, app, packLevelCollection, map[string]any{
						"pack":  packRecord.Id,
						"level": level,
					})
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to add pack level", nil)
					}
				}
				// We don't need to handle users that got their pack removed, because there will never be any on a newly created pack
				_, err = points.UpdateCompletedPacksByPackId(txDao, packRecord.Id)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update completed packs", nil)
				}
				err = points.UpdatePackPointsByPackId(txDao, packRecord.Id)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update pack points", nil)
				}
				err = points.UpdateUserPointsByPackId(txDao, packRecord.Id)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update user points", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerPackUpdate(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/update",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"pack_id":   {util.LoadString, true, nil, util.PackRules()},
				"name":      {util.LoadString, false, nil, util.PackRules()},
				"colour":    {util.LoadString, false, nil, util.PackRules()},
				"placement": {util.LoadInt, false, nil, util.PackRules()},
				"levels":    {util.LoadStringArray, false, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				packRecord, err := txDao.FindRecordById(names.TablePacks, c.Get("pack_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find pack", nil)
				}
				packForm := forms.NewRecordUpsert(app, packRecord)
				packForm.SetDao(txDao)
				err = packForm.LoadData(map[string]any{
					"name":   util.UseOtherIfNil(c.Get("name"), packRecord.GetString("name")),
					"colour": util.UseOtherIfNil(c.Get("colour"), packRecord.GetString("colour")),
				})
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to load pack data", nil)
				}
				err = packForm.Submit()
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update pack", nil)
				}
				if c.Get("placement") != nil {
					newPlacement := c.Get("placement").(int)
					oldPlacement := packRecord.GetInt("placement_order")
					moveInc := -1
					if newPlacement < oldPlacement {
						moveInc = 1
					}
					_, err = txDao.DB().Update(
						names.TablePacks,
						dbx.Params{"placement_order": dbx.NewExp("CASE WHEN placement_order = {:old} THEN {:new} ELSE placement_order + {:inc} END",
							dbx.Params{"old": oldPlacement, "new": newPlacement, "inc": moveInc})},
						dbx.Between("placement_order",
							mathutil.Min(newPlacement, oldPlacement),
							mathutil.Max(newPlacement, oldPlacement),
						)).Execute()
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to update", nil)
					}
				}
				if c.Get("levels") != nil {
					newLevels := c.Get("levels").([]string)
					if len(newLevels) < 2 {
						return apis.NewBadRequestError("Pack has to have at least two levels", nil)
					}
					type LevelData struct {
						Id string `db:"level"`
					}
					var oldLevelData []LevelData
					err = txDao.DB().Select("level").From(names.TablePackLevels).Where(dbx.HashExp{"pack": c.Get("pack_id")}).All(&oldLevelData)
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to fetch current pack levels", nil)
					}
					oldLevels := util.MapSlice(oldLevelData, func(value LevelData) string { return value.Id })
					addedLevels := util.SliceDifference(newLevels, oldLevels)
					for _, level := range addedLevels {
						_, err = util.AddRecordByCollectionName(txDao, app, names.TablePackLevels, map[string]any{
							"level": level,
							"pack":  packRecord.Id,
						})
						if err != nil {
							return apis.NewApiError(http.StatusInternalServerError, "Failed to add new level to pack", nil)
						}
					}
					removedLevels := util.SliceDifference(oldLevels, newLevels)
					removedLevelsAsInterface := util.MapSlice(removedLevels, func(value string) interface{} { return value })
					_, err = txDao.DB().Delete(names.TablePackLevels, dbx.In("level", removedLevelsAsInterface...)).Execute()
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to remove pack level", nil)
					}
					// users that got their pack removed
					removedUsers, err := points.UpdateCompletedPacksByPackId(txDao, packRecord.Id)
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to update completed packs", nil)
					}
					err = points.UpdateUserPointsByUserIds(txDao, removedUsers...)
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to update user points that got their pack removed", nil)
					}
					// update users that can have the pack now
					err = points.UpdateUserPointsByPackId(txDao, packRecord.Id)
					if err != nil {
						return apis.NewApiError(http.StatusInternalServerError, "Failed to update user points that have the pack", nil)
					}
				}
				return nil
			})
			return err
		},
	})
	return err
}

func registerPackDelete(e *echo.Echo, app *pocketbase.PocketBase) error {
	_, err := e.AddRoute(echo.Route{
		Method: http.MethodPost,
		Path:   pathPrefix + "/pack/delete",
		Middlewares: []echo.MiddlewareFunc{
			apis.ActivityLogger(app),
			util.CheckBanned(),
			util.RequirePermissionGroup(app, "manage_packs"),
			util.ValidateAndLoadParam(map[string]util.ValidationData{
				"pack_id": {util.LoadString, true, nil, util.PackRules()},
			}),
		},
		Handler: func(c echo.Context) error {
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				// store users that have the pack, so they can be updated once it gets removed
				type UserData struct {
					Id string `db:"user"`
				}
				var usersWithPackData []UserData
				err := txDao.DB().Select("user").From(names.TableCompletedPacks).Where(dbx.HashExp{"pack": c.Get("pack_id")}).All(&usersWithPackData)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to fetch users that completed the given pack", nil)
				}
				usersWithPack := util.MapSlice(usersWithPackData, func(value UserData) interface{} { return value.Id })
				// remove pack
				packRecord, err := txDao.FindRecordById(names.TablePacks, c.Get("pack_id").(string))
				if err != nil {
					return apis.NewBadRequestError("Could not find pack", nil)
				}
				err = txDao.DeleteRecord(packRecord)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to delete pack", nil)
				}
				// Move all levels up from the placement position
				_, err = txDao.DB().Update(names.TablePacks, dbx.Params{"placement_order": dbx.NewExp("placement_order-1")}, dbx.NewExp("placement_order>={:placement}", dbx.Params{"placement": packRecord.GetInt("placement_order")})).Execute()
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to move down other packs", nil)
				}
				// pack levels and pack completions will be deleted automatically by cascade
				_, err = txDao.DB().Delete(names.TablePackLevels, dbx.HashExp{"pack": c.Get("pack_id").(string)}).Execute()
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to delete completed packs for users", nil)
				}
				// update user points that got their pack removed
				err = points.UpdateUserPointsByUserIds(txDao, usersWithPack...)
				if err != nil {
					return apis.NewApiError(http.StatusInternalServerError, "Failed to update user points", nil)
				}
				return nil
			})
			return err
		},
	})
	return err
}

func queryMaxPlacementPosition(dao *daos.Dao) (int, error) {
	var position int
	err := dao.DB().Select("max(placement_order)").From(names.TablePacks).Row(&position)
	if err != nil {
		return 0, err
	}
	return position, nil
}
