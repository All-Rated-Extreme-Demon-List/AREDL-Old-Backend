package main

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/auth"
)

func RegisterUserAuth(app core.App) {
	app.OnRecordAuthRequest().Add(func(e *core.RecordAuthEvent) error {
		meta := e.Meta.(struct {
			*auth.AuthUser
			IsNew bool `json:"isNew"`
		})
		if meta.IsNew {
			e.Record.Set("role", "member")
			e.Record.Set("global_name", meta.RawUser["global_name"])
			e.Record.Set("discord_id", meta.RawUser["id"])
		}
		e.Record.Set("avatar_url", meta.AvatarUrl)
		e.Record.Set("banner_color", meta.RawUser["banner_color"])
		err := app.Dao().SaveRecord(e.Record)
		if err != nil {
			return err
		}
		return nil
	})
}
