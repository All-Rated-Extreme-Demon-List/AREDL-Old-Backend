package points

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/pocketbase/daos"
)

const packMultiplier = 0.5

func updatePackPointsByLevelRange(dao *daos.Dao, minPos int, maxPos int) error {
	return updatePackPoints(dao, fmt.Sprintf(`
		EXISTS (
			SELECT * FROM %v l, %v pl
			WHERE pl.pack = %v.id AND pl.level = l.id AND l.position BETWEEN %v AND %v
		)`, names.TableLevels, names.TablePackLevels, names.TablePacks, minPos, maxPos))
}

func updatePackPoints(dao *daos.Dao, condition string) error {
	if condition != "" {
		condition = fmt.Sprintf(`WHERE %v`, condition)
	}
	query := dao.DB().NewQuery(fmt.Sprintf(
		`UPDATE %v
			SET points=(
				SELECT ROUND(SUM(l.points)* %f,1)
				FROM %v pl, %v l
				WHERE %v.id = pl.pack AND pl.level = l.id
			) %v`,
		names.TablePacks, packMultiplier, names.TablePackLevels, names.TableLevels, names.TablePacks, condition))
	_, err := query.Execute()
	return err
}
