-- PMCS SBS Inspection Source Union
-- Migration: 011_extend_pmcs_inspection_sources.sql
--
-- Extends historical PMCS inspections so a row is either a guide inspection
-- or an immutable custom-checklist revision snapshot. Existing rows are
-- backfilled as guide inspections by the temporary source_type default.

BEGIN;

ALTER TABLE pmcs_sbs_inspections
  ADD COLUMN source_type TEXT NOT NULL DEFAULT 'guide',
  ADD COLUMN custom_checklist_id UUID,
  ADD COLUMN custom_revision_id UUID,
  ADD COLUMN custom_revision_number INTEGER,
  ADD COLUMN custom_checklist_name TEXT;

ALTER TABLE pmcs_sbs_inspections ALTER COLUMN guide_manual DROP NOT NULL;

ALTER TABLE pmcs_sbs_faults ADD COLUMN section_title TEXT;

ALTER TABLE pmcs_sbs_inspections
  DROP CONSTRAINT pmcs_sbs_inspections_nonblank_check,
  DROP CONSTRAINT pmcs_sbs_inspections_guide_manual_format_check;

ALTER TABLE pmcs_sbs_inspections
  ADD CONSTRAINT pmcs_sbs_inspections_source_type_check
    CHECK (source_type IN ('guide', 'custom')),
  ADD CONSTRAINT pmcs_sbs_inspections_equipment_id_nonblank_check
    CHECK (btrim(equipment_id) <> ''),
  ADD CONSTRAINT pmcs_sbs_inspections_source_shape_check CHECK (
    (source_type = 'guide' AND guide_manual IS NOT NULL
      AND guide_manual = btrim(guide_manual)
      AND guide_manual LIKE 'pmcs_sbs/%'
      AND right(guide_manual, 5) = '.json'
      AND custom_checklist_id IS NULL AND custom_revision_id IS NULL
      AND custom_revision_number IS NULL AND custom_checklist_name IS NULL)
    OR
    (source_type = 'custom' AND guide_manual IS NULL
      AND custom_checklist_id IS NOT NULL AND custom_revision_id IS NOT NULL
      AND custom_revision_number IS NOT NULL AND custom_revision_number >= 0
      AND custom_checklist_name IS NOT NULL
      AND custom_checklist_name = btrim(custom_checklist_name)
      AND btrim(custom_checklist_name) <> '')
  );

ALTER TABLE pmcs_sbs_inspections ALTER COLUMN source_type DROP DEFAULT;

COMMIT;
