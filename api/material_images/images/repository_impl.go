package images

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Create(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error) {
	stmt := MaterialImages.INSERT(
		MaterialImages.Niin,
		MaterialImages.UserID,
		MaterialImages.BlobName,
		MaterialImages.BlobURL,
		MaterialImages.OriginalFilename,
		MaterialImages.FileSizeBytes,
		MaterialImages.MimeType,
	).VALUES(
		image.Niin,
		image.UserID,
		image.BlobName,
		image.BlobURL,
		image.OriginalFilename,
		image.FileSizeBytes,
		image.MimeType,
	).RETURNING(MaterialImages.AllColumns)

	var createdImage model.MaterialImages
	err := stmt.Query(r.db, &createdImage)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	return &createdImage, nil
}

func (r *RepositoryImpl) GetByID(imageID string) (*model.MaterialImages, error) {
	stmt := SELECT(
		MaterialImages.AllColumns,
	).FROM(
		MaterialImages,
	).WHERE(
		MaterialImages.ID.EQ(UUID(uuid.MustParse(imageID))),
	)

	var image model.MaterialImages
	err := stmt.Query(r.db, &image)
	if err != nil {
		if err == qrm.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get image by ID: %w", err)
	}

	return &image, nil
}

func (r *RepositoryImpl) GetByNIIN(niin string, limit int, offset int) ([]ImageWithUser, int64, error) {
	rawSQL := `
		SELECT 
			mi.id,
			mi.niin,
			mi.user_id,
			mi.blob_name,
			mi.blob_url,
			mi.original_filename,
			mi.file_size_bytes,
			mi.mime_type,
			mi.upload_date,
			mi.is_active,
			mi.is_flagged,
			mi.flag_count,
			mi.downvote_count,
			mi.upvote_count,
			mi.net_votes,
			mi.created_at,
			mi.updated_at,
			COALESCE(u.username, 'Unknown') as username
		FROM material_images mi
		LEFT JOIN users u ON mi.user_id = u.uid
		WHERE mi.niin = $1 AND mi.is_active = true
		ORDER BY mi.net_votes DESC, mi.upload_date DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(rawSQL, niin, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images by NIIN: %w", err)
	}
	defer rows.Close()

	var imagesWithUsers []ImageWithUser
	for rows.Next() {
		var img ImageWithUser
		var username string

		err := rows.Scan(
			&img.ID,
			&img.Niin,
			&img.UserID,
			&img.BlobName,
			&img.BlobURL,
			&img.OriginalFilename,
			&img.FileSizeBytes,
			&img.MimeType,
			&img.UploadDate,
			&img.IsActive,
			&img.IsFlagged,
			&img.FlagCount,
			&img.DownvoteCount,
			&img.UpvoteCount,
			&img.NetVotes,
			&img.CreatedAt,
			&img.UpdatedAt,
			&username,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan image row: %w", err)
		}

		img.Username = &username
		imagesWithUsers = append(imagesWithUsers, img)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	countSQL := `
		SELECT COUNT(*) 
		FROM material_images mi 
		WHERE mi.niin = $1 AND mi.is_active = true
	`

	var count int64
	err = r.db.QueryRow(countSQL, niin).Scan(&count)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get image count: %w", err)
	}

	return imagesWithUsers, count, nil
}

func (r *RepositoryImpl) GetByUser(userID string, limit int, offset int) ([]ImageWithUser, int64, error) {
	rawSQL := `
		SELECT 
			mi.id,
			mi.niin,
			mi.user_id,
			mi.blob_name,
			mi.blob_url,
			mi.original_filename,
			mi.file_size_bytes,
			mi.mime_type,
			mi.upload_date,
			mi.is_active,
			mi.is_flagged,
			mi.flag_count,
			mi.downvote_count,
			mi.upvote_count,
			mi.net_votes,
			mi.created_at,
			mi.updated_at,
			COALESCE(u.username, 'Unknown') as username
		FROM material_images mi
		LEFT JOIN users u ON mi.user_id = u.uid
		WHERE mi.user_id = $1 AND mi.is_active = true
		ORDER BY mi.upload_date DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(rawSQL, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get images by user: %w", err)
	}
	defer rows.Close()

	var imagesWithUsers []ImageWithUser
	for rows.Next() {
		var img ImageWithUser
		var username string

		err := rows.Scan(
			&img.ID,
			&img.Niin,
			&img.UserID,
			&img.BlobName,
			&img.BlobURL,
			&img.OriginalFilename,
			&img.FileSizeBytes,
			&img.MimeType,
			&img.UploadDate,
			&img.IsActive,
			&img.IsFlagged,
			&img.FlagCount,
			&img.DownvoteCount,
			&img.UpvoteCount,
			&img.NetVotes,
			&img.CreatedAt,
			&img.UpdatedAt,
			&username,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan image row: %w", err)
		}

		img.Username = &username
		imagesWithUsers = append(imagesWithUsers, img)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	countSQL := `
		SELECT COUNT(*) 
		FROM material_images mi 
		WHERE mi.user_id = $1 AND mi.is_active = true
	`

	var count int64
	err = r.db.QueryRow(countSQL, userID).Scan(&count)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get image count: %w", err)
	}

	return imagesWithUsers, count, nil
}

func (r *RepositoryImpl) UpdateFlags(imageID string, flagCount int, isFlagged bool) error {
	stmt := MaterialImages.UPDATE(
		MaterialImages.FlagCount,
		MaterialImages.IsFlagged,
		MaterialImages.UpdatedAt,
	).SET(
		flagCount,
		isFlagged,
		TimestampT(time.Now()),
	).WHERE(
		MaterialImages.ID.EQ(UUID(uuid.MustParse(imageID))),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to update image flags: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) Delete(imageID string) error {
	stmt := MaterialImages.UPDATE(
		MaterialImages.IsActive,
		MaterialImages.UpdatedAt,
	).SET(
		false,
		TimestampT(time.Now()),
	).WHERE(
		MaterialImages.ID.EQ(UUID(uuid.MustParse(imageID))),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) GetUsernameByUserID(userID string) (string, error) {
	stmt := SELECT(
		Users.Username,
	).FROM(
		Users,
	).WHERE(
		Users.UID.EQ(String(userID)),
	)

	var user model.Users
	err := stmt.Query(r.db, &user)
	if err != nil {
		if err == qrm.ErrNoRows {
			return "Unknown", nil
		}
		return "", fmt.Errorf("failed to get username: %w", err)
	}

	if user.Username == nil {
		return "Unknown", nil
	}

	return *user.Username, nil
}
