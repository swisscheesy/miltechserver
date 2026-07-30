package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
)

type submittedNode struct {
	nodeType    string
	checklistID uuid.UUID
	revisionID  uuid.UUID
	parentID    uuid.UUID
}

func ReplaceDraftTree(
	ctx context.Context,
	tx *sql.Tx,
	checklistID uuid.UUID,
	prepared shared.PreparedRevision,
) error {
	submittedNodes, err := collectSubmittedNodes(checklistID, prepared.Input)
	if err != nil {
		return err
	}
	if err := preflightSubmittedNodeOwnership(
		ctx,
		tx,
		submittedNodes,
	); err != nil {
		return err
	}

	var revisionID uuid.UUID
	err = tx.QueryRowContext(
		ctx,
		`UPDATE user_pmcs_revisions
		 SET name = $1,
		     description = $2,
		     content_hash = $3,
		     updated_at = now()
		 WHERE id = $4
		   AND checklist_id = $5
		   AND state = 'draft'
		 RETURNING id`,
		prepared.Input.Name,
		prepared.Input.Description,
		prepared.Hash[:],
		prepared.Input.ID,
		checklistID,
	).Scan(&revisionID)
	if errors.Is(err, sql.ErrNoRows) {
		return shared.NewInvalidTransition(
			"revision is not the checklist's mutable draft",
			map[string]any{"revision_id": prepared.Input.ID},
		)
	}
	if err != nil {
		return fmt.Errorf("update draft revision: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_revision_models WHERE revision_id = $1`,
		revisionID,
	); err != nil {
		return fmt.Errorf("delete draft revision models: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM user_pmcs_sections WHERE revision_id = $1`,
		revisionID,
	); err != nil {
		return fmt.Errorf("delete draft sections: %w", err)
	}

	if err := insertPreparedTree(ctx, tx, prepared.Input); err != nil {
		return err
	}
	return nil
}

func DeleteRevisionTrees(
	ctx context.Context,
	tx *sql.Tx,
	revisionIDs []uuid.UUID,
) error {
	for start := 0; start < len(revisionIDs); start += maxBindParameters {
		end := min(start+maxBindParameters, len(revisionIDs))
		query, arguments := inStatement(
			"DELETE FROM user_pmcs_revisions WHERE id IN (",
			revisionIDs[start:end],
		)
		if _, err := tx.ExecContext(ctx, query, arguments...); err != nil {
			return fmt.Errorf("delete revision trees: %w", err)
		}
	}
	return nil
}

func collectSubmittedNodes(
	checklistID uuid.UUID,
	revision shared.RevisionInput,
) (map[uuid.UUID]submittedNode, error) {
	nodes := make(map[uuid.UUID]submittedNode)
	addNode := func(id uuid.UUID, node submittedNode) error {
		if id == uuid.Nil {
			return shared.NewValidationFailed(
				"revision tree contains a zero UUID",
				nil,
			)
		}
		if _, duplicate := nodes[id]; duplicate {
			return shared.NewValidationFailed(
				"revision tree contains a duplicate UUID",
				map[string]any{"id": id},
			)
		}
		nodes[id] = node
		return nil
	}

	if err := addNode(revision.ID, submittedNode{
		nodeType:    "revision",
		checklistID: checklistID,
		revisionID:  revision.ID,
	}); err != nil {
		return nil, err
	}
	for _, section := range revision.Sections {
		if err := addNode(section.ID, submittedNode{
			nodeType:    "section",
			checklistID: checklistID,
			revisionID:  revision.ID,
			parentID:    revision.ID,
		}); err != nil {
			return nil, err
		}
		for _, item := range section.Items {
			if err := addNode(item.ID, submittedNode{
				nodeType:    "item",
				checklistID: checklistID,
				revisionID:  revision.ID,
				parentID:    section.ID,
			}); err != nil {
				return nil, err
			}
			for _, notice := range item.Notices {
				if err := addNode(notice.ID, submittedNode{
					nodeType:    "notice",
					checklistID: checklistID,
					revisionID:  revision.ID,
					parentID:    item.ID,
				}); err != nil {
					return nil, err
				}
			}
			for _, step := range item.ProcedureSteps {
				if err := addNode(step.ID, submittedNode{
					nodeType:    "step",
					checklistID: checklistID,
					revisionID:  revision.ID,
					parentID:    item.ID,
				}); err != nil {
					return nil, err
				}
			}
		}
	}
	return nodes, nil
}

func preflightSubmittedNodeOwnership(
	ctx context.Context,
	tx *sql.Tx,
	nodes map[uuid.UUID]submittedNode,
) error {
	nodeIDs := make([]uuid.UUID, 0, len(nodes))
	for nodeID := range nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}

	for start := 0; start < len(nodeIDs); start += maxBindParameters {
		end := min(start+maxBindParameters, len(nodeIDs))
		placeholders, arguments := uuidPlaceholders(nodeIDs[start:end])
		query := ownershipQuery(placeholders)
		rows, err := tx.QueryContext(ctx, query, arguments...)
		if err != nil {
			return fmt.Errorf("query submitted UUID ownership: %w", err)
		}
		for rows.Next() {
			var (
				nodeType    string
				nodeID      uuid.UUID
				checklistID uuid.UUID
				revisionID  uuid.UUID
				parentID    *uuid.UUID
			)
			if err := rows.Scan(
				&nodeType,
				&nodeID,
				&checklistID,
				&revisionID,
				&parentID,
			); err != nil {
				return closeRowsWithError(
					rows,
					fmt.Errorf("scan submitted UUID ownership: %w", err),
					"submitted UUID ownership",
				)
			}
			expected := nodes[nodeID]
			actualParent := uuid.Nil
			if parentID != nil {
				actualParent = *parentID
			}
			if nodeType != expected.nodeType ||
				checklistID != expected.checklistID ||
				revisionID != expected.revisionID ||
				actualParent != expected.parentID {
				return closeRowsWithError(
					rows,
					shared.NewValidationFailed(
						"revision tree UUID belongs to another resource",
						map[string]any{"id": nodeID},
					),
					"submitted UUID ownership",
				)
			}
		}
		if err := rows.Err(); err != nil {
			return closeRowsWithError(
				rows,
				fmt.Errorf("iterate submitted UUID ownership: %w", err),
				"submitted UUID ownership",
			)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close submitted UUID ownership rows: %w", err)
		}
	}
	return nil
}

func ownershipQuery(placeholders string) string {
	return `
		SELECT 'revision', r.id, r.checklist_id, r.id, NULL::uuid
		FROM user_pmcs_revisions r
		WHERE r.id IN (` + placeholders + `)
		UNION ALL
		SELECT 'section', s.id, r.checklist_id, s.revision_id, s.revision_id
		FROM user_pmcs_sections s
		JOIN user_pmcs_revisions r ON r.id = s.revision_id
		WHERE s.id IN (` + placeholders + `)
		UNION ALL
		SELECT 'item', i.id, r.checklist_id, s.revision_id, i.section_id
		FROM user_pmcs_items i
		JOIN user_pmcs_sections s ON s.id = i.section_id
		JOIN user_pmcs_revisions r ON r.id = s.revision_id
		WHERE i.id IN (` + placeholders + `)
		UNION ALL
		SELECT 'notice', n.id, r.checklist_id, s.revision_id, n.item_id
		FROM user_pmcs_notices n
		JOIN user_pmcs_items i ON i.id = n.item_id
		JOIN user_pmcs_sections s ON s.id = i.section_id
		JOIN user_pmcs_revisions r ON r.id = s.revision_id
		WHERE n.id IN (` + placeholders + `)
		UNION ALL
		SELECT 'step', p.id, r.checklist_id, s.revision_id, p.item_id
		FROM user_pmcs_procedure_steps p
		JOIN user_pmcs_items i ON i.id = p.item_id
		JOIN user_pmcs_sections s ON s.id = i.section_id
		JOIN user_pmcs_revisions r ON r.id = s.revision_id
		WHERE p.id IN (` + placeholders + `)`
}

func insertPreparedTree(
	ctx context.Context,
	tx *sql.Tx,
	revision shared.RevisionInput,
) error {
	revisionModels := make([][]any, 0, len(revision.Models))
	sections := make([][]any, 0, len(revision.Sections))
	var sectionModels, items, notices, steps [][]any

	for _, model := range revision.Models {
		revisionModels = append(revisionModels, []any{
			revision.ID, model.DisplayText, model.NormalizedText,
		})
	}
	for _, section := range revision.Sections {
		sections = append(sections, []any{
			section.ID, revision.ID, section.Position, section.Title,
		})
		for _, model := range section.Models {
			sectionModels = append(sectionModels, []any{
				section.ID, model.DisplayText, model.NormalizedText,
			})
		}
		for _, item := range section.Items {
			items = append(items, []any{
				item.ID, section.ID, item.Position, item.Interval,
				item.ItemToBeCheckedOrServiced, item.PerformedBy,
			})
			for _, notice := range item.Notices {
				notices = append(notices, []any{
					notice.ID, item.ID, notice.Position,
					notice.Type, notice.NoticeText,
				})
			}
			for _, step := range item.ProcedureSteps {
				steps = append(steps, []any{
					step.ID, item.ID, step.Position,
					step.StepText, step.FaultFoundIf,
				})
			}
		}
	}

	insertions := []struct {
		table   string
		columns []string
		rows    [][]any
	}{
		{"user_pmcs_revision_models",
			[]string{"revision_id", "display_text", "normalized_text"},
			revisionModels},
		{"user_pmcs_sections",
			[]string{"id", "revision_id", "position", "title"},
			sections},
		{"user_pmcs_section_models",
			[]string{"section_id", "display_text", "normalized_text"},
			sectionModels},
		{"user_pmcs_items",
			[]string{"id", "section_id", "position", "interval",
				"item_to_be_checked_or_serviced", "performed_by"},
			items},
		{"user_pmcs_notices",
			[]string{"id", "item_id", "position", "type", "notice_text"},
			notices},
		{"user_pmcs_procedure_steps",
			[]string{"id", "item_id", "position", "step_text",
				"fault_found_if"},
			steps},
	}
	for _, insertion := range insertions {
		if err := insertRows(
			ctx,
			tx,
			insertion.table,
			insertion.columns,
			insertion.rows,
		); err != nil {
			return err
		}
	}
	return nil
}
