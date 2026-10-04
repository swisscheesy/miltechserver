package messages

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
)

const (
	maxImageSize = 5 * 1024 * 1024
	// LegacyCursorReloadMessage is the safe legacy error text when an ID anchor
	// can no longer supply its stored timestamp. Clients must reload their page.
	LegacyCursorReloadMessage = "Message cursor is unavailable; reload messages"
)

func legacyCursorReload() error {
	return &shared.Failure{Code: "reset_required", PublicMessage: LegacyCursorReloadMessage, Status: http.StatusConflict}
}

type ServiceImpl struct {
	repo               Repository
	blobs              BlobStore
	auth               shared.ShopAuthorization
	messageSyncEnabled bool
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
		auth: auth,
	}
}

// WithMessageSync returns a copy that serves the additive sync reads. Off by
// default: the counter migration and every writer must be verified first.
func (service *ServiceImpl) WithMessageSync(enabled bool) *ServiceImpl {
	copied := *service
	copied.messageSyncEnabled = enabled
	return &copied
}

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	copied := *service
	copied.auth = auth
	return &copied
}

func (service *ServiceImpl) WithBlobStore(blobs BlobStore) *ServiceImpl {
	copied := *service
	copied.blobs = blobs
	return &copied
}

func (service *ServiceImpl) CreateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, message.ShopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	message.ID = uuid.New().String()
	message.UserID = user.UserID
	now := time.Now()
	message.CreatedAt = &now
	message.UpdatedAt = &now
	message.IsEdited = func() *bool { b := false; return &b }()

	// Parent ownership must be checked by the repository under the same locks
	// as insertion; a service-level read would race parent deletion.
	createdMessage, err := service.repo.CreateShopMessage(ctx, user, message)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	slog.Info("Shop message created", "user_id", user.UserID, "shop_id", message.ShopID, "message_id", message.ID)
	return createdMessage, nil
}

func (service *ServiceImpl) GetShopMessages(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMessageResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	messages, err := service.repo.GetShopMessages(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	if messages == nil {
		return []response.ShopMessageResponse{}, nil
	}

	return messages, nil
}

func (service *ServiceImpl) GetShopMessagesPaginated(ctx context.Context, user *bootstrap.User, shopID string, req request.GetShopMessagesPaginatedRequest) (*response.PaginatedShopMessagesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	isMember, err := service.auth.IsUserMemberOfShop(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	if req.BeforeID != nil && req.AfterID != nil {
		return nil, errors.New("before_id and after_id cannot be used together")
	}

	if req.BeforeID != nil || req.AfterID != nil {
		cursorID := req.BeforeID
		isBefore := true
		if req.AfterID != nil {
			cursorID = req.AfterID
			isBefore = false
		}

		cursorMessage, err := service.repo.GetShopMessageByID(ctx, user, *cursorID)
		if errors.Is(err, ErrMessageNotFound) {
			return nil, legacyCursorReload()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load cursor message: %w", err)
		}
		if cursorMessage.ShopID != shopID {
			return nil, errors.New("cursor message does not belong to this shop")
		}
		if cursorMessage.CreatedAt == nil {
			return nil, legacyCursorReload()
		}

		messages, err := service.repo.GetShopMessagesByCursor(ctx, user, shopID, cursorMessage.ID, *cursorMessage.CreatedAt, isBefore, req.Limit+1)
		if err != nil {
			return nil, fmt.Errorf("failed to get cursor-based shop messages: %w", err)
		}

		hasMore := len(messages) > req.Limit
		if hasMore {
			messages = messages[:req.Limit]
		}

		if messages == nil {
			messages = []response.ShopMessageResponse{}
		}

		var nextCursor *string
		if hasMore && len(messages) > 0 {
			lastMessageID := messages[len(messages)-1].ID
			nextCursor = &lastMessageID
		}

		// Continuation follows selection order, before restoring legacy DESC display.
		if !isBefore {
			slices.Reverse(messages)
		}
		return &response.PaginatedShopMessagesResponse{
			Messages:   messages,
			Pagination: nil,
			NextCursor: nextCursor,
		}, nil
	}

	offset := (req.Page - 1) * req.Limit

	messages, err := service.repo.GetShopMessagesPaginated(ctx, user, shopID, offset, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated shop messages: %w", err)
	}

	totalCount, err := service.repo.GetShopMessagesCount(ctx, user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages count: %w", err)
	}

	totalPages := int((totalCount + int64(req.Limit) - 1) / int64(req.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	paginationMetadata := response.PaginationMetadata{
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
		HasNext:    req.Page < totalPages,
		HasPrev:    req.Page > 1,
	}

	if messages == nil {
		messages = []response.ShopMessageResponse{}
	}

	paginatedResponse := &response.PaginatedShopMessagesResponse{
		Messages:   messages,
		Pagination: &paginationMetadata,
	}

	return paginatedResponse, nil
}

func (service *ServiceImpl) UpdateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	message.UserID = user.UserID
	now := time.Now()
	message.UpdatedAt = &now
	message.IsEdited = func() *bool { b := true; return &b }()

	err := service.repo.UpdateShopMessage(ctx, user, message)
	if err != nil {
		return fmt.Errorf("failed to update shop message: %w", err)
	}

	slog.Info("Shop message updated", "user_id", user.UserID, "message_id", message.ID)
	return nil
}

func (service *ServiceImpl) DeleteShopMessage(ctx context.Context, user *bootstrap.User, messageID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.repo.DeleteShopMessage(ctx, user, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete shop message: %w", err)
	}

	slog.Info("Shop message deleted", "user_id", user.UserID, "message_id", messageID)
	return nil
}

func (service *ServiceImpl) UploadMessageImage(ctx context.Context, user *bootstrap.User, shopID string, imageData []byte, contentType string) (ImageUpload, error) {
	if user == nil || user.UserID == "" {
		return ImageUpload{}, errors.New("unauthorized user")
	}
	shop, err := uuid.Parse(shopID)
	if err != nil || shop == uuid.Nil {
		return ImageUpload{}, errors.New("invalid shop ID")
	}
	if len(imageData) == 0 || len(imageData) > maxImageSize {
		return ImageUpload{}, errors.New("invalid image size")
	}
	if service.blobs == nil {
		return ImageUpload{}, errors.New("asset storage unavailable")
	}
	if contentType == "" {
		contentType = http.DetectContentType(imageData)
	}
	ctx, cancel := context.WithTimeout(ctx, UploadOperationTimeout)
	defer cancel()
	asset, err := service.repo.ReserveMessageImage(ctx, user, shopID, getFileExtensionFromMIME(contentType))
	if err != nil {
		return ImageUpload{}, fmt.Errorf("reserve message image: %w", err)
	}
	err = service.blobs.Upload(ctx, asset, imageData, contentType)
	if err == nil {
		err = service.repo.FinalizeMessageImage(ctx, user, asset.ID)
	}
	if err != nil {
		// A canceled request cannot authorize detached business work. The lease lets
		// the cleanup worker recover interrupted PUTs. Conditional compensation also
		// protects ready assets when finalization committed but its response was lost.
		if ctx.Err() == nil {
			if cleanupErr := service.repo.FailMessageImage(ctx, asset); cleanupErr != nil {
				slog.Warn("Message upload compensation deferred", "upload_id", asset.ID)
				err = errors.Join(err, cleanupErr)
			}
		}
		return ImageUpload{}, fmt.Errorf("upload message image: %w", err)
	}
	return ImageUpload{MessageID: asset.ID, ShopID: asset.ShopID, ImageURL: asset.URL, FileExtension: asset.Extension}, nil
}

func (service *ServiceImpl) DeleteMessageImage(ctx context.Context, user *bootstrap.User, shopID string, messageID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.repo.DeleteMessageImageBlob(ctx, user, messageID, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete message image: %w", err)
	}

	slog.Info("shop_message_image_cleanup_queued", "user_id", user.UserID, "shop_id", shopID, "message_id", messageID)
	return nil
}

func (service *ServiceImpl) syncReader() (SyncReader, error) {
	if !service.messageSyncEnabled {
		return nil, syncUnavailable()
	}
	reader, ok := service.repo.(SyncReader)
	if !ok {
		return nil, syncUnavailable()
	}
	return reader, nil
}

func (service *ServiceImpl) InitialMessages(ctx context.Context, user *bootstrap.User, shopID string, limit int) (*MessageInitial, error) {
	reader, err := service.syncReader()
	if err != nil {
		return nil, err
	}
	return reader.InitialMessages(ctx, user, shopID, limit)
}

func (service *ServiceImpl) MessageHistory(ctx context.Context, user *bootstrap.User, shopID, cursor string, limit int) (*MessageHistory, error) {
	reader, err := service.syncReader()
	if err != nil {
		return nil, err
	}
	return reader.MessageHistory(ctx, user, shopID, cursor, limit)
}

func (service *ServiceImpl) CatchUpMessages(ctx context.Context, user *bootstrap.User, shopID, after string, through *string, limit int) (*MessageCatchUp, error) {
	reader, err := service.syncReader()
	if err != nil {
		return nil, err
	}
	return reader.CatchUpMessages(ctx, user, shopID, after, through, limit)
}

func (service *ServiceImpl) ReconcileMessages(ctx context.Context, user *bootstrap.User, shopID string, ids []string) (*MessageReconcile, error) {
	reader, err := service.syncReader()
	if err != nil {
		return nil, err
	}
	return reader.ReconcileMessages(ctx, user, shopID, ids)
}
