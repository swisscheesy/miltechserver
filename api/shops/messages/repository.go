package messages

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"
)

type Repository interface {
	CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error)
	GetShopMessages(user *bootstrap.User, shopID string) ([]response.ShopMessageResponse, error)
	GetShopMessagesPaginated(user *bootstrap.User, shopID string, offset int, limit int) ([]response.ShopMessageResponse, error)
	GetShopMessagesByCursor(user *bootstrap.User, shopID string, cursorTime time.Time, isBefore bool, limit int) ([]response.ShopMessageResponse, error)
	GetShopMessagesCount(user *bootstrap.User, shopID string) (int64, error)
	UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(user *bootstrap.User, messageID string) error
	GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error)
	UploadMessageImage(user *bootstrap.User, messageID string, shopID string, imageData []byte, contentType string) (string, string, error)
	DeleteMessageImageBlob(user *bootstrap.User, messageID string, shopID string) error
	DeleteBlobByURL(messageText string) error
	DeleteShopMessageBlobs(shopID string) error
}
