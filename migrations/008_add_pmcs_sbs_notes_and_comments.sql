-- PMCS SBS Inspection Notes + Comments
-- Migration: 008_add_pmcs_sbs_notes_and_comments.sql
--
-- Adds a free-text notes field to pmcs_sbs_inspections and a new
-- pmcs_sbs_inspection_comments table so any shop member with access to a
-- vehicle can leave a comment on a PMCS inspection. See ADR-018 in
-- docs/project_notes/decisions.md.

ALTER TABLE pmcs_sbs_inspections ADD COLUMN notes TEXT;

CREATE TABLE pmcs_sbs_inspection_comments (
    id          UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    pmcs_id     UUID NOT NULL,
    author_id   TEXT NOT NULL,
    text        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ,

    CONSTRAINT fk_pmcs_sbs_inspection_comments_pmcs_id
        FOREIGN KEY (pmcs_id) REFERENCES pmcs_sbs_inspections(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_pmcs_sbs_inspection_comments_author_id
        FOREIGN KEY (author_id) REFERENCES users(uid)
        ON UPDATE CASCADE,
    CONSTRAINT pmcs_sbs_inspection_comments_nonblank_check
        CHECK (btrim(text) <> '')
);

CREATE INDEX idx_pmcs_sbs_inspection_comments_pmcs_id
    ON pmcs_sbs_inspection_comments (pmcs_id, created_at ASC);
