-- Rollback: 011_rollback_extend_pmcs_inspection_sources.sql
--
-- Custom inspections carry provenance that cannot be represented by the
-- original guide-only schema, so this rollback refuses to discard them.

BEGIN;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pmcs_sbs_inspections
    WHERE source_type = 'custom'
  ) THEN
    RAISE EXCEPTION 'cannot roll back PMCS inspection source union while custom inspections exist';
  END IF;
END
$$;

ALTER TABLE pmcs_sbs_inspections
  DROP CONSTRAINT pmcs_sbs_inspections_source_type_check,
  DROP CONSTRAINT pmcs_sbs_inspections_source_shape_check;

ALTER TABLE pmcs_sbs_faults DROP COLUMN section_title;

ALTER TABLE pmcs_sbs_inspections
  DROP COLUMN source_type,
  DROP COLUMN custom_checklist_id,
  DROP COLUMN custom_revision_id,
  DROP COLUMN custom_revision_number,
  DROP COLUMN custom_checklist_name;

ALTER TABLE pmcs_sbs_inspections ALTER COLUMN guide_manual SET NOT NULL;

ALTER TABLE pmcs_sbs_inspections
  ADD CONSTRAINT pmcs_sbs_inspections_nonblank_check
    CHECK (btrim(equipment_id) <> '' AND btrim(guide_manual) <> ''),
  ADD CONSTRAINT pmcs_sbs_inspections_guide_manual_format_check
    CHECK (guide_manual = btrim(guide_manual)
      AND guide_manual LIKE 'pmcs_sbs/%'
      AND right(guide_manual, 5) = '.json');

COMMIT;
