package images

import (
	"context"
	"miltechserver/bootstrap"
)

type Service interface {
	Upload(ctx context.Context, user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
	Delete(ctx context.Context, user *bootstrap.User, itemID string, tableType string) error
	Get(ctx context.Context, user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
