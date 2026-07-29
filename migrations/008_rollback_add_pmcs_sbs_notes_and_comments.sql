-- Rollback: 008_rollback_add_pmcs_sbs_notes_and_comments.sql

DROP INDEX IF EXISTS idx_pmcs_sbs_inspection_comments_pmcs_id;
DROP TABLE IF EXISTS pmcs_sbs_inspection_comments;
ALTER TABLE pmcs_sbs_inspections DROP COLUMN IF EXISTS notes;
