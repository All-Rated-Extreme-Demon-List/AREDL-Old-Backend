package edel

import (
	"AREDL/demonlist"
	"context"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"os"
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

func matchLevels(id string, txDao *daos.Dao, levelCollection *models.Collection) ([]*models.Record, error) {
	levels, err := txDao.FindRecordsByExpr(levelCollection.Id, dbx.HashExp{"level_id": id})
	if err != nil {
		return nil, err
	}
	if len(levels) > 1 {
		return levels[1:], nil
	}
	return levels, nil
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
				spreadsheetId := "1VpqxxW4t-5tGSVVIAPyYnXKFD_bldhoHhm59FvfQ1EY"
				normalReadRange := "'IDS'!B:C"

				normalSheetData, err := getGoogleSheetData(spreadsheetId, normalReadRange, apiKey)
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

				for i := 1; i < len(normalSheetData); i++ {
					row := normalSheetData[i]
					if len(row) < 2 {
						continue
					}

					enjoymentValue, levelID := row[0].(string), row[1].(string)

					matchedLevels, err := matchLevels(levelID, txDao, levelCollection)
					if err != nil {
						return err
					}

					fmt.Printf("[%d/%d] %s\n", i, len(normalSheetData)-1, levelID)
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
				fmt.Printf("Scraped %d levels, %d not on AREDL\n", len(normalSheetData)-1, nomatch)
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
