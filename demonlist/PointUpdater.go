package demonlist

import (
	"AREDL/names"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"gopkg.in/Knetic/govaluate.v2"
	"math"
)

func RegisterUpdatePoints(app core.App) {
	app.OnRecordBeforeUpdateRequest(names.TablePointFormular).Add(func(e *core.RecordUpdateEvent) error {
		functions := map[string]govaluate.ExpressionFunction{
			"sqrt": func(args ...interface{}) (interface{}, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("sqrt exactly takes one argument")
				}
				value, ok := args[0].(float64)
				if !ok {
					return nil, fmt.Errorf("argument must be a float64 for sqrt")
				}
				return math.Sqrt(value), nil
			},
		}
		formular, err := govaluate.NewEvaluableExpressionWithFunctions(e.Record.GetString("formula"), functions)
		if err != nil {
			return err
		}
		var list ListData
		listName := e.Record.GetString("list")
		switch listName {
		case "aredl":
			list = Aredl()
		default:
			return fmt.Errorf("unknown list %s", listName)
		}
		parameters := make(map[string]interface{}, 1)
		generateTo := e.Record.GetInt("generate_to")
		err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			_, err := txDao.DB().Delete(list.PointLookupTableName, nil).Execute()
			if err != nil {
				return err
			}
			for i := 1; i <= generateTo; i++ {
				parameters["x"] = float64(i)
				result, err := formular.Evaluate(parameters)
				if err != nil {
					return err
				}
				value, ok := result.(float64)
				if !ok {
					return fmt.Errorf("resulting value is not a float64")
				}
				if value < 0.0 {
					value = 0.0
				}
				_, err = txDao.DB().Insert(list.PointLookupTableName, dbx.Params{
					"id":     i,
					"points": fmt.Sprintf("%.1f", math.Round(value*10)/10),
				}).Execute()
				if err != nil {
					return err
				}
			}
			return UpdateLevelListPointsByPositionRange(txDao, list, 1, generateTo)
		})
		return err
	})
}
