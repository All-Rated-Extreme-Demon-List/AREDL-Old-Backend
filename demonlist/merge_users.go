package demonlist

import (
	"AREDL/names"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"modernc.org/sqlite"
)

func MergeUsers(dao *daos.Dao, primaryId, secondaryId string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		aredl := Aredl()
		primaryUser, err := txDao.FindRecordById(names.TableUsers, primaryId)
		if err != nil {
			return err
		}
		secondaryUser, err := txDao.FindRecordById(names.TableUsers, secondaryId)
		if err != nil {
			return err
		}
		renassignTables := []struct {
			Name  string
			Field string
		}{
			{aredl.SubmissionsTableName, "submitted_by"},
			{aredl.RecordsTableName, "submitted_by"},
			{aredl.RecordsTableName, "reviewer"},
			{aredl.HistoryTableName, "action_by"},
			{aredl.CreatorTableName, "creator"},
			{names.TableNameChangeRequests, "user"},
			{names.TableRoles, "user"},
		}
		deleteTables := []struct {
			Name  string
			Field string
		}{
			{aredl.LeaderboardTableName, "user"},
			{aredl.Packs.CompletedPacksTableName, "user"},
		}
		for _, table := range renassignTables {
			err = mergeTableData(txDao, primaryId, secondaryId, table.Name, table.Field)
			if err != nil {
				return err
			}
		}
		for _, table := range deleteTables {
			_, err = txDao.DB().Delete(table.Name, dbx.HashExp{table.Field: secondaryId}).Execute()
			if err != nil {
				return err
			}
		}
		err = txDao.DeleteRecord(secondaryUser)
		if err != nil {
			return err
		}
		err = updateCompletedPacksByUser(txDao, aredl, primaryUser.Id)
		if err != nil {
			return err
		}
		err = UpdateLeaderboardByUserIds(txDao, aredl, []interface{}{primaryUser.Id})
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func mergeTableData(dao *daos.Dao, primaryId, secondaryId, tableName, fieldName string) error {
	records, err := dao.FindRecordsByExpr(tableName, dbx.HashExp{fieldName: secondaryId})
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set(fieldName, primaryId)
		err = dao.SaveRecord(record)
		if err != nil {
			var sqliteErr *sqlite.Error
			// check if uniqueness constraint fails (for example with duplicate records)
			if errors.As(err, &sqliteErr) && sqliteErr.Code() == 2067 {
				err = dao.DeleteRecord(record)
				if err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}
