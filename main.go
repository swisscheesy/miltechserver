package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
	"miltechserver/helper"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Start the engine

	engine := SetupEngine()

	err := engine.Run(":8080")
	helper.PanicOnError(err)

}

func SetupEngine() *gin.Engine {
	ctx := context.Background()
	env := bootstrap.NewEnv()
	app := bootstrap.App(ctx, env)
	db := app.Db

	server := gin.Default()

	route.Setup(db, server, app.FireAuth)

	// Cleanup server on crash or interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		if err := db.Close(); err != nil {
			log.Fatalf("Unable to disconnect from database: %s", err)
		}
		log.Println("Disconnected from database")
		os.Exit(1)
	}()

	return server
}
