package util

import (
	"fmt"
	"github.com/pocketbase/dbx"
	"reflect"
	"strings"
)

type PrefixResolver func(string) string

type QueryExtender func(*dbx.SelectQuery, PrefixResolver)

func nextTableAlias(tableFromPrefix map[string]string, tablePrefix string) string {
	alias := fmt.Sprintf("t%v", len(tableFromPrefix))
	tableFromPrefix[tablePrefix] = alias
	return alias
}

func LoadFromDb(db dbx.Builder, toLoad interface{}, tableNames map[string]string, queryExtender QueryExtender) error {
	toLoadType := reflect.TypeOf(toLoad)
	if toLoadType.Kind() != reflect.Ptr {
		return fmt.Errorf("value has to be a pointer")
	}
	toLoadType = toLoadType.Elem()
	if toLoadType.Kind() == reflect.Ptr {
		toLoadType = toLoadType.Elem()
	}
	isArray := false
	if toLoadType.Kind() == reflect.Array || toLoadType.Kind() == reflect.Slice {
		isArray = true
		toLoadType = toLoadType.Elem()
	}
	tableFromPrefix := make(map[string]string)
	query := db.Select()
	baseTableName, ok := tableNames["base"]
	if !ok {
		return fmt.Errorf("tables must include a base entry")
	}
	alias := nextTableAlias(tableFromPrefix, "")
	query.From(fmt.Sprintf("%v %v", baseTableName, alias))
	err := loadFields(query, toLoadType, false, "", tableFromPrefix, tableNames)
	if err != nil {
		return err
	}

	queryExtender(query, func(fieldString string) string {
		splitIndex := strings.LastIndex(fieldString, ".")
		if splitIndex == -1 {
			return fmt.Sprintf("%v.%v", tableFromPrefix[""], fieldString)
		}
		return fmt.Sprintf("%v.%v", tableFromPrefix[fieldString[:splitIndex+1]], fieldString[splitIndex+1:])
	})

	if isArray {
		return query.All(toLoad)
	}
	return query.One(toLoad)
}

func loadFields(query *dbx.SelectQuery, toLoad reflect.Type, optional bool, currentPrefix string, tableFromPrefix map[string]string, tableNames map[string]string) error {
	currentTableName := tableFromPrefix[currentPrefix]
	if toLoad.Kind() == reflect.Ptr {
		toLoad = toLoad.Elem()
	}
	var err error
	for index := 0; index < toLoad.NumField(); index++ {
		currentOptional := optional
		field := toLoad.Field(index)
		dbName, ok := field.Tag.Lookup("db")
		if !ok {
			continue
		}
		if field.Type.Kind() == reflect.Ptr {
			currentOptional = true
		}
		extendData, extend := field.Tag.Lookup("extend")
		if extend {
			extendDataSplit := strings.Split(extendData, ",")
			if len(extendDataSplit) != 3 {
				return fmt.Errorf("%v extend field has to have exactly two values separated by comma", field.Name)
			}
			srcName := extendDataSplit[0]
			destName := extendDataSplit[2]
			tableName, ok := tableNames[extendDataSplit[1]]
			if !ok {
				return fmt.Errorf("%v missing table tag %v", field.Name, extendDataSplit[1])
			}

			newPrefix := currentPrefix + dbName + "."

			alias := nextTableAlias(tableFromPrefix, newPrefix)
			if currentOptional {
				query.LeftJoin(fmt.Sprintf("%v %v", tableName, alias), dbx.NewExp(fmt.Sprintf("%v.%v = %v.%v", currentTableName, srcName, alias, destName)))
			} else {
				query.InnerJoin(fmt.Sprintf("%v %v", tableName, alias), dbx.NewExp(fmt.Sprintf("%v.%v = %v.%v", currentTableName, srcName, alias, destName)))
			}
			err = loadFields(query, field.Type, currentOptional, newPrefix, tableFromPrefix, tableNames)
			if err != nil {
				return err
			}
		} else {
			fieldName := fmt.Sprintf("%v.%v", currentTableName, dbName)
			if currentOptional {
				fieldName = fmt.Sprintf("COALESCE(%v, '')", fieldName)
			}
			if currentPrefix != "" {
				query.AndSelect(fmt.Sprintf("%v AS %v%v", fieldName, currentPrefix, dbName))
			} else {
				query.AndSelect(fieldName)
			}
		}
	}
	return nil
}

func IsNotNoResultError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() != "sql: no rows in result set"
}
