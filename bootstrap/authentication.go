package bootstrap

import (
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"fmt"
	"google.golang.org/api/option"
	"log/slog"
	"os"
)

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func NewFirebaseApp(ctx context.Context) (*firebase.App, error) {
	accountKey := os.Getenv("FIREBASE_AUTH_KEY")

	opt := option.WithCredentialsFile(accountKey)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase initialization error: %v", err)
	}
	return app, nil
}

// NewFireAuth creates a returns a new firebase auth client
func NewFireAuth(ctx context.Context) *auth.Client {
	slog.Info("Creating Firebase Auth client")
	fireApp, err := NewFirebaseApp(ctx)
	if err != nil {
		return nil
	}
	authClient, err := fireApp.Auth(ctx)
	if err != nil {
		slog.Error("error getting auth client: %v", err)
		return nil
	}
	slog.Info("Firebase Auth client created")
	return authClient
}
