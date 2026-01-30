package images

import "miltechserver/bootstrap"

type Repository interface {
	Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error)
	Delete(user *bootstrap.User, itemID string, tableType string) error
	Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error)
}
