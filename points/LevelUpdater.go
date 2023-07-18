package points

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/pocketbase/daos"
)

func UpdateListPoints(txDao *daos.Dao, minPos int, maxPos int) error {
	query := txDao.DB().NewQuery(fmt.Sprintf("UPDATE %v "+
		"SET points=("+
		"	SELECT p.points "+
		"	FROM %v p "+
		"	WHERE p.id=position "+
		")"+
		"WHERE position BETWEEN %d AND %d",
		txDao.DB().QuoteSimpleTableName(names.TableLevels),
		txDao.DB().QuoteSimpleTableName(names.TablePoints),
		minPos, maxPos))

	println(query.SQL())

	_, err := query.Execute()
	return err
}
