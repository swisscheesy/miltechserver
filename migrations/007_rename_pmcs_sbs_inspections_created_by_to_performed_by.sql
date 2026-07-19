-- PMCS SBS Performed-By Rename
-- Migration: 007_rename_pmcs_sbs_inspections_created_by_to_performed_by.sql
--
-- Repurposes the existing caller-derived created_by column as performed_by:
-- see docs/superpowers/specs/2026-07-18-pmcs-sbs-performed-by-design.md.
-- Pure rename, no data change — existing values carry over unchanged.

ALTER TABLE pmcs_sbs_inspections RENAME COLUMN created_by TO performed_by;
ALTER TABLE pmcs_sbs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_created_by TO fk_pmcs_sbs_inspections_performed_by;
