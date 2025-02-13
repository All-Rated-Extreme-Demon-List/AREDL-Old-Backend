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
	"slices"
	"strconv"
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
	User    int    `json:"user"`
	Link    string `json:"link"`
	Percent int    `json:"percent"`
	Mobile  bool   `json:"mobile"`
}

type Level struct {
	Id               int      `json:"id"`
	Name             string   `json:"name"`
	Author           int      `json:"author"`
	Creators         []int    `json:"creators"`
	Verifier         int      `json:"verifier"`
	Verification     string   `json:"verification"`
	PercentToQualify int      `json:"percentToQualify"`
	Password         string   `json:"password"`
	Records          []Record `json:"records"`
	Legacy           bool     `json:"-"`
}

type Pack struct {
	Name   string   `json:"name"`
	Colour string   `json:"colour"`
	Levels []string `json:"levels"`
}

type RoleList struct {
	Role    string `json:"role"`
	Members []struct {
		Name int    `json:"name"`
		Link string `json:"link"`
	} `json:"members"`
}

func addPlaceholder(txDao *daos.Dao, userJsonId int, oldIds map[int]string, jsonNameMap map[int]string, banned []int) (string, error) {
	userId, ok := oldIds[userJsonId]
	if !ok {
		userId = util.RandString(14)
	}
	username, ok := jsonNameMap[userJsonId]
	if !ok {
		return "", fmt.Errorf("unknown json id %v", userJsonId)
	}
	usedName := util.RandString(10)
	userToken := util.RandString(10)
	_, err := txDao.DB().Insert(names.TableUsers, dbx.Params{
		"id":               userId,
		"username":         usedName,
		"global_name":      username,
		"placeholder":      true,
		"passwordHash":     "",
		"tokenKey":         userToken,
		"json_id":          userJsonId,
		"banned_from_list": slices.Contains(banned, userJsonId),
	}).Execute()
	if err != nil {
		return "", err
	}
	return userId, nil
}

func resolveRole(role string) string {
	switch role {
	case "owner":
		return "listOwner"
	case "admin":
		return "listAdmin"
	case "mod":
		return "listMod"
	case "helper":
		return "listHelper"
	case "dev":
		return "developer"
	case "patreon":
		return "aredlPlus"
	}
	return "member"
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
				aredl := demonlist.Aredl()

				var jsonNameMapStr map[string]string
				var err = readFileIntoJson(path+"/_name_map.json", &jsonNameMapStr)
				if err != nil {
					return err
				}

				jsonNameMap := make(map[int]string)
				for idStr, name := range jsonNameMapStr {
					var id, err = strconv.Atoi(idStr)
					if err != nil {
						return err
					}
					jsonNameMap[id] = name
				}

				jsonIdFromName := make(map[string]int)
				for id, name := range jsonNameMap {
					jsonIdFromName[strings.ToLower(name)] = id
				}

				oldUserIds := make(map[int]string)
				oldLevelIds := make(map[string]string)

				oldLevels, err := txDao.FindRecordsByExpr(aredl.LevelTableName)
				if err != nil {
					return err
				}
				for _, oldLevel := range oldLevels {
					oldLevelIds[oldLevel.GetString("name")] = oldLevel.Id
				}

				println("Deleting current data")
				deleteDataTables := []string{
					aredl.LeaderboardTableName,
					aredl.LevelTableName,
					aredl.HistoryTableName,
					aredl.RecordsTableName,
					aredl.Packs.PackTableName,
					aredl.Packs.CompletedPacksTableName,
					aredl.Packs.PackLevelTableName,
					names.TableRoles,
					names.TableMergeRequests,
					names.TableNameChangeRequests}
				for _, table := range deleteDataTables {
					fmt.Printf("Deleting %v\n", table)
					_, err := txDao.DB().Delete(table, nil).Execute()
					if err != nil {
						return err
					}
				}
				println("Deleting placeholder users")
				userRecords, err := txDao.FindRecordsByExpr(names.TableUsers, dbx.HashExp{"placeholder": true})
				if err != nil {
					return err
				}
				for _, userRecord := range userRecords {
					var id = userRecord.GetInt("json_id")
					if id == 0 {
						var ok bool
						id, ok = jsonIdFromName[strings.TrimSpace(strings.ToLower(userRecord.GetString("global_name")))]
						if !ok {
							fmt.Printf("unknown existing name %s skipping\n", userRecord.GetString("global_name"))
							continue
						}
					}
					oldUserIds[id] = userRecord.Id
					if err = txDao.DeleteRecord(userRecord); err != nil {
						return err
					}
				}
				knownUsers := make(map[int]string)
				knownLevels := make(map[string]string)
				println("Migrating levels & records")
				var levelNames []string
				err = readFileIntoJson(path+"/_list.json", &levelNames)
				if err != nil {
					return err
				}
				var legacyLevelNames []string
				err = readFileIntoJson(path+"/_legacy.json", &legacyLevelNames)
				if err != nil {
					return err
				}
				var banned []int
				err = readFileIntoJson(path+"/_leaderboard_banned.json", &banned)
				if err != nil {
					return err
				}
				levelCollection, err := txDao.FindCollectionByNameOrId(aredl.LevelTableName)
				if err != nil {
					return err
				}
				recordsCollection, err := txDao.FindCollectionByNameOrId(aredl.RecordsTableName)
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
				type LevelData struct {
					Name   string
					Legacy bool
				}
				levels := util.MapSlice(levelNames, func(name string) LevelData { return LevelData{name, false} })
				for _, name := range legacyLevelNames {
					levels = append(levels, LevelData{name, true})
				}
				for position, levelData := range levels {
					fmt.Printf("[%d/%d] %s%v\n", position+1, len(levels), levelData.Name, util.If(levelData.Legacy, "(legacy)", ""))
					var level Level
					err = readFileIntoJson(path+"/"+levelData.Name+".json", &level)
					if err != nil {
						return err
					}
					twoPlayer := strings.HasSuffix(levelData.Name, "2p")
					if len(level.Creators) == 0 {
						level.Creators = []int{level.Author}
					}

					verifierId, exists := knownUsers[level.Verifier]
					if !exists {
						userId, err := addPlaceholder(txDao, level.Verifier, oldUserIds, jsonNameMap, banned)
						if err != nil {
							return err
						}
						verifierId = userId
						knownUsers[level.Verifier] = verifierId
					}
					publisherId, exists := knownUsers[level.Author]
					if !exists {
						userId, err := addPlaceholder(txDao, level.Author, oldUserIds, jsonNameMap, banned)
						if err != nil {
							return err
						}
						publisherId = userId
						knownUsers[level.Author] = publisherId
					}

					levelRecordData := map[string]any{
						"position":       position + 1,
						"name":           level.Name,
						"publisher":      publisherId,
						"level_id":       level.Id,
						"level_password": level.Password,
						"legacy":         levelData.Legacy,
						"two_player":     twoPlayer,
					}

					levelId, ok := oldLevelIds[level.Name]
					if ok {
						levelRecordData["id"] = levelId
					}

					levelRecord, err := util.AddRecord(txDao, app, levelCollection, levelRecordData)
					if err != nil {
						return err
					}
					knownLevels[levelData.Name] = levelRecord.Id
					for _, creator := range level.Creators {
						creatorId, exists := knownUsers[creator]
						if !exists {
							creatorId, err = addPlaceholder(txDao, creator, oldUserIds, jsonNameMap, banned)
							if err != nil {
								return err
							}
							knownUsers[creator] = creatorId
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
					addSubmissionRecord := func(username int, recordOrder int, url string, percent int, mobile bool) (*models.Record, error) {
						playerId, exists := knownUsers[username]
						if !exists {
							userId, err := addPlaceholder(txDao, username, oldUserIds, jsonNameMap, banned)
							if err != nil {
								return nil, err
							}
							playerId = userId
							knownUsers[username] = playerId
						}
						submissionRecord, err := util.AddRecord(txDao, app, recordsCollection, map[string]any{
							"video_url":       strings.Replace(url, " ", "", -1),
							"level":           levelRecord.Id,
							"submitted_by":    playerId,
							"placement_order": recordOrder + 1,
							"mobile":          mobile,
						})
						if err != nil {
							return nil, err
						}
						return submissionRecord, nil
					}

					_, err = addSubmissionRecord(level.Verifier, 0, level.Verification, 60, false)
					if err != nil {
						return err
					}

					for submissionOrder, playerRecord := range level.Records {
						_, err := addSubmissionRecord(playerRecord.User, submissionOrder+1, playerRecord.Link, playerRecord.Percent, playerRecord.Mobile)
						if err != nil {
							return err
						}
					}
				}
				println("Loading editors")
				var editors []RoleList
				err = readFileIntoJson(path+"/_editors.json", &editors)
				if err != nil {
					return err
				}
				for _, editorList := range editors {
					for _, member := range editorList.Members {
						memberId, ok := knownUsers[member.Name]
						if !ok {
							memberId, err = addPlaceholder(txDao, member.Name, oldUserIds, jsonNameMap, banned)
							if err != nil {
								return err
							}
							knownUsers[member.Name] = memberId
						}
						_, err = txDao.DB().Insert(names.TableRoles, dbx.Params{
							"user": memberId,
							"role": resolveRole(editorList.Role),
						}).Execute()
						if err != nil {
							return err
						}
					}
				}

				println("Loading supporters")
				var supporters []RoleList
				err = readFileIntoJson(path+"/_supporters.json", &supporters)
				if err != nil {
					return err
				}
				for _, supporterList := range supporters {
					for _, member := range supporterList.Members {
						memberId, ok := knownUsers[member.Name]
						if !ok {
							memberId, err = addPlaceholder(txDao, member.Name, oldUserIds, jsonNameMap, banned)
							if err != nil {
								return err
							}
							knownUsers[member.Name] = memberId
						}
						_, err = txDao.DB().Insert(names.TableRoles, dbx.Params{
							"user": memberId,
							"role": resolveRole(supporterList.Role),
						}).Execute()
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
							return fmt.Errorf("failed to load pack %v, because level %v was not found", pack.Name, levelName)
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
				err = demonlist.UpdatePointTable(txDao, aredl)
				if err != nil {
					return err
				}
				err = demonlist.UpdateLevelListPointsByPositionRange(txDao, aredl, 1, len(levelNames))
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				println("Failed to migrate: ", err.Error())
				os.Exit(1)
			} else {
				println("Finished migrating")
			}
			return
		},
	})
}
