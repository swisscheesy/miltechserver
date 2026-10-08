package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func NewFirebaseApp(ctx context.Context) (*firebase.App, error) {
	accountKey := os.Getenv("FIREBASE_AUTH_KEY")
	if accountKey == "" {
		return nil, errors.New("FIREBASE_AUTH_KEY is required")
	}
	// Read the runtime mount before initialization so missing credentials fail before other clients start.
	credentials, err := os.ReadFile(accountKey)
	if err != nil {
		return nil, errors.New("unable to read Firebase credentials")
	}
	opt := option.WithCredentialsJSON(credentials)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, errors.New("firebase initialization failed")
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
