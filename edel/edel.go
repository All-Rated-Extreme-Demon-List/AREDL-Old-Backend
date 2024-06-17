package edel

import (
	"context"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"os"
	"regexp"
	"strings"

	"AREDL/demonlist"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/spf13/cobra"
)

func getGoogleSheetData(spreadsheetId string, readRange string, apiKey string) ([][]interface{}, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return nil, err
	}

	return resp.Values, nil
}

func cleanLevelName(name string) string {
	re := regexp.MustCompile(`[^\w\s]+`)
	cleanedName := re.ReplaceAllString(name, "")
	cleanedName = strings.ReplaceAll(cleanedName, "\n", "")
	return strings.TrimSpace(cleanedName)
}

func matchLevelNames(levelName string, txDao *daos.Dao, levelCollection *models.Collection) ([]*models.Record, error) {
	suffixes := []string{"", " ", " (Solo)", " (2P)"}
	var matchedLevels []*models.Record
	for _, suffix := range suffixes {
		fullName := levelName + suffix
		levels, err := txDao.FindRecordsByExpr(levelCollection.Id, dbx.HashExp{"name": fullName})
		if err != nil {
			return nil, err
		}
		matchedLevels = append(matchedLevels, levels...)
	}
	return matchedLevels, nil
}

func Register(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(&cobra.Command{
		Use: "edel",
		Run: func(command *cobra.Command, args []string) {
			if len(args) != 1 {
				print("Please provide the Google Sheets API Key to use as param")
				return
			}
			apiKey := args[0]
			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				spreadsheetId := "1-2-n2aU__rQya_IESevHjEU0f1xbegKUZEslk7fV38Q"
				readRange := "'The Enjoyments'!A:B"

				sheetData, err := getGoogleSheetData(spreadsheetId, readRange, apiKey)
				if err != nil {
					return err
				}

				aredl := demonlist.Aredl()
				levelCollection, err := txDao.FindCollectionByNameOrId(aredl.LevelTableName)
				if err != nil {
					return err
				}

				println("Updating level enjoyment data...")
				nomatch := 0

				for i, row := range sheetData {
					if len(row) < 2 {
						continue
					}

					levelName, enjoymentValue := row[0].(string), row[1].(string)
					cleanedLevelName := cleanLevelName(levelName)

					matchedLevels, err := matchLevelNames(cleanedLevelName, txDao, levelCollection)
					if err != nil {
						return err
					}

					fmt.Printf("[%d/%d] %s\n", i+1, len(sheetData)-1, levelName)
					if len(matchedLevels) > 0 {
						for _, level := range matchedLevels {
							level.Set("enjoyment", enjoymentValue)

							err = txDao.SaveRecord(level)
							if err != nil {
								return err
							}
						}
					} else {
						nomatch++
						println("\tCouldn't find a matching level on the list")
					}
				}
				fmt.Printf("Scraped %d levels, %d not on AREDL\n", len(sheetData), nomatch)
				return nil
			})
			if err != nil {
				println("Failed to fetch EDEL data: ", err.Error())
				os.Exit(1)
			} else {
				println("Updated EDEL data successfully")
			}
			return
		},
	})
}
