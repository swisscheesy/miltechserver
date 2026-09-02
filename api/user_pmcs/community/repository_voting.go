package community

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

func (repository *RepositoryImpl) PutVote(
	ctx context.Context,
	voterUID string,
	checklistID uuid.UUID,
	direction int16,
) (*shared.CommunityVoteMutation, error) {
	if !isCommunityVoteDirection(direction) {
		return nil, shared.NewInvalidRequest(
			"community vote direction must be 1 or -1",
			nil,
		)
	}

	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*shared.CommunityVoteMutation, error) {
			if err := requireCommunityVoteAccount(ctx, tx, voterUID); err != nil {
				return nil, err
			}
			ownerUID, found, err := lockCommunityVoteSource(
				ctx,
				tx,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, communityVoteNotFoundError()
			}
			if ownerUID.Valid && ownerUID.String == voterUID {
				return nil, shared.NewForbidden(
					"owners cannot vote on their own checklist",
					nil,
				)
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO user_pmcs_community_votes (
				     checklist_id, voter_uid, direction
				 ) VALUES ($1, $2, $3)
				 ON CONFLICT (checklist_id, voter_uid) DO UPDATE
				 SET direction = EXCLUDED.direction,
				     updated_at = now()`,
				checklistID,
				voterUID,
				direction,
			); err != nil {
				return nil, fmt.Errorf("upsert community vote: %w", err)
			}
			return loadCommunityVoteMutation(ctx, tx, checklistID, voterUID)
		},
	)
}

func (repository *RepositoryImpl) DeleteVote(
	ctx context.Context,
	voterUID string,
	checklistID uuid.UUID,
) (*shared.CommunityVoteMutation, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*shared.CommunityVoteMutation, error) {
			if err := requireCommunityVoteAccount(ctx, tx, voterUID); err != nil {
				return nil, err
			}
			ownerUID, found, err := lockCommunityVoteSource(
				ctx,
				tx,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, communityVoteNotFoundError()
			}
			if ownerUID.Valid && ownerUID.String == voterUID {
				return nil, shared.NewForbidden(
					"owners cannot vote on their own checklist",
					nil,
				)
			}
			if _, err := tx.ExecContext(
				ctx,
				`DELETE FROM user_pmcs_community_votes
				 WHERE checklist_id = $1 AND voter_uid = $2`,
				checklistID,
				voterUID,
			); err != nil {
				return nil, fmt.Errorf("delete community vote: %w", err)
			}
			return loadCommunityVoteMutation(ctx, tx, checklistID, voterUID)
		},
	)
}

func isCommunityVoteDirection(direction int16) bool {
	return direction == 1 || direction == -1
}

func requireCommunityVoteAccount(
	ctx context.Context,
	queryer persistence.Queryer,
	voterUID string,
) error {
	var exists bool
	if err := queryer.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE uid = $1)`,
		voterUID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("verify community voter account: %w", err)
	}
	if !exists {
		return shared.NewAccountNotInitialized("account is not initialized", nil)
	}
	return nil
}

func lockCommunityVoteSource(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) (sql.NullString, bool, error) {
	var ownerUID sql.NullString
	err := tx.QueryRowContext(
		ctx,
		`SELECT checklist.owner_uid
		 FROM user_pmcs_community_sources AS source
		 JOIN user_pmcs_checklists AS checklist ON checklist.id = source.checklist_id
		 WHERE source.checklist_id = $1
		   AND source.status = 'active'
		   AND checklist.deleted_at IS NULL
		 FOR SHARE OF source, checklist`,
		checklistID,
	).Scan(&ownerUID)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullString{}, false, nil
	}
	if err != nil {
		return sql.NullString{}, false, fmt.Errorf("lock community vote source: %w", err)
	}
	return ownerUID, true, nil
}

func loadCommunityVoteMutation(
	ctx context.Context,
	queryer persistence.Queryer,
	checklistID uuid.UUID,
	voterUID string,
) (*shared.CommunityVoteMutation, error) {
	mutation := shared.CommunityVoteMutation{ChecklistID: checklistID}
	var myVote sql.NullInt16
	if err := queryer.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(vote.direction), 0)::BIGINT,
		        (SELECT direction
		         FROM user_pmcs_community_votes
		         WHERE checklist_id = $1 AND voter_uid = $2)
		 FROM user_pmcs_community_votes AS vote
		 WHERE vote.checklist_id = $1`,
		checklistID,
		voterUID,
	).Scan(&mutation.Score, &myVote); err != nil {
		return nil, fmt.Errorf("load community vote mutation: %w", err)
	}
	if myVote.Valid {
		direction := int16(myVote.Int16)
		mutation.MyVote = &direction
	}
	return &mutation, nil
}

func communityVoteNotFoundError() *shared.APIError {
	return shared.NewResourceNotFound("community checklist not found", nil)
}
