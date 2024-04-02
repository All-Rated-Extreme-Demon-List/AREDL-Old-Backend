package main

import (
	"AREDL/demonlist"
	"AREDL/endpoints/aredl"
	"AREDL/endpoints/global"
	"AREDL/migration"
	"github.com/Simolater/echo-swagger"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"log"

	_ "AREDL/docs"
)

//	@title			Aredl API
//	@version		1.0
//	@description	Backend for the all rated extreme demon list
//	@contact.name	Discord server
//	@contact.url	https://discord.gg/VbqrUBtTfX
//	@contact.name	Aredl
//	@contact.url	https://aredl.net/
//	@host			api.aredl.net
//	@schemes		https
//	@BasePath		/api

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						api-key
// @description				Perform actions as a user. It is also used to access endpoints that require user permissions.
func main() {
	app := pocketbase.New()

	migration.Register(app)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/docs/*", echoSwagger.WrapHandler)
		return nil
	})

	//app.OnBeforeServe().Add(demonlist.RegisterLevelDataRequester)

	global.RegisterEndpoints(app)
	aredl.RegisterEndpoints(app)

	RegisterUserAuth(app)

	demonlist.RegisterUpdatePoints(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}

}
