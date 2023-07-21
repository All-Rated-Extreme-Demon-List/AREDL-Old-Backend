package main

import (
	"AREDL/migration"
	"AREDL/moderation"
	"AREDL/user"
	"github.com/pocketbase/pocketbase"
	"log"
)

func main() {
	app := pocketbase.New()

	migration.Register(app)

	moderation.RegisterEndpoints(app)
	user.RegisterEndpoints(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
