package images

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db         *sql.DB
	blobClient *azblob.Client
	env        *bootstrap.Env
}

func NewRepository(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *RepositoryImpl {
	return &RepositoryImpl{db: db, blobClient: blobClient, env: env}
}

// Upload uploads an item image to Azure Blob Storage and updates the database.
func (repo *RepositoryImpl) Upload(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	if repo.blobClient == nil {
		return "", fmt.Errorf("blob client is not configured")
	}

	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	_, err := repo.blobClient.UploadBuffer(context.TODO(), containerName, blobName, imageData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		repo.env.BlobAccountName, containerName, blobName)

	updated, err := repo.updateImageInTable(user, itemID, blobURL, tableType)
	if err != nil {
		slog.Error("Failed to update image URL in database", "error", err, "blob_url", blobURL, "table_type", tableType)
		return "", fmt.Errorf("failed to update image URL in database for table %s: %w", tableType, err)
	}

	if !updated {
		slog.Error("No rows updated - item not found", "user_id", user.UserID, "item_id", itemID, "table_type", tableType)
		return "", fmt.Errorf("item with ID %s not found in %s table for user %s", itemID, tableType, user.UserID)
	}

	slog.Info("user item image uploaded successfully", "user_id", user.UserID, "item_id", itemID, "table_type", tableType, "blob_url", blobURL)
	return blobURL, nil
}

// Delete deletes an item image from Azure Blob Storage and clears the database.
func (repo *RepositoryImpl) Delete(user *bootstrap.User, itemID string, tableType string) error {
	if repo.blobClient == nil {
		return fmt.Errorf("blob client is not configured")
	}

	ctx := context.Background()
	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	updated, err := repo.updateImageInTable(user, itemID, "", tableType)
	if err != nil {
		slog.Error("Failed to clear image URL in database", "error", err, "item_id", itemID, "table_type", tableType)
		return fmt.Errorf("failed to clear image URL in database for table %s: %w", tableType, err)
	}

	if !updated {
		slog.Error("No rows updated - item not found for deletion", "user_id", user.UserID, "item_id", itemID, "table_type", tableType)
		return fmt.Errorf("item with ID %s not found in %s table for user %s", itemID, tableType, user.UserID)
	}

	_, err = repo.blobClient.DeleteBlob(ctx, containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	slog.Info("image deleted successfully", "user_id", user.UserID, "item_id", itemID, "table_type", tableType)
	return nil
}

// Get retrieves an item image from Azure Blob Storage.
func (repo *RepositoryImpl) Get(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	if repo.blobClient == nil {
		return nil, "", fmt.Errorf("blob client is not configured")
	}

	imageURL, err := repo.getImageURLFromTable(user, itemID, tableType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get image URL from database: %w", err)
	}

	if imageURL == "" {
		return nil, "", fmt.Errorf("no image found for item %s in %s table", itemID, tableType)
	}

	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	ctx := context.Background()
	response, err := repo.blobClient.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image from blob storage: %w", err)
	}
	defer response.Body.Close()

	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	contentType := "image/jpeg"
	if response.ContentType != nil {
		contentType = *response.ContentType
	}

	slog.Info("image retrieved successfully", "user_id", user.UserID, "item_id", itemID, "table_type", tableType, "content_type", contentType)
	return imageData, contentType, nil
}

func (repo *RepositoryImpl) getImageURLFromTable(user *bootstrap.User, itemID string, tableType string) (string, error) {
	validTables := map[string]bool{
		"quick":       true,
		"serialized":  true,
		"categorized": true,
		"category":    true,
	}

	if !validTables[tableType] {
		return "", fmt.Errorf("invalid table type: %s. Valid types are: quick, serialized, categorized, category", tableType)
	}

	var imageURL *string
	var err error

	switch tableType {
	case "quick":
		var item model.UserItemsQuick
		stmt := SELECT(UserItemsQuick.Image).
			FROM(UserItemsQuick).
			WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
				AND(UserItemsQuick.ID.EQ(String(itemID))))
		err = stmt.Query(repo.db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "serialized":
		var item model.UserItemsSerialized
		stmt := SELECT(UserItemsSerialized.Image).
			FROM(UserItemsSerialized).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
				AND(UserItemsSerialized.ID.EQ(String(itemID))))
		err = stmt.Query(repo.db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "categorized":
		var item model.UserItemsCategorized
		stmt := SELECT(UserItemsCategorized.Image).
			FROM(UserItemsCategorized).
			WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.ID.EQ(String(itemID))))
		err = stmt.Query(repo.db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "category":
		var item model.UserItemCategory
		stmt := SELECT(UserItemCategory.Image).
			FROM(UserItemCategory).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
				AND(UserItemCategory.ID.EQ(String(itemID))))
		err = stmt.Query(repo.db, &item)
		if err == nil {
			imageURL = item.Image
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to query %s table: %w", tableType, err)
	}

	if imageURL == nil {
		return "", nil
	}

	return *imageURL, nil
}

func (repo *RepositoryImpl) updateImageInTable(user *bootstrap.User, itemID string, imageURL string, tableType string) (bool, error) {
	validTables := map[string]bool{
		"quick":       true,
		"serialized":  true,
		"categorized": true,
		"category":    true,
	}

	if !validTables[tableType] {
		return false, fmt.Errorf("invalid table type: %s. Valid types are: quick, serialized, categorized, category", tableType)
	}

	var result sql.Result
	var err error

	switch tableType {
	case "quick":
		stmt := UserItemsQuick.UPDATE(UserItemsQuick.Image).
			SET(String(imageURL)).
			WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
				AND(UserItemsQuick.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.db)
	case "serialized":
		stmt := UserItemsSerialized.UPDATE(UserItemsSerialized.Image).
			SET(String(imageURL)).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
				AND(UserItemsSerialized.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.db)
	case "categorized":
		stmt := UserItemsCategorized.UPDATE(UserItemsCategorized.Image).
			SET(String(imageURL)).
			WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.db)
	case "category":
		stmt := UserItemCategory.UPDATE(UserItemCategory.Image).
			SET(String(imageURL)).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
				AND(UserItemCategory.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.db)
	default:
		return false, fmt.Errorf("unknown table type: %s", tableType)
	}

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}
