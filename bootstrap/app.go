package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"

	"firebase.google.com/go/v4/auth"
)

type Application struct {
	Db       *sql.DB
	FireAuth *auth.Client
}

func App(ctx context.Context, env *Env) Application {
	slog.Info("Starting application")
	app := &Application{}
	app.Db = NewSqlClient(env)
	app.FireAuth = NewFireAuth(ctx)

	return *app
}
