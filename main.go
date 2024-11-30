package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	app := bootstrap.App()
	db := app.PostgresDB

	server := gin.Default()

	route.Setup(db, server)

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

	// Start the server
	err := server.Run(":8080")
	if err != nil {
		return
	}

}
