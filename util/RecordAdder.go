package util

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
)

func AddRecordByCollectionName(dao *daos.Dao, app core.App, collectionName string, values map[string]any) (*models.Record, error) {
	collection, err := dao.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, err
	}
	return AddRecord(dao, app, collection, values)
}

func AddRecord(dao *daos.Dao, app core.App, collection *models.Collection, values map[string]any) (*models.Record, error) {
	record := models.NewRecord(collection)
	recordForm := forms.NewRecordUpsert(app, record)
	recordForm.SetDao(dao)
	err := recordForm.LoadData(values)
	if err != nil {
		return nil, err
	}
	err = recordForm.Submit()
	if err != nil {
		return nil, err
	}
	return record, nil
}
