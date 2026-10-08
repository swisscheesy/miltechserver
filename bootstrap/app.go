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

func App(ctx context.Context, env *Env) (Application, error) {
	slog.Info("Starting application, or not, we'll see.")
	authClient, err := NewFireAuth(ctx)
	if err != nil {
		return Application{}, err
	}
	app := &Application{FireAuth: authClient}
	app.Db = NewSqlClient(env)
	app.BlobClient = NewAzureBlobClient(env)

	return *app, nil
}
