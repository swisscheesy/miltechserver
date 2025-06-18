package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	. "github.com/go-jet/jet/v2/postgres"
)

type UserSavesRepositoryImpl struct {
	Db         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
}

func NewUserSavesRepositoryImpl(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *UserSavesRepositoryImpl {
	return &UserSavesRepositoryImpl{Db: db, BlobClient: blobClient, Env: env}
}

func (repo *UserSavesRepositoryImpl) GetQuickSaveItemsByUserId(user *bootstrap.User) ([]model.UserItemsQuick, error) {
	var items []model.UserItemsQuick

	if user != nil {
		stmt := SELECT(
			UserItemsQuick.AllColumns).FROM(UserItemsQuick).WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &items)
		if err != nil {
			return nil, errors.New("user saves not found")
		} else {
			slog.Info("quick saves retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}

}

func (repo *UserSavesRepositoryImpl) UpsertQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
		UserItemsQuick.Image, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID).
		MODEL(quick).
		ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ID).
		DO_UPDATE(
			SET(
				UserItemsQuick.LastUpdated.SET(TimestampT(*quick.LastUpdated)),
				UserItemsQuick.Image.SET(String(*quick.Image)),
				UserItemsQuick.ItemComment.SET(String(*quick.ItemComment))).
				WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
					AND(UserItemsQuick.ID.EQ(String(quick.ID))).
					AND(UserItemsQuick.UserID.EQ(String(user.UserID))).
					AND(UserItemsQuick.Niin.EQ(String(quick.Niin))))).
		RETURNING(UserItemsQuick.AllColumns)

	err := stmt.Query(repo.Db, &quick)

	if err != nil {
		return errors.New("error saving quick item " + err.Error())
	} else {
		slog.Info("quick save item saved", "user_id", user.UserID, "niin", quick.Niin)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) DeleteQuickSaveItemByUser(user *bootstrap.User, quick model.UserItemsQuick) error {
	stmt := UserItemsQuick.
		DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
			AND(UserItemsQuick.Niin.EQ(String(quick.Niin))))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting quick item")
	} else {
		slog.Info("quick save item deleted", "user_id", user.UserID, "niin", quick.Niin)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) UpsertQuickSaveItemListByUser(user *bootstrap.User, quickItems []model.UserItemsQuick) error {

	var failedNiins []string
	for _, val := range quickItems {
		stmt := UserItemsQuick.INSERT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ItemName,
			UserItemsQuick.Image, UserItemsQuick.ItemComment, UserItemsQuick.SaveTime, UserItemsQuick.LastUpdated, UserItemsQuick.ID).
			MODEL(val).
			ON_CONFLICT(UserItemsQuick.UserID, UserItemsQuick.Niin, UserItemsQuick.ID).
			DO_UPDATE(
				SET(
					UserItemsQuick.LastUpdated.SET(TimestampT(*val.LastUpdated)),
					UserItemsQuick.Image.SET(String(*val.Image)),
					UserItemsQuick.ItemComment.SET(String(*val.ItemComment))).
					WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)).
						AND(UserItemsQuick.ID.EQ(String(val.ID))).
						AND(UserItemsQuick.Niin.EQ(String(val.Niin))))).
			RETURNING(UserItemsQuick.AllColumns)

		err := stmt.Query(repo.Db, &quickItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}

	}

	if len(failedNiins) > 0 {
		return fmt.Errorf(fmt.Sprintf("failed to save following items: %s", failedNiins))
	} else {
		slog.Info("quick save item list inserted", "user_id", user.UserID)
		return nil
	}

}

func (repo *UserSavesRepositoryImpl) DeleteAllQuickSaveItemsByUser(user *bootstrap.User) error {
	stmt := UserItemsQuick.DELETE().
		WHERE(UserItemsQuick.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting quick items")
	} else {
		slog.Info("all quick save items deleted", "user_id", user.UserID)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) GetSerializedItemsByUserId(user *bootstrap.User) ([]model.UserItemsSerialized, error) {
	var items []model.UserItemsSerialized

	if user != nil {
		stmt := SELECT(
			UserItemsSerialized.AllColumns).
			FROM(UserItemsSerialized).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &items)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving serialized saves for user %s", user.UserID))
		} else {
			slog.Info("serialized saves retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}
}

func (repo *UserSavesRepositoryImpl) UpsertSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.INSERT(UserItemsSerialized.ID, UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
		UserItemsSerialized.Serial,
		UserItemsSerialized.Image, UserItemsSerialized.SaveTime, UserItemsSerialized.ItemComment, UserItemsSerialized.LastUpdated).
		MODEL(serializedItem).
		ON_CONFLICT(UserItemsSerialized.ID, UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.Serial).
		DO_UPDATE(
			SET(UserItemsSerialized.Image.
				SET(String(*serializedItem.Image)),
				UserItemsSerialized.ItemComment.
					SET(String(*serializedItem.ItemComment)),
				UserItemsSerialized.LastUpdated.
					SET(TimestampT(*serializedItem.LastUpdated))).
				WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
					AND(UserItemsSerialized.ID.EQ(String(serializedItem.ID))).
					AND(UserItemsSerialized.Serial.EQ(String(serializedItem.Serial))).
					AND(UserItemsSerialized.Niin.EQ(String(serializedItem.Niin))))).
		RETURNING(
			UserItemsSerialized.AllColumns)

	err := stmt.Query(repo.Db, &serializedItem)

	if err != nil {
		return errors.New("error saving serialized item")
	}

	slog.Info("serialized save item saved", "user_id", user.UserID, "niin", serializedItem.Niin)
	return nil
}

func (repo *UserSavesRepositoryImpl) DeleteAllSerializedItemsByUser(user *bootstrap.User) error {
	stmt := UserItemsSerialized.DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting serialized items")
	} else {
		slog.Info("all serialized save items deleted", "user_id", user.UserID)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) UpsertSerializedSaveItemListByUser(user *bootstrap.User, serializedItems []model.UserItemsSerialized) error {

	var failedNiins []string
	for _, val := range serializedItems {
		stmt := UserItemsSerialized.INSERT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.ItemName,
			UserItemsSerialized.Serial, UserItemsSerialized.SaveTime,
			UserItemsSerialized.Image, UserItemsSerialized.ItemComment, UserItemsSerialized.ID, UserItemsSerialized.LastUpdated).
			MODEL(val).
			ON_CONFLICT(UserItemsSerialized.UserID, UserItemsSerialized.Niin, UserItemsSerialized.Serial).
			DO_UPDATE(
				SET(UserItemsSerialized.Image.
					SET(String(*val.Image)),
					UserItemsSerialized.ItemComment.
						SET(String(*val.ItemComment)),
					UserItemsSerialized.LastUpdated.
						SET(TimestampT(*val.LastUpdated))).
					WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
						AND(UserItemsSerialized.ID.EQ(String(val.ID))).
						AND(UserItemsSerialized.Niin.EQ(String(val.Niin)).
							AND(UserItemsSerialized.Serial.EQ(String(val.Serial))))))

		err := stmt.Query(repo.Db, &serializedItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}

	}
	if len(failedNiins) > 0 {
		return errors.New(fmt.Sprintf("failed to save following items: %s", failedNiins))
	} else {
		slog.Info("serialized save item list inserted", "user_id", user.UserID)
		return nil
	}
}

func (repo *UserSavesRepositoryImpl) DeleteSerializedSaveItemByUser(user *bootstrap.User, serializedItem model.UserItemsSerialized) error {
	stmt := UserItemsSerialized.
		DELETE().
		WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
			AND(UserItemsSerialized.Niin.EQ(String(serializedItem.Niin)).
				AND(UserItemsSerialized.Serial.EQ(String(serializedItem.Serial)))))

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting serialized item")
	}

	slog.Info("serialized save item deleted", "user_id", user.UserID, "niin", serializedItem.Niin)
	return nil
}

func (repo *UserSavesRepositoryImpl) GetUserItemCategories(user *bootstrap.User) ([]model.UserItemCategory, error) {
	var categories []model.UserItemCategory

	if user != nil {
		stmt := SELECT(UserItemCategory.AllColumns).
			FROM(UserItemCategory).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &categories)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving item categories for user %s", user.UserID))
		} else {
			slog.Info("item categories retrieved for user", "user_id", user.UserID)
			return categories, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}
}

func (repo *UserSavesRepositoryImpl) UpsertUserItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	stmt := UserItemCategory.
		INSERT(UserItemCategory.ID, UserItemCategory.UserUID,
			UserItemCategory.Name, UserItemCategory.Comment,
			UserItemCategory.Image, UserItemCategory.LastUpdated).
		MODEL(itemCategory).
		ON_CONFLICT(UserItemCategory.ID, UserItemCategory.UserUID).
		DO_UPDATE(
			SET(UserItemCategory.Name.
				SET(String(itemCategory.Name)),
				UserItemCategory.Comment.
					SET(String(*itemCategory.Comment)), //TODO Was *itemCategory.Comment
				UserItemCategory.Image.
					SET(String(*itemCategory.Image)),
				UserItemCategory.LastUpdated.
					SET(TimestampT(*itemCategory.LastUpdated))).
				WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))).
		RETURNING(UserItemCategory.AllColumns)

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error saving item category")
	}

	slog.Info("item category saved", "user_id", user.UserID, "category_name", itemCategory.Name)
	return nil
}

// Deletes a single item category and all items in that category
func (repo *UserSavesRepositoryImpl) DeleteUserItemCategory(user *bootstrap.User, itemCategory model.UserItemCategory) error {
	cat_stmt := UserItemCategory.DELETE().
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
			AND(UserItemCategory.ID.EQ(String(itemCategory.ID))))

	_, err := cat_stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting item category")
	}

	items_stmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.CategoryID.EQ(String(itemCategory.ID)))

	_, err = items_stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting categorized items")
	}

	slog.Info("item category deleted", "user_id", user.UserID, "category_uuid", itemCategory.ID)
	return nil
}

// DeleteAllUserItemCategories deletes all item categories and their associated categorized items for a user
func (repo *UserSavesRepositoryImpl) DeleteAllUserItemCategories(user *bootstrap.User) error {
	// First delete all categorized items for this user
	items_stmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

	_, err := items_stmt.Exec(repo.Db)
	if err != nil {
		return errors.New("error deleting all categorized items: " + err.Error())
	}

	// Then delete all item categories for this user
	cat_stmt := UserItemCategory.DELETE().
		WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)))

	_, err = cat_stmt.Exec(repo.Db)
	if err != nil {
		return errors.New("error deleting all item categories: " + err.Error())
	}

	slog.Info("all item categories and categorized items deleted", "user_id", user.UserID)
	return nil
}

func (repo *UserSavesRepositoryImpl) GetCategorizedItemsByCategory(user *bootstrap.User, category model.UserItemCategory) ([]model.UserItemsCategorized, error) {
	var items []model.UserItemsCategorized

	if user != nil {
		stmt :=
			SELECT(
				UserItemsCategorized.AllColumns,
			).WHERE(
				UserItemsCategorized.UserID.EQ(String(user.UserID)).
					AND(UserItemsCategorized.CategoryID.EQ(String(category.ID))))

		err := stmt.Query(repo.Db, &items)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving categorized items for user %s", user.UserID))
		} else {
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}

}

func (repo *UserSavesRepositoryImpl) GetCategorizedItemsByUser(user *bootstrap.User) ([]model.UserItemsCategorized, error) {
	var items []model.UserItemsCategorized

	if user != nil {
		stmt := SELECT(UserItemsCategorized.AllColumns).
			FROM(UserItemsCategorized).
			WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

		err := stmt.Query(repo.Db, &items)

		if err != nil {
			return nil, errors.New(fmt.Sprintf("error retrieving categorized items for user %s", user.UserID))
		} else {
			slog.Info("categorized items retrieved for user", "user_id", user.UserID)
			return items, nil
		}
	} else {
		return nil, errors.New("valid user not found")
	}
}

func (repo *UserSavesRepositoryImpl) UpsertUserItemsCategorized(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	stmt := UserItemsCategorized.
		INSERT(
			UserItemsCategorized.UserID,
			UserItemsCategorized.Niin,
			UserItemsCategorized.ItemName,
			UserItemsCategorized.Quantity,
			UserItemsCategorized.EquipModel,
			UserItemsCategorized.Uoc,
			UserItemsCategorized.CategoryID,
			UserItemsCategorized.SaveTime,
			UserItemsCategorized.Image,
			UserItemsCategorized.LastUpdated,
			UserItemsCategorized.ID,
		).
		MODEL(categorizedItem).
		ON_CONFLICT(
			UserItemsCategorized.Niin,
			UserItemsCategorized.CategoryID,
			UserItemsCategorized.ID,
		).
		DO_UPDATE(
			SET(
				UserItemsCategorized.ItemName.SET(String(*categorizedItem.ItemName)),
				UserItemsCategorized.Quantity.SET(Int32(*categorizedItem.Quantity)),
				UserItemsCategorized.EquipModel.SET(String(*categorizedItem.EquipModel)),
				UserItemsCategorized.Uoc.SET(String(*categorizedItem.Uoc)),
				UserItemsCategorized.SaveTime.SET(TimestampT(*categorizedItem.SaveTime)),
				UserItemsCategorized.Image.SET(String(*categorizedItem.Image)),
				UserItemsCategorized.LastUpdated.SET(TimestampT(*categorizedItem.LastUpdated)),
			).
				WHERE(
					UserItemsCategorized.UserID.EQ(String(user.UserID)).
						AND(UserItemsCategorized.Niin.EQ(String(categorizedItem.Niin))).
						AND(UserItemsCategorized.CategoryID.EQ(String(categorizedItem.CategoryID))).
						AND(UserItemsCategorized.ID.EQ(String(categorizedItem.ID))),
				),
		).
		RETURNING(UserItemsCategorized.AllColumns)

	err := stmt.Query(repo.Db, &categorizedItem)

	if err != nil {
		return errors.New("error saving categorized item: " + err.Error())
	}

	slog.Info("categorized item saved", "user_id", user.UserID, "niin", categorizedItem.Niin, "category_id", categorizedItem.CategoryID)
	return nil
}

func (repo *UserSavesRepositoryImpl) UpsertUserItemsCategorizedList(user *bootstrap.User, categorizedItems []model.UserItemsCategorized) error {
	var failedNiins []string
	for _, val := range categorizedItems {
		stmt := UserItemsCategorized.
			INSERT(UserItemsCategorized.UserID, UserItemsCategorized.Niin, UserItemsCategorized.ItemName,
				UserItemsCategorized.Quantity, UserItemsCategorized.EquipModel, UserItemsCategorized.Uoc,
				UserItemsCategorized.CategoryID, UserItemsCategorized.SaveTime, UserItemsCategorized.Image,
				UserItemsCategorized.LastUpdated, UserItemsCategorized.ID).
			MODEL(val).
			ON_CONFLICT(UserItemsCategorized.UserID, UserItemsCategorized.Niin, UserItemsCategorized.CategoryID, UserItemsCategorized.ID).
			DO_UPDATE(
				SET(
					UserItemsCategorized.ItemName.SET(String(*val.ItemName)),
					UserItemsCategorized.Quantity.SET(Int32(*val.Quantity)),
					UserItemsCategorized.EquipModel.SET(String(*val.EquipModel)),
					UserItemsCategorized.Uoc.SET(String(*val.Uoc)),
					UserItemsCategorized.SaveTime.SET(TimestampT(*val.SaveTime)),
					UserItemsCategorized.Image.SET(String(*val.Image)),
					UserItemsCategorized.LastUpdated.SET(TimestampT(*val.LastUpdated)),
				).
					WHERE(
						UserItemsCategorized.UserID.EQ(String(user.UserID)).
							AND(UserItemsCategorized.Niin.EQ(String(val.Niin))).
							AND(UserItemsCategorized.CategoryID.EQ(String(val.CategoryID))).
							AND(UserItemsCategorized.ID.EQ(String(val.ID))),
					),
			)

		err := stmt.Query(repo.Db, &categorizedItems)

		if err != nil {
			failedNiins = append(failedNiins, val.Niin)
		}
	}

	if len(failedNiins) > 0 {
		return errors.New(fmt.Sprintf("failed to save following items: %s", failedNiins))
	} else {
		slog.Info("categorized item list inserted", "user_id", user.UserID)
	}
	return nil
}

func (repo *UserSavesRepositoryImpl) DeleteUserItemsCategorized(user *bootstrap.User, categorizedItem model.UserItemsCategorized) error {
	stmt := UserItemsCategorized.
		DELETE().
		WHERE(
			UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.Niin.EQ(String(categorizedItem.Niin))).
				AND(UserItemsCategorized.CategoryID.EQ(String(categorizedItem.CategoryID))).
				AND(UserItemsCategorized.ID.EQ(String(categorizedItem.ID))),
		)

	_, err := stmt.Exec(repo.Db)

	if err != nil {
		return errors.New("error deleting categorized item: " + err.Error())
	}

	slog.Info("categorized item deleted", "user_id", user.UserID, "niin", categorizedItem.Niin, "category_id", categorizedItem.CategoryID)
	return nil
}

// DeleteAllUserItemsCategorized deletes all categorized items for a user
func (repo *UserSavesRepositoryImpl) DeleteAllUserItemsCategorized(user *bootstrap.User) error {
	stmt := UserItemsCategorized.DELETE().
		WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return errors.New("error deleting all categorized items: " + err.Error())
	}

	slog.Info("all categorized items deleted", "user_id", user.UserID)
	return nil
}

// Image management methods

// UploadItemImage uploads an item image to Azure Blob Storage and updates the database
// \param user - the user who owns the item
// \param itemID - unique identifier for the item
// \param tableType - the type of table to update (quick, serialized, categorized, category)
// \param imageData - the image data as bytes
// \return the blob URL and an error if the operation fails
func (repo *UserSavesRepositoryImpl) UploadItemImage(user *bootstrap.User, itemID string, tableType string, imageData []byte) (string, error) {
	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	// Upload the blob
	_, err := repo.BlobClient.UploadBuffer(context.TODO(), containerName, blobName, imageData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Construct the blob URL
	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		repo.Env.BlobAccountName, containerName, blobName)

	// Update the image column in the specified table
	updated, err := repo.updateImageInTable(user, itemID, blobURL, tableType)
	if err != nil {
		// If database update fails, we should consider deleting the uploaded blob
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

// DeleteItemImage deletes an item image from Azure Blob Storage and clears the database
// \param user - the user who owns the item
// \param itemID - unique identifier for the item
// \param tableType - the type of table to update (quick, serialized, categorized, category)
// \return an error if the operation fails
func (repo *UserSavesRepositoryImpl) DeleteItemImage(user *bootstrap.User, itemID string, tableType string) error {
	ctx := context.Background()
	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	// Update the database first to clear the image URL
	updated, err := repo.updateImageInTable(user, itemID, "", tableType)
	if err != nil {
		slog.Error("Failed to clear image URL in database", "error", err, "item_id", itemID, "table_type", tableType)
		return fmt.Errorf("failed to clear image URL in database for table %s: %w", tableType, err)
	}

	if !updated {
		slog.Error("No rows updated - item not found for deletion", "user_id", user.UserID, "item_id", itemID, "table_type", tableType)
		return fmt.Errorf("item with ID %s not found in %s table for user %s", itemID, tableType, user.UserID)
	}

	// Delete the blob
	_, err = repo.BlobClient.DeleteBlob(ctx, containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	slog.Info("image deleted successfully", "user_id", user.UserID, "item_id", itemID, "table_type", tableType)
	return nil
}

// GetItemImage retrieves an item image from Azure Blob Storage
// \param user - the user who owns the item
// \param itemID - unique identifier for the item
// \param tableType - the type of table to query (quick, serialized, categorized, category)
// \return image data, content type, and an error if the operation fails
func (repo *UserSavesRepositoryImpl) GetItemImage(user *bootstrap.User, itemID string, tableType string) ([]byte, string, error) {
	// First, get the image URL from the database
	imageURL, err := repo.getImageURLFromTable(user, itemID, tableType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get image URL from database: %w", err)
	}

	if imageURL == "" {
		return nil, "", fmt.Errorf("no image found for item %s in %s table", itemID, tableType)
	}

	// Extract blob info from URL and download from Azure
	containerName := "user-item-images"
	blobName := fmt.Sprintf("%s/%s.jpg", user.UserID, itemID)

	// Download the blob
	ctx := context.Background()
	response, err := repo.BlobClient.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image from blob storage: %w", err)
	}
	defer response.Body.Close()

	// Read the blob data
	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Determine content type (default to image/jpeg for .jpg files)
	contentType := "image/jpeg"
	if response.ContentType != nil {
		contentType = *response.ContentType
	}

	slog.Info("image retrieved successfully", "user_id", user.UserID, "item_id", itemID, "table_type", tableType, "content_type", contentType)
	return imageData, contentType, nil
}

// getImageURLFromTable retrieves the image URL from the specified table type
func (repo *UserSavesRepositoryImpl) getImageURLFromTable(user *bootstrap.User, itemID string, tableType string) (string, error) {
	// Validate table type
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
		err = stmt.Query(repo.Db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "serialized":
		var item model.UserItemsSerialized
		stmt := SELECT(UserItemsSerialized.Image).
			FROM(UserItemsSerialized).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
				AND(UserItemsSerialized.ID.EQ(String(itemID))))
		err = stmt.Query(repo.Db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "categorized":
		var item model.UserItemsCategorized
		stmt := SELECT(UserItemsCategorized.Image).
			FROM(UserItemsCategorized).
			WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.ID.EQ(String(itemID))))
		err = stmt.Query(repo.Db, &item)
		if err == nil {
			imageURL = item.Image
		}
	case "category":
		var item model.UserItemCategory
		stmt := SELECT(UserItemCategory.Image).
			FROM(UserItemCategory).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
				AND(UserItemCategory.ID.EQ(String(itemID))))
		err = stmt.Query(repo.Db, &item)
		if err == nil {
			imageURL = item.Image
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to query %s table: %w", tableType, err)
	}

	if imageURL == nil {
		return "", nil // No image URL found, but no error
	}

	return *imageURL, nil
}

// updateImageInTable updates the image column in a specific table type
func (repo *UserSavesRepositoryImpl) updateImageInTable(user *bootstrap.User, itemID string, imageURL string, tableType string) (bool, error) {
	// Validate table type
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
		result, err = stmt.Exec(repo.Db)
	case "serialized":
		stmt := UserItemsSerialized.UPDATE(UserItemsSerialized.Image).
			SET(String(imageURL)).
			WHERE(UserItemsSerialized.UserID.EQ(String(user.UserID)).
				AND(UserItemsSerialized.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.Db)
	case "categorized":
		stmt := UserItemsCategorized.UPDATE(UserItemsCategorized.Image).
			SET(String(imageURL)).
			WHERE(UserItemsCategorized.UserID.EQ(String(user.UserID)).
				AND(UserItemsCategorized.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.Db)
	case "category":
		stmt := UserItemCategory.UPDATE(UserItemCategory.Image).
			SET(String(imageURL)).
			WHERE(UserItemCategory.UserUID.EQ(String(user.UserID)).
				AND(UserItemCategory.ID.EQ(String(itemID))))
		result, err = stmt.Exec(repo.Db)
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
