package main

import (
	"encoding/json"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/auth"
	"io"
	"net/http"
	"time"
)

const VisibilityEveryone = 1

type Connection struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Revoked    bool   `json:"revoked"`
	Verified   bool   `json:"verified"`
	Visibility int    `json:"visibility"`
}

func loadConnections(record *models.Record, accessToken string) error {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://discord.com/api/users/@me/connections"), nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(response.Body)
	var connections []Connection
	err = json.NewDecoder(response.Body).Decode(&connections)
	if err != nil {
		return err
	}
	for _, connection := range connections {
		// only store verified and non revoked connections
		if connection.Revoked || !connection.Verified {
			continue
		}
		// only store public connections
		if connection.Visibility != VisibilityEveryone {
			continue
		}
		if connection.Type == "youtube" {
			record.Set("youtube_id", connection.Id)
		}
		if connection.Type == "twitch" {
			record.Set("twitch_id", connection.Name)
		}
		if connection.Type == "twitter" {
			record.Set("twitter_id", connection.Name)
		}
	}
	return nil
}

func RegisterUserAuth(app core.App) {
	app.OnRecordAuthRequest().Add(func(e *core.RecordAuthEvent) error {
		meta := e.Meta.(struct {
			*auth.AuthUser
			IsNew bool `json:"isNew"`
		})
		if meta.IsNew {
			e.Record.Set("global_name", meta.RawUser["global_name"])
			e.Record.Set("discord_id", meta.RawUser["id"])
		}
		err := e.Record.SetEmail("")
		if err != nil {
			return err
		}
		e.Record.Set("avatar_url", meta.AvatarUrl)
		e.Record.Set("banner_color", meta.RawUser["banner_color"])
		err = loadConnections(e.Record, meta.AccessToken)
		if err != nil {
			return err
		}
		err = app.Dao().SaveRecord(e.Record)
		if err != nil {
			return err
		}

		return nil
	})
}
