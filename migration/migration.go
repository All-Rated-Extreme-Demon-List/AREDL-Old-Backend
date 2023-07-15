package migration

import (
	"AREDL/names"
	"encoding/json"
	"errors"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/spf13/cobra"
	"io"
	"math/rand"
	"os"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

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
				var levelNames []string
				err := readFileIntoJson(path+"/_list.json", &levelNames)
				if err != nil {
					return err
				}
				levelCollection, err := app.Dao().FindCollectionByNameOrId(names.TableLevels)
				if err != nil {
					return err
				}
				userCollection, err := app.Dao().FindCollectionByNameOrId(names.TableUsers)
				if err != nil {
					return err
				}
				submissionCollection, err := app.Dao().FindCollectionByNameOrId(names.TableSubmissions)
				if err != nil {
					return err
				}
				packCollection, err := app.Dao().FindCollectionByNameOrId(names.TablePacks)
				if err != nil {
					return err
				}
				knownUsers := make(map[string]string)
				knownLevels := make(map[string]string)
				for position, levelName := range levelNames {
					println(levelName)
					var level Level
					err = readFileIntoJson(path+"/"+levelName+".json", &level)
					if err != nil {
						return err
					}
					levelRecord := models.NewRecord(levelCollection)
					levelForm := forms.NewRecordUpsert(app, levelRecord)
					levelForm.SetDao(txDao)
					err = levelForm.LoadData(map[string]any{
						"position":           position + 1,
						"name":               level.Name,
						"creators":           strings.Join(level.Creators, ","),
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
					err = levelForm.Submit()
					if err != nil {
						return err
					}
					knownLevels[levelName] = levelRecord.Id
					for submissionOrder, playerRecord := range level.Records {
						playerId, exists := knownUsers[strings.ToLower(playerRecord.User)]
						if !exists {
							// create legacy user
							userRecord := models.NewRecord(userCollection)
							userForm := forms.NewRecordUpsert(app, userRecord)
							userForm.SetDao(txDao)
							password := RandString(20)
							usedName := RandString(10)
							err = userForm.LoadData(map[string]any{
								"username":    usedName,
								"permissions": "member",
								"global_name": playerRecord.User,
								"legacy":      true,
								"email":       usedName + "@none.com",
							})
							userForm.Password = password
							userForm.PasswordConfirm = password
							if err != nil {
								return err
							}
							err = userForm.Submit()
							if err != nil {
								return err
							}
							playerId = userRecord.Id
							knownUsers[strings.ToLower(playerRecord.User)] = playerId
						}

						submissionRecord := models.NewRecord(submissionCollection)
						submissionForm := forms.NewRecordUpsert(app, submissionRecord)
						submissionForm.SetDao(txDao)
						err = submissionForm.LoadData(map[string]any{
							"status":       "accepted",
							"video_url":    strings.Replace(playerRecord.Link, " ", "", -1),
							"level":        levelRecord.Id,
							"submitted_by": playerId,
							"fps":          playerRecord.Framerate,
							"percentage":   playerRecord.Percent,
							"order":        submissionOrder + 1,
						})
						err = submissionForm.Submit()
						if err != nil {
							return err
						}
					}
				}

				var packs []Pack
				err = readFileIntoJson(path+"/_packlist.json", &packs)
				if err != nil {
					return err
				}
				for packOrder, pack := range packs {
					packRecord := models.NewRecord(packCollection)
					packForm := forms.NewRecordUpsert(app, packRecord)
					packForm.SetDao(txDao)

					var levels []string
					for _, levelName := range pack.Levels {
						levelId, exists := knownLevels[levelName]
						if !exists {
							return errors.New("Unknown level: " + levelName)
						}
						levels = append(levels, levelId)
					}

					err = packForm.LoadData(map[string]any{
						"order":  packOrder + 1,
						"name":   pack.Name,
						"colour": pack.Colour,
						"levels": levels,
					})
					if err != nil {
						return err
					}
					err = packForm.Submit()
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				print("Failed to migrate: ", err.Error())
			}
		},
	})
}
