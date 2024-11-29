package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
)

func main() {

	app := bootstrap.App()
	db := app.PostgresDB

	//defer here?
	defer func() {
		if err := app.PostgresDB.Disconnect(); err != nil {
			log.Fatalf("Unable to disconnect from database: %s", err)
		}
	}()

	server := gin.Default()

	route.Setup(db, server)

	err := server.Run(":8080")
	if err != nil {
		return
	}

}
