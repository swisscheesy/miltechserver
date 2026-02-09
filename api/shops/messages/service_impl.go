package messages

import (
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/request"
	"miltechserver/api/response"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/google/uuid"
)

const (
	maxImageSize = 5 * 1024 * 1024
)

type ServiceImpl struct {
	repo Repository
	auth shared.ShopAuthorization
}

func NewService(repo Repository, auth shared.ShopAuthorization) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
		auth: auth,
	}
}

func (service *ServiceImpl) WithAuthorization(auth shared.ShopAuthorization) shared.AuthorizationAware {
	return &ServiceImpl{
		repo: service.repo,
		auth: auth,
	}
}

func (service *ServiceImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, message.ShopID)
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

	createdMessage, err := service.repo.CreateShopMessage(user, message)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	slog.Info("Shop message created", "user_id", user.UserID, "shop_id", message.ShopID, "message_id", message.ID)
	return createdMessage, nil
}

func (service *ServiceImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	if !isMember {
		return nil, errors.New("access denied: user is not a member of this shop")
	}

	messages, err := service.repo.GetShopMessages(user, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	if messages == nil {
		return []model.ShopMessages{}, nil
	}

	return messages, nil
}

func (service *ServiceImpl) GetShopMessagesPaginated(user *bootstrap.User, shopID string, req request.GetShopMessagesPaginatedRequest) (*response.PaginatedShopMessagesResponse, error) {
	if user == nil {
		return nil, errors.New("unauthorized user")
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	isMember, err := service.auth.IsUserMemberOfShop(user, shopID)
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

		cursorMessage, err := service.repo.GetShopMessageByID(user, *cursorID)
		if err != nil {
			return nil, fmt.Errorf("failed to load cursor message: %w", err)
		}
		if cursorMessage.ShopID != shopID {
			return nil, errors.New("cursor message does not belong to this shop")
		}
		if cursorMessage.CreatedAt == nil {
			return nil, errors.New("cursor message missing created_at")
		}

		messages, err := service.repo.GetShopMessagesByCursor(user, shopID, *cursorMessage.CreatedAt, isBefore, req.Limit+1)
		if err != nil {
			return nil, fmt.Errorf("failed to get cursor-based shop messages: %w", err)
		}

		hasMore := len(messages) > req.Limit
		if hasMore {
			messages = messages[:req.Limit]
		}

		if messages == nil {
			messages = []model.ShopMessages{}
		}

		var nextCursor *string
		if hasMore && len(messages) > 0 {
			lastMessageID := messages[len(messages)-1].ID
			nextCursor = &lastMessageID
		}

		return &response.PaginatedShopMessagesResponse{
			Messages:   messages,
			Pagination: nil,
			NextCursor: nextCursor,
		}, nil
	}

	offset := (req.Page - 1) * req.Limit

	messages, err := service.repo.GetShopMessagesPaginated(user, shopID, offset, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated shop messages: %w", err)
	}

	totalCount, err := service.repo.GetShopMessagesCount(user, shopID)
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
		messages = []model.ShopMessages{}
	}

	paginatedResponse := &response.PaginatedShopMessagesResponse{
		Messages:   messages,
		Pagination: &paginationMetadata,
	}

	return paginatedResponse, nil
}

func (service *ServiceImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	message.UserID = user.UserID
	now := time.Now()
	message.UpdatedAt = &now
	message.IsEdited = func() *bool { b := true; return &b }()

	err := service.repo.UpdateShopMessage(user, message)
	if err != nil {
		return fmt.Errorf("failed to update shop message: %w", err)
	}

	slog.Info("Shop message updated", "user_id", user.UserID, "message_id", message.ID)
	return nil
}

func (service *ServiceImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	message, err := service.repo.GetShopMessageByID(user, messageID)
	if err != nil {
		return fmt.Errorf("failed to get shop message: %w", err)
	}

	err = service.repo.DeleteShopMessage(user, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete shop message: %w", err)
	}

	if message != nil && message.Message != "" {
		err = service.repo.DeleteBlobByURL(message.Message)
		if err != nil {
			slog.Warn("Failed to delete blob during message deletion",
				"message_id", messageID,
				"user_id", user.UserID,
				"error", err)
		}
	}

	slog.Info("Shop message deleted", "user_id", user.UserID, "message_id", messageID)
	return nil
}

func (service *ServiceImpl) UploadMessageImage(user *bootstrap.User, shopID string, imageData []byte, contentType string) (string, string, string, error) {
	if user == nil {
		return "", "", "", errors.New("unauthorized user")
	}

	if len(imageData) > maxImageSize {
		return "", "", "", fmt.Errorf("image size exceeds maximum allowed size of %d bytes", maxImageSize)
	}

	if len(imageData) == 0 {
		return "", "", "", errors.New("image data is empty")
	}

	messageID := uuid.New().String()

	fileExtension, imageURL, err := service.repo.UploadMessageImage(user, messageID, shopID, imageData, contentType)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to upload message image: %w", err)
	}

	slog.Info("Shop message image uploaded via service", "user_id", user.UserID, "shop_id", shopID, "message_id", messageID)
	return messageID, fileExtension, imageURL, nil
}

func (service *ServiceImpl) DeleteMessageImage(user *bootstrap.User, shopID string, messageID string) error {
	if user == nil {
		return errors.New("unauthorized user")
	}

	err := service.repo.DeleteMessageImageBlob(user, messageID, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete message image: %w", err)
	}

	slog.Info("Shop message image deleted", "user_id", user.UserID, "shop_id", shopID, "message_id", messageID)
	return nil
}
