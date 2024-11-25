package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"miltechserver/bootstrap"
	"miltechserver/prisma/db"
)

func main() {

	server := gin.Default()

	pDB, dbErr := bootstrap.ConnectDB()
	if dbErr != nil {
		log.Fatal("Cannot connect to database")
	}

	server.GET("/ping", func(ctx *gin.Context) {
		client := pDB.Client
		test, err := client.Nsn.FindFirst(db.Nsn.Niin.Equals("013469317")).Exec(pDB.Context)
		if err != nil {
			ctx.JSON(200, gin.H{
				"message": "error",
				"data":    err,
			})
			return
		}
		ctx.JSON(200, gin.H{
			"message": "pong",
			"data":    test,
		})
	})

	defer pDB.Client.Disconnect()

	err := server.Run(":8080")
	if err != nil {
		return
	}

}
