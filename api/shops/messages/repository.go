package messages

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type Repository interface {
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error)
	GetShopMessagesPaginated(user *bootstrap.User, shopID string, offset int, limit int) ([]model.ShopMessages, error)
	GetShopMessagesCount(user *bootstrap.User, shopID string) (int64, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error
	GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error)
	UploadMessageImage(user *bootstrap.User, messageID string, shopID string, imageData []byte, contentType string) (string, string, error)
	DeleteMessageImageBlob(user *bootstrap.User, messageID string, shopID string) error
	DeleteBlobByURL(messageText string) error
	DeleteShopMessageBlobs(shopID string) error
}
