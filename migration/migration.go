package migration

import (
	"AREDL/demonlist"
	"AREDL/names"
	"AREDL/util"
	"encoding/json"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
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
				aredl := demonlist.Aredl()
				deleteDataTables := []string{
					aredl.LeaderboardTableName,
					aredl.LevelTableName,
					aredl.HistoryTableName,
					aredl.SubmissionTableName,
					aredl.Packs.PackTableName,
					aredl.Packs.CompletedPacksTableName,
					aredl.Packs.PackLevelTableName,
					names.TableMergeRequests,
					names.TableNameChangeRequests}
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
				levelCollection, err := txDao.FindCollectionByNameOrId(aredl.LevelTableName)
				if err != nil {
					return err
				}
				userCollection, err := txDao.FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return err
				}
				submissionCollection, err := txDao.FindCollectionByNameOrId(aredl.SubmissionTableName)
				if err != nil {
					return err
				}
				packCollection, err := txDao.FindCollectionByNameOrId(aredl.Packs.PackTableName)
				if err != nil {
					return err
				}
				packLevelCollection, err := txDao.FindCollectionByNameOrId(aredl.Packs.PackLevelTableName)
				if err != nil {
					return err
				}
				creatorCollection, err := txDao.FindCollectionByNameOrId(aredl.CreatorTableName)
				if err != nil {
					return err
				}
				positionHistoryCollection, err := txDao.FindCollectionByNameOrId(aredl.HistoryTableName)
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
					verifierId, exists := knownUsers[strings.ToLower(level.Verifier)]
					if !exists {
						userRecord, err := util.CreatePlaceholderUser(app, txDao, userCollection, level.Verifier)
						if err != nil {
							return err
						}
						verifierId = userRecord.Id
						knownUsers[strings.ToLower(level.Verifier)] = verifierId
					}
					publisherId, exists := knownUsers[strings.ToLower(level.Author)]
					if !exists {
						userRecord, err := util.CreatePlaceholderUser(app, txDao, userCollection, level.Author)
						if err != nil {
							return err
						}
						publisherId = userRecord.Id
						knownUsers[strings.ToLower(level.Author)] = publisherId
					}

					levelRecord, err := util.AddRecord(txDao, app, levelCollection, map[string]any{
						"position":           position + 1,
						"name":               level.Name,
						"publisher":          publisherId,
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
					addSubmissionRecord := func(username string, recordOrder int, url string, framerate int, percent int, mobile bool) (*models.Record, error) {
						playerId, exists := knownUsers[strings.ToLower(username)]
						if !exists {
							userRecord, err := util.CreatePlaceholderUser(app, txDao, userCollection, username)
							if err != nil {
								return nil, err
							}
							playerId = userRecord.Id
							knownUsers[strings.ToLower(username)] = playerId
						}
						submissionRecord, err := util.AddRecord(txDao, app, submissionCollection, map[string]any{
							"status":          "accepted",
							"video_url":       strings.Replace(url, " ", "", -1),
							"level":           levelRecord.Id,
							"submitted_by":    playerId,
							"fps":             framerate,
							"percentage":      percent,
							"placement_order": recordOrder + 1,
							"mobile":          mobile,
						})
						if err != nil {
							return nil, err
						}
						return submissionRecord, nil
					}

					_, err = addSubmissionRecord(level.Verifier, 0, level.Verification, 60, 100, false)
					if err != nil {
						return err
					}

					for submissionOrder, playerRecord := range level.Records {
						_, err := addSubmissionRecord(playerRecord.User, submissionOrder+1, playerRecord.Link, playerRecord.Framerate, playerRecord.Percent, playerRecord.Mobile)
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
						"color":           pack.Colour,
					})
					if err != nil {
						return err
					}
					// Add levels to pack
					for _, levelName := range pack.Levels {
						levelId, exists := knownLevels[levelName]
						if !exists {
							println("Ignoring " + levelName + " for pack " + pack.Name)
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
				err = demonlist.UpdateAllCompletedPacks(txDao, aredl)
				if err != nil {
					return err
				}
				println("Updating demonlist")
				err = demonlist.UpdateLevelListPointsByPositionRange(txDao, aredl, 1, len(levelNames))
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				println("Failed to migrate: ", err.Error())
			} else {
				println("Finished migrating")
			}
		},
	})
}
