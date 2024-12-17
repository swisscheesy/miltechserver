package bootstrap

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"gorm.io/gorm"
	"log/slog"
)

type Application struct {
	Db       *gorm.DB
	FireAuth *auth.Client
}

func App(ctx context.Context, env *Env) Application {
	slog.Info("Creating application")
	app := &Application{}
	app.Db = NewGormClient(env)
	app.FireAuth = NewFireAuth(ctx)

	return *app
}
