package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/bootstrap"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
)

// Constants for shop message image uploads
const (
	ShopMessageImagesContainer = "shop-message-images"
	MaxImageSize               = 5 * 1024 * 1024 // 5MB in bytes
)

type ShopsRepositoryImpl struct {
	Db         *sql.DB
	BlobClient *azblob.Client
	Env        *bootstrap.Env
}

func NewShopsRepositoryImpl(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *ShopsRepositoryImpl {
	return &ShopsRepositoryImpl{
		Db:         db,
		BlobClient: blobClient,
		Env:        env,
	}
}

// Shop Operations
func (repo *ShopsRepositoryImpl) CreateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	stmt := Shops.INSERT(
		Shops.ID,
		Shops.Name,
		Shops.Details,
		Shops.CreatedBy,
		Shops.CreatedAt,
		Shops.UpdatedAt,
	).MODEL(shop).RETURNING(Shops.AllColumns)

	var createdShop model.Shops
	err := stmt.Query(repo.Db, &createdShop)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop: %w", err)
	}

	slog.Info("Shop created in database", "shop_id", shop.ID, "created_by", user.UserID)
	return &createdShop, nil
}

func (repo *ShopsRepositoryImpl) UpdateShop(user *bootstrap.User, shop model.Shops) (*model.Shops, error) {
	// Set updated timestamp
	now := time.Now()
	shop.UpdatedAt = &now

	// Build the update statement based on whether details is provided
	updateStmt := Shops.UPDATE(
		Shops.Name,
		Shops.UpdatedAt,
	).SET(
		Shops.Name.SET(String(shop.Name)),
		Shops.UpdatedAt.SET(TimestampzT(*shop.UpdatedAt)),
		Shops.AdminOnlyLists.SET(Bool(shop.AdminOnlyLists)),
	)

	// If details is provided, add it to the update
	if shop.Details != nil {
		updateStmt = Shops.UPDATE(
			Shops.Name,
			Shops.Details,
			Shops.UpdatedAt,
		).SET(
			Shops.Name.SET(String(shop.Name)),
			Shops.Details.SET(String(*shop.Details)),
			Shops.UpdatedAt.SET(TimestampzT(*shop.UpdatedAt)),
			Shops.AdminOnlyLists.SET(Bool(shop.AdminOnlyLists)),
		)
	}

	stmt := updateStmt.WHERE(
		Shops.ID.EQ(String(shop.ID)).
			AND(Shops.CreatedBy.EQ(String(user.UserID))),
	).RETURNING(Shops.AllColumns)

	var updatedShop model.Shops
	err := stmt.Query(repo.Db, &updatedShop)
	if err != nil {
		return nil, fmt.Errorf("failed to update shop: %w", err)
	}

	slog.Info("Shop updated in database", "shop_id", shop.ID, "updated_by", user.UserID)
	return &updatedShop, nil
}

func (repo *ShopsRepositoryImpl) DeleteShop(user *bootstrap.User, shopID string) error {
	// Delete shop will cascade delete related records (members, messages, vehicles, etc.)
	stmt := Shops.DELETE().WHERE(
		Shops.ID.EQ(String(shopID)).
			AND(Shops.CreatedBy.EQ(String(user.UserID))),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop not found or user not authorized to delete")
	}

	slog.Info("Shop deleted from database", "shop_id", shopID, "deleted_by", user.UserID)
	return nil
}

func (repo *ShopsRepositoryImpl) GetShopsByUser(user *bootstrap.User) ([]model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(
			Shops.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(Shops.ID)),
		).
		WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
		ORDER_BY(Shops.CreatedAt.DESC())

	var shops []model.Shops
	err := stmt.Query(repo.Db, &shops)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops for user: %w", err)
	}

	slog.Info("Shops retrieved for user", "user_id", user.UserID, "count", len(shops))
	return shops, nil
}

func (repo *ShopsRepositoryImpl) GetShopByID(user *bootstrap.User, shopID string) (*model.Shops, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var shop model.Shops
	err := stmt.Query(repo.Db, &shop)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop: %w", err)
	}

	return &shop, nil
}

func (repo *ShopsRepositoryImpl) GetShopsWithStatsForUser(user *bootstrap.User) ([]response.ShopWithStats, error) {
	rawSQL := `
		SELECT
			s.id,
			s.name,
			s.details,
			s.created_by,
			s.created_at,
			s.updated_at,
			s.admin_only_lists,
			COALESCE(member_stats.member_count, 0) as member_count,
			COALESCE(vehicle_stats.vehicle_count, 0) as vehicle_count,
			CASE WHEN admin_check.user_id IS NOT NULL THEN true ELSE false END as is_admin
		FROM shops s
		INNER JOIN shop_members sm ON s.id = sm.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as member_count
			FROM shop_members
			GROUP BY shop_id
		) member_stats ON s.id = member_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, COUNT(*) as vehicle_count
			FROM shop_vehicle
			GROUP BY shop_id
		) vehicle_stats ON s.id = vehicle_stats.shop_id
		LEFT JOIN (
			SELECT shop_id, user_id
			FROM shop_members
			WHERE role = 'admin'
		) admin_check ON s.id = admin_check.shop_id AND admin_check.user_id = $1
		WHERE sm.user_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := repo.Db.Query(rawSQL, user.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shops with stats: %w", err)
	}
	defer rows.Close()

	var results []response.ShopWithStats
	for rows.Next() {
		var shop model.Shops
		var memberCount, vehicleCount int64
		var isAdmin bool

		err := rows.Scan(
			&shop.ID,
			&shop.Name,
			&shop.Details,
			&shop.CreatedBy,
			&shop.CreatedAt,
			&shop.UpdatedAt,
			&shop.AdminOnlyLists,
			&memberCount,
			&vehicleCount,
			&isAdmin,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shop row: %w", err)
		}

		results = append(results, response.ShopWithStats{
			Shop:             shop,
			MemberCount:      memberCount,
			VehicleCount:     vehicleCount,
			IsAdmin:          isAdmin,
			IsListsAdminOnly: shop.AdminOnlyLists,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	slog.Info("Shops with stats retrieved for user", "user_id", user.UserID, "count", len(results))
	return results, nil
}

func (repo *ShopsRepositoryImpl) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))).
				AND(ShopMembers.Role.EQ(String("admin"))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}

	return result.Count > 0, nil
}

// Shop Member Operations
func (repo *ShopsRepositoryImpl) AddMemberToShop(user *bootstrap.User, shopID string, role string) error {
	curTime := time.Now().UTC()
	member := model.ShopMembers{
		ID:       fmt.Sprintf("%s_%s", shopID, user.UserID), // Simple composite ID
		ShopID:   shopID,
		UserID:   user.UserID,
		Role:     role,
		JoinedAt: &curTime,
	}

	stmt := ShopMembers.INSERT(
		ShopMembers.ID,
		ShopMembers.ShopID,
		ShopMembers.UserID,
		ShopMembers.Role,
		ShopMembers.JoinedAt,
	).MODEL(member).
		ON_CONFLICT(ShopMembers.ShopID, ShopMembers.UserID).
		DO_UPDATE(SET(ShopMembers.Role.SET(String(role))))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to add member to shop: %w", err)
	}

	slog.Info("Member added to shop", "shop_id", shopID, "user_id", user.UserID, "role", role)
	return nil
}

func (repo *ShopsRepositoryImpl) RemoveMemberFromShop(user *bootstrap.User, shopID string, targetUserID string) error {
	stmt := ShopMembers.DELETE().
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(targetUserID))),
		)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to remove member from shop: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("member not found in shop")
	}

	slog.Info("Member removed from shop", "shop_id", shopID, "removed_user_id", targetUserID, "removed_by", user.UserID)
	return nil
}

func (repo *ShopsRepositoryImpl) UpdateMemberRole(user *bootstrap.User, shopID string, targetUserID string, newRole string) error {
	stmt := ShopMembers.UPDATE(
		ShopMembers.Role,
	).SET(
		newRole,
	).WHERE(
		ShopMembers.ShopID.EQ(String(shopID)).
			AND(ShopMembers.UserID.EQ(String(targetUserID))),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("member not found in shop")
	}

	slog.Info("Member role updated", "shop_id", shopID, "target_user_id", targetUserID, "new_role", newRole, "updated_by", user.UserID)
	return nil
}

func (repo *ShopsRepositoryImpl) GetShopMembers(user *bootstrap.User, shopID string) ([]response.ShopMemberWithUsername, error) {
	rawSQL := `
		SELECT 
			sm.id,
			sm.shop_id,
			sm.user_id,
			sm.role,
			sm.joined_at,
			u.username
		FROM shop_members sm
		LEFT JOIN users u ON sm.user_id = u.uid
		WHERE sm.shop_id = $1
		ORDER BY sm.joined_at ASC
	`

	rows, err := repo.Db.Query(rawSQL, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop members: %w", err)
	}
	defer rows.Close()

	var members []response.ShopMemberWithUsername
	for rows.Next() {
		var member response.ShopMemberWithUsername
		err := rows.Scan(&member.ID, &member.ShopID, &member.UserID, &member.Role, &member.JoinedAt, &member.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return members, nil
}

func (repo *ShopsRepositoryImpl) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return result.Count > 0, nil
}

func (repo *ShopsRepositoryImpl) GetShopMemberCount(user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopMembers.ID).AS("count")).
		FROM(ShopMembers).
		WHERE(ShopMembers.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}

	return result.Count, nil
}

func (repo *ShopsRepositoryImpl) GetShopVehicleCount(user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopVehicle.ID).AS("count")).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get vehicle count: %w", err)
	}

	return result.Count, nil
}

// Shop Invite Code Operations
func (repo *ShopsRepositoryImpl) CreateInviteCode(user *bootstrap.User, inviteCode model.ShopInviteCodes) (*model.ShopInviteCodes, error) {
	stmt := ShopInviteCodes.INSERT(
		ShopInviteCodes.ID,
		ShopInviteCodes.ShopID,
		ShopInviteCodes.Code,
		ShopInviteCodes.CreatedBy,
		ShopInviteCodes.IsActive,
		ShopInviteCodes.CreatedAt,
	).MODEL(inviteCode).RETURNING(ShopInviteCodes.AllColumns)

	var createdCode model.ShopInviteCodes
	err := stmt.Query(repo.Db, &createdCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create invite code: %w", err)
	}

	return &createdCode, nil
}

func (repo *ShopsRepositoryImpl) GetInviteCodeByCode(code string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.Code.EQ(String(code)))

	var inviteCode model.ShopInviteCodes
	err := stmt.Query(repo.Db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *ShopsRepositoryImpl) GetInviteCodeByID(codeID string) (*model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	var inviteCode model.ShopInviteCodes
	err := stmt.Query(repo.Db, &inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite code not found: %w", err)
	}

	return &inviteCode, nil
}

func (repo *ShopsRepositoryImpl) GetInviteCodesByShop(user *bootstrap.User, shopID string) ([]model.ShopInviteCodes, error) {
	stmt := SELECT(ShopInviteCodes.AllColumns).
		FROM(ShopInviteCodes).
		WHERE(ShopInviteCodes.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopInviteCodes.CreatedAt.DESC())

	var codes []model.ShopInviteCodes
	err := stmt.Query(repo.Db, &codes)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite codes: %w", err)
	}

	return codes, nil
}

func (repo *ShopsRepositoryImpl) DeactivateInviteCode(user *bootstrap.User, codeID string) error {
	stmt := ShopInviteCodes.UPDATE(
		ShopInviteCodes.IsActive,
	).SET(
		ShopInviteCodes.IsActive.SET(Bool(false)),
	).WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite code not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteInviteCode(user *bootstrap.User, codeID string) error {
	stmt := ShopInviteCodes.DELETE().WHERE(ShopInviteCodes.ID.EQ(String(codeID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete invite code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite code not found")
	}

	slog.Info("Invite code deleted from database", "code_id", codeID, "deleted_by", user.UserID)
	return nil
}

// Shop Message Operations
func (repo *ShopsRepositoryImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
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
	err := stmt.Query(repo.Db, &createdMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop message: %w", err)
	}

	return &createdMessage, nil
}

func (repo *ShopsRepositoryImpl) GetShopMessages(user *bootstrap.User, shopID string) ([]model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.ASC())

	var messages []model.ShopMessages
	err := stmt.Query(repo.Db, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	return messages, nil
}

func (repo *ShopsRepositoryImpl) GetShopMessagesPaginated(user *bootstrap.User, shopID string, offset int, limit int) ([]model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	var messages []model.ShopMessages
	err := stmt.Query(repo.Db, &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated shop messages: %w", err)
	}

	return messages, nil
}

func (repo *ShopsRepositoryImpl) GetShopMessagesCount(user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopMessages.ID)).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.Query(repo.Db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get shop messages count: %w", err)
	}

	return result.Count, nil
}

func (repo *ShopsRepositoryImpl) UpdateShopMessage(user *bootstrap.User, message model.ShopMessages) error {
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

	result, err := stmt.Exec(repo.Db)
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

func (repo *ShopsRepositoryImpl) DeleteShopMessage(user *bootstrap.User, messageID string) error {
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

	result, err := stmt.Exec(repo.Db)
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
		return ".jpg" // Default to jpg if unknown
	}
}

// extractImageURLFromMessage extracts the image URL from a message text containing [IMAGE:url] tag
// Returns empty string if no image tag found
func extractImageURLFromMessage(messageText string) string {
	// Pattern matches: [IMAGE:https://...]
	re := regexp.MustCompile(`\[IMAGE:(https://[^\]]+)\]`)
	matches := re.FindStringSubmatch(messageText)
	if len(matches) > 1 {
		return matches[1] // Return the captured URL
	}
	return ""
}

// parseBlobNameFromURL extracts the blob name from an Azure Blob Storage URL
// URL format: https://{account}.blob.core.windows.net/{container}/{blob_path}
// Returns the blob_path portion (e.g., "shopID/messageID.jpg")
func parseBlobNameFromURL(url string, expectedContainer string) (string, error) {
	if url == "" {
		return "", errors.New("empty URL")
	}

	// Find the container name in the URL
	containerPrefix := fmt.Sprintf("/%s/", expectedContainer)
	idx := strings.Index(url, containerPrefix)
	if idx == -1 {
		return "", fmt.Errorf("container '%s' not found in URL", expectedContainer)
	}

	// Extract everything after the container name
	blobName := url[idx+len(containerPrefix):]
	if blobName == "" {
		return "", errors.New("blob name is empty")
	}

	return blobName, nil
}

// UploadMessageImage uploads an image for a shop message to Azure Blob Storage
// Returns: file extension, blob URL, error
func (repo *ShopsRepositoryImpl) UploadMessageImage(user *bootstrap.User, messageID string, shopID string, imageData []byte, contentType string) (string, string, error) {
	ctx := context.Background()

	// Verify user is a member of the shop
	isMember, err := repo.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return "", "", errors.New("access denied: user is not a member of this shop")
	}

	// Detect content type if not provided
	if contentType == "" {
		contentType = http.DetectContentType(imageData)
	}

	// Get file extension based on content type
	fileExtension := getFileExtensionFromMIME(contentType)

	// Construct blob name: shopID/messageID.ext
	blobName := fmt.Sprintf("%s/%s%s", shopID, messageID, fileExtension)

	// Upload the blob
	_, err = repo.BlobClient.UploadBuffer(ctx, ShopMessageImagesContainer, blobName, imageData, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Construct the blob URL
	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		repo.Env.BlobAccountName, ShopMessageImagesContainer, blobName)

	slog.Info("shop message image uploaded successfully", "user_id", user.UserID, "message_id", messageID, "shop_id", shopID, "blob_url", blobURL)
	return fileExtension, blobURL, nil
}

// DeleteMessageImageBlob deletes a shop message image from Azure Blob Storage
// This is a cleanup endpoint for when upload succeeds but message creation fails
func (repo *ShopsRepositoryImpl) DeleteMessageImageBlob(user *bootstrap.User, messageID string, shopID string) error {
	ctx := context.Background()

	// Verify user is a member of the shop
	isMember, err := repo.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return errors.New("access denied: user is not a member of this shop")
	}

	// Try to delete all possible file extensions (we don't know which one was used)
	extensions := []string{".jpg", ".png", ".gif", ".webp"}
	deleted := false

	for _, ext := range extensions {
		blobName := fmt.Sprintf("%s/%s%s", shopID, messageID, ext)
		_, err := repo.BlobClient.DeleteBlob(ctx, ShopMessageImagesContainer, blobName, nil)
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

// GetShopMessageByID retrieves a single shop message by its ID
func (repo *ShopsRepositoryImpl) GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error) {
	stmt := SELECT(ShopMessages.AllColumns).
		FROM(ShopMessages).
		WHERE(ShopMessages.ID.EQ(String(messageID)))

	var message model.ShopMessages
	err := stmt.Query(repo.Db, &message)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

// DeleteBlobByURL deletes a blob from Azure Blob Storage given a message text that may contain [IMAGE:url]
// Extracts the URL from the [IMAGE:url] tag format and deletes the blob
func (repo *ShopsRepositoryImpl) DeleteBlobByURL(messageText string) error {
	if messageText == "" {
		return nil // Nothing to delete
	}

	// Extract image URL from message text (format: [IMAGE:https://...])
	imageURL := extractImageURLFromMessage(messageText)
	if imageURL == "" {
		// No image URL found in message - this is normal for text-only messages
		return nil
	}

	ctx := context.Background()

	// Parse the blob name from the URL
	// Expected format: https://{account}.blob.core.windows.net/shop-message-images/{shopID}/{messageID}.{ext}
	blobName, err := parseBlobNameFromURL(imageURL, ShopMessageImagesContainer)
	if err != nil {
		slog.Warn("Failed to parse blob name from URL", "url", imageURL, "error", err)
		return nil // Don't fail deletion if we can't parse the URL
	}

	// Delete the blob
	_, err = repo.BlobClient.DeleteBlob(ctx, ShopMessageImagesContainer, blobName, nil)
	if err != nil {
		// Log warning but don't fail - blob may already be deleted
		slog.Warn("Failed to delete blob from Azure", "blob_name", blobName, "error", err)
		return nil // Graceful failure
	}

	slog.Info("Blob deleted successfully from Azure", "blob_name", blobName, "image_url", imageURL)
	return nil
}

// DeleteShopMessageBlobs deletes all message image blobs for a shop
// Called when shop is deleted to clean up orphaned blobs in Azure Blob Storage
// Uses graceful failure - logs errors but doesn't fail the operation
func (repo *ShopsRepositoryImpl) DeleteShopMessageBlobs(shopID string) error {
	ctx := context.Background()

	// Get container client
	containerClient := repo.BlobClient.ServiceClient().NewContainerClient(ShopMessageImagesContainer)

	// List all blobs with shopID prefix (e.g., "shop-uuid-123/")
	prefix := fmt.Sprintf("%s/", shopID)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	deletedCount := 0
	errorCount := 0

	// Iterate through all pages of results
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Warn("Failed to list blobs for shop deletion", "shop_id", shopID, "error", err)
			continue // Try to delete what we can
		}

		// Delete each blob found
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}

			_, err := repo.BlobClient.DeleteBlob(ctx, ShopMessageImagesContainer, *blob.Name, nil)
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

	return nil // Graceful failure - don't fail shop deletion if blob cleanup has issues
}

// Shop Vehicle Operations
func (repo *ShopsRepositoryImpl) CreateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	stmt := ShopVehicle.INSERT(
		ShopVehicle.ID,
		ShopVehicle.CreatorID,
		ShopVehicle.Niin,
		ShopVehicle.Admin,
		ShopVehicle.Model,
		ShopVehicle.Serial,
		ShopVehicle.Uoc,
		ShopVehicle.Mileage,
		ShopVehicle.Hours,
		ShopVehicle.Comment,
		ShopVehicle.SaveTime,
		ShopVehicle.LastUpdated,
		ShopVehicle.ShopID,
	).MODEL(vehicle).RETURNING(ShopVehicle.AllColumns)

	var createdVehicle model.ShopVehicle
	err := stmt.Query(repo.Db, &createdVehicle)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop vehicle: %w", err)
	}

	return &createdVehicle, nil
}

func (repo *ShopsRepositoryImpl) GetShopVehicles(user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicle.SaveTime.DESC())

	var vehicles []model.ShopVehicle
	err := stmt.Query(repo.Db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *ShopsRepositoryImpl) GetShopVehicleByID(user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.Query(repo.Db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("shop vehicle not found: %w", err)
	}

	return &vehicle, nil
}

func (repo *ShopsRepositoryImpl) UpdateShopVehicle(user *bootstrap.User, vehicle model.ShopVehicle) error {
	// Build base SET clauses for non-nullable fields
	setClauses := []postgres.ColumnAssigment{
		ShopVehicle.Model.SET(String(vehicle.Model)),
		ShopVehicle.Serial.SET(String(vehicle.Serial)),
		ShopVehicle.Niin.SET(String(vehicle.Niin)),
		ShopVehicle.Uoc.SET(String(vehicle.Uoc)),
		ShopVehicle.Mileage.SET(Int32(vehicle.Mileage)),
		ShopVehicle.Hours.SET(Int32(vehicle.Hours)),
		ShopVehicle.Comment.SET(String(vehicle.Comment)),
		ShopVehicle.LastUpdated.SET(TimestampzT(vehicle.LastUpdated)),
	}

	// Add nullable fields conditionally
	if vehicle.TrackedMileage != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(*vehicle.TrackedMileage)))
	}

	if vehicle.TrackedHours != nil {
		setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(*vehicle.TrackedHours)))
	}

	// Convert to interface{} slice for variadic function
	setArgs := make([]interface{}, len(setClauses))
	for i, clause := range setClauses {
		setArgs[i] = clause
	}

	stmt := ShopVehicle.UPDATE().SET(setArgs[0], setArgs[1:]...).WHERE(ShopVehicle.ID.EQ(String(vehicle.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteShopVehicle(user *bootstrap.User, vehicleID string) error {
	stmt := ShopVehicle.DELETE().
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("vehicle not found")
	}

	return nil
}

// Shop Vehicle Notification Operations
func (repo *ShopsRepositoryImpl) CreateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) (*model.ShopVehicleNotifications, error) {
	stmt := ShopVehicleNotifications.INSERT(
		ShopVehicleNotifications.ID,
		ShopVehicleNotifications.ShopID,
		ShopVehicleNotifications.VehicleID,
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.SaveTime,
		ShopVehicleNotifications.LastUpdated,
	).MODEL(notification).RETURNING(ShopVehicleNotifications.AllColumns)

	var createdNotification model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &createdNotification)
	if err != nil {
		return nil, fmt.Errorf("failed to create vehicle notification: %w", err)
	}

	return &createdNotification, nil
}

func (repo *ShopsRepositoryImpl) GetVehicleNotifications(user *bootstrap.User, vehicleID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.VehicleID.EQ(String(vehicleID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	return notifications, nil
}

func (repo *ShopsRepositoryImpl) GetVehicleNotificationsWithItems(user *bootstrap.User, vehicleID string) ([]response.VehicleNotificationWithItems, error) {
	// First get all notifications for the vehicle
	notifications, err := repo.GetVehicleNotifications(user, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notifications: %w", err)
	}

	var result []response.VehicleNotificationWithItems

	// For each notification, get its items
	for _, notification := range notifications {
		items, err := repo.GetNotificationItems(user, notification.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get items for notification %s: %w", notification.ID, err)
		}

		// If no items found, provide empty slice instead of nil
		if items == nil {
			items = []model.ShopNotificationItems{}
		}

		result = append(result, response.VehicleNotificationWithItems{
			Notification: notification,
			Items:        items,
		})
	}

	return result, nil
}

func (repo *ShopsRepositoryImpl) GetShopNotifications(user *bootstrap.User, shopID string) ([]model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicleNotifications.SaveTime.DESC())

	var notifications []model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &notifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notifications: %w", err)
	}

	return notifications, nil
}

func (repo *ShopsRepositoryImpl) GetVehicleNotificationByID(user *bootstrap.User, notificationID string) (*model.ShopVehicleNotifications, error) {
	stmt := SELECT(ShopVehicleNotifications.AllColumns).
		FROM(ShopVehicleNotifications).
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	var notification model.ShopVehicleNotifications
	err := stmt.Query(repo.Db, &notification)
	if err != nil {
		return nil, fmt.Errorf("vehicle notification not found: %w", err)
	}

	return &notification, nil
}

func (repo *ShopsRepositoryImpl) UpdateVehicleNotification(user *bootstrap.User, notification model.ShopVehicleNotifications) error {
	stmt := ShopVehicleNotifications.UPDATE(
		ShopVehicleNotifications.Title,
		ShopVehicleNotifications.Description,
		ShopVehicleNotifications.Type,
		ShopVehicleNotifications.Completed,
		ShopVehicleNotifications.LastUpdated,
	).SET(
		ShopVehicleNotifications.Title.SET(String(notification.Title)),
		ShopVehicleNotifications.Description.SET(String(notification.Description)),
		ShopVehicleNotifications.Type.SET(String(notification.Type)),
		ShopVehicleNotifications.Completed.SET(Bool(notification.Completed)),
		ShopVehicleNotifications.LastUpdated.SET(TimestampzT(notification.LastUpdated)),
	).WHERE(ShopVehicleNotifications.ID.EQ(String(notification.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteVehicleNotification(user *bootstrap.User, notificationID string) error {
	stmt := ShopVehicleNotifications.DELETE().
		WHERE(ShopVehicleNotifications.ID.EQ(String(notificationID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification not found")
	}

	return nil
}

// Shop Notification Item Operations
func (repo *ShopsRepositoryImpl) CreateNotificationItem(user *bootstrap.User, item model.ShopNotificationItems) (*model.ShopNotificationItems, error) {
	stmt := ShopNotificationItems.INSERT(
		ShopNotificationItems.ID,
		ShopNotificationItems.ShopID,
		ShopNotificationItems.NotificationID,
		ShopNotificationItems.Niin,
		ShopNotificationItems.Nomenclature,
		ShopNotificationItems.Quantity,
		ShopNotificationItems.SaveTime,
	).MODEL(item).RETURNING(ShopNotificationItems.AllColumns)

	var createdItem model.ShopNotificationItems
	err := stmt.Query(repo.Db, &createdItem)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification item: %w", err)
	}

	return &createdItem, nil
}

func (repo *ShopsRepositoryImpl) GetNotificationItems(user *bootstrap.User, notificationID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.NotificationID.EQ(String(notificationID))).
		ORDER_BY(ShopNotificationItems.SaveTime.ASC())

	var items []model.ShopNotificationItems
	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	return items, nil
}

func (repo *ShopsRepositoryImpl) GetShopNotificationItems(user *bootstrap.User, shopID string) ([]model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopNotificationItems.SaveTime.DESC())

	var items []model.ShopNotificationItems
	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification items: %w", err)
	}

	return items, nil
}

func (repo *ShopsRepositoryImpl) GetNotificationItemByID(user *bootstrap.User, itemID string) (*model.ShopNotificationItems, error) {
	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ID.EQ(String(itemID)))

	var item model.ShopNotificationItems
	err := stmt.Query(repo.Db, &item)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification item: %w", err)
	}

	return &item, nil
}

func (repo *ShopsRepositoryImpl) GetNotificationItemsByIDs(user *bootstrap.User, itemIDs []string) ([]model.ShopNotificationItems, error) {
	if len(itemIDs) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	// Convert string slice to expressions for the IN clause
	var expressions []Expression
	for _, id := range itemIDs {
		expressions = append(expressions, String(id))
	}

	stmt := SELECT(ShopNotificationItems.AllColumns).
		FROM(ShopNotificationItems).
		WHERE(ShopNotificationItems.ID.IN(expressions...))

	var items []model.ShopNotificationItems
	err := stmt.Query(repo.Db, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification items: %w", err)
	}

	// Note: Result may contain fewer items than requested IDs if some don't exist
	// This is intentional for lenient bulk operations
	return items, nil
}

func (repo *ShopsRepositoryImpl) CreateNotificationItemList(user *bootstrap.User, items []model.ShopNotificationItems) ([]model.ShopNotificationItems, error) {
	if len(items) == 0 {
		return []model.ShopNotificationItems{}, nil
	}

	stmt := ShopNotificationItems.INSERT(
		ShopNotificationItems.ID,
		ShopNotificationItems.ShopID,
		ShopNotificationItems.NotificationID,
		ShopNotificationItems.Niin,
		ShopNotificationItems.Nomenclature,
		ShopNotificationItems.Quantity,
		ShopNotificationItems.SaveTime,
	).MODELS(items).RETURNING(ShopNotificationItems.AllColumns)

	var createdItems []model.ShopNotificationItems
	err := stmt.Query(repo.Db, &createdItems)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification items: %w", err)
	}

	return createdItems, nil
}

func (repo *ShopsRepositoryImpl) DeleteNotificationItem(user *bootstrap.User, itemID string) error {
	stmt := ShopNotificationItems.DELETE().
		WHERE(ShopNotificationItems.ID.EQ(String(itemID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete notification item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("notification item not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteNotificationItemList(user *bootstrap.User, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Convert string slice to expressions for the IN clause
	var expressions []Expression
	for _, id := range itemIDs {
		expressions = append(expressions, String(id))
	}

	stmt := ShopNotificationItems.DELETE().
		WHERE(ShopNotificationItems.ID.IN(expressions...))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete notification items: %w", err)
	}

	return nil
}

// Shop List Operations
func (repo *ShopsRepositoryImpl) CreateShopList(user *bootstrap.User, list model.ShopLists) (*response.ShopListWithUsername, error) {
	stmt := ShopLists.INSERT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
	).MODEL(list)

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return nil, fmt.Errorf("failed to create shop list: %w", err)
	}

	// Get the created list with username
	selectStmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ID.EQ(String(list.ID)),
	)

	var result struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err = selectStmt.Query(repo.Db, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get created shop list with username: %w", err)
	}

	// Convert to response type
	createdListWithUsername := &response.ShopListWithUsername{
		ID:                result.ID,
		ShopID:            result.ShopID,
		CreatedBy:         result.CreatedBy,
		CreatedByUsername: result.CreatedByUsername,
		Description:       result.Description,
		CreatedAt:         &result.CreatedAt,
		UpdatedAt:         &result.UpdatedAt,
	}

	return createdListWithUsername, nil
}

func (repo *ShopsRepositoryImpl) GetShopLists(user *bootstrap.User, shopID string) ([]response.ShopListWithUsername, error) {
	stmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ShopID.EQ(String(shopID)),
	).ORDER_BY(ShopLists.CreatedAt.DESC())

	var results []struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err := stmt.Query(repo.Db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop lists with usernames: %w", err)
	}

	// Convert to response type
	lists := make([]response.ShopListWithUsername, len(results))
	for i, r := range results {
		lists[i] = response.ShopListWithUsername{
			ID:                r.ID,
			ShopID:            r.ShopID,
			CreatedBy:         r.CreatedBy,
			CreatedByUsername: r.CreatedByUsername,
			Description:       r.Description,
			CreatedAt:         &r.CreatedAt,
			UpdatedAt:         &r.UpdatedAt,
		}
	}

	return lists, nil
}

func (repo *ShopsRepositoryImpl) GetShopListByID(user *bootstrap.User, listID string) (*response.ShopListWithUsername, error) {
	stmt := SELECT(
		ShopLists.ID,
		ShopLists.ShopID,
		ShopLists.CreatedBy,
		ShopLists.Description,
		ShopLists.CreatedAt,
		ShopLists.UpdatedAt,
		Users.Username.AS("created_by_username"),
	).FROM(
		ShopLists.
			LEFT_JOIN(Users, Users.UID.EQ(ShopLists.CreatedBy)),
	).WHERE(
		ShopLists.ID.EQ(String(listID)),
	)

	var result struct {
		model.ShopLists
		CreatedByUsername *string `sql:"created_by_username"`
	}

	err := stmt.Query(repo.Db, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("shop list not found")
		}
		return nil, fmt.Errorf("failed to get shop list: %w", err)
	}

	// Convert to response type
	listWithUsername := &response.ShopListWithUsername{
		ID:                result.ID,
		ShopID:            result.ShopID,
		CreatedBy:         result.CreatedBy,
		CreatedByUsername: result.CreatedByUsername,
		Description:       result.Description,
		CreatedAt:         &result.CreatedAt,
		UpdatedAt:         &result.UpdatedAt,
	}

	return listWithUsername, nil
}

func (repo *ShopsRepositoryImpl) UpdateShopList(user *bootstrap.User, list model.ShopLists) error {
	stmt := ShopLists.UPDATE(
		ShopLists.Description,
		ShopLists.UpdatedAt,
	).MODEL(list).
		WHERE(ShopLists.ID.EQ(String(list.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update shop list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop list not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) DeleteShopList(user *bootstrap.User, listID string) error {
	stmt := ShopLists.DELETE().
		WHERE(ShopLists.ID.EQ(String(listID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to delete shop list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop list not found")
	}

	return nil
}

// Shop List Item Operations
func (repo *ShopsRepositoryImpl) AddListItem(user *bootstrap.User, item model.ShopListItems) (*response.ShopListItemWithUsername, error) {
	stmt := ShopListItems.INSERT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODEL(item)

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return nil, fmt.Errorf("failed to add list item: %w", err)
	}

	// Get the created item with username
	selectStmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ID.EQ(String(item.ID)),
	)

	var result struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err = selectStmt.Query(repo.Db, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get created list item with username: %w", err)
	}

	// Convert to response type
	createdItemWithUsername := &response.ShopListItemWithUsername{
		ID:              result.ID,
		ListID:          result.ListID,
		Niin:            result.Niin,
		Nomenclature:    result.Nomenclature,
		Quantity:        result.Quantity,
		AddedBy:         result.AddedBy,
		AddedByUsername: result.AddedByUsername,
		CreatedAt:       &result.CreatedAt,
		UpdatedAt:       &result.UpdatedAt,
		Nickname:        result.Nickname,
		UnitOfMeasure:   result.UnitOfMeasure,
	}

	return createdItemWithUsername, nil
}

func (repo *ShopsRepositoryImpl) GetListItems(user *bootstrap.User, listID string) ([]response.ShopListItemWithUsername, error) {
	stmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ListID.EQ(String(listID)),
	).ORDER_BY(ShopListItems.CreatedAt.ASC())

	var results []struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err := stmt.Query(repo.Db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items with usernames: %w", err)
	}

	// Convert to response type
	items := make([]response.ShopListItemWithUsername, len(results))
	for i, r := range results {
		items[i] = response.ShopListItemWithUsername{
			ID:              r.ID,
			ListID:          r.ListID,
			Niin:            r.Niin,
			Nomenclature:    r.Nomenclature,
			Quantity:        r.Quantity,
			AddedBy:         r.AddedBy,
			AddedByUsername: r.AddedByUsername,
			CreatedAt:       &r.CreatedAt,
			UpdatedAt:       &r.UpdatedAt,
			Nickname:        r.Nickname,
			UnitOfMeasure:   r.UnitOfMeasure,
		}
	}

	return items, nil
}

func (repo *ShopsRepositoryImpl) GetListItemByID(user *bootstrap.User, itemID string) (*model.ShopListItems, error) {
	stmt := SELECT(ShopListItems.AllColumns).
		FROM(ShopListItems).
		WHERE(ShopListItems.ID.EQ(String(itemID)))

	var item model.ShopListItems
	err := stmt.Query(repo.Db, &item)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("list item not found")
		}
		return nil, fmt.Errorf("failed to get list item: %w", err)
	}

	return &item, nil
}

func (repo *ShopsRepositoryImpl) UpdateListItem(user *bootstrap.User, item model.ShopListItems) error {
	stmt := ShopListItems.UPDATE(
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODEL(item).
		WHERE(ShopListItems.ID.EQ(String(item.ID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("list item not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) RemoveListItem(user *bootstrap.User, itemID string) error {
	stmt := ShopListItems.DELETE().
		WHERE(ShopListItems.ID.EQ(String(itemID)))

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to remove list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("list item not found")
	}

	return nil
}

func (repo *ShopsRepositoryImpl) AddListItemBatch(user *bootstrap.User, items []model.ShopListItems) ([]response.ShopListItemWithUsername, error) {
	if len(items) == 0 {
		return []response.ShopListItemWithUsername{}, nil
	}

	stmt := ShopListItems.INSERT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
	).MODELS(items)

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return nil, fmt.Errorf("failed to add list items: %w", err)
	}

	// Get the created items with usernames
	itemIDs := make([]postgres.Expression, len(items))
	for i, item := range items {
		itemIDs[i] = String(item.ID)
	}

	selectStmt := SELECT(
		ShopListItems.ID,
		ShopListItems.ListID,
		ShopListItems.Niin,
		ShopListItems.Nomenclature,
		ShopListItems.Quantity,
		ShopListItems.AddedBy,
		ShopListItems.CreatedAt,
		ShopListItems.UpdatedAt,
		ShopListItems.Nickname,
		ShopListItems.UnitOfMeasure,
		Users.Username.AS("added_by_username"),
	).FROM(
		ShopListItems.
			LEFT_JOIN(Users, Users.UID.EQ(ShopListItems.AddedBy)),
	).WHERE(
		ShopListItems.ID.IN(itemIDs...),
	).ORDER_BY(ShopListItems.CreatedAt.ASC())

	var results []struct {
		model.ShopListItems
		AddedByUsername *string `sql:"added_by_username"`
	}

	err = selectStmt.Query(repo.Db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get created list items with usernames: %w", err)
	}

	// Convert to response type
	createdItemsWithUsername := make([]response.ShopListItemWithUsername, len(results))
	for i, r := range results {
		createdItemsWithUsername[i] = response.ShopListItemWithUsername{
			ID:              r.ID,
			ListID:          r.ListID,
			Niin:            r.Niin,
			Nomenclature:    r.Nomenclature,
			Quantity:        r.Quantity,
			AddedBy:         r.AddedBy,
			AddedByUsername: r.AddedByUsername,
			CreatedAt:       &r.CreatedAt,
			UpdatedAt:       &r.UpdatedAt,
			Nickname:        r.Nickname,
			UnitOfMeasure:   r.UnitOfMeasure,
		}
	}

	return createdItemsWithUsername, nil
}

func (repo *ShopsRepositoryImpl) RemoveListItemBatch(user *bootstrap.User, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Convert string slice to expressions for the IN clause
	var expressions []Expression
	for _, id := range itemIDs {
		expressions = append(expressions, String(id))
	}

	stmt := ShopListItems.DELETE().
		WHERE(ShopListItems.ID.IN(expressions...))

	_, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to remove list items: %w", err)
	}

	return nil
}

// Helper method for permissions
func (repo *ShopsRepositoryImpl) GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error) {
	stmt := SELECT(ShopMembers.Role).
		FROM(ShopMembers).
		WHERE(
			ShopMembers.ShopID.EQ(String(shopID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		)

	var role string
	err := stmt.Query(repo.Db, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("user is not a member of this shop")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// Notification Change Tracking (Audit Trail) Operations

// CreateNotificationChange records a change to a vehicle notification
func (repo *ShopsRepositoryImpl) CreateNotificationChange(
	user *bootstrap.User,
	change model.ShopVehicleNotificationChanges,
) error {
	rawSQL := `
		INSERT INTO shop_vehicle_notification_changes (
			notification_id,
			shop_id,
			vehicle_id,
			changed_by,
			change_type,
			field_changes
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := repo.Db.Exec(
		rawSQL,
		change.NotificationID,
		change.ShopID,
		change.VehicleID,
		change.ChangedBy,
		change.ChangeType,
		change.FieldChanges,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification change record: %w", err)
	}

	return nil
}

// GetNotificationChanges retrieves all change history for a specific notification
func (repo *ShopsRepositoryImpl) GetNotificationChanges(
	user *bootstrap.User,
	notificationID string,
) ([]response.NotificationChangeWithUsername, error) {
	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		WHERE c.notification_id = $1
		ORDER BY c.changed_at DESC
	`

	rows, err := repo.Db.Query(rawSQL, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		// Parse JSONB field_changes
		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			// Simple JSON parsing - will be handled by the service layer if needed
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
}

// GetNotificationChangesByShop retrieves recent changes for all notifications in a shop
func (repo *ShopsRepositoryImpl) GetNotificationChangesByShop(
	user *bootstrap.User,
	shopID string,
	limit int,
) ([]response.NotificationChangeWithUsername, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 500 {
		limit = 500 // Maximum limit
	}

	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes,
			COALESCE(n.title, 'Deleted Notification') as notification_title
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
		WHERE c.shop_id = $1
		ORDER BY c.changed_at DESC
		LIMIT $2
	`

	rows, err := repo.Db.Query(rawSQL, shopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
			&change.NotificationTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		// Parse JSONB field_changes
		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
}

// GetNotificationChangesByVehicle retrieves all changes for notifications on a specific vehicle
func (repo *ShopsRepositoryImpl) GetNotificationChangesByVehicle(
	user *bootstrap.User,
	vehicleID string,
) ([]response.NotificationChangeWithUsername, error) {
	rawSQL := `
		SELECT
			c.id,
			c.notification_id,
			c.shop_id,
			c.vehicle_id,
			c.changed_by,
			COALESCE(u.username, 'Unknown User') as changed_by_username,
			c.changed_at,
			c.change_type,
			c.field_changes,
			COALESCE(n.title, 'Deleted Notification') as notification_title
		FROM shop_vehicle_notification_changes c
		LEFT JOIN users u ON c.changed_by = u.uid
		LEFT JOIN shop_vehicle_notifications n ON c.notification_id = n.id
		WHERE c.vehicle_id = $1
		ORDER BY c.changed_at DESC
	`

	rows, err := repo.Db.Query(rawSQL, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle notification changes: %w", err)
	}
	defer rows.Close()

	var changes []response.NotificationChangeWithUsername
	for rows.Next() {
		var change response.NotificationChangeWithUsername
		var fieldChangesJSON string

		err := rows.Scan(
			&change.ID,
			&change.NotificationID,
			&change.ShopID,
			&change.VehicleID,
			&change.ChangedBy,
			&change.ChangedByUsername,
			&change.ChangedAt,
			&change.ChangeType,
			&fieldChangesJSON,
			&change.NotificationTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan change row: %w", err)
		}

		// Parse JSONB field_changes
		change.FieldChanges = make(map[string]interface{})
		if fieldChangesJSON != "" {
			change.FieldChanges["raw"] = fieldChangesJSON
		}

		changes = append(changes, change)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change rows: %w", err)
	}

	return changes, nil
}

// Shop Settings Operations

// GetShopAdminOnlyListsSetting retrieves the admin_only_lists setting for a shop
func (repo *ShopsRepositoryImpl) GetShopAdminOnlyListsSetting(shopID string) (bool, error) {
	stmt := SELECT(Shops.AllColumns).
		FROM(Shops).
		WHERE(Shops.ID.EQ(String(shopID)))

	var shop model.Shops
	err := stmt.Query(repo.Db, &shop)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.New("shop not found")
		}
		return false, fmt.Errorf("failed to get admin_only_lists setting: %w", err)
	}

	return shop.AdminOnlyLists, nil
}

// UpdateShopAdminOnlyListsSetting updates the admin_only_lists setting for a shop
func (repo *ShopsRepositoryImpl) UpdateShopAdminOnlyListsSetting(shopID string, adminOnlyLists bool) error {
	now := time.Now()

	stmt := Shops.UPDATE(
		Shops.AdminOnlyLists,
		Shops.UpdatedAt,
	).SET(
		Shops.AdminOnlyLists.SET(Bool(adminOnlyLists)),
		Shops.UpdatedAt.SET(TimestampzT(now)),
	).WHERE(
		Shops.ID.EQ(String(shopID)),
	)

	result, err := stmt.Exec(repo.Db)
	if err != nil {
		return fmt.Errorf("failed to update admin_only_lists setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("shop not found")
	}

	slog.Info("Shop admin_only_lists setting updated", "shop_id", shopID, "admin_only_lists", adminOnlyLists)
	return nil
}
