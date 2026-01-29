package messages

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error)
	GetShopMessagesPaginated(user *bootstrap.User, shopID string, page int, limit int) (*response.PaginatedShopMessagesResponse, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error
	UploadMessageImage(user *bootstrap.User, shopID string, imageData []byte, contentType string) (string, string, string, error)
	DeleteMessageImage(user *bootstrap.User, shopID string, messageID string) error
}
