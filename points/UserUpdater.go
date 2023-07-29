package points

import (
	"AREDL/names"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
)

func updateUserPointsByLevelRange(dao *daos.Dao, minPos int, maxPos int) error {
	_, err := dao.DB().NewQuery(`
		UPDATE ` + names.TableUsers + `
		SET aredl_points = ROUND(
    	(
    		SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    		FROM ` + names.TableSubmissions + ` rs, ` + names.TableLevels + ` l
    		WHERE ` + names.TableUsers + `.id = rs.submitted_by AND rs.level = l.id AND rs.status = 'accepted'
    	) + (
    		SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    		FROM ` + names.TableCompletedPacks + ` cp, ` + names.TablePacks + ` p
    		WHERE ` + names.TableUsers + `.id = cp.user AND cp.pack = p.id
    	), 1)
		WHERE EXISTS (
			SELECT * FROM ` + names.TableSubmissions + ` rs, ` + names.TableLevels + ` l
			WHERE ` + names.TableUsers + `.id = rs.submitted_by AND rs.status = 'accepted' AND rs.level = l.id AND l.position BETWEEN {:min} AND {:max}
		)
	`).Bind(dbx.Params{"min": minPos, "max": maxPos}).Execute()
	return err
}

func UpdateUserPointsByUserIds(dao *daos.Dao, userIds ...interface{}) error {
	params := dbx.Params{}
	_, err := dao.DB().NewQuery(`
		UPDATE ` + names.TableUsers + `
		SET aredl_points = ROUND(
    	(
    		SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    		FROM ` + names.TableSubmissions + ` rs, ` + names.TableLevels + ` l
    		WHERE ` + names.TableUsers + `.id = rs.submitted_by AND rs.level = l.id AND rs.status = 'accepted'
    	) + (
    		SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    		FROM ` + names.TableCompletedPacks + ` cp, ` + names.TablePacks + ` p
    		WHERE ` + names.TableUsers + `.id = cp.user AND cp.pack = p.id
    	), 1)
		` + dao.DB().QueryBuilder().BuildWhere(dbx.In("id", userIds...), params) + `
	`).Bind(params).Execute()
	query := dao.DB().NewQuery(`
		UPDATE ` + names.TableUsers + `
		SET aredl_points = ROUND(
    	(
    		SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    		FROM ` + names.TableSubmissions + ` rs, ` + names.TableLevels + ` l
    		WHERE ` + names.TableUsers + `.id = rs.submitted_by AND rs.level = l.id AND rs.status = 'accepted'
    	) + (
    		SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    		FROM ` + names.TableCompletedPacks + ` cp, ` + names.TablePacks + ` p
    		WHERE ` + names.TableUsers + `.id = cp.user AND cp.pack = p.id
    	), 1)
		` + dao.DB().QueryBuilder().BuildWhere(dbx.In("id", userIds...), params) + `
	`).Bind(params)
	println(query.SQL())
	return err
}

func UpdateUserPointsByPackId(dao *daos.Dao, packId string) error {
	_, err := dao.DB().NewQuery(`
		UPDATE ` + names.TableUsers + `
		SET aredl_points = ROUND(
    	(
    		SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    		FROM ` + names.TableSubmissions + ` rs, ` + names.TableLevels + ` l
    		WHERE ` + names.TableUsers + `.id = rs.submitted_by AND rs.level = l.id AND rs.status = 'accepted'
    	) + (
    		SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    		FROM ` + names.TableCompletedPacks + ` cp, ` + names.TablePacks + ` p
    		WHERE ` + names.TableUsers + `.id = cp.user AND cp.pack = p.id
    	), 1)
		WHERE EXISTS (
			SELECT * FROM ` + names.TableCompletedPacks + ` cp WHERE cp.user = ` + names.TableUsers + `.id AND cp.pack = {:packId}
		)
	`).Bind(dbx.Params{"packId": packId}).Execute()
	return err
}
