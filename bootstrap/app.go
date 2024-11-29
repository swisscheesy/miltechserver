package bootstrap

import (
	"log/slog"
	"miltechserver/prisma/db"
)

type Application struct {
	Env        *Env
	PostgresDB *db.PrismaClient
}

func App() Application {
	slog.Info("Creating application")
	app := &Application{}
	app.Env = NewEnv()
	app.PostgresDB = NewPrismaClient(app.Env)

	return *app
}
