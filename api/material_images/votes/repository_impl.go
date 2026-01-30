package votes

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Upsert(vote model.MaterialImagesVotes) error {
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

func (r *RepositoryImpl) Delete(imageID string, userID string) error {
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

func (r *RepositoryImpl) GetUserVote(imageID string, userID string) (*model.MaterialImagesVotes, error) {
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

func (r *RepositoryImpl) UpdateImageCounts(imageID string) error {
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
