package points

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/pocketbase/daos"
)

func UpdateUserPointsByLevelRange(dao *daos.Dao, minPos int, maxPos int) error {
	return updateUserPoints(dao, fmt.Sprintf(`
		EXISTS (
			SELECT * FROM %v rs, %v l
			WHERE %v.id = rs.submitted_by AND rs.status = 'accepted' AND rs.level = l.id AND l.position BETWEEN %v AND %v
		)
		`, names.TableSubmissions, names.TableLevels, names.TableUsers, minPos, maxPos))
}

func updateUserPoints(dao *daos.Dao, condition string) error {
	if condition != "" {
		condition = fmt.Sprintf(`WHERE %v`, condition)
	}
	query := dao.DB().NewQuery(fmt.Sprintf(`
		UPDATE %v
		SET aredl_points = ROUND(
    	(
    		SELECT ROUND(COALESCE(SUM(l.points), 0), 1)
    		FROM %v rs, %v l
    		WHERE %v.id = rs.submitted_by AND rs.level = l.id AND rs.status = 'accepted'
    	) + (
    		SELECT ROUND(COALESCE(SUM(p.points), 0), 1)
    		FROM %v cp, %v p
    		WHERE %v.id = cp.user AND cp.pack = p.id
    	), 1) %v`, names.TableUsers, names.TableSubmissions, names.TableLevels, names.TableUsers, names.TableCompletedPacks, names.TablePacks, names.TableUsers, condition))
	_, err := query.Execute()
	return err
}

func UpdateCompletedPacks(dao *daos.Dao) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		query := txDao.DB().NewQuery(fmt.Sprintf(`DELETE FROM %v
			WHERE (
				SELECT COUNT(*) FROM %v pl WHERE pl.pack = %v.pack
			) <> (
				SELECT COUNT(*) FROM %v pl, %v rs
				WHERE pl.pack = %v.pack AND pl.level = rs.level AND rs.submitted_by = user AND rs.status='accepted'
			)`, names.TableCompletedPacks, names.TablePackLevels, names.TableCompletedPacks, names.TablePackLevels, names.TableSubmissions, names.TableCompletedPacks))
		_, err := query.Execute()
		if err != nil {
			return err
		}
		query = txDao.DB().NewQuery(fmt.Sprintf(
			`INSERT INTO %v (user, pack)
			SELECT u.id as user, p.id as pack
			FROM %v u, %v p
			WHERE (
				SELECT COUNT(*) FROM %v pl WHERE pl.pack = p.id
			)=(
				SELECT COUNT(*) FROM %v pl, %v rs
				WHERE pl.pack = p.id AND rs.submitted_by = u.id AND rs.level = pl.level AND rs.status = 'accepted'
			) ON CONFLICT DO NOTHING`, names.TableCompletedPacks, names.TableUsers, names.TablePacks, names.TablePackLevels, names.TablePackLevels, names.TableSubmissions))
		_, err = query.Execute()
		return err
	})
	return err
}
