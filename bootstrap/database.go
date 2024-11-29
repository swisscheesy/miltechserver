package bootstrap

import (
	"log/slog"
	"miltechserver/prisma/db"
)

func NewPrismaClient(env *Env) *db.PrismaClient {
	client := db.NewClient()
	slog.Info("Connecting to Database: %s", env.ServerAddress)

	if err := client.Connect(); err != nil {
		slog.Error("Unable to connect to database: %s", err)
	}

	slog.Info("Connected to Database!")

	return client
}
