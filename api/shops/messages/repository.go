package messages

import (
	"context"
	"errors"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"time"
)

// ErrMessageNotFound distinguishes an absent anchor from a failed or canceled read.
var ErrMessageNotFound = errors.New("message not found")

type Repository interface {
	CreateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error)
	GetShopMessages(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMessageResponse, error)
	GetShopMessagesPaginated(ctx context.Context, user *bootstrap.User, shopID string, offset int, limit int) ([]response.ShopMessageResponse, error)
	GetShopMessagesByCursor(ctx context.Context, user *bootstrap.User, shopID, cursorID string, cursorTime time.Time, isBefore bool, limit int) ([]response.ShopMessageResponse, error)
	GetShopMessagesCount(ctx context.Context, user *bootstrap.User, shopID string) (int64, error)
	UpdateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(ctx context.Context, user *bootstrap.User, messageID string) error
	GetShopMessageByID(ctx context.Context, user *bootstrap.User, messageID string) (*response.ShopMessageResponse, error)
	ReserveMessageImage(ctx context.Context, user *bootstrap.User, shopID, extension string) (Asset, error)
	FinalizeMessageImage(ctx context.Context, user *bootstrap.User, uploadID string) error
	FailMessageImage(ctx context.Context, asset Asset) error
	DeleteMessageImageBlob(ctx context.Context, user *bootstrap.User, messageID string, shopID string) error
}
