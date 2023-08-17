package queryhelper

import (
	"fmt"
	"github.com/pocketbase/dbx"
	"reflect"
)

type Extend struct {
	FieldName string
	Fields    []interface{}
}

func nextTableAlias(tableFromPrefix map[string]string, tablePrefix string) string {
	alias := fmt.Sprintf("t%v", len(tableFromPrefix))
	tableFromPrefix[tablePrefix] = alias
	return alias
}

func Build(db dbx.Builder, toLoad interface{}, fields []interface{}) (*dbx.SelectQuery, map[string]string, error) {
	toLoadType := reflect.TypeOf(toLoad)
	if toLoadType.Kind() == reflect.Ptr {
		toLoadType = toLoadType.Elem()
	}
	if toLoadType.Kind() == reflect.Array || toLoadType.Kind() == reflect.Slice {
		toLoadType = toLoadType.Elem()
	}
	tableFromPrefix := make(map[string]string)
	query := db.Select()
	tableName := dbx.GetTableName(reflect.Zero(toLoadType).Interface())
	if tableName == "" {
		return nil, nil, fmt.Errorf("could not load table name")
	}
	alias := nextTableAlias(tableFromPrefix, "")
	query.From(fmt.Sprintf("%v %v", tableName, alias))
	err := loadFields(query, toLoadType, "", tableFromPrefix, fields)
	return query, tableFromPrefix, err
}

func loadFields(query *dbx.SelectQuery, toLoad reflect.Type, currentPrefix string, tableFromPrefix map[string]string, fields []interface{}) error {
	currentTableName := tableFromPrefix[currentPrefix]
	var err error
	for _, field := range fields {
		switch field.(type) {
		case string:
			name := field.(string)
			if currentPrefix != "" {
				query.AndSelect(fmt.Sprintf("%v.%v AS %v%v", currentTableName, name, currentPrefix, name))
			} else {
				query.AndSelect(fmt.Sprintf("%v.%v", currentTableName, name))
			}
		case Extend:
			extend := field.(Extend)
			loadedField, ok := toLoad.FieldByName(extend.FieldName)
			if !ok {
				return fmt.Errorf("could not find extend field %v", extend.FieldName)
			}
			extendName, ok := loadedField.Tag.Lookup("db")
			if !ok {
				return fmt.Errorf("could extend field %v has to have db tag", extend.FieldName)
			}
			newPrefix := currentPrefix + extendName + "."
			tableName := dbx.GetTableName(reflect.Zero(loadedField.Type).Interface())
			if tableName == "" {
				return fmt.Errorf("could not load table name for extend %v", newPrefix)
			}
			alias := nextTableAlias(tableFromPrefix, newPrefix)
			query.InnerJoin(fmt.Sprintf("%v %v", tableName, alias), dbx.NewExp(fmt.Sprintf("%v.%v = %v.id", currentTableName, extendName, alias)))
			err = loadFields(query, loadedField.Type, newPrefix, tableFromPrefix, extend.Fields)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("fields may only contain")
		}
	}
	return nil
}
