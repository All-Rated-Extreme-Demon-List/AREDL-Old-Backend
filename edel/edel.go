package edel

import (
	"context"
	"fmt"
	"github.com/pocketbase/dbx"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"os"

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
				for i, row := range sheetData {
					if len(row) < 2 {
						continue
					}
					levelName, enjoymentValue := row[0].(string), row[1].(string)
					fmt.Printf("[%d/%d] %s\n", i+1, len(sheetData)-1, levelName)
					levels, err := txDao.FindRecordsByExpr(levelCollection.Id, dbx.HashExp{"name": levelName})
					if err != nil {
						return err
					}

					if len(levels) > 0 {
						level := levels[0]
						level.Set("enjoyment", enjoymentValue)

						err = txDao.SaveRecord(level)
						if err != nil {
							return err
						}
					}
				}

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
