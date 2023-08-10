package demonlist

import (
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"modernc.org/mathutil"
)

func updatePackPointsByLevelRange(dao *daos.Dao, list ListData, minPos int, maxPos int) error {
	params := dbx.Params{}
	condition := dao.DB().QueryBuilder().BuildWhere(dbx.Exists(dbx.NewExp(fmt.Sprintf(`
		SELECT * FROM %s l, %s pl
		WHERE pl.pack = %s.id AND pl.level = l.id AND l.position BETWEEN {:min} AND {:max}`,
		list.LevelTableName,
		list.Packs.PackLevelTableName,
		list.Packs.PackTableName), dbx.Params{"min": minPos, "max": maxPos})), params)
	return updatePackPoints(dao, list, condition, params)
}

func updatePackPointsByPackId(dao *daos.Dao, list ListData, packId string) error {
	params := dbx.Params{}
	condition := dao.DB().QueryBuilder().BuildWhere(dbx.HashExp{"id": packId}, params)
	return updatePackPoints(dao, list, condition, params)
}

func updatePackPoints(dao *daos.Dao, list ListData, condition string, params dbx.Params) error {
	_, err := dao.DB().NewQuery(fmt.Sprintf(`
		UPDATE %s 
		SET points=(
			SELECT ROUND(SUM(l.points)*%v,1) 
			FROM %s pl, %v l 
			WHERE %s.id = pl.pack AND pl.level = l.id
		) %s`,
		list.Packs.PackTableName,
		list.Packs.PackMultiplier,
		list.Packs.PackLevelTableName,
		list.LevelTableName,
		list.Packs.PackTableName,
		condition,
	)).Bind(params).Execute()
	return err
}

func UpdateAllCompletedPacks(dao *daos.Dao, list ListData) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		_, err := txDao.DB().NewQuery(fmt.Sprintf(`
			DELETE FROM %s 
			WHERE (
				SELECT COUNT(*) FROM %s pl 
				WHERE pl.pack = %s.pack
			) <> (
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = %s.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			)`,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName,
			list.Packs.CompletedPacksTableName)).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM %s u, %s p 
			WHERE (
				SELECT COUNT(*) FROM %s pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) ON CONFLICT DO NOTHING`,
			list.Packs.CompletedPacksTableName,
			names.TableUsers,
			list.Packs.PackTableName,
			list.Packs.PackLevelTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName)).Execute()
		return err
	})
	return err
}

func updateCompletedPacksByUser(dao *daos.Dao, list ListData, userId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		_, err := txDao.DB().NewQuery(fmt.Sprintf(`
			DELETE FROM %s 
			WHERE (
				SELECT COUNT(*) FROM %s pl 
				WHERE pl.pack = %s.pack
			) <> (
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = %s.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			) AND user = {:userId}`,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName,
			list.Packs.CompletedPacksTableName,
		)).Bind(dbx.Params{"userId": userId}).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM %s u, %s p 
			WHERE (
				SELECT COUNT(*) FROM %s pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) AND u.id = {:userId} ON CONFLICT DO NOTHING`,
			list.Packs.CompletedPacksTableName,
			names.TableUsers,
			list.Packs.PackTableName,
			list.Packs.PackLevelTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName)).Bind(dbx.Params{"userId": userId}).Execute()
		return err
	})
	return err
}

func updateCompletedPacksByPackId(dao *daos.Dao, list ListData, packId string) ([]interface{}, error) {
	var usersToRemove []interface{}
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		type UserData struct {
			Id string `db:"user"`
		}
		var usersToRemoveData []UserData
		err := txDao.DB().NewQuery(fmt.Sprintf(`
			SELECT user
			FROM %s
			WHERE (
				SELECT COUNT(*) FROM %s pl 
				WHERE pl.pack = %s.pack
			) <> (
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = %s.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			) AND pack = {:packId}`,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.Packs.CompletedPacksTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName,
			list.Packs.CompletedPacksTableName)).Bind(dbx.Params{"packId": packId}).All(&usersToRemoveData)
		if err != nil {
			return err
		}
		usersToRemove = util.MapSlice(usersToRemoveData, func(value UserData) interface{} { return value.Id })
		_, err = txDao.DB().Delete(list.Packs.CompletedPacksTableName, dbx.And(dbx.In("user", usersToRemove...), dbx.HashExp{"pack": packId})).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM %s u, %s p 
			WHERE (
				SELECT COUNT(*) FROM %s pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM %s pl, %s rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) AND p.id = {:packId} ON CONFLICT DO NOTHING`,
			list.Packs.CompletedPacksTableName,
			names.TableUsers,
			list.Packs.PackTableName,
			list.Packs.PackLevelTableName,
			list.Packs.PackLevelTableName,
			list.SubmissionTableName)).Bind(dbx.Params{"packId": packId}).Execute()
		return err
	})
	return usersToRemove, err
}

func UpsertPack(dao *daos.Dao, app core.App, listData ListData, packData map[string]interface{}) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		maxPlacementPos, err := queryMaxPlacementPosition(dao, listData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to query max placement position")
		}
		var packRecord *models.Record
		var oldPos int
		if packData["id"] != nil {
			packRecordResult, err := txDao.FindRecordById(listData.Packs.PackTableName, packData["id"].(string))
			if err != nil {
				return util.NewErrorResponse(err, "Failed to fetch pack")
			}
			packRecord = packRecordResult
			oldPos = packRecord.GetInt("placement_order")
		} else {
			packCollection, err := txDao.FindCollectionByNameOrId(listData.Packs.PackTableName)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to fetch pack collection")
			}
			packRecord = models.NewRecord(packCollection)
			oldPos = maxPlacementPos + 1
			if packData["placement_order"] == nil {
				packData["placement_order"] = maxPlacementPos + 1
			}
		}
		if packData["placement_order"] != nil {
			newPos := packData["placement_order"].(int)
			if oldPos != newPos {
				_, err = txDao.DB().Update(
					listData.Packs.PackTableName,
					dbx.Params{"placement_order": dbx.NewExp("placement_order + {:inc}",
						dbx.Params{"inc": util.If(oldPos > newPos, 1, -1)})},
					dbx.Between("placement_order",
						mathutil.Min(newPos, oldPos),
						mathutil.Max(newPos, oldPos))).Execute()
				if err != nil {
					return util.NewErrorResponse(err, "Failed to update other level positions")
				}
			}
		}
		packForm := forms.NewRecordUpsert(app, packRecord)
		packForm.SetDao(txDao)
		err = packForm.LoadData(packData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to load pack data")
		}
		err = packForm.Submit()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to save pack")
		}
		if packData["levels"] != nil {
			type LevelData struct {
				Id string `db:"level"`
			}
			var oldLevelData []LevelData
			err = txDao.DB().Select("level").
				From(listData.Packs.PackLevelTableName).
				Where(dbx.HashExp{"pack": packRecord.Id}).
				All(&oldLevelData)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to fetch old pack levels")
			}
			oldLevels := util.MapSlice(oldLevelData, func(v LevelData) string { return v.Id })
			newLevels := packData["levels"].([]string)
			if len(newLevels) < 2 {
				return util.NewErrorResponse(nil, "Pack needs to have at least two levels")
			}
			addedLevels := list.SubtractSlice(newLevels, oldLevels)
			for _, level := range addedLevels {
				_, err = util.AddRecordByCollectionName(txDao, app, listData.Packs.PackLevelTableName, map[string]any{
					"level": level,
					"pack":  packRecord.Id,
				})
			}
			removedLevels := list.SubtractSlice(oldLevels, newLevels)
			_, err = txDao.DB().Delete(listData.Packs.PackLevelTableName, dbx.In("level", list.ToInterfaceSlice(removedLevels)...)).Execute()
			if err != nil {
				return util.NewErrorResponse(err, "Failed to delete removed levels")
			}
			err = updatePackPointsByPackId(txDao, listData, packRecord.Id)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to uptate pack points")
			}
			removedUsers, err := updateCompletedPacksByPackId(txDao, listData, packRecord.Id)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to update completed packs")
			}
			err = UpdateLeaderboardByUserIds(txDao, listData, removedUsers)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to update users that got their pack removed")
			}
			err = updateLeaderboardByPackId(txDao, listData, packRecord.Id)
			if err != nil {
				return util.NewErrorResponse(err, "Failed tu update users related to the pack")
			}
		}
		return nil
	})
	return err
}

func DeletePack(dao *daos.Dao, listData ListData, recordId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		packRecord, err := txDao.FindRecordById(listData.Packs.PackTableName, recordId)
		if err != nil {
			return util.NewErrorResponse(err, "Could not find pack")
		}
		_, err = txDao.DB().Delete(listData.Packs.PackLevelTableName, dbx.HashExp{"pack": packRecord.Id}).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete pack level")
		}
		type UserData struct {
			Id string `db:"user"`
		}
		var usersToUpdateData []UserData
		err = txDao.DB().NewQuery(fmt.Sprintf(`
			DELETE FROM %s
			WHERE pack = {:packId}
			RETURNING user`,
			listData.Packs.CompletedPacksTableName)).
			Bind(dbx.Params{"packId": packRecord.Id}).
			All(&usersToUpdateData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete completed packs")
		}
		_, err = txDao.DB().Update(
			listData.Packs.PackTableName,
			dbx.Params{"placement_order": dbx.NewExp("placement_order - 1")},
			dbx.NewExp("placement_order >={:placement}", dbx.Params{"placement": packRecord.GetInt("placement_order")})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to move other level")
		}
		err = txDao.DeleteRecord(packRecord)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to delete pack")
		}
		err = UpdateLeaderboardByUserIds(txDao, listData, util.MapSlice(usersToUpdateData, func(v UserData) interface{} { return v.Id }))
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update users")
		}
		return nil
	})
	return err
}

func queryMaxPlacementPosition(dao *daos.Dao, listData ListData) (int, error) {
	var position int
	err := dao.DB().Select("max(placement_order)").From(listData.Packs.PackTableName).Row(&position)
	if err != nil {
		return 0, err
	}
	return position, nil
}
