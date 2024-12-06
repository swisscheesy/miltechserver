package bootstrap

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"log/slog"
	"miltechserver/prisma/db"
)

type Application struct {
	PostgresDB *db.PrismaClient
	FireAuth   *auth.Client
}

func App(ctx context.Context) Application {
	slog.Info("Creating application")
	app := &Application{}
	app.PostgresDB = NewPrismaClient()
	app.FireAuth = NewFireAuth(ctx)

	return *app
}
