package bootstrap

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"log/slog"
	"miltechserver/prisma/db"
)

type Application struct {
	Env        *Env
	PostgresDB *db.PrismaClient
	FireAuth   *auth.Client
}

func App(ctx context.Context) Application {
	slog.Info("Creating application")
	app := &Application{}
	app.Env = NewEnv()
	app.PostgresDB = NewPrismaClient(app.Env)
	app.FireAuth = NewFireAuth(ctx)

	return *app
}
