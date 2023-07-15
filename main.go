package main

import (
	"AREDL/migration"
	"AREDL/moderation"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"log"
)

func main() {
	app := pocketbase.New()

	migration.Register(app)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		err := moderation.RegisterEndpoints(e.Router, app)
		if err != nil {
			return err
		}
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
