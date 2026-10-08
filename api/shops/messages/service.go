package messages

import (
	"context"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
)

type Service interface {
	CreateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error)
	GetShopMessages(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMessageResponse, error)
	GetShopMessagesPaginated(ctx context.Context, user *bootstrap.User, shopID string, req request.GetShopMessagesPaginatedRequest) (*response.PaginatedShopMessagesResponse, error)
	UpdateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) error
	DeleteShopMessage(ctx context.Context, user *bootstrap.User, messageID string) error
	UploadMessageImage(ctx context.Context, user *bootstrap.User, shopID string, imageData []byte, contentType string) (ImageUpload, error)
	DeleteMessageImage(ctx context.Context, user *bootstrap.User, shopID string, messageID string) error
}
