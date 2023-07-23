package points

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/pocketbase/daos"
)

func UpdateListPoints(dao *daos.Dao, minPos int, maxPos int) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		query := txDao.DB().NewQuery(fmt.Sprintf(`UPDATE %v 
		SET points=(
			SELECT p.points 
			FROM %v p 
			WHERE p.id=position 
		)
		WHERE position BETWEEN %d AND %d`, names.TableLevels, names.TablePoints, minPos, maxPos))

		_, err := query.Execute()
		if err != nil {
			return err
		}
		err = updatePackPointsByLevelRange(txDao, minPos, maxPos)
		if err != nil {
			return err
		}
		err = updateUserPointsByLevelRange(txDao, minPos, maxPos)
		return nil
	})
	return err
}
