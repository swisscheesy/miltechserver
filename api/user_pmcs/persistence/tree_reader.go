package persistence

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"miltechserver/api/user_pmcs/shared"
)

type sectionLocation struct {
	revisionID uuid.UUID
	index      int
}

type itemLocation struct {
	revisionID   uuid.UUID
	sectionIndex int
	itemIndex    int
}

func LoadRevisionTrees(
	ctx context.Context,
	queryer Queryer,
	revisionIDs []uuid.UUID,
) (map[uuid.UUID]shared.Revision, error) {
	revisions := make(map[uuid.UUID]shared.Revision, len(revisionIDs))
	if len(revisionIDs) == 0 {
		return revisions, nil
	}
	idArray := pq.Array(revisionIDs)

	if err := loadRevisionRoots(ctx, queryer, idArray, revisions); err != nil {
		return nil, err
	}
	if err := loadRevisionModels(ctx, queryer, idArray, revisions); err != nil {
		return nil, err
	}
	sectionLocations, err := loadSections(
		ctx,
		queryer,
		idArray,
		revisions,
	)
	if err != nil {
		return nil, err
	}
	if err := loadSectionModels(
		ctx,
		queryer,
		idArray,
		revisions,
		sectionLocations,
	); err != nil {
		return nil, err
	}
	itemLocations, err := loadItems(
		ctx,
		queryer,
		idArray,
		revisions,
		sectionLocations,
	)
	if err != nil {
		return nil, err
	}
	if err := loadNotices(
		ctx,
		queryer,
		idArray,
		revisions,
		itemLocations,
	); err != nil {
		return nil, err
	}
	if err := loadProcedureSteps(
		ctx,
		queryer,
		idArray,
		revisions,
		itemLocations,
	); err != nil {
		return nil, err
	}
	return revisions, nil
}

func loadRevisionRoots(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
) error {
	return queryRows(
		ctx,
		queryer,
		"revision roots",
		`SELECT id, revision_number, name, description, state,
		        created_at, updated_at, published_at
		 FROM user_pmcs_revisions
		 WHERE id = ANY($1)
		 ORDER BY id`,
		idArray,
		func(rows *sql.Rows) error {
			var revision shared.Revision
			if err := rows.Scan(
				&revision.ID, &revision.RevisionNumber, &revision.Name,
				&revision.Description, &revision.State, &revision.CreatedAt,
				&revision.UpdatedAt, &revision.PublishedAt,
			); err != nil {
				return err
			}
			revision.Models = []shared.ModelValue{}
			revision.Sections = []shared.Section{}
			revisions[revision.ID] = revision
			return nil
		},
	)
}

func loadRevisionModels(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
) error {
	return queryRows(
		ctx,
		queryer,
		"revision models",
		`SELECT revision_id, display_text, normalized_text
		 FROM user_pmcs_revision_models
		 WHERE revision_id = ANY($1)
		 ORDER BY revision_id, normalized_text`,
		idArray,
		func(rows *sql.Rows) error {
			var revisionID uuid.UUID
			var model shared.ModelValue
			if err := rows.Scan(
				&revisionID, &model.DisplayText, &model.NormalizedText,
			); err != nil {
				return err
			}
			revision, exists := revisions[revisionID]
			if exists {
				revision.Models = append(revision.Models, model)
				revisions[revisionID] = revision
			}
			return nil
		},
	)
}

func loadSections(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
) (map[uuid.UUID]sectionLocation, error) {
	locations := make(map[uuid.UUID]sectionLocation)
	err := queryRows(
		ctx,
		queryer,
		"sections",
		`SELECT id, revision_id, position, title
		 FROM user_pmcs_sections
		 WHERE revision_id = ANY($1)
		 ORDER BY revision_id, position`,
		idArray,
		func(rows *sql.Rows) error {
			var section shared.Section
			var revisionID uuid.UUID
			if err := rows.Scan(
				&section.ID, &revisionID, &section.Position, &section.Title,
			); err != nil {
				return err
			}
			section.Models = []shared.ModelValue{}
			section.Items = []shared.Item{}
			revision, exists := revisions[revisionID]
			if exists {
				locations[section.ID] = sectionLocation{
					revisionID: revisionID,
					index:      len(revision.Sections),
				}
				revision.Sections = append(revision.Sections, section)
				revisions[revisionID] = revision
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return locations, nil
}

func loadSectionModels(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
	locations map[uuid.UUID]sectionLocation,
) error {
	return queryRows(
		ctx,
		queryer,
		"section models",
		`SELECT sm.section_id, sm.display_text, sm.normalized_text
		 FROM user_pmcs_section_models sm
		 JOIN user_pmcs_sections s ON s.id = sm.section_id
		 WHERE s.revision_id = ANY($1)
		 ORDER BY sm.section_id, sm.normalized_text`,
		idArray,
		func(rows *sql.Rows) error {
			var sectionID uuid.UUID
			var model shared.ModelValue
			if err := rows.Scan(
				&sectionID, &model.DisplayText, &model.NormalizedText,
			); err != nil {
				return err
			}
			if location, exists := locations[sectionID]; exists {
				revision := revisions[location.revisionID]
				revision.Sections[location.index].Models = append(
					revision.Sections[location.index].Models,
					model,
				)
				revisions[location.revisionID] = revision
			}
			return nil
		},
	)
}

func loadItems(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
	sections map[uuid.UUID]sectionLocation,
) (map[uuid.UUID]itemLocation, error) {
	locations := make(map[uuid.UUID]itemLocation)
	err := queryRows(
		ctx,
		queryer,
		"items",
		`SELECT i.id, i.section_id, i.position, i.interval,
		        i.item_to_be_checked_or_serviced, i.performed_by
		 FROM user_pmcs_items i
		 JOIN user_pmcs_sections s ON s.id = i.section_id
		 WHERE s.revision_id = ANY($1)
		 ORDER BY i.section_id, i.position`,
		idArray,
		func(rows *sql.Rows) error {
			var item shared.Item
			var sectionID uuid.UUID
			if err := rows.Scan(
				&item.ID, &sectionID, &item.Position, &item.Interval,
				&item.ItemToBeCheckedOrServiced, &item.PerformedBy,
			); err != nil {
				return err
			}
			item.Notices = []shared.NoticeInput{}
			item.ProcedureSteps = []shared.ProcedureStepInput{}
			if section, exists := sections[sectionID]; exists {
				revision := revisions[section.revisionID]
				items := revision.Sections[section.index].Items
				locations[item.ID] = itemLocation{
					revisionID:   section.revisionID,
					sectionIndex: section.index,
					itemIndex:    len(items),
				}
				revision.Sections[section.index].Items = append(items, item)
				revisions[section.revisionID] = revision
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return locations, nil
}

func loadNotices(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
	items map[uuid.UUID]itemLocation,
) error {
	return queryRows(
		ctx,
		queryer,
		"notices",
		`SELECT n.id, n.item_id, n.position, n.type, n.notice_text
		 FROM user_pmcs_notices n
		 JOIN user_pmcs_items i ON i.id = n.item_id
		 JOIN user_pmcs_sections s ON s.id = i.section_id
		 WHERE s.revision_id = ANY($1)
		 ORDER BY n.item_id, n.position`,
		idArray,
		func(rows *sql.Rows) error {
			var notice shared.NoticeInput
			var itemID uuid.UUID
			if err := rows.Scan(
				&notice.ID, &itemID, &notice.Position,
				&notice.Type, &notice.NoticeText,
			); err != nil {
				return err
			}
			if location, exists := items[itemID]; exists {
				revision := revisions[location.revisionID]
				item := &revision.Sections[location.sectionIndex].
					Items[location.itemIndex]
				item.Notices = append(item.Notices, notice)
				revisions[location.revisionID] = revision
			}
			return nil
		},
	)
}

func loadProcedureSteps(
	ctx context.Context,
	queryer Queryer,
	idArray any,
	revisions map[uuid.UUID]shared.Revision,
	items map[uuid.UUID]itemLocation,
) error {
	return queryRows(
		ctx,
		queryer,
		"procedure steps",
		`SELECT p.id, p.item_id, p.position, p.step_text, p.fault_found_if
		 FROM user_pmcs_procedure_steps p
		 JOIN user_pmcs_items i ON i.id = p.item_id
		 JOIN user_pmcs_sections s ON s.id = i.section_id
		 WHERE s.revision_id = ANY($1)
		 ORDER BY p.item_id, p.position`,
		idArray,
		func(rows *sql.Rows) error {
			var step shared.ProcedureStepInput
			var itemID uuid.UUID
			if err := rows.Scan(
				&step.ID, &itemID, &step.Position,
				&step.StepText, &step.FaultFoundIf,
			); err != nil {
				return err
			}
			if location, exists := items[itemID]; exists {
				revision := revisions[location.revisionID]
				item := &revision.Sections[location.sectionIndex].
					Items[location.itemIndex]
				item.ProcedureSteps = append(item.ProcedureSteps, step)
				revisions[location.revisionID] = revision
			}
			return nil
		},
	)
}
