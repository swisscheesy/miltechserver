package bootstrap

import (
	"context"
	"database/sql"
	"firebase.google.com/go/v4/auth"
	"log/slog"
)

type Application struct {
	Db       *sql.DB
	FireAuth *auth.Client
}

func App(ctx context.Context, env *Env) Application {
	slog.Info("Creating application")
	app := &Application{}
	app.Db = NewSqlClient(env)
	app.FireAuth = NewFireAuth(ctx)

	return *app
}
