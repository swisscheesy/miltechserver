package community

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type RepositoryImpl struct {
	store  persistence.Store
	config shared.Config
}

func NewRepository(store persistence.Store, config shared.Config) Repository {
	return &RepositoryImpl{store: store, config: config}
}

func (repository *RepositoryImpl) Release(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*ReleaseMutationResult, error) {
			if _, err := persistence.LockAccountVersion(
				ctx,
				tx,
				ownerUID,
			); err != nil {
				return nil, err
			}

			checklist, found, err := lockChecklist(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !found || checklist.ownerUID == nil ||
				*checklist.ownerUID != ownerUID {
				return nil, hiddenChecklistError()
			}
			if checklist.deletedAt != nil {
				return nil, staleChecklistError()
			}

			source, sourceExists, err := lockSource(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			revision, revisionExists, err := lockRevision(
				ctx,
				tx,
				checklistID,
				revisionID,
			)
			if err != nil {
				return nil, err
			}
			if !revisionExists ||
				(revision.state != "published" &&
					revision.state != "superseded") ||
				revision.revisionNumber == nil {
				return nil, invalidReleaseTransition(
					"community release requires a published revision",
				)
			}
			if err := validateImmutableRevision(
				ctx,
				tx,
				revisionID,
				revision,
				repository.config,
			); err != nil {
				return nil, err
			}

			if sourceExists &&
				source.status == "active" &&
				source.currentRevisionID != nil &&
				*source.currentRevisionID == revisionID {
				aggregate, err := loadAggregate(
					ctx,
					tx,
					ownerUID,
					checklistID,
				)
				if err != nil {
					return nil, err
				}
				return &ReleaseMutationResult{
					Aggregate:  *aggregate,
					Idempotent: true,
				}, nil
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}
			if sourceExists &&
				*revision.revisionNumber <= source.latestRevisionNumber {
				return nil, invalidReleaseTransition(
					"community release revision number must increase",
				)
			}

			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO user_pmcs_community_releases
				     (revision_id, checklist_id)
				 VALUES ($1, $2)`,
				revisionID,
				checklistID,
			); err != nil {
				return nil, fmt.Errorf("insert community release: %w", err)
			}
			if sourceExists {
				if _, err := tx.ExecContext(
					ctx,
					`UPDATE user_pmcs_community_sources
					 SET current_release_revision_id = $1,
					     status = 'active',
					     latest_release_revision_number = $2,
					     updated_at = now(),
					     retired_at = NULL
					 WHERE checklist_id = $3`,
					revisionID,
					*revision.revisionNumber,
					checklistID,
				); err != nil {
					return nil, fmt.Errorf("advance community source: %w", err)
				}
			} else if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO user_pmcs_community_sources
				     (checklist_id, status, current_release_revision_id,
				      latest_release_revision_number, first_released_at,
				      updated_at, retired_at)
				 VALUES ($1, 'active', $2, $3, now(), now(), NULL)`,
				checklistID,
				revisionID,
				*revision.revisionNumber,
			); err != nil {
				return nil, fmt.Errorf("insert community source: %w", err)
			}

			if err := advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &ReleaseMutationResult{Aggregate: *aggregate}, nil
		},
	)
}

func (repository *RepositoryImpl) Retire(
	ctx context.Context,
	ownerUID string,
	checklistID uuid.UUID,
	precondition shared.Precondition,
) (*ReleaseMutationResult, error) {
	return persistence.WithWriteTx(
		ctx,
		repository.store.DB,
		repository.store.MaxWriteAttempts,
		func(tx *sql.Tx) (*ReleaseMutationResult, error) {
			if _, err := persistence.LockAccountVersion(
				ctx,
				tx,
				ownerUID,
			); err != nil {
				return nil, err
			}
			checklist, found, err := lockChecklist(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !found || checklist.ownerUID == nil ||
				*checklist.ownerUID != ownerUID {
				return nil, hiddenChecklistError()
			}
			if checklist.deletedAt != nil {
				return nil, staleChecklistError()
			}
			source, sourceExists, err := lockSource(ctx, tx, checklistID)
			if err != nil {
				return nil, err
			}
			if !sourceExists {
				return nil, invalidReleaseTransition(
					"community source has not been released",
				)
			}
			if source.status == "retired" {
				aggregate, err := loadAggregate(
					ctx,
					tx,
					ownerUID,
					checklistID,
				)
				if err != nil {
					return nil, err
				}
				return &ReleaseMutationResult{
					Aggregate:  *aggregate,
					Idempotent: true,
				}, nil
			}
			if !precondition.Matches(shared.MakeChecklistETag(
				checklistID,
				checklist.syncVersion,
			)) {
				return nil, staleChecklistError()
			}
			if _, err := tx.ExecContext(
				ctx,
				`UPDATE user_pmcs_community_sources
				 SET current_release_revision_id = NULL,
				     status = 'retired',
				     updated_at = now(),
				     retired_at = now()
				 WHERE checklist_id = $1`,
				checklistID,
			); err != nil {
				return nil, fmt.Errorf("retire community source: %w", err)
			}
			if err := advanceChecklist(
				ctx,
				tx,
				ownerUID,
				checklistID,
			); err != nil {
				return nil, err
			}
			aggregate, err := loadAggregate(
				ctx,
				tx,
				ownerUID,
				checklistID,
			)
			if err != nil {
				return nil, err
			}
			return &ReleaseMutationResult{Aggregate: *aggregate}, nil
		},
	)
}

func (repository *RepositoryImpl) Browse(
	ctx context.Context,
	filter shared.CommunityBrowseFilter,
) (*shared.CommunityPage, error) {
	startedAt := time.Now()
	defer func() {
		shared.RecordDBDuration(ctx, time.Since(startedAt))
	}()

	query, arguments := communityBrowseQuery(filter)
	rows, err := repository.store.DB.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("browse active community sources: %w", err)
	}
	defer rows.Close()

	items := make([]shared.CommunitySummary, 0, filter.Limit+1)
	for rows.Next() {
		var (
			item     shared.CommunitySummary
			username sql.NullString
			myVote   sql.NullInt16
		)
		if err := rows.Scan(
			&item.ChecklistID,
			&item.RevisionID,
			&item.RevisionNumber,
			&item.Name,
			&item.Description,
			&username,
			&item.ReleasedAt,
			&item.UpdatedAt,
			&item.Score,
			&myVote,
			&item.CanVote,
		); err != nil {
			return nil, fmt.Errorf("scan community summary: %w", err)
		}
		item.Models = []shared.ModelValue{}
		item.CreatorDisplayName = creatorDisplayName(username)
		if myVote.Valid {
			vote := int16(myVote.Int16)
			item.MyVote = &vote
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate community summaries: %w", err)
	}

	hasMore := len(items) > filter.Limit
	if hasMore {
		items = items[:filter.Limit]
	}
	if err := loadSummaryModels(ctx, repository.store.DB, items); err != nil {
		return nil, err
	}
	page := &shared.CommunityPage{
		HasMore: hasMore,
		Items:   items,
	}
	if hasMore {
		last := items[len(items)-1]
		cursorValue := shared.CommunityCursor{
			Version:   2,
			Sort:      communityBrowseSort(filter),
			UpdatedAt: last.UpdatedAt,
			Checklist: last.ChecklistID,
		}
		if cursorValue.Sort == shared.CommunitySortTop {
			score := last.Score
			cursorValue.Score = &score
		}
		cursor, err := shared.EncodeCommunityCursor(cursorValue)
		if err != nil {
			return nil, fmt.Errorf("encode community cursor: %w", err)
		}
		page.NextCursor = &cursor
	}
	return page, nil
}

func containsModelPattern(normalizedModel string) string {
	escaped := strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	).Replace(normalizedModel)
	return "%" + escaped + "%"
}

func communityBrowseQuery(
	filter shared.CommunityBrowseFilter,
) (string, []any) {
	sort := communityBrowseSort(filter)
	arguments := make([]any, 0, 6)
	viewerSelect := `NULL::SMALLINT AS my_vote,
	                 FALSE AS can_vote`
	viewerJoin := ""
	if filter.ViewerUID != "" {
		arguments = append(arguments, filter.ViewerUID)
		viewerSelect = fmt.Sprintf(
			`viewer_vote.direction AS my_vote,
		     checklist.owner_uid <> $%d AS can_vote`,
			len(arguments),
		)
		viewerJoin = fmt.Sprintf(
			`LEFT JOIN user_pmcs_community_votes AS viewer_vote
		        ON viewer_vote.checklist_id = source.checklist_id
		       AND viewer_vote.voter_uid = $%d`,
			len(arguments),
		)
	}

	query := fmt.Sprintf(
		`WITH ranked AS (
		    SELECT source.checklist_id, revision.id AS revision_id,
		           revision.revision_number, revision.name,
		           revision.description, owner.username,
		           release.released_at, source.updated_at,
		           vote_total.score,
		           %s
		    FROM user_pmcs_community_sources AS source
		    JOIN user_pmcs_community_releases AS release
		      ON release.checklist_id = source.checklist_id
		     AND release.revision_id = source.current_release_revision_id
		    JOIN user_pmcs_revisions AS revision
		      ON revision.checklist_id = source.checklist_id
		     AND revision.id = source.current_release_revision_id
		    JOIN user_pmcs_checklists AS checklist
		      ON checklist.id = source.checklist_id
		    LEFT JOIN users AS owner
		      ON owner.uid = checklist.owner_uid
		    LEFT JOIN LATERAL (
		        SELECT COALESCE(SUM(vote.direction), 0)::BIGINT AS score
		        FROM user_pmcs_community_votes AS vote
		        WHERE vote.checklist_id = source.checklist_id
		    ) AS vote_total ON TRUE
		    %s
		    WHERE source.status = 'active'
		      AND checklist.deleted_at IS NULL`,
		viewerSelect,
		viewerJoin,
	)
	cursorArgumentStart := 0
	if filter.After != nil {
		cursorArgumentStart = len(arguments) + 1
		if sort == shared.CommunitySortTop {
			arguments = append(
				arguments,
				*filter.After.Score,
				filter.After.UpdatedAt,
				filter.After.Checklist,
			)
		} else {
			arguments = append(
				arguments,
				filter.After.UpdatedAt,
				filter.After.Checklist,
			)
		}
	}
	if filter.NormalizedModel != "" {
		arguments = append(
			arguments,
			containsModelPattern(filter.NormalizedModel),
		)
		query += fmt.Sprintf(
			` AND EXISTS (
			      SELECT 1
			      FROM user_pmcs_revision_models AS model
			      WHERE model.normalized_text LIKE $%d ESCAPE '!'
			        AND model.revision_id =
			            source.current_release_revision_id
			  )`,
			len(arguments),
		)
	}
	query += `
	)
	SELECT ranked.checklist_id, ranked.revision_id,
	       ranked.revision_number, ranked.name,
	       ranked.description, ranked.username,
	       ranked.released_at, ranked.updated_at,
	       ranked.score, ranked.my_vote, ranked.can_vote
	FROM ranked`
	if filter.After != nil {
		if sort == shared.CommunitySortTop {
			query += fmt.Sprintf(
				` WHERE ranked.score < $%d
			     OR (ranked.score = $%d AND ranked.updated_at < $%d)
			     OR (ranked.score = $%d AND ranked.updated_at = $%d
			         AND ranked.checklist_id > $%d)`,
				cursorArgumentStart,
				cursorArgumentStart,
				cursorArgumentStart+1,
				cursorArgumentStart,
				cursorArgumentStart+1,
				cursorArgumentStart+2,
			)
		} else {
			query += fmt.Sprintf(
				` WHERE ranked.updated_at < $%d
			     OR (ranked.updated_at = $%d
			         AND ranked.checklist_id > $%d)`,
				cursorArgumentStart,
				cursorArgumentStart,
				cursorArgumentStart+1,
			)
		}
	}
	arguments = append(arguments, filter.Limit+1)
	if sort == shared.CommunitySortTop {
		query += fmt.Sprintf(
			` ORDER BY ranked.score DESC, ranked.updated_at DESC, ranked.checklist_id ASC
		      LIMIT $%d`,
			len(arguments),
		)
	} else {
		query += fmt.Sprintf(
			` ORDER BY ranked.updated_at DESC, ranked.checklist_id ASC
		      LIMIT $%d`,
			len(arguments),
		)
	}
	return query, arguments
}

func communityBrowseSort(
	filter shared.CommunityBrowseFilter,
) shared.CommunitySort {
	if filter.Sort == "" {
		return shared.CommunitySortTop
	}
	return filter.Sort
}

func loadSummaryModels(
	ctx context.Context,
	queryer persistence.Queryer,
	items []shared.CommunitySummary,
) error {
	if len(items) == 0 {
		return nil
	}
	revisionIDs := make([]uuid.UUID, len(items))
	itemIndexes := make(map[uuid.UUID]int, len(items))
	for index := range items {
		revisionIDs[index] = items[index].RevisionID
		itemIndexes[items[index].RevisionID] = index
	}
	rows, err := queryer.QueryContext(
		ctx,
		`SELECT revision_id, display_text, normalized_text
		 FROM user_pmcs_revision_models
		 WHERE revision_id = ANY($1)
		 ORDER BY revision_id, normalized_text`,
		pq.Array(revisionIDs),
	)
	if err != nil {
		return fmt.Errorf("load community summary models: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			revisionID uuid.UUID
			model      shared.ModelValue
		)
		if err := rows.Scan(
			&revisionID,
			&model.DisplayText,
			&model.NormalizedText,
		); err != nil {
			return fmt.Errorf("scan community summary model: %w", err)
		}
		index, exists := itemIndexes[revisionID]
		if exists {
			items[index].Models = append(items[index].Models, model)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate community summary models: %w", err)
	}
	return nil
}

func (repository *RepositoryImpl) GetCurrentRelease(
	ctx context.Context,
	checklistID uuid.UUID,
) (*shared.PublicChecklistRelease, error) {
	startedAt := time.Now()
	defer func() {
		shared.RecordDBDuration(ctx, time.Since(startedAt))
	}()

	var (
		release     shared.PublicChecklistRelease
		revisionID  uuid.UUID
		contentHash []byte
		username    sql.NullString
	)
	err := repository.store.DB.QueryRowContext(
		ctx,
		`SELECT source.checklist_id, source.current_release_revision_id,
		        release.released_at, revision.content_hash, owner.username
		 FROM user_pmcs_community_sources AS source
		 JOIN user_pmcs_community_releases AS release
		   ON release.checklist_id = source.checklist_id
		  AND release.revision_id = source.current_release_revision_id
		 JOIN user_pmcs_revisions AS revision
		   ON revision.checklist_id = source.checklist_id
		  AND revision.id = source.current_release_revision_id
		 JOIN user_pmcs_checklists AS checklist
		   ON checklist.id = source.checklist_id
		 LEFT JOIN users AS owner
		   ON owner.uid = checklist.owner_uid
		 WHERE source.checklist_id = $1
		   AND source.status = 'active'
		   AND checklist.deleted_at IS NULL`,
		checklistID,
	).Scan(
		&release.ChecklistID,
		&revisionID,
		&release.ReleasedAt,
		&contentHash,
		&username,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, communityNotFoundError()
	}
	if err != nil {
		return nil, fmt.Errorf("load current community release: %w", err)
	}

	revisions, err := persistence.LoadRevisionTrees(
		ctx,
		repository.store.DB,
		[]uuid.UUID{revisionID},
	)
	if err != nil {
		return nil, err
	}
	revision, found := revisions[revisionID]
	if !found {
		return nil, fmt.Errorf("current community release revision disappeared")
	}
	canonicalHash, err := shared.CanonicalRevisionHash(revisionInput(revision))
	if err != nil {
		return nil, err
	}
	if len(contentHash) != sha256.Size ||
		!bytes.Equal(contentHash, canonicalHash[:]) {
		return nil, fmt.Errorf("current community release content hash mismatch")
	}
	release.CreatorDisplayName = creatorDisplayName(username)
	release.Revision = revision
	return &release, nil
}

func creatorDisplayName(username sql.NullString) string {
	if !username.Valid {
		return "Deleted user"
	}
	return username.String
}

func communityNotFoundError() *shared.APIError {
	return shared.NewResourceNotFound("community checklist not found", nil)
}

func advanceChecklist(
	ctx context.Context,
	tx *sql.Tx,
	ownerUID string,
	checklistID uuid.UUID,
) error {
	accountVersion, err := persistence.AdvanceAccountVersion(
		ctx,
		tx,
		ownerUID,
	)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE user_pmcs_checklists
		 SET sync_version = sync_version + 1,
		     account_change_version = $1,
		     updated_at = now()
		 WHERE id = $2`,
		accountVersion,
		checklistID,
	); err != nil {
		return fmt.Errorf("advance released checklist: %w", err)
	}
	return nil
}

func loadAggregate(
	ctx context.Context,
	queryer persistence.Queryer,
	ownerUID string,
	checklistID uuid.UUID,
) (*shared.ChecklistAggregate, error) {
	var aggregate shared.ChecklistAggregate
	err := queryer.QueryRowContext(
		ctx,
		`SELECT id, sync_version, account_change_version,
		        created_at, updated_at, deleted_at
		 FROM user_pmcs_checklists
		 WHERE id = $1 AND owner_uid = $2`,
		checklistID,
		ownerUID,
	).Scan(
		&aggregate.ID,
		&aggregate.SyncVersion,
		&aggregate.AccountChangeVersion,
		&aggregate.CreatedAt,
		&aggregate.UpdatedAt,
		&aggregate.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, hiddenChecklistError()
	}
	if err != nil {
		return nil, fmt.Errorf("load released checklist root: %w", err)
	}
	if aggregate.DeletedAt != nil {
		return &aggregate, nil
	}

	if err := loadCurrentRevisions(
		ctx,
		queryer,
		checklistID,
		&aggregate,
	); err != nil {
		return nil, err
	}
	if err := loadSourceSummary(
		ctx,
		queryer,
		checklistID,
		&aggregate,
	); err != nil {
		return nil, err
	}
	return &aggregate, nil
}
