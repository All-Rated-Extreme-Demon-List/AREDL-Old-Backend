package demonlist

import (
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/tools/list"
	"gopkg.in/Knetic/govaluate.v2"
	"math"
	"modernc.org/mathutil"
	"strings"
)

func PlaceLevel(dao *daos.Dao, app core.App, userId string, listData ListData, levelData map[string]interface{}, verificationData map[string]interface{}, creatorIds []string) error {
	var legacy bool
	if levelData["legacy"] == nil {
		levelData["legacy"] = false
		legacy = false
	} else {
		legacy = levelData["legacy"].(bool)
	}
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		highestPosition, err := queryMaxPosition(txDao, listData, legacy)
		if err != nil {
			return util.NewErrorResponse(err, "Could not query max pos")
		}
		lowestPosition := 1
		if legacy {
			lowestPosition, err = queryMaxPosition(txDao, listData, false)
			if err != nil {
				return util.NewErrorResponse(err, "Could not query low pos")
			}
			lowestPosition++
		}
		position := levelData["position"].(int)
		if position > highestPosition || position < lowestPosition {
			return util.NewErrorResponse(err, "New position is outside the list")
		}
		_, err = txDao.DB().Update(listData.LevelTableName,
			dbx.Params{"position": dbx.NewExp("position+1")}, dbx.NewExp("position>={:position}", dbx.Params{"position": position})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to move other levels")
		}
		levelRecord, err := util.AddRecordByCollectionName(txDao, app, listData.LevelTableName, levelData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to add new level")
		}
		err = updateCreators(txDao, listData, levelRecord.Id, creatorIds)
		if err != nil {
			return err
		}
		verificationData["level"] = levelRecord.Id
		verificationData["reviewer"] = userId
		verificationData["placement_order"] = 1
		_, err = util.AddRecordByCollectionName(txDao, app, listData.RecordsTableName, verificationData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to add verification")
		}
		err = UpdatePointTable(txDao, listData)
		if err != nil {
			return err
		}
		err = UpdateLevelListPointsByPositionRange(txDao, listData, 1, highestPosition)
		if err != nil {
			return err
		}
		// history
		_, err = util.AddRecordByCollectionName(txDao, app, listData.HistoryTableName, map[string]any{
			"level":        levelRecord.Id,
			"action":       "placed",
			"new_position": position,
			"cause":        levelRecord.Id,
			"action_by":    userId,
		})
		if err != nil {
			return util.NewErrorResponse(err, "Failed to add placement into history")
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (level, action, new_position, cause, action_by)
			SELECT l.id AS level, {:action} AS action, l.position AS new_position, {:cause} AS cause, {:action_by} AS action_by
			FROM %s l
			WHERE l.position > {:position}`,
			listData.HistoryTableName,
			listData.LevelTableName)).Bind(dbx.Params{
			"action":    "placedAbove",
			"cause":     levelRecord.Id,
			"action_by": userId,
			"position":  position,
		}).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update level history")
		}
		return nil
	})
	return err
}

func UpdateLevel(dao *daos.Dao, app core.App, recordId string, userId string, listData ListData, levelData map[string]interface{}, creatorIds interface{}) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		if levelData["position"] != nil || levelData["legacy"] != nil {
			err := moveLevel(txDao, app, listData, recordId, userId, levelData["legacy"], levelData["position"])
			if err != nil {
				return err
			}
		}
		levelRecord, err := txDao.FindRecordById(listData.LevelTableName, recordId)
		if err != nil {
			return util.NewErrorResponse(err, "Level not found")
		}
		levelForm := forms.NewRecordUpsert(app, levelRecord)
		levelForm.SetDao(txDao)
		err = levelForm.LoadData(levelData)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update level")
		}
		if creatorIds != nil {
			err = updateCreators(txDao, listData, recordId, creatorIds.([]string))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func moveLevel(dao *daos.Dao, app core.App, listData ListData, levelId string, userId string, legacyI interface{}, newPosI interface{}) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		levelRecord, err := txDao.FindRecordById(listData.LevelTableName, levelId)
		if err != nil {
			return util.NewErrorResponse(err, "Could not find level")
		}
		legacy := levelRecord.GetBool("legacy")
		if legacyI != nil {
			legacy = legacyI.(bool)
		}
		legacyChanged := levelRecord.GetBool("legacy") != legacy
		oldPos := levelRecord.GetInt("position")
		newPos := oldPos
		if newPosI != nil {
			newPos = newPosI.(int)
		}
		if newPos == oldPos && !legacyChanged {
			// nothing changed
			return nil
		}
		if legacyChanged {
			levelRecord.Set("legacy", legacy)
			err = txDao.SaveRecord(levelRecord)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to update legacy")
			}
		}
		var lowestPosition, highestPosition int
		if !legacy {
			lowestPosition = 1
			highestPosition, err = queryMaxPosition(txDao, listData, false)
			if err != nil {
				return util.NewErrorResponse(err, "Could not query max pos")
			}
			if legacyChanged {
				highestPosition++
			}
		} else {
			lowestPosition, err = queryMaxPosition(txDao, listData, false)
			if err != nil {
				return util.NewErrorResponse(err, "Could not query max pos")
			}
			if !legacyChanged {
				lowestPosition++
			}
			highestPosition, err = queryMaxPosition(txDao, listData, true)
			if err != nil {
				return util.NewErrorResponse(err, "Could not query max pos including legacy")
			}
		}
		if newPos > highestPosition || newPos < lowestPosition {
			return util.NewErrorResponse(err, fmt.Sprintf("New position is outside the applicable range (%v, %v)", lowestPosition, highestPosition))
		}
		moveUp := newPos < oldPos
		_, err = txDao.DB().Update(
			listData.LevelTableName,
			dbx.Params{"position": dbx.NewExp("CASE WHEN position = {:old} THEN {:new} ELSE position + {:inc} END",
				dbx.Params{"old": oldPos, "new": newPos, "inc": util.If(moveUp, 1, -1)})},
			dbx.Between("position",
				mathutil.Min(newPos, oldPos),
				mathutil.Max(newPos, oldPos))).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update positions")
		}
		levelMovedStatus := util.If(moveUp, "movedUp", "movedDown")
		levelOtherStatus := util.If(moveUp, "movedPastUp", "movedPastDown")
		if legacyChanged {
			// moved into or out of legacy
			levelMovedStatus = util.If(legacy, "movedToLegacy", "movedFromLegacy")
			err := UpdatePointTable(txDao, listData)
			if err != nil {
				return err
			}
		}
		err = UpdateLevelListPointsByPositionRange(txDao, listData, mathutil.Min(newPos, oldPos), mathutil.Max(newPos, oldPos))
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update level listData points")
		}
		// write to history
		_, err = util.AddRecordByCollectionName(txDao, app, listData.HistoryTableName, map[string]any{
			"level":        levelRecord.Id,
			"action":       levelMovedStatus,
			"new_position": newPos,
			"cause":        levelRecord.Id,
			"action_by":    userId,
		})
		if err != nil {
			return util.NewErrorResponse(err, "Failed to write place into the position history")
		}
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			INSERT INTO %s (level, action, new_position, cause, action_by) 
			SELECT l.id AS level, {:status} AS status, l.position AS new_position, {:cause} AS cause, {:action_by} AS action_by
			FROM %s l
			WHERE l.position BETWEEN {:minPos} AND {:maxPos} AND l.position <> {:newPos}`,
			listData.HistoryTableName,
			listData.LevelTableName)).Bind(dbx.Params{
			"status":    levelOtherStatus,
			"minPos":    mathutil.Min(newPos, oldPos),
			"maxPos":    mathutil.Max(newPos, oldPos),
			"cause":     levelRecord.Id,
			"action_by": userId,
			"newPos":    newPos,
		}).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to write history")
		}
		return nil
	})
	return err
}

func updateCreators(dao *daos.Dao, listData ListData, recordId string, newCreatos []string) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		type Creator struct {
			ID string `db:"creator"`
		}
		var currentCreatorsdata []Creator
		err := txDao.DB().Select("creator").From(listData.CreatorTableName).Where(dbx.HashExp{"level": recordId}).All(&currentCreatorsdata)
		if err != nil {
			return util.NewErrorResponse(err, "Failed to fetch current creators")
		}
		currentCreators := util.MapSlice(currentCreatorsdata, func(v Creator) string { return v.ID })
		creatorsToRemove := list.SubtractSlice(currentCreators, newCreatos)
		creatorsToAdd := list.SubtractSlice(newCreatos, currentCreators)
		_, err = txDao.DB().Delete(listData.CreatorTableName, dbx.And(dbx.In("creator", list.ToInterfaceSlice(creatorsToRemove)...), dbx.HashExp{"level": recordId})).Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to remove creators")
		}
		for _, creator := range creatorsToAdd {
			_, err = txDao.DB().Insert(listData.CreatorTableName, dbx.Params{"creator": creator, "level": recordId}).Execute()
			if err != nil {
				return util.NewErrorResponse(err, "Failed to add creator")
			}
		}
		return nil
	})
	return err
}

func UpdateLevelListPointsByPositionRange(dao *daos.Dao, list ListData, minPos int, maxPos int) error {
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		query := txDao.DB().NewQuery(fmt.Sprintf(`
		UPDATE %s
		SET points=(
			SELECT p.points 
			FROM %s p 
			WHERE p.id=position 
		)
		WHERE position BETWEEN {:minPos} AND {:maxPos}`, list.LevelTableName, list.PointLookupTableName)).Bind(dbx.Params{
			"minPos": minPos,
			"maxPos": maxPos,
		})

		_, err := query.Execute()
		if err != nil {
			return util.NewErrorResponse(err, "Failed to update level list points")
		}
		err = updatePackPointsByLevelRange(txDao, list, minPos, maxPos)
		if err != nil {
			return err
		}
		err = updateLeaderboardByLevelRange(txDao, list, minPos, maxPos)
		return err
	})
	return err
}

func UpdatePointTable(dao *daos.Dao, list ListData) error {
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
	err := dao.RunInTransaction(func(txDao *daos.Dao) error {
		formulaData, err := txDao.FindFirstRecordByData("point_formula", "list", list.Name)
		if err != nil {
			return util.NewErrorResponse(nil, "failed to load formula data")
		}
		levelCount, err := queryMaxPosition(txDao, list, false)
		if err != nil {
			return util.NewErrorResponse(nil, "failed to query max pos")
		}
		levelCount--
		totalLevelCount, err := queryMaxPosition(txDao, list, true)
		if err != nil {
			return util.NewErrorResponse(nil, "failed to query max legacy pos")
		}
		totalLevelCount--
		parameters := make(map[string]interface{}, 1)
		parameters["level_count"] = levelCount
		for _, precalc := range strings.Split(formulaData.GetString("precalc"), ",") {
			if len(precalc) == 0 {
				continue
			}
			calc := strings.SplitN(precalc, "=", 2)
			if len(calc) != 2 {
				return util.NewErrorResponse(nil, "invalid format")
			}
			calcFormula, err := govaluate.NewEvaluableExpressionWithFunctions(calc[1], functions)
			if err != nil {
				return util.NewErrorResponse(nil, "failed to parse precalc formula")
			}
			result, err := calcFormula.Evaluate(parameters)
			if err != nil {
				return util.NewErrorResponse(nil, "failed to evaluate precalc formula")
			}
			parameters[calc[0]] = result
		}
		formula, err := govaluate.NewEvaluableExpressionWithFunctions(formulaData.GetString("formula"), functions)
		if err != nil {
			return util.NewErrorResponse(nil, "failed to parse formula")
		}
		_, err = txDao.DB().Delete(list.PointLookupTableName, nil).Execute()
		if err != nil {
			return util.NewErrorResponse(nil, "failed to delete old points")
		}
		for i := 1; i <= levelCount+1; i++ {
			parameters["x"] = float64(i)
			result, err := formula.Evaluate(parameters)
			if err != nil {
				return util.NewErrorResponse(nil, "failed to evaluate formula")
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
				return util.NewErrorResponse(nil, "failed to insert new points")
			}
		}
		for i := levelCount + 2; i <= totalLevelCount+1; i++ {
			_, err = txDao.DB().Insert(list.PointLookupTableName, dbx.Params{
				"id":     i,
				"points": fmt.Sprintf("%.1f", 0.0),
			}).Execute()
			if err != nil {
				return util.NewErrorResponse(nil, "failed to insert new remaining legacy points")
			}
		}
		return nil
	})
	return err
}

func RegisterUpdatePoints(app core.App) {
	app.OnRecordAfterUpdateRequest(names.TablePointFormular).Add(func(e *core.RecordUpdateEvent) error {
		var listData ListData
		listName := e.Record.GetString("list")
		switch listName {
		case "aredl":
			listData = Aredl()
		default:
			return fmt.Errorf("unknown list %s", listName)
		}
		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			err := UpdatePointTable(txDao, listData)
			if err != nil {
				return err
			}
			maxPos, err := queryMaxPosition(txDao, listData, true)
			if err != nil {
				return util.NewErrorResponse(nil, "Failed to query max pos")
			}
			err = UpdateLevelListPointsByPositionRange(txDao, listData, 1, maxPos)
			if err != nil {
				return util.NewErrorResponse(err, "Failed to update list points")
			}
			return nil
		})
		return err
	})
}

func queryMaxPosition(dao *daos.Dao, list ListData, includeLegacy bool) (int, error) {
	condition := util.If(includeLegacy, dbx.HashExp{}, dbx.HashExp{"legacy": false})
	var position int
	err := dao.DB().Select("COALESCE(max(position), 0)").Where(condition).From(list.LevelTableName).Row(&position)
	if err != nil {
		return 0, err
	}
	return position, nil
}
