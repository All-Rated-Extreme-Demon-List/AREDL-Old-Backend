package demonlist

import (
	"AREDL/names"
	"AREDL/util"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type SongData struct {
	ID   int
	Name string
	Link string
}

type LevelData struct {
	LevelName     string
	Version       int
	Downloads     int
	GameVersion   int
	OfficialSong  int
	CustomSongId  int
	Likes         int
	Length        string
	FeatureScore  int
	Coins         int
	VerifiedCoins bool
	Objects       int
	Songs         []SongData
}

func RegisterLevelDataRequester(e *core.ServeEvent) error {
	scheduler := cron.New()
	scheduler.MustAdd("leveldata", "* * * * *", RequestLevelData(e.App))

	scheduler.Start()
	return nil
}

func RequestLevelData(app core.App) func() {
	return func() {

		l := app.Logger()

		levelIds, err := getMissingLevelIds(app)

		if len(levelIds) == 0 {
			return
		}

		// Select random levelID
		levelId := levelIds[rand.Intn(len(levelIds))]
		l = l.With("level_id", levelId)

		levelDataRaw, err := requestLevelDataFromGDServer(levelId)
		if err != nil {
			l.Error("Failed request gd level data", "error", err)
			return
		}

		l = l.With("raw_data", levelDataRaw)

		levelData, err := parseLevelData(levelDataRaw)
		if err != nil {
			l.Error("Failed to parse gd level data", "error", err)
			return
		}

		recordData := map[string]any{
			"level_id":       levelId,
			"name":           levelData.LevelName,
			"version":        levelData.Version,
			"downloads":      levelData.Downloads,
			"game_version":   levelData.GameVersion,
			"official_song":  levelData.OfficialSong,
			"custom_song":    levelData.CustomSongId,
			"likes":          levelData.Likes,
			"length":         levelData.Length,
			"feature_score":  levelData.FeatureScore,
			"coins":          levelData.Coins,
			"verified_coins": levelData.VerifiedCoins,
			"object_count":   levelData.Objects,
			"songs":          levelData.Songs,
		}

		_, err = util.AddRecordByCollectionName(app.Dao(), app, names.TableLevelInfo, recordData)
		if err != nil {
			l.Error("Failed to add record data", "error", err)
			return
		}

		l.Info("Requested level data", "data", levelData)
	}
}

func getMissingLevelIds(app core.App) ([]string, error) {
	query := app.Dao().DB().NewQuery(fmt.Sprintf(`
			SELECT level_id
			FROM %v level 
			WHERE NOT EXISTS (
				SELECT NULL
				FROM %v level_info
				WHERE level.level_id == level_info.level_id
			)
		`, Aredl().LevelTableName, names.TableLevelInfo))

	type LevelData struct {
		LevelId string `db:"level_id"`
	}

	var levelIds []LevelData

	err := query.All(&levelIds)
	if err != nil {
		return []string{}, fmt.Errorf("level id query error: %w", err)
	}
	stringIds := util.MapSlice(levelIds, func(value LevelData) string { return value.LevelId })
	return stringIds, nil
}

func parseLevelData(data string) (LevelData, error) {
	dataComponents := strings.Split(data, "#")
	var levelData LevelData

	if len(dataComponents) != 5 {
		return LevelData{}, fmt.Errorf("invalid data format: %s", data)
	}

	levelInfoKV := strings.Split(dataComponents[0], ":")

	if len(levelInfoKV)%2 != 0 {
		return LevelData{}, fmt.Errorf("invalid number of key-value pairs")
	}

	for i := 0; i < len(levelInfoKV); i += 2 {
		key := levelInfoKV[i]
		value := levelInfoKV[i+1]
		var valueInt int
		var err error
		switch key {
		case "2":
			levelData.LevelName = value
		case "5":
			valueInt, err = strconv.Atoi(value)
			levelData.Version = valueInt
		case "10":
			valueInt, err = strconv.Atoi(value)
			levelData.Downloads = valueInt
		case "12":
			valueInt, err = strconv.Atoi(value)
			levelData.OfficialSong = valueInt
		case "13":
			valueInt, err = strconv.Atoi(value)
			levelData.GameVersion = valueInt
		case "14":
			valueInt, err = strconv.Atoi(value)
			levelData.Likes = valueInt
		case "15":
			switch value {
			case "0":
				levelData.Length = "Tiny"
			case "1":
				levelData.Length = "Short"
			case "2":
				levelData.Length = "Medium"
			case "3":
				levelData.Length = "Long"
			case "4":
				levelData.Length = "XL"
			default:
				return LevelData{}, fmt.Errorf("invalid level length value: %s", value)
			}
		case "19":
			valueInt, err = strconv.Atoi(value)
			levelData.FeatureScore = valueInt
		case "35":
			valueInt, err = strconv.Atoi(value)
			levelData.CustomSongId = valueInt
		case "37":
			valueInt, err = strconv.Atoi(value)
			levelData.Coins = valueInt
		case "38":
			if value == "0" {
				levelData.VerifiedCoins = false
			} else if value == "1" {
				levelData.VerifiedCoins = true
			} else {
				return LevelData{}, fmt.Errorf("invalid verified coins value: %s", value)
			}
		case "45":
			valueInt, err = strconv.Atoi(value)
			levelData.Objects = valueInt
		}
		if err != nil {
			return LevelData{}, fmt.Errorf("error parsing data %w", err)
		}
	}

	songs := strings.Split(dataComponents[2], "~:~")

	if len(songs) == 0 {
		return levelData, nil
	}

	for _, song := range songs {
		songKV := strings.Split(song, "~|~")
		songData := SongData{}

		if len(songKV)%2 != 0 {
			return LevelData{}, fmt.Errorf("invalid number of key-value pairs in song %s", song)
		}

		for i := 0; i < len(songKV); i += 2 {
			key := songKV[i]
			value := songKV[i+1]
			var err error
			switch key {
			case "1":
				songData.ID, err = strconv.Atoi(value)
				if err != nil {
					return LevelData{}, fmt.Errorf("failed to convert data: %w", err)
				}
			case "2":
				songData.Name = value
			case "10":
				songData.Link, err = url.QueryUnescape(value)
				if err != nil {
					return LevelData{}, fmt.Errorf("failed to unescape level song: %w", err)
				}
			}
		}
		levelData.Songs = append(levelData.Songs, songData)
	}

	return levelData, nil
}

func requestLevelDataFromGDServer(levelId string) (string, error) {
	urlStr := "http://www.boomlings.com/database/getGJLevels21.php"

	params := url.Values{}
	params.Set("str", levelId)
	params.Set("secret", "Wmfd2893gb7")
	params.Set("type", "0")

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request error: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request data error: %w", err)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("response read error: %w", err)
	}

	return string(body), nil
}
