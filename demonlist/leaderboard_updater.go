package demonlist

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
)

func updateLeaderboardByLevelRange(dao *daos.Dao, listData ListData, minPos int, maxPos int) error {
	params := dbx.Params{}
	condition := dao.DB().QueryBuilder().BuildWhere(dbx.Exists(dbx.NewExp(fmt.Sprintf(`
		SELECT NULL FROM %s rs, %s l 
		WHERE u.id = rs.submitted_by AND rs.level = l.id AND l.position BETWEEN {:min} AND {:max}`,
		listData.RecordsTableName,
		listData.LevelTableName,
	), dbx.Params{"min": minPos, "max": maxPos})), params)
	return updateLeaderboard(dao, listData, condition, params)
}

func UpdateLeaderboardByUserIds(dao *daos.Dao, listData ListData, userIds []interface{}) error {
	params := dbx.Params{}
	condition := dao.DB().QueryBuilder().BuildWhere(dbx.In("user", userIds...), params)
	return updateLeaderboard(dao, listData, condition, params)
}

func updateLeaderboardByPackId(dao *daos.Dao, listData ListData, packId string) error {
	params := dbx.Params{}
	condition := dao.DB().QueryBuilder().BuildWhere(dbx.Exists(dbx.NewExp(fmt.Sprintf(`
		SELECT NULL FROM %s cp WHERE cp.user = u.id AND cp.pack = {:packId}`,
		listData.Packs.CompletedPacksTableName),
		dbx.Params{"packId": packId})), params)
	return updateLeaderboard(dao, listData, condition, params)
}

func updateLeaderboard(dao *daos.Dao, listData ListData, condition string, params dbx.Params) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		_, err := txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (user, points) 
			SELECT u.id as user, (
				ROUND(
    			(
    				SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    				FROM %s rs, %s l
    				WHERE u.id = rs.submitted_by AND rs.level = l.id
    			) + (
    				SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    				FROM %s cp, %s p
    				WHERE u.id = cp.user AND cp.pack = p.id
    			), 1)
			) as points 
			FROM %s u 
			%s 
			ON CONFLICT DO UPDATE SET points = excluded.points`,
			listData.LeaderboardTableName,
			listData.RecordsTableName,
			listData.LevelTableName,
			listData.Packs.CompletedPacksTableName,
			listData.Packs.PackTableName,
			names.TableUsers,
			condition)).Bind(params).Execute()
		if err != nil {
			return err
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			DELETE FROM %s
			WHERE user IN (
				SELECT u.id
				FROM %s al
				JOIN %s u ON u.id = al.user
				WHERE u.banned_from_list = 1 OR al.points = 0
			)`,
			listData.LeaderboardTableName,
			listData.LeaderboardTableName,
			names.TableUsers)).Execute()
		if err != nil {
			return err
		}
		err = updateLeaderboardRanks(txDao, listData)
		return err
	})
	return err
}

func updateLeaderboardRanks(dao *daos.Dao, listData ListData) error {
	_, err := dao.DB().NewQuery(fmt.Sprintf(`
		WITH ranking AS (
			SELECT user, RANK() OVER (ORDER BY points DESC) AS position 
			FROM %s
		)
		UPDATE %s 
		SET rank = position
		FROM ranking
		WHERE ranking.user = %s.user`,
		listData.LeaderboardTableName,
		listData.LeaderboardTableName,
		listData.LeaderboardTableName)).Execute()
	return err
}

func UpdateLeaderboardAndPacksForUser(dao *daos.Dao, listData ListData, userId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		err := updateCompletedPacksByUser(txDao, listData, userId)
		if err != nil {
			return err
		}
		return UpdateLeaderboardByUserIds(txDao, listData, []interface{}{userId})
	})
	return err
}
