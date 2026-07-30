package community

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/persistence"
	"miltechserver/api/user_pmcs/shared"
)

type Repository interface {
	Release(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		revisionID uuid.UUID,
		precondition shared.Precondition,
	) (*ReleaseMutationResult, error)
	Retire(
		ctx context.Context,
		ownerUID string,
		checklistID uuid.UUID,
		precondition shared.Precondition,
	) (*ReleaseMutationResult, error)
	Browse(
		ctx context.Context,
		filter shared.CommunityBrowseFilter,
	) (*shared.CommunityPage, error)
	GetCurrentRelease(
		ctx context.Context,
		checklistID uuid.UUID,
	) (*shared.PublicChecklistRelease, error)
}

type ReleaseMutationResult struct {
	Aggregate  shared.ChecklistAggregate
	Idempotent bool
}

type lockedChecklist struct {
	ownerUID    *string
	syncVersion int64
	deletedAt   *time.Time
}

type lockedSource struct {
	status               string
	currentRevisionID    *uuid.UUID
	latestRevisionNumber int32
}

type lockedRevision struct {
	state          string
	revisionNumber *int32
	contentHash    []byte
}

func lockChecklist(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) (lockedChecklist, bool, error) {
	var checklist lockedChecklist
	err := tx.QueryRowContext(
		ctx,
		`SELECT owner_uid, sync_version, deleted_at
		 FROM user_pmcs_checklists
		 WHERE id = $1
		 FOR UPDATE`,
		checklistID,
	).Scan(
		&checklist.ownerUID,
		&checklist.syncVersion,
		&checklist.deletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedChecklist{}, false, nil
	}
	if err != nil {
		return lockedChecklist{}, false, fmt.Errorf("lock community checklist: %w", err)
	}
	return checklist, true, nil
}

func lockSource(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
) (lockedSource, bool, error) {
	var (
		source          lockedSource
		currentRevision uuid.NullUUID
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT status, current_release_revision_id,
		        latest_release_revision_number
		 FROM user_pmcs_community_sources
		 WHERE checklist_id = $1
		 FOR UPDATE`,
		checklistID,
	).Scan(
		&source.status,
		&currentRevision,
		&source.latestRevisionNumber,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedSource{}, false, nil
	}
	if err != nil {
		return lockedSource{}, false, fmt.Errorf("lock community source: %w", err)
	}
	if currentRevision.Valid {
		revisionID := currentRevision.UUID
		source.currentRevisionID = &revisionID
	}
	return source, true, nil
}

func lockRevision(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
	revisionID uuid.UUID,
) (lockedRevision, bool, error) {
	var revision lockedRevision
	var revisionNumber sql.NullInt32
	err := tx.QueryRowContext(
		ctx,
		`SELECT state, revision_number, content_hash
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1 AND id = $2
		 FOR UPDATE`,
		checklistID,
		revisionID,
	).Scan(
		&revision.state,
		&revisionNumber,
		&revision.contentHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedRevision{}, false, nil
	}
	if err != nil {
		return lockedRevision{}, false, fmt.Errorf("lock community release revision: %w", err)
	}
	if revisionNumber.Valid {
		value := revisionNumber.Int32
		revision.revisionNumber = &value
	}
	return revision, true, nil
}

func validateImmutableRevision(
	ctx context.Context,
	tx *sql.Tx,
	revisionID uuid.UUID,
	locked lockedRevision,
	config shared.Config,
) error {
	revisions, err := persistence.LoadRevisionTrees(
		ctx,
		tx,
		[]uuid.UUID{revisionID},
	)
	if err != nil {
		return err
	}
	revision, found := revisions[revisionID]
	if !found {
		return fmt.Errorf("community release revision disappeared")
	}
	prepared, err := shared.PreparePublication(
		revisionInput(revision),
		config,
	)
	if err != nil {
		return err
	}
	if len(locked.contentHash) != sha256.Size ||
		!bytes.Equal(locked.contentHash, prepared.Hash[:]) {
		return fmt.Errorf("community release revision content hash mismatch")
	}
	return nil
}

func revisionInput(revision shared.Revision) shared.RevisionInput {
	input := shared.RevisionInput{
		ID:             revision.ID,
		RevisionNumber: revision.RevisionNumber,
		Name:           revision.Name,
		Description:    revision.Description,
		Models:         make([]shared.ModelInput, len(revision.Models)),
		Sections:       make([]shared.SectionInput, len(revision.Sections)),
	}
	for index, model := range revision.Models {
		input.Models[index] = shared.ModelInput{
			DisplayText: model.DisplayText,
		}
	}
	for sectionIndex, section := range revision.Sections {
		inputSection := shared.SectionInput{
			ID:       section.ID,
			Position: section.Position,
			Title:    section.Title,
			Models:   make([]shared.ModelInput, len(section.Models)),
			Items:    make([]shared.ItemInput, len(section.Items)),
		}
		for modelIndex, model := range section.Models {
			inputSection.Models[modelIndex] = shared.ModelInput{
				DisplayText: model.DisplayText,
			}
		}
		for itemIndex, item := range section.Items {
			inputSection.Items[itemIndex] = shared.ItemInput{
				ID:                        item.ID,
				Position:                  item.Position,
				Interval:                  item.Interval,
				ItemToBeCheckedOrServiced: item.ItemToBeCheckedOrServiced,
				PerformedBy:               item.PerformedBy,
				Notices:                   item.Notices,
				ProcedureSteps:            item.ProcedureSteps,
			}
		}
		input.Sections[sectionIndex] = inputSection
	}
	return input
}

func loadCurrentRevisions(
	ctx context.Context,
	queryer persistence.Queryer,
	checklistID uuid.UUID,
	aggregate *shared.ChecklistAggregate,
) error {
	rows, err := queryer.QueryContext(
		ctx,
		`SELECT id, state
		 FROM user_pmcs_revisions
		 WHERE checklist_id = $1 AND state IN ('draft', 'published')
		 ORDER BY state, id`,
		checklistID,
	)
	if err != nil {
		return fmt.Errorf("query released checklist revisions: %w", err)
	}
	var revisionIDs []uuid.UUID
	revisionStates := make(map[uuid.UUID]string)
	for rows.Next() {
		var revisionID uuid.UUID
		var state string
		if err := rows.Scan(&revisionID, &state); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan released checklist revision: %w", err)
		}
		revisionIDs = append(revisionIDs, revisionID)
		revisionStates[revisionID] = state
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate released checklist revisions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close released checklist revisions: %w", err)
	}

	revisions, err := persistence.LoadRevisionTrees(ctx, queryer, revisionIDs)
	if err != nil {
		return err
	}
	for revisionID, state := range revisionStates {
		revision := revisions[revisionID]
		if state == "draft" {
			aggregate.Draft = &revision
		} else {
			aggregate.Publication = &revision
		}
	}
	return nil
}

func loadSourceSummary(
	ctx context.Context,
	queryer persistence.Queryer,
	checklistID uuid.UUID,
	aggregate *shared.ChecklistAggregate,
) error {
	var (
		summary         shared.CommunitySourceSummary
		currentRevision uuid.NullUUID
		retiredAt       sql.NullTime
	)
	err := queryer.QueryRowContext(
		ctx,
		`SELECT status, current_release_revision_id,
		        latest_release_revision_number, first_released_at,
		        updated_at, retired_at
		 FROM user_pmcs_community_sources
		 WHERE checklist_id = $1`,
		checklistID,
	).Scan(
		&summary.Status,
		&currentRevision,
		&summary.LatestReleaseRevisionNumber,
		&summary.FirstReleasedAt,
		&summary.UpdatedAt,
		&retiredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load community source summary: %w", err)
	}
	if currentRevision.Valid {
		revisionID := currentRevision.UUID
		summary.CurrentReleaseRevisionID = &revisionID
	}
	if retiredAt.Valid {
		retired := retiredAt.Time
		summary.RetiredAt = &retired
	}
	aggregate.Community = &summary
	return nil
}

func hiddenChecklistError() *shared.APIError {
	return shared.NewResourceNotFound("checklist not found", nil)
}

func staleChecklistError() *shared.APIError {
	return shared.NewStalePrecondition("resource changed", nil)
}

func invalidReleaseTransition(message string) *shared.APIError {
	return shared.NewInvalidTransition(message, nil)
}
