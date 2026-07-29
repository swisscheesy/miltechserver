# User-Created PMCS Database and Shared Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development`. Use one fresh implementer and one
> independent reviewer per task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Establish the relational schema, generated Jet models, shared
contracts, validation, hashing, and persistence primitives required by every
user-PMCS endpoint.

**Architecture:** Migration 009 creates normalized revision trees, stable
checklist/subscription roots, public release references, and the per-user sync
counter. A shared Go package validates and canonicalizes complete trees before
short transactions; a shared persistence package performs deterministic
locking and batched relational reads/writes.

**Tech Stack:** PostgreSQL, Jet 2.13.0, Go 1.23, `database/sql`,
`github.com/google/uuid`, `github.com/clipperhouse/uax29/v2/graphemes`
v2.4.0, `testify/require`.

## Global constraints

Read and inherit every constraint from
`docs/superpowers/plans/2026-07-29-user-pmcs-server-implementation.md`.
Migration number 009 is correct at planning HEAD but must be reverified before
execution. Do not hand-edit generated files.

---

### Task 1: Migration, integrity tests, and Jet generation

**Files:**

- Create: `migrations/009_create_user_pmcs_sync.sql`
- Create: `migrations/009_rollback_user_pmcs_sync.sql`
- Create: `tests/user_pmcs/main_test.go`
- Create: `tests/user_pmcs/migration_schema_test.go`
- Modify (generated): `.gen/miltech_ng/public/model/user_pmcs_*.go`
- Modify (generated): `.gen/miltech_ng/public/table/user_pmcs_*.go`

**Interfaces:**

- Produces the twelve Jet table variables and model types named after the twelve
  `user_pmcs_*` tables below.
- Produces restrictive composite release references used by deletion logic.
- Produces `content_hash []byte` on `model.UserPmcsRevisions`.

- [ ] **Step 1: Write the failing schema-integrity test**

Create a table-driven Postgres integration test that queries
`information_schema.tables`, `pg_constraint`, and `pg_indexes`:

```go
func TestUserPmcsSchemaIntegrity(t *testing.T) {
	requiredTables := []string{
		"user_pmcs_sync_state", "user_pmcs_checklists",
		"user_pmcs_revisions", "user_pmcs_revision_models",
		"user_pmcs_sections", "user_pmcs_section_models",
		"user_pmcs_items", "user_pmcs_notices",
		"user_pmcs_procedure_steps", "user_pmcs_community_sources",
		"user_pmcs_community_releases", "user_pmcs_subscriptions",
	}
	for _, tableName := range requiredTables {
		t.Run(tableName, func(t *testing.T) {
			var exists bool
			err := testDB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables
					WHERE table_schema = 'public' AND table_name = $1
				)`, tableName).Scan(&exists)
			require.NoError(t, err)
			require.True(t, exists)
		})
	}
}
```

Add focused assertions that the source-to-release composite FK and both
release-to-revision FK columns have delete action `RESTRICT`, that partial
draft/published indexes exist, and that every child FK has a leading index.

- [ ] **Step 2: Run the test to prove the schema is absent**

Run:

```bash
go test ./tests/user_pmcs -run TestUserPmcsSchemaIntegrity -count=1
```

Expected: FAIL because `user_pmcs_sync_state` and the remaining tables do not
exist.

- [ ] **Step 3: Write the forward migration**

Use this table order and exact integrity policy:

```sql
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
    revision_id    UUID NOT NULL,
    display_text   TEXT NOT NULL,
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
    section_id     UUID NOT NULL,
    display_text   TEXT NOT NULL,
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
```

There is intentionally no standalone FK from
`user_pmcs_subscriptions.checklist_id` to `user_pmcs_community_sources`.
An active subscription is protected by the composite installed-release FK and
service validation. An unsubscribed permanent tombstone has a null installed
revision and must be allowed to outlive a source removed during owner-account
cleanup. The checklist-leading subscription index still supports source
retention and cleanup queries.

The implementation may split long SQL comments but must not weaken these
constraints. Run the repository's missing-FK-index audit against the result.

- [ ] **Step 4: Write the rollback**

Use one transaction and this child-to-parent order:

```sql
BEGIN;
DROP TABLE IF EXISTS user_pmcs_subscriptions;
ALTER TABLE IF EXISTS user_pmcs_community_sources
    DROP CONSTRAINT IF EXISTS fk_user_pmcs_community_sources_current_release;
DROP TABLE IF EXISTS user_pmcs_community_releases;
DROP TABLE IF EXISTS user_pmcs_community_sources;
DROP TABLE IF EXISTS user_pmcs_procedure_steps;
DROP TABLE IF EXISTS user_pmcs_notices;
DROP TABLE IF EXISTS user_pmcs_items;
DROP TABLE IF EXISTS user_pmcs_section_models;
DROP TABLE IF EXISTS user_pmcs_sections;
DROP TABLE IF EXISTS user_pmcs_revision_models;
DROP TABLE IF EXISTS user_pmcs_revisions;
DROP TABLE IF EXISTS user_pmcs_checklists;
DROP TABLE IF EXISTS user_pmcs_sync_state;
COMMIT;
```

- [ ] **Step 5: Verify forward, rollback, and forward**

Apply the forward migration to the configured disposable test database, run
the schema test, apply rollback, prove the tables are absent, then reapply
forward and rerun the test. Use `TEST_DATABASE_URL`; do not embed credentials
or a fallback DSN in new tests.

Expected: every command succeeds and the final schema test passes.

- [ ] **Step 6: Regenerate Jet**

Run the repository's current Jet generator against the migrated development
schema. Confirm generated models use `uuid.UUID`, nullable pointers, `[]byte`
for `content_hash`, and no generated UUID defaults.

Run:

```bash
gofmt -w .gen/miltech_ng/public/model .gen/miltech_ng/public/table
go test ./tests/user_pmcs -run TestUserPmcsSchemaIntegrity -count=1
go test ./... -run '^$'
```

Expected: schema test PASS and all packages compile.

- [ ] **Step 7: Commit**

```bash
git add migrations/009_create_user_pmcs_sync.sql \
  migrations/009_rollback_user_pmcs_sync.sql \
  .gen/miltech_ng/public/model \
  .gen/miltech_ng/public/table \
  tests/user_pmcs/main_test.go \
  tests/user_pmcs/migration_schema_test.go
git commit -m "feat(user-pmcs): add sync and sharing schema"
```

---

### Task 2: Shared configuration, domain contracts, and HTTP primitives

**Files:**

- Modify: `bootstrap/env.go`
- Create: `api/user_pmcs/shared/config.go`
- Create: `api/user_pmcs/shared/domain.go`
- Create: `api/user_pmcs/shared/errors.go`
- Create: `api/user_pmcs/shared/http.go`
- Create: `api/user_pmcs/shared/cursor.go`
- Create: `api/user_pmcs/shared/http_test.go`
- Create: `api/user_pmcs/shared/cursor_test.go`

**Interfaces:**

- Produces `shared.Config`, `shared.RevisionInput`,
  `shared.ChecklistAggregate`, `shared.APIError`, `shared.Precondition`,
  `shared.MakeChecklistETag`, `shared.MakeSubscriptionETag`,
  `shared.DecodeStrictJSON`, and opaque cursor codecs.

- [ ] **Step 1: Write failing table-driven tests**

Cover weak ETags, multiple ETags, malformed quoted values, missing headers,
`If-None-Match: *`, current/stale `If-Match`, unknown JSON fields, trailing
JSON, malformed cursor base64, and cursor round trips.

```go
func TestParseExistingPrecondition(t *testing.T) {
	current := shared.MakeChecklistETag(uuid.MustParse(
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), 7)
	got, err := shared.ParseExistingPrecondition(current)
	require.NoError(t, err)
	require.Equal(t, current, got.ETag)
}
```

- [ ] **Step 2: Run tests and verify failure**

```bash
go test ./api/user_pmcs/shared -run 'Test(Parse|Decode|Cursor)' -count=1
```

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Add concrete configuration**

`bootstrap.Env` owns deployment configuration. Add a `UserPmcs` field whose
defaults exactly match the master plan. `shared.Config` contains:

```go
type Config struct {
	MaxOwnedChecklists       int
	MaxActiveSubscriptions  int
	MaxChecklistModels      int
	MaxSections             int
	MaxSectionModels        int
	MaxSectionModelsTotal   int
	MaxItemsPerSection      int
	MaxItemsTotal           int
	MaxNoticesPerItem       int
	MaxNoticesTotal         int
	MaxStepsPerItem         int
	MaxStepsTotal           int
	MaxMutationBodyBytes    int64
	MaxDeltaResponseBytes   int
	DeltaDefaultLimit       int
	DeltaMaxLimit           int
	UpdatesDefaultLimit     int
	UpdatesMaxLimit         int
	CommunityDefaultLimit   int
	CommunityMaxLimit       int
	TransactionMaxAttempts  int
}

func DefaultConfig() Config
func ConfigFromEnv(env *bootstrap.Env) (Config, error)
```

Map fields to these environment names:

| Field | Environment name |
|---|---|
| `MaxOwnedChecklists` | `USER_PMCS_MAX_OWNED_CHECKLISTS` |
| `MaxActiveSubscriptions` | `USER_PMCS_MAX_ACTIVE_SUBSCRIPTIONS` |
| `MaxChecklistModels` | `USER_PMCS_MAX_CHECKLIST_MODELS` |
| `MaxSections` | `USER_PMCS_MAX_SECTIONS` |
| `MaxSectionModels` | `USER_PMCS_MAX_SECTION_MODELS_PER_SECTION` |
| `MaxSectionModelsTotal` | `USER_PMCS_MAX_SECTION_MODELS_TOTAL` |
| `MaxItemsPerSection` | `USER_PMCS_MAX_ITEMS_PER_SECTION` |
| `MaxItemsTotal` | `USER_PMCS_MAX_ITEMS_TOTAL` |
| `MaxNoticesPerItem` | `USER_PMCS_MAX_NOTICES_PER_ITEM` |
| `MaxNoticesTotal` | `USER_PMCS_MAX_NOTICES_TOTAL` |
| `MaxStepsPerItem` | `USER_PMCS_MAX_STEPS_PER_ITEM` |
| `MaxStepsTotal` | `USER_PMCS_MAX_STEPS_TOTAL` |
| `MaxMutationBodyBytes` | `USER_PMCS_MAX_MUTATION_BODY_BYTES` |
| `MaxDeltaResponseBytes` | `USER_PMCS_MAX_DELTA_RESPONSE_BYTES` |
| `DeltaDefaultLimit` | `USER_PMCS_DELTA_DEFAULT_LIMIT` |
| `DeltaMaxLimit` | `USER_PMCS_DELTA_MAX_LIMIT` |
| `UpdatesDefaultLimit` | `USER_PMCS_UPDATES_DEFAULT_LIMIT` |
| `UpdatesMaxLimit` | `USER_PMCS_UPDATES_MAX_LIMIT` |
| `CommunityDefaultLimit` | `USER_PMCS_COMMUNITY_DEFAULT_LIMIT` |
| `CommunityMaxLimit` | `USER_PMCS_COMMUNITY_MAX_LIMIT` |
| `TransactionMaxAttempts` | `USER_PMCS_TRANSACTION_MAX_ATTEMPTS` |

Reject nonpositive configured values during application startup rather than
silently accepting them.

- [ ] **Step 4: Add domain input and output types**

Use value types independent of Jet:

```go
type RevisionInput struct {
	ID             uuid.UUID      `json:"id"`
	RevisionNumber *int32         `json:"revision_number,omitempty"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Models         []ModelInput   `json:"models"`
	Sections       []SectionInput `json:"sections"`
}

type ModelInput struct {
	DisplayText    string `json:"display_text"`
	NormalizedText string `json:"-"`
}

type SectionInput struct {
	ID       uuid.UUID    `json:"id"`
	Position int32        `json:"position"`
	Title    string       `json:"title"`
	Models   []ModelInput `json:"models"`
	Items    []ItemInput  `json:"items"`
}

type ItemInput struct {
	ID                          uuid.UUID            `json:"id"`
	Position                    int32                `json:"position"`
	Interval                    string               `json:"interval"`
	ItemToBeCheckedOrServiced   string               `json:"item_to_be_checked_or_serviced"`
	PerformedBy                 string               `json:"performed_by"`
	Notices                     []NoticeInput        `json:"notices"`
	ProcedureSteps              []ProcedureStepInput `json:"procedure_steps"`
}

type NoticeInput struct {
	ID         uuid.UUID `json:"id"`
	Position   int32     `json:"position"`
	Type       *string   `json:"type"`
	NoticeText string    `json:"notice_text"`
}

type ProcedureStepInput struct {
	ID           uuid.UUID `json:"id"`
	Position     int32     `json:"position"`
	StepText     string    `json:"step_text"`
	FaultFoundIf string    `json:"fault_found_if"`
}

type ModelValue struct {
	DisplayText    string `json:"display_text"`
	NormalizedText string `json:"normalized_text"`
}

type Section struct {
	ID       uuid.UUID    `json:"id"`
	Position int32        `json:"position"`
	Title    string       `json:"title"`
	Models   []ModelValue `json:"models"`
	Items    []Item       `json:"items"`
}

type Item struct {
	ID                        uuid.UUID       `json:"id"`
	Position                  int32           `json:"position"`
	Interval                  string          `json:"interval"`
	ItemToBeCheckedOrServiced string          `json:"item_to_be_checked_or_serviced"`
	PerformedBy               string          `json:"performed_by"`
	Notices                   []NoticeInput   `json:"notices"`
	ProcedureSteps            []ProcedureStepInput `json:"procedure_steps"`
}

type Revision struct {
	ID             uuid.UUID    `json:"id"`
	RevisionNumber *int32       `json:"revision_number,omitempty"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Models         []ModelValue `json:"models"`
	Sections       []Section    `json:"sections"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type CommunitySourceSummary struct {
	Status                       string     `json:"status"`
	CurrentReleaseRevisionID     *uuid.UUID `json:"current_release_revision_id,omitempty"`
	LatestReleaseRevisionNumber  int32      `json:"latest_release_revision_number"`
	FirstReleasedAt              time.Time  `json:"first_released_at"`
	UpdatedAt                    time.Time  `json:"updated_at"`
	RetiredAt                    *time.Time `json:"retired_at,omitempty"`
}

type ChecklistAggregate struct {
	ID                   uuid.UUID               `json:"id"`
	SyncVersion          int64                   `json:"sync_version"`
	AccountChangeVersion int64                   `json:"account_change_version"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
	DeletedAt            *time.Time              `json:"deleted_at,omitempty"`
	Draft                *Revision               `json:"draft,omitempty"`
	Publication          *Revision               `json:"publication,omitempty"`
	Community            *CommunitySourceSummary `json:"community,omitempty"`
}

type Subscription struct {
	ChecklistID          uuid.UUID  `json:"checklist_id"`
	InstalledRevisionID  *uuid.UUID `json:"installed_revision_id,omitempty"`
	SyncVersion          int64      `json:"sync_version"`
	AccountChangeVersion int64      `json:"account_change_version"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty"`
}

type InstalledChecklistRelease struct {
	ChecklistID        uuid.UUID `json:"checklist_id"`
	SourceStatus       string    `json:"source_status"`
	CreatorDisplayName string    `json:"creator_display_name"`
	ReleasedAt         time.Time `json:"released_at"`
	Revision           Revision  `json:"revision"`
}
```

Public/subscription/delta wrappers compose these value types without exposing
owner or subscriber UID; they do not duplicate authored-tree structs.

- [ ] **Step 5: Implement stable errors and response envelopes**

`APIError` carries HTTP status, stable code, safe message, and optional safe
details. Implement constructors for every code in specification §15.3.

```go
type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }
func (e *APIError) Unwrap() error { return e.Cause }
```

The handler serializer emits `{status,message,data:null,error:{code,details}}`
and never serializes `Cause`.

- [ ] **Step 6: Implement strict JSON and conditional parsing**

`DecodeStrictJSON` wraps the body with `http.MaxBytesReader`, requires
`application/json` with no content-encoding, calls `DisallowUnknownFields`,
decodes exactly one value, and maps body overflow to 413.

ETags are strong quoted SHA-256 validators:

```go
func makeETag(kind, identity string, version int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d",
		kind, identity, version)))
	return `"` + base64.RawURLEncoding.EncodeToString(sum[:]) + `"`
}
```

Only one strong ETag is accepted for mutation. `*` is accepted only by create
parsing. Every nested checklist mutation uses the checklist validator.

Use these exact precondition types:

```go
type PreconditionMode string

const (
	PreconditionMatch    PreconditionMode = "match"
	PreconditionCreate   PreconditionMode = "create"
)

type Precondition struct {
	Mode PreconditionMode
	ETag string
}

func ParseExistingPrecondition(header string) (Precondition, error)
func ParseCreatePrecondition(header string) (Precondition, error)
```

- [ ] **Step 7: Implement opaque cursors**

Use versioned JSON structs encoded with base64url:

```go
type CommunityCursor struct {
	Version   int       `json:"v"`
	UpdatedAt time.Time `json:"updated_at"`
	Checklist uuid.UUID `json:"checklist_id"`
}

type SubscriptionUpdateCursor struct {
	Version   int       `json:"v"`
	Checklist uuid.UUID `json:"checklist_id"`
}
```

Reject unknown versions and trailing bytes. Cursor contents are opaque but do
not require signing because they contain no authority and every query
revalidates access and filters.

- [ ] **Step 8: Verify and commit**

```bash
go test ./api/user_pmcs/shared -count=1
go test ./bootstrap ./api/route -run '^$'
git add bootstrap/env.go api/user_pmcs/shared
git commit -m "feat(user-pmcs): add shared API contracts"
```

---

### Task 3: Unicode normalization, validation, and canonical hashing

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`
- Create: `api/user_pmcs/shared/normalize.go`
- Create: `api/user_pmcs/shared/validation.go`
- Create: `api/user_pmcs/shared/content_hash.go`
- Create: `api/user_pmcs/shared/testdata/unicode_v16.json`
- Create: `api/user_pmcs/shared/validation_test.go`
- Create: `api/user_pmcs/shared/content_hash_test.go`

**Interfaces:**

- Produces `NormalizeModel`, `PrepareDraft`, `PreparePublication`,
  `CanonicalRevisionHash`, `PreparedRevision`, `TreeCounts`, and typed
  ceiling/validation errors.

- [ ] **Step 1: Add failing normalization and boundary tests**

Fixtures include ASCII/non-ASCII whitespace, composed/decomposed sequences,
ZWJ emoji, flags, skin tones, combining marks, and duplicate models that
normalize identically. Test exactly 200/201 and 4,000/4,001 grapheme
boundaries, 8 KiB/64 KiB byte boundaries, every per-parent limit, and every
total limit.

```go
func TestPublicationRejectsDuplicateNormalizedModels(t *testing.T) {
	input := validPublication()
	input.Models = []shared.ModelInput{
		{DisplayText: " M1152A1 "},
		{DisplayText: "m1152a1"},
	}
	err := shared.ValidatePublication(input, shared.DefaultConfig())
	requireAPIError(t, err, http.StatusUnprocessableEntity,
		"validation_failed")
}
```

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/shared -run 'Test(Normalize|Draft|Publication|Canonical)' -count=1
```

- [ ] **Step 3: Pin the Unicode-compatible library**

```bash
go get github.com/clipperhouse/uax29/v2@v2.4.0
```

Do not use `rivo/uniseg` v0.4.7: its Unicode 15 data does not match the
mobile client's Unicode 16 `characters` 1.4.1.

- [ ] **Step 4: Implement normalization and validation**

`NormalizeModel` trims Unicode whitespace, collapses each Unicode whitespace
run to one ASCII space, and lowercases. Validate UTF-8 before segmentation.

Count graphemes with `graphemes.FromString`. Draft validation permits blank
authored fields and incomplete structure but still enforces UUID uniqueness,
positions, types when non-null, text ceilings, and node ceilings. Publication
adds every completeness rule from specification §16.

All UUIDs are unique across the entire submitted tree, even across different
node types. Positions must be contiguous one-based siblings.

Preparation returns the only tree shape repositories accept:

```go
type PreparedRevision struct {
	Input  RevisionInput
	Hash   [32]byte
	Counts TreeCounts
}

func PrepareDraft(input RevisionInput, config Config) (PreparedRevision, error)
func PreparePublication(input RevisionInput, config Config) (PreparedRevision, error)
```

Both functions fill every model's `NormalizedText`, validate the resulting
canonical tree, compute counts and hash, and finish before a transaction opens.

- [ ] **Step 5: Implement deterministic hashing**

Hash the canonical semantic input before transactions. Write each scalar using
a one-byte field tag and big-endian length prefix; sort model sets by
`normalized_text`; sort sections/items/notices/steps by position; include
every client UUID and byte-exact authored string; exclude server timestamps,
revision state, and revision number.

```go
func CanonicalRevisionHash(input RevisionInput) ([32]byte, error) {
	canonical, err := canonicalizeRevision(input)
	if err != nil {
		return [32]byte{}, err
	}
	h := sha256.New()
	writeRevision(h, canonical)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result, nil
}
```

Tests prove reordered model sets hash equally, changed text/UUID/position hashes
differently, and the same normalized input is stable across repeated calls.

- [ ] **Step 6: Verify and commit**

```bash
go test ./api/user_pmcs/shared -count=1
go test -race ./api/user_pmcs/shared -count=1
git add go.mod go.sum api/user_pmcs/shared
git commit -m "feat(user-pmcs): validate and hash revision trees"
```

---

### Task 4: Transaction retry, account locking, and batched tree persistence

**Files:**

- Create: `api/user_pmcs/persistence/store.go`
- Create: `api/user_pmcs/persistence/retry.go`
- Create: `api/user_pmcs/persistence/sync_state.go`
- Create: `api/user_pmcs/persistence/tree_reader.go`
- Create: `api/user_pmcs/persistence/tree_writer.go`
- Create: `api/user_pmcs/persistence/retry_test.go`
- Create: `api/user_pmcs/persistence/tree_writer_test.go`
- Create: `tests/user_pmcs/helpers_test.go`
- Create: `tests/user_pmcs/tree_persistence_test.go`

**Interfaces:**

- Produces `persistence.Store`, `persistence.WithWriteTx`,
  `persistence.LockAccountVersion`, `persistence.LoadRevisionTrees`,
  `persistence.ReplaceDraftTree`, and `persistence.DeleteRevisionTrees`.

```go
type Queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func LoadRevisionTrees(ctx context.Context, queryer Queryer,
	revisionIDs []uuid.UUID) (map[uuid.UUID]shared.Revision, error)

func ReplaceDraftTree(ctx context.Context, tx *sql.Tx,
	checklistID uuid.UUID, prepared shared.PreparedRevision) error

func DeleteRevisionTrees(ctx context.Context, tx *sql.Tx,
	revisionIDs []uuid.UUID) error
```

- [ ] **Step 1: Write failing retry and integration tests**

Test SQLSTATE classification without a database and tree round trips with the
real test database. Test cross-revision UUID grafting rejection and prove tree
loading uses a fixed number of queries for 1 and 25 revision roots.

```go
func TestIsRetryableTransactionError(t *testing.T) {
	require.True(t, persistence.IsRetryable(
		&pq.Error{Code: pq.ErrorCode("40P01")}))
	require.True(t, persistence.IsRetryable(
		&pq.Error{Code: pq.ErrorCode("40001")}))
	require.False(t, persistence.IsRetryable(
		&pq.Error{Code: pq.ErrorCode("23505")}))
}
```

- [ ] **Step 2: Run and prove failure**

```bash
go test ./api/user_pmcs/persistence ./tests/user_pmcs \
  -run 'Test(IsRetryable|Tree)' -count=1
```

- [ ] **Step 3: Implement bounded transaction retry**

`WithWriteTx` begins `READ COMMITTED`, runs the callback, rolls back on error,
commits on success, and retries only SQLSTATE 40P01/40001 with context-aware
short jitter for at most configured attempts. Never retry validation,
authorization, conditional, or constraint errors.

```go
type TxFunc[T any] func(*sql.Tx) (T, error)

func WithWriteTx[T any](
	ctx context.Context,
	db *sql.DB,
	maxAttempts int,
	fn TxFunc[T],
) (T, error)
```

- [ ] **Step 4: Implement account initialization and locking**

`LockAccountVersion` first verifies `users(uid)` exists, inserts sync state
with `ON CONFLICT DO NOTHING`, then selects the row `FOR UPDATE`. It returns
`account_not_initialized` before any PMCS mutation if the user row is absent.

`AdvanceAccountVersion` increments once and returns the committed logical
change number used on the mutated root.

- [ ] **Step 5: Implement batched tree writes**

`ReplaceDraftTree`:

1. checks every submitted UUID against existing node tables in batched `ANY`
   queries;
2. rejects any UUID owned by another submitted parent/revision/checklist;
3. deletes only current mutable children;
4. inserts revision models, sections, section models, items, notices, and steps
   in bounded multi-row batches; and
5. writes the precomputed 32-byte hash.

Use a maximum 1,000 bind parameters per statement so requests remain below
Postgres's parameter limit. Never loop one INSERT per node.

- [ ] **Step 6: Implement batched tree reads**

`LoadRevisionTrees(ctx, queryer, revisionIDs)` performs one query per table,
each filtered by the entire selected revision/parent ID set, and assembles
trees in memory by UUID. It never joins every child table into a Cartesian row
explosion and never runs per-revision child queries.

- [ ] **Step 7: Verify query plans and tests**

```bash
go test ./api/user_pmcs/persistence -count=1
go test ./tests/user_pmcs -run 'TestTree' -count=1
go test -race ./api/user_pmcs/persistence -count=1
```

Run `EXPLAIN (ANALYZE, BUFFERS)` for each batched parent lookup and confirm
the planned parent-leading indexes are used at representative node ceilings.

- [ ] **Step 8: Commit**

```bash
git add api/user_pmcs/persistence tests/user_pmcs
git commit -m "feat(user-pmcs): add transactional tree persistence"
```

## Foundation completion gate

Before the owned-sync plan begins:

```bash
go test ./api/user_pmcs/shared ./api/user_pmcs/persistence -count=1
go test ./tests/user_pmcs -run 'Test(UserPmcsSchema|Tree)' -count=1
go test ./... -run '^$'
git status --short
```

Record exact results and HEAD in the progress ledger. A clean build does not
count as endpoint delivery; no routes exist until the next plan.
