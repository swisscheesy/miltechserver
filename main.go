package main

import (
	"github.com/gin-gonic/gin"
	"miltechserver/api/route"
	"miltechserver/bootstrap"
	"time"
)

func main() {
	app := bootstrap.App()
	env := app.Env
	db := app.PostgresDB
	//defer here?

	timeout := time.Duration(env.ContextTimeout) * time.Second

	server := gin.Default()

	route.Setup(env, timeout, db, server)

	err := server.Run(":8080")
	if err != nil {
		return
	}

}
