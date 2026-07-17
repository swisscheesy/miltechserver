-- Rollback: 006_rollback_pmcs_sbs_inspections.sql

DROP TABLE IF EXISTS pmcs_sbs_faults;
DROP INDEX IF EXISTS idx_pmcs_sbs_inspections_equipment_performed;
DROP TABLE IF EXISTS pmcs_sbs_inspections;

CREATE TABLE pmcs_sbs_faults (
    equipment_id       text NOT NULL,
    guide_manual       text NOT NULL,
    section_id         text NOT NULL,
    item_index         integer NOT NULL,
    item_no            text NOT NULL,
    status             text NOT NULL,
    fault_text         text NOT NULL,
    corrective_action  text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (equipment_id, guide_manual, section_id, item_index),
    CONSTRAINT fk_pmcs_sbs_faults_equipment_id
        FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT pmcs_sbs_faults_item_index_check CHECK (item_index >= 0),
    CONSTRAINT pmcs_sbs_faults_status_check CHECK (status = ANY (ARRAY['x','slash','dash'])),
    CONSTRAINT pmcs_sbs_faults_nonblank_fields_check CHECK (
        btrim(equipment_id) <> '' AND btrim(guide_manual) <> '' AND
        btrim(section_id) <> '' AND btrim(item_no) <> '' AND btrim(fault_text) <> ''
    ),
    CONSTRAINT pmcs_sbs_faults_guide_manual_format_check CHECK (
        guide_manual = btrim(guide_manual) AND guide_manual LIKE 'pmcs_sbs/%' AND right(guide_manual, 5) = '.json'
    )
);
