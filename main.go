package main

import (
	"AREDL/demonlist"
	"AREDL/endpoints/aredl_moderation"
	"AREDL/endpoints/aredl_public"
	"AREDL/endpoints/aredl_user"
	"AREDL/endpoints/moderation"
	"AREDL/endpoints/user"
	"AREDL/migration"
	"github.com/Simolater/echo-swagger"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"log"

	_ "AREDL/docs"
)

//	@title			Aredl API
//	@version		0.1
//	@description	Backend for the all rated extreme demon list
//	@contact.name	Discord server
//	@contact.url	https://discord.gg/VbqrUBtTfX
//	@host			api.aredl.com
//	@BasePath		/api

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						authorization
// @description				Perform actions as a user. It is also used to access endpoints that require user permissions.
func main() {
	app := pocketbase.New()

	migration.Register(app)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/docs/*", echoSwagger.WrapHandler)
		return nil
	})

	moderation.RegisterEndpoints(app)
	aredl_moderation.RegisterEndpoints(app)
	user.RegisterEndpoints(app)
	aredl_user.RegisterEndpoints(app)
	aredl_public.RegisterEndpoints(app)

	RegisterUserAuth(app)

	demonlist.RegisterUpdatePoints(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
