package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Start the engine

	engine := SetupEngine()

	err := engine.Run(":8080")
	if err != nil {
		return
	}

}

func SetupEngine() *gin.Engine {
	ctx := context.Background()
	app := bootstrap.App(ctx)
	db := app.PostgresDB

	server := gin.Default()

	route.Setup(db, server, app.FireAuth)

	// Cleanup server on crash or interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-c
		if err := app.PostgresDB.Disconnect(); err != nil {
			log.Fatalf("Unable to disconnect from database: %s", err)
		}
		log.Println("Disconnected from database")
		os.Exit(1)
	}()

	return server
}
