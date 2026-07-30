BEGIN;

CREATE TABLE user_pmcs_sync_state (
    user_uid        TEXT PRIMARY KEY,
    current_version BIGINT NOT NULL DEFAULT 0 CHECK (current_version >= 0),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_user_pmcs_sync_state_user
        FOREIGN KEY (user_uid) REFERENCES users(uid)
        ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE user_pmcs_checklists (
    id                     UUID PRIMARY KEY,
    owner_uid              TEXT,
    sync_version           BIGINT NOT NULL CHECK (sync_version > 0),
    account_change_version BIGINT NOT NULL CHECK (account_change_version > 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ,
    CONSTRAINT fk_user_pmcs_checklists_owner
        FOREIGN KEY (owner_uid) REFERENCES users(uid)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT user_pmcs_checklists_owner_state_check
        CHECK (owner_uid IS NOT NULL OR deleted_at IS NOT NULL)
);

CREATE TABLE user_pmcs_revisions (
    id              UUID PRIMARY KEY,
    checklist_id    UUID NOT NULL,
    state           TEXT NOT NULL,
    revision_number INTEGER,
    name            TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    content_hash    BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at    TIMESTAMPTZ,
    CONSTRAINT fk_user_pmcs_revisions_checklist
        FOREIGN KEY (checklist_id) REFERENCES user_pmcs_checklists(id)
        ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT user_pmcs_revisions_state_check CHECK (
        (state = 'draft' AND revision_number IS NULL AND published_at IS NULL)
        OR
        (state IN ('published', 'superseded')
         AND revision_number > 0 AND published_at IS NOT NULL)
    ),
    CONSTRAINT user_pmcs_revisions_hash_check
        CHECK (octet_length(content_hash) = 32),
    CONSTRAINT user_pmcs_revisions_text_bytes_check CHECK (
        octet_length(name) <= 8192
        AND octet_length(description) <= 65536
    ),
    CONSTRAINT user_pmcs_revisions_checklist_id_id_key
        UNIQUE (checklist_id, id)
);

CREATE UNIQUE INDEX user_pmcs_revisions_one_draft_idx
    ON user_pmcs_revisions (checklist_id) WHERE state = 'draft';
CREATE UNIQUE INDEX user_pmcs_revisions_one_published_idx
    ON user_pmcs_revisions (checklist_id) WHERE state = 'published';
CREATE UNIQUE INDEX user_pmcs_revisions_number_idx
    ON user_pmcs_revisions (checklist_id, revision_number)
    WHERE revision_number IS NOT NULL;

CREATE TABLE user_pmcs_revision_models (
    revision_id     UUID NOT NULL,
    display_text    TEXT NOT NULL,
    normalized_text TEXT NOT NULL,
    PRIMARY KEY (revision_id, normalized_text),
    CONSTRAINT fk_user_pmcs_revision_models_revision
        FOREIGN KEY (revision_id) REFERENCES user_pmcs_revisions(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_revision_models_bytes_check CHECK (
        octet_length(display_text) <= 8192
        AND octet_length(normalized_text) <= 8192
    )
);
CREATE INDEX user_pmcs_revision_models_lookup_idx
    ON user_pmcs_revision_models (normalized_text, revision_id);

CREATE TABLE user_pmcs_sections (
    id          UUID PRIMARY KEY,
    revision_id UUID NOT NULL,
    position    INTEGER NOT NULL CHECK (position > 0),
    title       TEXT NOT NULL DEFAULT '',
    CONSTRAINT fk_user_pmcs_sections_revision
        FOREIGN KEY (revision_id) REFERENCES user_pmcs_revisions(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_sections_position_key
        UNIQUE (revision_id, position),
    CONSTRAINT user_pmcs_sections_title_bytes_check
        CHECK (octet_length(title) <= 8192)
);

CREATE TABLE user_pmcs_section_models (
    section_id      UUID NOT NULL,
    display_text    TEXT NOT NULL,
    normalized_text TEXT NOT NULL,
    PRIMARY KEY (section_id, normalized_text),
    CONSTRAINT fk_user_pmcs_section_models_section
        FOREIGN KEY (section_id) REFERENCES user_pmcs_sections(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_section_models_bytes_check CHECK (
        octet_length(display_text) <= 8192
        AND octet_length(normalized_text) <= 8192
    )
);

CREATE TABLE user_pmcs_items (
    id                             UUID PRIMARY KEY,
    section_id                     UUID NOT NULL,
    position                       INTEGER NOT NULL CHECK (position > 0),
    interval                       TEXT NOT NULL DEFAULT '',
    item_to_be_checked_or_serviced TEXT NOT NULL DEFAULT '',
    performed_by                   TEXT NOT NULL DEFAULT '',
    CONSTRAINT fk_user_pmcs_items_section
        FOREIGN KEY (section_id) REFERENCES user_pmcs_sections(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_items_position_key UNIQUE (section_id, position),
    CONSTRAINT user_pmcs_items_text_bytes_check CHECK (
        octet_length(interval) <= 8192
        AND octet_length(performed_by) <= 8192
        AND octet_length(item_to_be_checked_or_serviced) <= 65536
    )
);

CREATE TABLE user_pmcs_notices (
    id          UUID PRIMARY KEY,
    item_id     UUID NOT NULL,
    position    INTEGER NOT NULL CHECK (position > 0),
    type        TEXT CHECK (type IN ('warning', 'caution', 'note')),
    notice_text TEXT NOT NULL DEFAULT '',
    CONSTRAINT fk_user_pmcs_notices_item
        FOREIGN KEY (item_id) REFERENCES user_pmcs_items(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_notices_position_key UNIQUE (item_id, position),
    CONSTRAINT user_pmcs_notices_text_bytes_check
        CHECK (octet_length(notice_text) <= 65536)
);

CREATE TABLE user_pmcs_procedure_steps (
    id             UUID PRIMARY KEY,
    item_id        UUID NOT NULL,
    position       INTEGER NOT NULL CHECK (position > 0),
    step_text      TEXT NOT NULL DEFAULT '',
    fault_found_if TEXT NOT NULL DEFAULT '',
    CONSTRAINT fk_user_pmcs_procedure_steps_item
        FOREIGN KEY (item_id) REFERENCES user_pmcs_items(id)
        ON DELETE CASCADE,
    CONSTRAINT user_pmcs_procedure_steps_position_key
        UNIQUE (item_id, position),
    CONSTRAINT user_pmcs_procedure_steps_text_bytes_check CHECK (
        octet_length(step_text) <= 65536
        AND octet_length(fault_found_if) <= 65536
    )
);

CREATE TABLE user_pmcs_community_sources (
    checklist_id                   UUID PRIMARY KEY,
    status                         TEXT NOT NULL CHECK (status IN ('active', 'retired')),
    current_release_revision_id    UUID,
    latest_release_revision_number INTEGER NOT NULL CHECK (latest_release_revision_number > 0),
    first_released_at              TIMESTAMPTZ NOT NULL,
    updated_at                     TIMESTAMPTZ NOT NULL,
    retired_at                     TIMESTAMPTZ,
    CONSTRAINT fk_user_pmcs_community_sources_checklist
        FOREIGN KEY (checklist_id) REFERENCES user_pmcs_checklists(id)
        ON DELETE RESTRICT,
    CONSTRAINT user_pmcs_community_sources_state_check CHECK (
        (status = 'active'
         AND current_release_revision_id IS NOT NULL
         AND retired_at IS NULL)
        OR
        (status = 'retired'
         AND current_release_revision_id IS NULL
         AND retired_at IS NOT NULL)
    )
);

CREATE TABLE user_pmcs_community_releases (
    revision_id UUID PRIMARY KEY,
    checklist_id UUID NOT NULL,
    released_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_pmcs_community_releases_checklist_revision_key
        UNIQUE (checklist_id, revision_id),
    CONSTRAINT fk_user_pmcs_community_releases_revision
        FOREIGN KEY (checklist_id, revision_id)
        REFERENCES user_pmcs_revisions(checklist_id, id)
        ON DELETE RESTRICT
);

ALTER TABLE user_pmcs_community_sources
    ADD CONSTRAINT fk_user_pmcs_community_sources_current_release
    FOREIGN KEY (checklist_id, current_release_revision_id)
    REFERENCES user_pmcs_community_releases(checklist_id, revision_id)
    ON DELETE RESTRICT;

CREATE TABLE user_pmcs_subscriptions (
    subscriber_uid         TEXT NOT NULL,
    checklist_id           UUID NOT NULL,
    installed_revision_id  UUID,
    sync_version           BIGINT NOT NULL CHECK (sync_version > 0),
    account_change_version BIGINT NOT NULL CHECK (account_change_version > 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ,
    PRIMARY KEY (subscriber_uid, checklist_id),
    CONSTRAINT fk_user_pmcs_subscriptions_subscriber
        FOREIGN KEY (subscriber_uid) REFERENCES users(uid)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_pmcs_subscriptions_installed_release
        FOREIGN KEY (checklist_id, installed_revision_id)
        REFERENCES user_pmcs_community_releases(checklist_id, revision_id)
        ON DELETE RESTRICT,
    CONSTRAINT user_pmcs_subscriptions_state_check CHECK (
        (deleted_at IS NULL AND installed_revision_id IS NOT NULL)
        OR
        (deleted_at IS NOT NULL AND installed_revision_id IS NULL)
    )
);

CREATE INDEX user_pmcs_checklists_owner_delta_idx
    ON user_pmcs_checklists (owner_uid, account_change_version);
CREATE INDEX user_pmcs_community_sources_recent_idx
    ON user_pmcs_community_sources (updated_at DESC, checklist_id)
    WHERE status = 'active';
CREATE INDEX user_pmcs_community_releases_history_idx
    ON user_pmcs_community_releases (checklist_id, released_at DESC);
CREATE INDEX user_pmcs_subscriptions_delta_idx
    ON user_pmcs_subscriptions (subscriber_uid, account_change_version);
CREATE INDEX user_pmcs_subscriptions_source_idx
    ON user_pmcs_subscriptions (checklist_id, subscriber_uid);
CREATE INDEX user_pmcs_subscriptions_active_update_idx
    ON user_pmcs_subscriptions (subscriber_uid, checklist_id, installed_revision_id)
    WHERE deleted_at IS NULL;
CREATE INDEX user_pmcs_subscriptions_active_pin_idx
    ON user_pmcs_subscriptions (installed_revision_id)
    WHERE deleted_at IS NULL;

COMMIT;
