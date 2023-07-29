package points

import (
	"AREDL/names"
	"AREDL/util"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
)

const packMultiplier = "0.5"

func updatePackPointsByLevelRange(dao *daos.Dao, minPos int, maxPos int) error {
	_, err := dao.DB().NewQuery(`
			UPDATE ` + names.TablePacks + ` 
			SET points=(
				SELECT ROUND(SUM(l.points)*` + packMultiplier + `,1) 
				FROM ` + names.TablePackLevels + ` pl, ` + names.TableLevels + ` l 
				WHERE ` + names.TablePacks + `.id = pl.pack AND pl.level = l.id
			) WHERE EXISTS (
				SELECT * FROM ` + names.TableLevels + ` l, ` + names.TablePackLevels + ` pl 
				WHERE pl.pack = ` + names.TablePacks + `.id AND pl.level = l.id AND l.position BETWEEN {:min} AND {:max}
			)`).Bind(dbx.Params{"min": minPos, "max": maxPos}).Execute()
	return err
}

func UpdatePackPointsByPackId(dao *daos.Dao, packId string) error {
	_, err := dao.DB().NewQuery(`
			UPDATE ` + names.TablePacks + ` 
			SET points=(
				SELECT ROUND(SUM(l.points)*` + packMultiplier + `,1) 
				FROM ` + names.TablePackLevels + ` pl, ` + names.TableLevels + ` l 
				WHERE ` + names.TablePacks + `.id = pl.pack AND pl.level = l.id
			) WHERE id = {:packId}
			`).Bind(dbx.Params{"packId": packId}).Execute()
	return err
}

func UpdateAllCompletedPacks(dao *daos.Dao) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		_, err := txDao.DB().NewQuery(`
			DELETE FROM ` + names.TableCompletedPacks + ` 
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack
			) <> (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			)`).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(`
			INSERT INTO ` + names.TableCompletedPacks + ` (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM ` + names.TableUsers + ` u, ` + names.TablePacks + ` p 
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) ON CONFLICT DO NOTHING`).Execute()
		return err
	})
	return err
}

func UpdateCompletedPacksByUser(dao *daos.Dao, userId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		_, err := txDao.DB().NewQuery(`
			DELETE FROM ` + names.TableCompletedPacks + ` 
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack
			) <> (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			) AND user = {:userId}`).Bind(dbx.Params{"userId": userId}).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(`
			INSERT INTO ` + names.TableCompletedPacks + ` (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM ` + names.TableUsers + ` u, ` + names.TablePacks + ` p 
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) AND u.id = {:userId} ON CONFLICT DO NOTHING`).Bind(dbx.Params{"userId": userId}).Execute()
		return err
	})
	return err
}

func UpdateCompletedPacksByPackId(dao *daos.Dao, packId string) ([]interface{}, error) {
	var usersToRemove []interface{}
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		type UserData struct {
			Id string `db:"user"`
		}
		var usersToRemoveData []UserData
		err := txDao.DB().NewQuery(`
			SELECT user
			FROM ` + names.TableCompletedPacks + `
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack
			) <> (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = ` + names.TableCompletedPacks + `.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			) AND pack = {:packId}`).Bind(dbx.Params{"packId": packId}).All(&usersToRemoveData)
		if err != nil {
			return err
		}
		usersToRemove = util.MapSlice(usersToRemoveData, func(value UserData) interface{} { return value.Id })
		_, err = txDao.DB().Delete(names.TableCompletedPacks, dbx.And(dbx.In("user", usersToRemove...), dbx.HashExp{"pack": packId})).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(`
			INSERT INTO ` + names.TableCompletedPacks + ` (user, pack) 
			SELECT u.id as user, p.id as pack 
			FROM ` + names.TableUsers + ` u, ` + names.TablePacks + ` p 
			WHERE (
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM ` + names.TablePackLevels + ` pl, ` + names.TableSubmissions + ` rs 
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) AND p.id = {:packId} ON CONFLICT DO NOTHING`).Bind(dbx.Params{"packId": packId}).Execute()
		return err
	})
	return usersToRemove, err
}
