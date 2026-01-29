package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	. "github.com/go-jet/jet/v2/postgres"
)

const (
	shopMessageImagesContainer = "shop-message-images"
)

type RepositoryImpl struct {
	db         *sql.DB
	blobClient *azblob.Client
	env        *bootstrap.Env
}

func NewRepository(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *RepositoryImpl {
	return &RepositoryImpl{
		db:         db,
		blobClient: blobClient,
		env:        env,
	}
}

func (repo *RepositoryImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
	stmt := ShopMessages.INSERT(
		ShopMessages.ID,
		ShopMessages.ShopID,
		ShopMessages.UserID,
		ShopMessages.Message,
		ShopMessages.CreatedAt,
		ShopMessages.UpdatedAt,
		ShopMessages.IsEdited,
	).MODEL(message).RETURNING(ShopMessages.AllColumns)

	var createdMessage model.ShopMessages
	err := stmt.Query(repo.db, &createdMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	return &createdMessage, nil
}

func (repo *RepositoryImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.ASC())

	var messages []model.ShopMessages
	err := stmt.Query(repo.db, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	return messages, nil
}

func (repo *RepositoryImpl) GetShopMessagesPaginated(user *bootstrap.User, shopID string, offset int, limit int) ([]model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	var messages []model.ShopMessages
	err := stmt.Query(repo.db, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated shop messages: %w", err)
	}

	return messages, nil
}

func (repo *RepositoryImpl) GetShopMessagesCount(user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopMessages.ID)).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get shop messages count: %w", err)
	}

	return result.Count, nil
}

func (repo *RepositoryImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
	stmt := ShopMessages.UPDATE(
		ShopMessages.Message,
		ShopMessages.UpdatedAt,
		ShopMessages.IsEdited,
	).SET(
		ShopMessages.Message.SET(String(message.Message)),
		ShopMessages.UpdatedAt.SET(TimestampzT(*message.UpdatedAt)),
		ShopMessages.IsEdited.SET(Bool(*message.IsEdited)),
	).WHERE(
		ShopMessages.ID.EQ(String(message.ID)).
			AND(ShopMessages.UserID.EQ(String(user.UserID))),
	)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to update shop message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("message not found or user not authorized to update")
	}

	return nil
}

func (repo *RepositoryImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
	stmt := ShopMessages.DELETE().
		WHERE(
			ShopMessages.ID.EQ(String(messageID)).
				AND(
					ShopMessages.UserID.EQ(String(user.UserID)).
						OR(
							ShopMessages.ShopID.IN(
								SELECT(ShopMembers.ShopID).
									FROM(ShopMembers).
									WHERE(
										ShopMembers.UserID.EQ(String(user.UserID)).
											AND(ShopMembers.Role.EQ(String("admin"))),
									),
							),
						),
				),
		)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete shop message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("message not found or user not authorized to delete")
	}

	return nil
}

func (repo *RepositoryImpl) GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ID.EQ(String(messageID)))

	var message model.ShopMessages
	err := stmt.Query(repo.db, &message)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (repo *RepositoryImpl) UploadMessageImage(user *bootstrap.User, messageID string, shopID string, imageData []byte, contentType string) (string, string, error) {
	ctx := context.Background()

	// Verify user is a member of the shop
	isMember, err := repo.isUserMemberOfShop(user, shopID)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return "", "", errors.New("access denied: user is not a member of this shop")
	}

	if contentType == "" {
		contentType = http.DetectContentType(imageData)
	}

	fileExtension := getFileExtensionFromMIME(contentType)

	blobName := fmt.Sprintf("%s/%s%s", shopID, messageID, fileExtension)

	_, err = repo.blobClient.UploadBuffer(ctx, shopMessageImagesContainer, blobName, imageData, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload image: %w", err)
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		repo.env.BlobAccountName, shopMessageImagesContainer, blobName)

	slog.Info("shop message image uploaded successfully", "user_id", user.UserID, "message_id", messageID, "shop_id", shopID, "blob_url", blobURL)
	return fileExtension, blobURL, nil
}

func (repo *RepositoryImpl) DeleteMessageImageBlob(user *bootstrap.User, messageID string, shopID string) error {
	ctx := context.Background()

	isMember, err := repo.isUserMemberOfShop(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	extensions := []string{".jpg", ".png", ".gif", ".webp"}
	deleted := false

	for _, ext := range extensions {
		blobName := fmt.Sprintf("%s/%s%s", shopID, messageID, ext)
		_, err := repo.blobClient.DeleteBlob(ctx, shopMessageImagesContainer, blobName, nil)
		if err == nil {
			deleted = true
			slog.Info("shop message image blob deleted successfully", "message_id", messageID, "shop_id", shopID, "extension", ext)
			break
		}
	}

	if !deleted {
		slog.Warn("Failed to delete any image blob - may not exist", "message_id", messageID, "shop_id", shopID)
	}

	return nil
}

func (repo *RepositoryImpl) DeleteBlobByURL(messageText string) error {
	if messageText == "" {
		return nil
	}

	imageURL := extractImageURLFromMessage(messageText)
	if imageURL == "" {
		return nil
	}

	ctx := context.Background()

	blobName, err := parseBlobNameFromURL(imageURL, shopMessageImagesContainer)
	if err != nil {
		slog.Warn("Failed to parse blob name from URL", "url", imageURL, "error", err)
		return nil
	}

	_, err = repo.blobClient.DeleteBlob(ctx, shopMessageImagesContainer, blobName, nil)
	if err != nil {
		slog.Warn("Failed to delete blob from Azure", "blob_name", blobName, "error", err)
		return nil
	}

	slog.Info("Blob deleted successfully from Azure", "blob_name", blobName, "image_url", imageURL)
	return nil
}

func (repo *RepositoryImpl) DeleteShopMessageBlobs(shopID string) error {
	ctx := context.Background()

	containerClient := repo.blobClient.ServiceClient().NewContainerClient(shopMessageImagesContainer)

	prefix := fmt.Sprintf("%s/", shopID)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	deletedCount := 0
	errorCount := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Warn("Failed to list blobs for shop deletion", "shop_id", shopID, "error", err)
			continue
		}

		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			_, err := repo.blobClient.DeleteBlob(ctx, shopMessageImagesContainer, *blob.Name, nil)
			if err != nil {
				slog.Warn("Failed to delete shop message blob",
					"shop_id", shopID,
					"blob_name", *blob.Name,
					"error", err)
				errorCount++
			} else {
				deletedCount++
			}
		}
	}

	slog.Info("Shop message blobs cleanup completed",
		"shop_id", shopID,
		"deleted_count", deletedCount,
		"error_count", errorCount)

	return nil
}

// getFileExtensionFromMIME returns the file extension for a given MIME type
func getFileExtensionFromMIME(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func extractImageURLFromMessage(messageText string) string {
	re := regexp.MustCompile(`\\[IMAGE:(https://[^\\]]+)\\]`)
	matches := re.FindStringSubmatch(messageText)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func parseBlobNameFromURL(url string, expectedContainer string) (string, error) {
	if url == "" {
		return "", errors.New("empty URL")
	}

	containerPrefix := fmt.Sprintf("/%s/", expectedContainer)
	idx := strings.Index(url, containerPrefix)
	if idx == -1 {
		return "", fmt.Errorf("container '%s' not found in URL", expectedContainer)
	}

	blobName := url[idx+len(containerPrefix):]
	if blobName == "" {
		return "", errors.New("blob name is empty")
	}

	return blobName, nil
}

func (repo *RepositoryImpl) isUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return result.Count > 0, nil
}
