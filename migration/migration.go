package migration

import (
	"AREDL/names"
	"AREDL/points"
	"AREDL/util"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
)

func readFileIntoJson(path string, v any) error {
	list, err := os.Open(path)
	if err != nil {
		return err
	}
	listBytes, err := io.ReadAll(list)
	if err != nil {
		return err
	}
	err = json.Unmarshal(listBytes, &v)
	if err != nil {
		return err
	}
	return nil
}

type Record struct {
	User      string `json:"user"`
	Link      string `json:"link"`
	Percent   int    `json:"percent"`
	Framerate int    `json:"hz"`
	Mobile    bool   `json:"mobile"`
}

type Level struct {
	Id               int      `json:"id"`
	Name             string   `json:"name"`
	Author           string   `json:"author"`
	Creators         []string `json:"creators"`
	Verifier         string   `json:"verifier"`
	Verification     string   `json:"verification"`
	PercentToQualify int      `json:"percentToQualify"`
	Password         string   `json:"password"`
	Records          []Record `json:"records"`
}

type Pack struct {
	Name   string   `json:"name"`
	Colour string   `json:"colour"`
	Levels []string `json:"levels"`
}

func Register(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(&cobra.Command{
		Use: "migrate",
		Run: func(command *cobra.Command, args []string) {
			if len(args) != 1 {
				print("Migrate takes only the data path as argument")
				return
			}
			path := args[0]
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				println("Deleting current data")
				deleteDataTables := []string{names.TableLevels, names.TableLevelHistory, names.TableSubmissions, names.TablePacks, names.TableCompletedPacks, names.TablePackLevels, names.TableMergeRequests, names.TableNameChangeRequests}
				for _, table := range deleteDataTables {
					_, err := txDao.DB().Delete(table, nil).Execute()
					if err != nil {
						return err
					}
				}
				userRecords, err := txDao.FindRecordsByExpr(names.TableUsers, dbx.HashExp{"placeholder": true})
				if err != nil {
					return err
				}
				for _, userRecord := range userRecords {
					if err = txDao.DeleteRecord(userRecord); err != nil {
						return err
					}
				}
				println("Migrating levels & records")
				var levelNames []string
				err = readFileIntoJson(path+"/_list.json", &levelNames)
				if err != nil {
					return err
				}
				levelCollection, err := txDao.FindCollectionByNameOrId(names.TableLevels)
				if err != nil {
					return err
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return err
				}
				submissionCollection, err := txDao.FindCollectionByNameOrId(names.TableSubmissions)
				if err != nil {
					return err
				}
				packCollection, err := txDao.FindCollectionByNameOrId(names.TablePacks)
				if err != nil {
					return err
				}
				packLevelCollection, err := txDao.FindCollectionByNameOrId(names.TablePackLevels)
				if err != nil {
					return err
				}
				creatorCollection, err := txDao.FindCollectionByNameOrId(names.TableCreators)
				if err != nil {
					return err
				}
				positionHistoryCollection, err := txDao.FindCollectionByNameOrId(names.TableLevelHistory)
				if err != nil {
					return err
				}
				knownUsers := make(map[string]string)
				knownLevels := make(map[string]string)
				for position, levelName := range levelNames {
					fmt.Printf("[%d/%d] %s\n", position+1, len(levelNames), levelName)
					var level Level
					err = readFileIntoJson(path+"/"+levelName+".json", &level)
					if err != nil {
						return err
					}

					levelRecord, err := util.AddRecord(txDao, app, levelCollection, map[string]any{
						"position":           position + 1,
						"name":               level.Name,
						"verifier":           level.Verifier,
						"publisher":          level.Author,
						"video_id":           level.Verification,
						"level_id":           level.Id,
						"level_password":     level.Password,
						"qualifying_percent": level.PercentToQualify,
					})
					if err != nil {
						return err
					}
					knownLevels[levelName] = levelRecord.Id
					for _, creator := range level.Creators {
						creatorId, exists := knownUsers[strings.ToLower(creator)]
						if !exists {
							userRecord, err := util.CreatePlaceholderUser(app, txDao, userCollection, creator)
							if err != nil {
								return err
							}
							creatorId = userRecord.Id
							knownUsers[strings.ToLower(creator)] = creatorId
						}
						_, err = util.AddRecord(txDao, app, creatorCollection, map[string]any{
							"creator": creatorId,
							"level":   levelRecord.Id,
						})
						if err != nil {
							return err
						}
					}

					_, err = util.AddRecord(txDao, app, positionHistoryCollection, map[string]any{
						"level":        levelRecord.Id,
						"action":       "placed",
						"new_position": position + 1,
						"cause":        levelRecord.Id,
					})
					if err != nil {
						return err
					}

					// level submissions
					addSubmissionRecord := func(username string, recordOrder int, url string, framerate int, percent int, mobile bool) error {
						playerId, exists := knownUsers[strings.ToLower(username)]
						if !exists {
							userRecord, err := util.CreatePlaceholderUser(app, txDao, userCollection, username)
							if err != nil {
								return err
							}
							playerId = userRecord.Id
							knownUsers[strings.ToLower(username)] = playerId
						}

						device := "pc"
						if mobile {
							device = "mobile"
						}
						_, err := util.AddRecord(txDao, app, submissionCollection, map[string]any{
							"status":          "accepted",
							"video_url":       strings.Replace(url, " ", "", -1),
							"level":           levelRecord.Id,
							"submitted_by":    playerId,
							"fps":             framerate,
							"percentage":      percent,
							"placement_order": recordOrder + 1,
							"device":          device,
						})
						if err != nil {
							return err
						}
						return nil
					}

					err = addSubmissionRecord(level.Verifier, 0, level.Verification, 60, 100, false)
					if err != nil {
						return err
					}

					for submissionOrder, playerRecord := range level.Records {
						err := addSubmissionRecord(playerRecord.User, submissionOrder+1, playerRecord.Link, playerRecord.Framerate, playerRecord.Percent, playerRecord.Mobile)
						if err != nil {
							return err
						}
					}
				}

				println("Migrating packs")
				var packs []Pack
				err = readFileIntoJson(path+"/_packlist.json", &packs)
				if err != nil {
					return err
				}
				for packOrder, pack := range packs {
					packRecord, err := util.AddRecord(txDao, app, packCollection, map[string]any{
						"placement_order": packOrder + 1,
						"name":            pack.Name,
						"colour":          pack.Colour,
					})
					if err != nil {
						return err
					}
					// Add levels to pack
					for _, levelName := range pack.Levels {
						levelId, exists := knownLevels[levelName]
						if !exists {
							return errors.New("Unknown level: " + levelName)
						}
						_, err := util.AddRecord(txDao, app, packLevelCollection, map[string]any{
							"level": levelId,
							"pack":  packRecord.Id,
						})
						if err != nil {
							return err
						}
					}
				}
				println("Updating users")
				err = points.UpdateAllCompletedPacks(txDao)
				if err != nil {
					return err
				}
				println("Updating list points")
				err = points.UpdateListPointsByLevelRange(txDao, 1, len(levelNames))
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				print("Failed to migrate: ", err.Error())
			} else {
				println("Finished migrating")
			}
		},
	})
}
