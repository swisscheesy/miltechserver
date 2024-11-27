package bootstrap

import (
	"fmt"
	"log"
	"miltechserver/prisma/db"
)

func NewPrismaClient(env *Env) *db.PrismaClient {
	client := db.NewClient()
	_ = fmt.Sprintf("Connecgting to Database: %s", env.ServerAddress)

	if err := client.Connect(); err != nil {
		log.Fatalf("Unable to connect to database: %s", err)
	}

	log.Println("Connected to Database!")

	return client
}
