package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type Application struct {
	Db         *sql.DB
	FireAuth   *auth.Client
	BlobClient *azblob.Client
}

func App(ctx context.Context, env *Env) Application {
	slog.Info("Starting application")
	app := &Application{}
	app.Db = NewSqlClient(env)
	app.FireAuth = NewFireAuth(ctx)
	app.BlobClient = NewAzureBlobClient(env)

	return *app
}
