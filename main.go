package main

import (
	"AREDL/migration"
	"AREDL/moderation"
	"github.com/pocketbase/pocketbase"
	"log"
)

func main() {
	app := pocketbase.New()

	migration.Register(app)

	moderation.RegisterEndpoints(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
