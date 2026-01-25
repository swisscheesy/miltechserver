package repository

import (
	"database/sql"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type ItemCommentsRepositoryImpl struct {
	db *sql.DB
}

func NewItemCommentsRepositoryImpl(db *sql.DB) *ItemCommentsRepositoryImpl {
	return &ItemCommentsRepositoryImpl{db: db}
}

func (repo *ItemCommentsRepositoryImpl) GetCommentsByNiin(niin string) ([]ItemCommentWithAuthor, error) {
	rawSQL := `
		SELECT
			c.id,
			c.comment_niin,
			c.author_id,
			c.text,
			c.parent_id,
			c.created_at,
			c.updated_at,
			u.username AS author_display_name
		FROM item_comments c
		LEFT JOIN users u ON c.author_id = u.uid
		WHERE c.comment_niin = $1
		ORDER BY c.created_at ASC
	`

	rows, err := repo.db.Query(rawSQL, niin)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments by NIIN: %w", err)
	}
	defer rows.Close()

	var comments []ItemCommentWithAuthor
	for rows.Next() {
		var (
			commentID   uuid.UUID
			commentNiin string
			authorID    string
			text        string
			parentID    sql.NullString
			createdAt   time.Time
			updatedAt   sql.NullTime
			username    sql.NullString
		)

		if scanErr := rows.Scan(
			&commentID,
			&commentNiin,
			&authorID,
			&text,
			&parentID,
			&createdAt,
			&updatedAt,
			&username,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan comment row: %w", scanErr)
		}

		var parsedParentID *uuid.UUID
		if parentID.Valid {
			parsed, parseErr := uuid.Parse(parentID.String)
			if parseErr != nil {
				return nil, fmt.Errorf("failed to parse parent_id: %w", parseErr)
			}
			parsedParentID = &parsed
		}

		var parsedUpdatedAt *time.Time
		if updatedAt.Valid {
			parsedUpdatedAt = &updatedAt.Time
		}

		comment := model.ItemComments{
			ID:          commentID,
			CommentNiin: commentNiin,
			AuthorID:    authorID,
			Text:        text,
			ParentID:    parsedParentID,
			CreatedAt:   createdAt,
			UpdatedAt:   parsedUpdatedAt,
		}

		var authorDisplayName *string
		if username.Valid {
			authorDisplayName = &username.String
		}

		comments = append(comments, ItemCommentWithAuthor{
			ItemComments:      comment,
			AuthorDisplayName: authorDisplayName,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comment rows: %w", err)
	}

	return comments, nil
}

func (repo *ItemCommentsRepositoryImpl) GetCommentByID(commentID uuid.UUID) (*model.ItemComments, error) {
	stmt := SELECT(
		table.ItemComments.AllColumns,
	).FROM(
		table.ItemComments,
	).WHERE(
		table.ItemComments.ID.EQ(UUID(commentID)),
	)

	var comment model.ItemComments
	err := stmt.Query(repo.db, &comment)
	if err != nil {
		if err == qrm.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return &comment, nil
}

func (repo *ItemCommentsRepositoryImpl) CreateComment(comment model.ItemComments) (*model.ItemComments, error) {
	parentValue := NULL
	if comment.ParentID != nil {
		parentValue = UUID(*comment.ParentID)
	}

	stmt := table.ItemComments.INSERT(
		table.ItemComments.CommentNiin,
		table.ItemComments.AuthorID,
		table.ItemComments.Text,
		table.ItemComments.ParentID,
	).VALUES(
		comment.CommentNiin,
		comment.AuthorID,
		comment.Text,
		parentValue,
	).RETURNING(
		table.ItemComments.AllColumns,
	)

	var created model.ItemComments
	err := stmt.Query(repo.db, &created)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return &created, nil
}

func (repo *ItemCommentsRepositoryImpl) UpdateCommentText(commentID uuid.UUID, text string) (*model.ItemComments, error) {
	now := time.Now()

	stmt := table.ItemComments.UPDATE().
		SET(
			table.ItemComments.Text.SET(String(text)),
			table.ItemComments.UpdatedAt.SET(TimestampT(now)),
		).
		WHERE(table.ItemComments.ID.EQ(UUID(commentID))).
		RETURNING(table.ItemComments.AllColumns)

	var updated model.ItemComments
	err := stmt.Query(repo.db, &updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return &updated, nil
}

func (repo *ItemCommentsRepositoryImpl) FlagComment(flag model.ItemCommentFlags) error {
	stmt := table.ItemCommentFlags.INSERT(
		table.ItemCommentFlags.CommentID,
		table.ItemCommentFlags.FlaggerID,
	).VALUES(
		flag.CommentID,
		flag.FlaggerID,
	).ON_CONFLICT(
		table.ItemCommentFlags.CommentID,
		table.ItemCommentFlags.FlaggerID,
	).DO_NOTHING()

	_, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to flag comment: %w", err)
	}

	return nil
}
