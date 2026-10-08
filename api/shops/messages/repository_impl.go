package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	sharedb "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

func shopMessageAuthorUsernameProjection() Projection {
	return NULLIF(BTRIM(Users.Username), String("")).AS("author_username")
}

const (
	shopMessageImagesContainer = "shop-message-images"
)

type RepositoryImpl struct {
	db     *sql.DB
	assets AssetRepository
}

// Keep this as an alias to an anonymous struct. Jet's qrm mapper uses the
// anonymous embedded-model shape when matching aliased projections.
type shopMessageResponseRow = struct {
	model.ShopMessages
	AuthorUsername *string `sql:"author_username"`
}

func mapShopMessageResponseRows(rows []shopMessageResponseRow) []response.ShopMessageResponse {
	messages := make([]response.ShopMessageResponse, len(rows))
	for i, row := range rows {
		messages[i] = response.NewShopMessageResponse(row.ShopMessages, row.AuthorUsername)
	}
	return messages
}

func NewRepository(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env) *RepositoryImpl {
	storage := AssetStorage{Container: shopMessageImagesContainer}
	if env != nil {
		storage.Account = env.BlobAccountName
	}
	return &RepositoryImpl{
		assets: NewAssetRepository(db, storage),
		db:     db,
	}
}

func (repo *RepositoryImpl) CreateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) (*response.ShopMessageResponse, error) {
	var createdMessage response.ShopMessageResponse
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if _, _, err := shared.LockShopMutation(ctx, tx, message.ShopID, user.UserID); err != nil {
			return err
		}
		if message.ParentID != nil {
			// Filter before locking so a foreign parent never acquires another Shop's
			// resource lock. Retain the parent lock through insertion and references.
			var parent model.ShopMessages
			err := SELECT(ShopMessages.ID, ShopMessages.ShopID).FROM(ShopMessages).
				WHERE(ShopMessages.ID.EQ(String(*message.ParentID)).AND(ShopMessages.ShopID.EQ(String(message.ShopID)))).
				FOR(KEY_SHARE()).QueryContext(ctx, tx, &parent)
			if errors.Is(err, qrm.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				return errors.New("reply parent is unavailable in this shop")
			}
			if err != nil {
				return fmt.Errorf("lock reply parent: %w", err)
			}
		}
		stmt := ShopMessages.INSERT(
			ShopMessages.ID,
			ShopMessages.ShopID,
			ShopMessages.UserID,
			ShopMessages.Message,
			ShopMessages.CreatedAt,
			ShopMessages.UpdatedAt,
			ShopMessages.IsEdited,
			ShopMessages.ParentID,
		).MODEL(message)

		if _, err := stmt.ExecContext(ctx, tx); err != nil {
			return fmt.Errorf("failed to create shop message: %w", err)
		}

		if err := repo.assets.ReplaceReferences(ctx, tx, user, message.ID, message.Message); err != nil {
			return err
		}
		var err error
		createdMessage, err = getShopMessageResponseByID(ctx, tx, message.ID)
		if err != nil {
			return fmt.Errorf("failed to get created shop message: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &createdMessage, nil
}

func getShopMessageResponseByID(ctx context.Context, queryable qrm.Queryable, messageID string) (response.ShopMessageResponse, error) {
	stmt := SELECT(legacyMessageColumns(), shopMessageAuthorUsernameProjection()).
		FROM(
			ShopMessages.
				LEFT_JOIN(Users, Users.UID.EQ(ShopMessages.UserID)),
		).
		WHERE(ShopMessages.ID.EQ(String(messageID)))

	var row shopMessageResponseRow
	if err := stmt.QueryContext(ctx, queryable, &row); err != nil {
		return response.ShopMessageResponse{}, err
	}

	return response.NewShopMessageResponse(row.ShopMessages, row.AuthorUsername), nil
}

func (repo *RepositoryImpl) GetShopMessages(ctx context.Context, user *bootstrap.User, shopID string) ([]response.ShopMessageResponse, error) {
	stmt := SELECT(legacyMessageColumns(), shopMessageAuthorUsernameProjection()).
		FROM(
			ShopMessages.
				LEFT_JOIN(Users, Users.UID.EQ(ShopMessages.UserID)),
		).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.ASC(), ShopMessages.ID.ASC())

	var rows []shopMessageResponseRow
	err := stmt.QueryContext(ctx, repo.db, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop messages: %w", err)
	}

	return mapShopMessageResponseRows(rows), nil
}

func (repo *RepositoryImpl) GetShopMessagesPaginated(ctx context.Context, user *bootstrap.User, shopID string, offset int, limit int) ([]response.ShopMessageResponse, error) {
	stmt := SELECT(legacyMessageColumns(), shopMessageAuthorUsernameProjection()).
		FROM(
			ShopMessages.
				LEFT_JOIN(Users, Users.UID.EQ(ShopMessages.UserID)),
		).
		WHERE(ShopMessages.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopMessages.CreatedAt.DESC(), ShopMessages.ID.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	var rows []shopMessageResponseRow
	err := stmt.QueryContext(ctx, repo.db, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated shop messages: %w", err)
	}

	return mapShopMessageResponseRows(rows), nil
}

func (repo *RepositoryImpl) GetShopMessagesByCursor(ctx context.Context, user *bootstrap.User, shopID, cursorID string, cursorTime time.Time, isBefore bool, limit int) ([]response.ShopMessageResponse, error) {
	condition := ShopMessages.CreatedAt.LT(TimestampzT(cursorTime)).OR(
		ShopMessages.CreatedAt.EQ(TimestampzT(cursorTime)).AND(ShopMessages.ID.LT(String(cursorID))),
	)
	order := []OrderByClause{ShopMessages.CreatedAt.DESC(), ShopMessages.ID.DESC()}
	if !isBefore {
		condition = ShopMessages.CreatedAt.GT(TimestampzT(cursorTime)).OR(
			ShopMessages.CreatedAt.EQ(TimestampzT(cursorTime)).AND(ShopMessages.ID.GT(String(cursorID))),
		)
		// Select the nearest newer tuples so a bounded read cannot skip a burst.
		order = []OrderByClause{ShopMessages.CreatedAt.ASC(), ShopMessages.ID.ASC()}
	}

	stmt := SELECT(legacyMessageColumns(), shopMessageAuthorUsernameProjection()).
		FROM(
			ShopMessages.
				LEFT_JOIN(Users, Users.UID.EQ(ShopMessages.UserID)),
		).
		WHERE(
			ShopMessages.ShopID.EQ(String(shopID)).
				AND(condition),
		).
		ORDER_BY(order...).
		LIMIT(int64(limit))

	var rows []shopMessageResponseRow
	err := stmt.QueryContext(ctx, repo.db, &rows)
	if err != nil {
		return nil, fmt.Errorf("failed to get cursor-based shop messages: %w", err)
	}

	return mapShopMessageResponseRows(rows), nil
}

func (repo *RepositoryImpl) GetShopMessagesCount(ctx context.Context, user *bootstrap.User, shopID string) (int64, error) {
	stmt := SELECT(COUNT(ShopMessages.ID)).
		FROM(ShopMessages).
		WHERE(ShopMessages.ShopID.EQ(String(shopID)))

	var result struct {
		Count int64 `sql:"primary_key"`
	}
	err := stmt.QueryContext(ctx, repo.db, &result)
	if err != nil {
		return 0, fmt.Errorf("failed to get shop messages count: %w", err)
	}

	return result.Count, nil
}

func (repo *RepositoryImpl) UpdateShopMessage(ctx context.Context, user *bootstrap.User, message model.ShopMessages) error {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeOwnedMutation(ctx, tx, user.UserID, "message", message.ID, false); err != nil {
		return err
	}

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

	result, err := stmt.ExecContext(ctx, tx)
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

	if err := repo.assets.ReplaceReferences(ctx, tx, user, message.ID, message.Message); err != nil {
		return err
	}
	return tx.Commit()
}

func (repo *RepositoryImpl) DeleteShopMessage(ctx context.Context, user *bootstrap.User, messageID string) error {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := shared.AuthorizeOwnedMutation(ctx, tx, user.UserID, "message", messageID, true); err != nil {
		return err
	}

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

	result, err := stmt.ExecContext(ctx, tx)
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

	return tx.Commit()
}

func (repo *RepositoryImpl) GetShopMessageByID(ctx context.Context, user *bootstrap.User, messageID string) (*response.ShopMessageResponse, error) {
	message, err := getShopMessageResponseByID(ctx, repo.db, messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (repo *RepositoryImpl) ReserveMessageImage(ctx context.Context, user *bootstrap.User, shopID, extension string) (Asset, error) {
	var asset Asset
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		var err error
		asset, err = repo.assets.Reserve(ctx, tx, user, shopID, extension)
		return err
	})
	if err != nil {
		return Asset{}, err
	}
	return asset, nil
}
func (repo *RepositoryImpl) FinalizeMessageImage(ctx context.Context, user *bootstrap.User, uploadID string) error {
	return sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error { return repo.assets.Finalize(ctx, tx, user, uploadID) })
}
func (repo *RepositoryImpl) FailMessageImage(ctx context.Context, asset Asset) error {
	return sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error { return repo.assets.FailReserved(ctx, tx, asset, "upload_not_finalized") })
}

func (repo *RepositoryImpl) DeleteMessageImageBlob(ctx context.Context, user *bootstrap.User, messageID string, shopID string) error {
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		return repo.assets.Discard(ctx, tx, user, shopID, messageID)
	})
	return err
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

// Keep legacy reads compatible with both pre-018 and expanded generated models.
func legacyMessageColumns() ProjectionList {
	return ProjectionList{ShopMessages.ID, ShopMessages.ShopID, ShopMessages.UserID, ShopMessages.Message, ShopMessages.CreatedAt, ShopMessages.UpdatedAt, ShopMessages.IsEdited, ShopMessages.ParentID}
}
