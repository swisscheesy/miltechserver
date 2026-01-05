package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"
	"miltechserver/api/websocket"

	"firebase.google.com/go/v4/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type Application struct {
	Db             *sql.DB
	FireAuth       *auth.Client
	BlobClient     *azblob.Client
	BlobCredential *azblob.SharedKeyCredential
	Hub            *websocket.Hub
}

func App(ctx context.Context, env *Env) Application {
	slog.Info("Starting application, or not, we'll see.")
	app := &Application{}
	app.Db = NewSqlClient(env)
	app.FireAuth = NewFireAuth(ctx)
	app.BlobClient, app.BlobCredential = NewAzureBlobClient(env)

	// Initialize WebSocket Hub if enabled
	if env.WebSocketEnabled {
		app.Hub = websocket.NewHub()
		go app.Hub.Run()
		slog.Info("WebSocket Hub initialized and running")
	} else {
		slog.Info("WebSocket functionality disabled")
	}

	return *app
}
