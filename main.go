package main

import (
	"AREDL/demonlist"
	"AREDL/migration"
	"AREDL/moderation"
	"AREDL/public"
	"AREDL/user"
	"github.com/pocketbase/pocketbase"
	"log"
)

func main() {
	app := pocketbase.New()

	migration.Register(app)

	moderation.RegisterEndpoints(app)
	user.RegisterEndpoints(app)
	user.RegisterUserAuth(app)
	public.RegisterEndpoints(app)

	demonlist.RegisterUpdatePoints(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
