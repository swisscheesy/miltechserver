package repository

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

type MaterialImagesRepositoryImpl struct {
	db *sql.DB
}

func NewMaterialImagesRepositoryImpl(db *sql.DB) MaterialImagesRepository {
	return &MaterialImagesRepositoryImpl{
		db: db,
	}
}

// Image operations

func (r *MaterialImagesRepositoryImpl) CreateImage(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error) {
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

func (r *MaterialImagesRepositoryImpl) GetImageByID(imageID string) (*model.MaterialImages, error) {
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

func (r *MaterialImagesRepositoryImpl) GetImagesByNIIN(niin string, limit int, offset int) ([]MaterialImageWithUser, int64, error) {
	// Use raw SQL query similar to GetShopMembers to ensure proper JOIN handling
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

	var imagesWithUsers []MaterialImageWithUser
	for rows.Next() {
		var img MaterialImageWithUser
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

	// Get total count using raw SQL
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

func (r *MaterialImagesRepositoryImpl) GetImagesByUser(userID string, limit int, offset int) ([]MaterialImageWithUser, int64, error) {
	// Use raw SQL query similar to GetShopMembers to ensure proper JOIN handling
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

	var imagesWithUsers []MaterialImageWithUser
	for rows.Next() {
		var img MaterialImageWithUser
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

	// Get total count using raw SQL
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

func (r *MaterialImagesRepositoryImpl) UpdateImageFlags(imageID string, flagCount int, isFlagged bool) error {
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

func (r *MaterialImagesRepositoryImpl) DeleteImage(imageID string) error {
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

// Vote operations

func (r *MaterialImagesRepositoryImpl) UpsertVote(vote model.MaterialImagesVotes) error {
	stmt := MaterialImagesVotes.INSERT(
		MaterialImagesVotes.ImageID,
		MaterialImagesVotes.UserID,
		MaterialImagesVotes.VoteType,
	).VALUES(
		vote.ImageID,
		vote.UserID,
		vote.VoteType,
	).ON_CONFLICT(
		MaterialImagesVotes.ImageID,
		MaterialImagesVotes.UserID,
	).DO_UPDATE(
		SET(
			MaterialImagesVotes.VoteType.SET(String(vote.VoteType)),
			MaterialImagesVotes.UpdatedAt.SET(TimestampT(time.Now())),
		),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to upsert vote: %w", err)
	}

	return nil
}

func (r *MaterialImagesRepositoryImpl) DeleteVote(imageID string, userID string) error {
	stmt := MaterialImagesVotes.DELETE().WHERE(
		MaterialImagesVotes.ImageID.EQ(UUID(uuid.MustParse(imageID))).
			AND(MaterialImagesVotes.UserID.EQ(String(userID))),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to delete vote: %w", err)
	}

	return nil
}

func (r *MaterialImagesRepositoryImpl) GetUserVoteForImage(imageID string, userID string) (*model.MaterialImagesVotes, error) {
	stmt := SELECT(
		MaterialImagesVotes.AllColumns,
	).FROM(
		MaterialImagesVotes,
	).WHERE(
		MaterialImagesVotes.ImageID.EQ(UUID(uuid.MustParse(imageID))).
			AND(MaterialImagesVotes.UserID.EQ(String(userID))),
	)

	var vote model.MaterialImagesVotes
	err := stmt.Query(r.db, &vote)
	if err != nil {
		if err == qrm.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user vote: %w", err)
	}

	return &vote, nil
}

func (r *MaterialImagesRepositoryImpl) UpdateImageVoteCounts(imageID string) error {
	// Calculate upvotes
	upvoteStmt := SELECT(
		COUNT(MaterialImagesVotes.ImageID).AS("count"),
	).FROM(
		MaterialImagesVotes,
	).WHERE(
		MaterialImagesVotes.ImageID.EQ(UUID(uuid.MustParse(imageID))).
			AND(MaterialImagesVotes.VoteType.EQ(String("upvote"))),
	)

	var upvoteCount struct {
		Count int32 `sql:"count"`
	}
	err := upvoteStmt.Query(r.db, &upvoteCount)
	if err != nil {
		return fmt.Errorf("failed to get upvote count: %w", err)
	}

	// Calculate downvotes
	downvoteStmt := SELECT(
		COUNT(MaterialImagesVotes.ImageID).AS("count"),
	).FROM(
		MaterialImagesVotes,
	).WHERE(
		MaterialImagesVotes.ImageID.EQ(UUID(uuid.MustParse(imageID))).
			AND(MaterialImagesVotes.VoteType.EQ(String("downvote"))),
	)

	var downvoteCount struct {
		Count int32 `sql:"count"`
	}
	err = downvoteStmt.Query(r.db, &downvoteCount)
	if err != nil {
		return fmt.Errorf("failed to get downvote count: %w", err)
	}

	// Update image counts
	updateStmt := MaterialImages.UPDATE(
		MaterialImages.UpvoteCount,
		MaterialImages.DownvoteCount,
		MaterialImages.UpdatedAt,
	).SET(
		upvoteCount.Count,
		downvoteCount.Count,
		TimestampT(time.Now()),
	).WHERE(
		MaterialImages.ID.EQ(UUID(uuid.MustParse(imageID))),
	)

	_, err = updateStmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to update vote counts: %w", err)
	}

	return nil
}

// Flag operations

func (r *MaterialImagesRepositoryImpl) CreateFlag(flag model.MaterialImagesFlags) error {
	stmt := MaterialImagesFlags.INSERT(
		MaterialImagesFlags.ImageID,
		MaterialImagesFlags.UserID,
		MaterialImagesFlags.Reason,
		MaterialImagesFlags.Description,
	).VALUES(
		flag.ImageID,
		flag.UserID,
		flag.Reason,
		flag.Description,
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to create flag: %w", err)
	}

	// Update flag count on image
	countStmt := SELECT(
		COUNT(MaterialImagesFlags.ID).AS("count"),
	).FROM(
		MaterialImagesFlags,
	).WHERE(
		MaterialImagesFlags.ImageID.EQ(UUID(flag.ImageID)),
	)

	var count struct {
		Count int32 `sql:"count"`
	}
	err = countStmt.Query(r.db, &count)
	if err != nil {
		return fmt.Errorf("failed to get flag count: %w", err)
	}

	// Flag image immediately when any flag is created
	isFlagged := count.Count >= 1

	err = r.UpdateImageFlags(flag.ImageID.String(), int(count.Count), isFlagged)
	if err != nil {
		return fmt.Errorf("failed to update image flag status: %w", err)
	}

	return nil
}

func (r *MaterialImagesRepositoryImpl) GetFlagsByImage(imageID string) ([]model.MaterialImagesFlags, error) {
	stmt := SELECT(
		MaterialImagesFlags.AllColumns,
	).FROM(
		MaterialImagesFlags,
	).WHERE(
		MaterialImagesFlags.ImageID.EQ(UUID(uuid.MustParse(imageID))),
	).ORDER_BY(
		MaterialImagesFlags.CreatedAt.DESC(),
	)

	var flags []model.MaterialImagesFlags
	err := stmt.Query(r.db, &flags)
	if err != nil {
		return nil, fmt.Errorf("failed to get flags: %w", err)
	}

	return flags, nil
}

// Rate limiting

func (r *MaterialImagesRepositoryImpl) CheckUploadLimit(userID string, niin string) (bool, *time.Time, error) {
	stmt := SELECT(
		MaterialImagesUploadLimits.AllColumns,
	).FROM(
		MaterialImagesUploadLimits,
	).WHERE(
		MaterialImagesUploadLimits.UserID.EQ(String(userID)).
			AND(MaterialImagesUploadLimits.Niin.EQ(String(niin))),
	)

	var limit model.MaterialImagesUploadLimits
	err := stmt.Query(r.db, &limit)
	if err != nil {
		if err == qrm.ErrNoRows {
			// No previous upload, allowed
			return true, nil, nil
		}
		return false, nil, fmt.Errorf("failed to check upload limit: %w", err)
	}

	// Check if 1 hour has passed - ensure we use UTC for consistent comparison
	nextAllowedTime := limit.LastUploadTime.Add(1 * time.Hour)
	now := time.Now().UTC()  // Convert to UTC for consistent comparison
	
	if now.After(nextAllowedTime) {
		return true, nil, nil
	}

	return false, &nextAllowedTime, nil
}

func (r *MaterialImagesRepositoryImpl) UpdateUploadLimit(userID string, niin string) error {
	now := time.Now().UTC()  // Use UTC consistently
	
	stmt := MaterialImagesUploadLimits.INSERT(
		MaterialImagesUploadLimits.UserID,
		MaterialImagesUploadLimits.Niin,
		MaterialImagesUploadLimits.LastUploadTime,
		MaterialImagesUploadLimits.UploadCount,
	).VALUES(
		userID,
		niin,
		TimestampT(now),
		1,
	).ON_CONFLICT(
		MaterialImagesUploadLimits.UserID,
		MaterialImagesUploadLimits.Niin,
	).DO_UPDATE(
		SET(
			MaterialImagesUploadLimits.LastUploadTime.SET(TimestampT(now)),
			MaterialImagesUploadLimits.UploadCount.SET(
				MaterialImagesUploadLimits.UploadCount.ADD(Int(1)),
			),
		),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to update upload limit: %w", err)
	}

	return nil
}

func (r *MaterialImagesRepositoryImpl) CleanupOldLimits(olderThan time.Time) error {
	stmt := MaterialImagesUploadLimits.DELETE().WHERE(
		MaterialImagesUploadLimits.LastUploadTime.LT(TimestampT(olderThan)),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to cleanup old limits: %w", err)
	}

	return nil
}

// GetUsernameByUserID looks up a username by user ID
func (r *MaterialImagesRepositoryImpl) GetUsernameByUserID(userID string) (string, error) {
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
