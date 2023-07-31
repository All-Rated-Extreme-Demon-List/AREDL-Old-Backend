package points

import (
	"AREDL/names"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
)

func UpdateListPointsByLevelRange(dao *daos.Dao, minPos int, maxPos int) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		query := txDao.DB().NewQuery(`UPDATE ` + names.TableLevels + `
		SET points=(
			SELECT p.points 
			FROM ` + names.TablePoints + ` p 
			WHERE p.id=position 
		)
		WHERE position BETWEEN {:minPos} AND {:maxPos}`).Bind(dbx.Params{
			"minPos": minPos,
			"maxPos": maxPos,
		})

		_, err := query.Execute()
		if err != nil {
			return err
		}
		err = updatePackPointsByLevelRange(txDao, minPos, maxPos)
		if err != nil {
			return err
		}
		err = updateUserPointsByLevelRange(txDao, minPos, maxPos)
		return err
	})
	return err
}
