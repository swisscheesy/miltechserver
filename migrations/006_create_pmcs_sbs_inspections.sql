-- PMCS SBS Inspection History
-- Migration: 006_create_pmcs_sbs_inspections.sql
--
-- Introduces pmcs_sbs_inspections as the parent of pmcs_sbs_faults so a
-- vehicle can have many dated PMCS inspections over time, each with its own
-- faults, instead of one overwritable fault state per checklist item.
--
-- Existing pmcs_sbs_faults rows are current-state-only snapshots with no
-- date-performed concept and are not preserved; see
-- docs/superpowers/specs/2026-07-16-pmcs-sbs-inspection-history-design.md.

DROP TABLE IF EXISTS pmcs_sbs_faults;

CREATE TABLE pmcs_sbs_inspections (
    id              UUID NOT NULL PRIMARY KEY,
    equipment_id    TEXT NOT NULL,
    guide_manual    TEXT NOT NULL,
    performed_date  TIMESTAMPTZ NOT NULL,
    created_by      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_pmcs_sbs_inspections_equipment_id
        FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_pmcs_sbs_inspections_created_by
        FOREIGN KEY (created_by) REFERENCES users(uid)
        ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT pmcs_sbs_inspections_nonblank_check
        CHECK (btrim(equipment_id) <> '' AND btrim(guide_manual) <> ''),
    CONSTRAINT pmcs_sbs_inspections_guide_manual_format_check
        CHECK (guide_manual = btrim(guide_manual) AND guide_manual LIKE 'pmcs_sbs/%' AND right(guide_manual, 5) = '.json')
);

CREATE INDEX idx_pmcs_sbs_inspections_equipment_performed
    ON pmcs_sbs_inspections (equipment_id, performed_date DESC);

CREATE TABLE pmcs_sbs_faults (
    pmcs_id            UUID NOT NULL,
    section_id         TEXT NOT NULL,
    item_index         INTEGER NOT NULL,
    item_no            TEXT NOT NULL,
    status             TEXT NOT NULL,
    fault_text         TEXT NOT NULL,
    corrective_action  TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (pmcs_id, section_id, item_index),
    CONSTRAINT fk_pmcs_sbs_faults_pmcs_id
        FOREIGN KEY (pmcs_id) REFERENCES pmcs_sbs_inspections(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT pmcs_sbs_faults_item_index_check CHECK (item_index >= 0),
    CONSTRAINT pmcs_sbs_faults_status_check CHECK (status = ANY (ARRAY['x','slash','dash'])),
    CONSTRAINT pmcs_sbs_faults_nonblank_fields_check
        CHECK (btrim(section_id) <> '' AND btrim(item_no) <> '' AND btrim(fault_text) <> '')
);
