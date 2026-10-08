package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// defaultFirebaseKeyPath is relative to the working directory: the Dockerfile
// copies the key into WORKDIR /app, and local runs start from the repo root
// where the gitignored key lives. The container does not load .env, so without
// this default it would have no way to locate the key.
const defaultFirebaseKeyPath = "fire_auth_key.json"

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// firebaseKeyPath returns FIREBASE_AUTH_KEY when set, so a deployment can
// still point at a mounted secret instead of the baked-in key.
func firebaseKeyPath() string {
	if path := os.Getenv("FIREBASE_AUTH_KEY"); path != "" {
		return path
	}
	return defaultFirebaseKeyPath
}

func NewFirebaseApp(ctx context.Context) (*firebase.App, error) {
	keyPath := firebaseKeyPath()
	slog.Info("Loading Firebase credentials", "path", keyPath)

	// The file is read lazily when a service client is built; NewFireAuth builds
	// one at startup, so a missing key still stops the server before it serves.
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(keyPath))
	if err != nil {
		return nil, fmt.Errorf("firebase initialization error using credentials %q: %w", keyPath, err)
	}
	return app, nil
}

// NewFireAuth creates a Firebase auth client or returns its initialization error.
func NewFireAuth(ctx context.Context) (*auth.Client, error) {
	slog.Info("Creating Firebase Auth client")
	fireApp, err := NewFirebaseApp(ctx)
	if err != nil {
		return nil, err
	}
	authClient, err := fireApp.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth client initialization error: %w", err)
	}
	slog.Info("Firebase Auth client created")
	return authClient, nil
}
