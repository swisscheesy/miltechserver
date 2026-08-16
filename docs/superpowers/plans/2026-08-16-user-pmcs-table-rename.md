# User PMCS Server Table Rename Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the three legacy PostgreSQL PMCS persistence tables to the approved `user_pmcs_` names in `miltech_ng_test` and `miltech_ng`, regenerate Jet, and update every server/test reference without changing data, behavior, or the HTTP contract.

**Architecture:** Perform an atomic, metadata-only PostgreSQL rename with an exact inverse rollback. Rehearse forward/rollback/forward on `miltech_ng_test`, apply forward once to `miltech_ng`, regenerate Jet from migrated development, then mechanically replace internal generated identifiers and raw SQL names. Keep routes, JSON, domain DTOs, package names, guide paths, columns, constraints, index definitions, and historical migrations unchanged.

**Tech Stack:** PostgreSQL 14, SQL migrations, Go 1.23, Gin, Jet v2.13, `database/sql`, `lib/pq`, Testify.

**Spec:** `docs/superpowers/specs/2026-08-16-user-pmcs-table-rename-design.md`

## Global Constraints

- Modify only the two development databases `miltech_ng_test` and `miltech_ng`. Never connect to, inspect, or mutate production.
- Before every migration-sensitive command, run `SELECT current_database()` on that same connection and require the exact expected database name.
- Stop the local server and other database-backed test processes before acquiring locks or capturing fingerprints. Old and new binaries are intentionally not schema-compatible.
- Do not edit migrations `006` through `013`; they remain the historical construction path. Migration `014` is the current-schema rename.
- Do not rename `/api/v1/auth/pmcs-sbs/...`, request/response JSON, `pmcs_id`, `pmcs_sbs/` guide paths, the `api/pmcs_sbs_progress` package, response DTOs, or Flutter storage.
- Do not add compatibility views, copy tables, dual writes, triggers, or aliases.
- Never hand-edit Jet output. Use the exact JSON-tag generator template from `main.go:56-96` against migrated `miltech_ng`.
- Preserve the current generated-file tracking boundary: replace the four tracked inspection/fault model/table files with their renamed generated counterparts. The generated comment files and `table_use_schema.go` remain ignored local generator output, as they are today; do not force-add the entire ignored `.gen/miltech_ng` tree.
- Preserve the pre-existing untracked `.worktrees/` directory and any unrelated user changes.
- Run database-backed packages sequentially. Do not run packages sharing `miltech_ng_test` in parallel.
- Use test-first sequencing within each task. Observe each new test fail for the intended reason before adding the production/migration change that makes it pass.
- Make only the commits named in this plan, one logical commit per task. Do not push, merge, deploy, or touch production.

---

## Task 1: Encode the reversible migration contract

**Files:**

- Create: `api/pmcs_sbs_progress/table_rename_migration_test.go`
- Create: `migrations/014_rename_pmcs_tables.sql`
- Create: `migrations/014_rollback_rename_pmcs_tables.sql`

### 1.1 Write the failing static migration test

- [ ] Add `api/pmcs_sbs_progress/table_rename_migration_test.go` in package `pmcs_sbs_progress`. The test must read both SQL files relative to the package, normalize whitespace only for diagnostics, and assert:

  - each file contains `BEGIN`, `SET LOCAL lock_timeout`, one deterministic three-table `LOCK TABLE ... IN ACCESS EXCLUSIVE MODE`, and `COMMIT`;
  - the forward file contains every exact old-to-new table, constraint, and explicit-index rename once;
  - the rollback file contains the exact inverse of every rename once;
  - both files contain ordinary-table/source-presence and destination-absence assertions;
  - neither file contains `DROP TABLE`, `CREATE TABLE`, `CASCADE`, or `IF EXISTS` (case-insensitive);
  - neither file attempts a separate `ALTER INDEX` for a primary-key backing index.

  Use a table-driven mapping so omission of any approved object fails visibly:

  ```go
  var tableRenameStatements = [][2]string{
      {"ALTER TABLE public.pmcs_sbs_inspections RENAME TO user_pmcs_inspections;", "ALTER TABLE public.user_pmcs_inspections RENAME TO pmcs_sbs_inspections;"},
      {"ALTER TABLE public.pmcs_sbs_faults RENAME TO user_pmcs_faults;", "ALTER TABLE public.user_pmcs_faults RENAME TO pmcs_sbs_faults;"},
      {"ALTER TABLE public.pmcs_sbs_inspection_comments RENAME TO user_pmcs_inspection_comments;", "ALTER TABLE public.user_pmcs_inspection_comments RENAME TO pmcs_sbs_inspection_comments;"},
  }

  var explicitIndexRenameStatements = [][2]string{
      {"ALTER INDEX public.idx_pmcs_sbs_inspections_equipment_performed RENAME TO user_pmcs_inspections_equipment_performed_idx;", "ALTER INDEX public.user_pmcs_inspections_equipment_performed_idx RENAME TO idx_pmcs_sbs_inspections_equipment_performed;"},
      {"ALTER INDEX public.idx_pmcs_sbs_inspection_comments_pmcs_id RENAME TO user_pmcs_inspection_comments_pmcs_id_idx;", "ALTER INDEX public.user_pmcs_inspection_comments_pmcs_id_idx RENAME TO idx_pmcs_sbs_inspection_comments_pmcs_id;"},
  }
  ```

  Add the following 15 constraint pairs to the same test as full `ALTER TABLE ... RENAME CONSTRAINT ... TO ...;` statements:

  | Target table | Legacy constraint | Target constraint |
  |---|---|---|
  | `user_pmcs_inspections` | `pmcs_sbs_inspections_pkey` | `user_pmcs_inspections_pkey` |
  | `user_pmcs_inspections` | `fk_pmcs_sbs_inspections_equipment_id` | `fk_user_pmcs_inspections_equipment_id` |
  | `user_pmcs_inspections` | `fk_pmcs_sbs_inspections_performed_by` | `fk_user_pmcs_inspections_performed_by` |
  | `user_pmcs_inspections` | `pmcs_sbs_inspections_equipment_id_nonblank_check` | `user_pmcs_inspections_equipment_id_nonblank_check` |
  | `user_pmcs_inspections` | `pmcs_sbs_inspections_source_shape_check` | `user_pmcs_inspections_source_shape_check` |
  | `user_pmcs_inspections` | `pmcs_sbs_inspections_source_type_check` | `user_pmcs_inspections_source_type_check` |
  | `user_pmcs_faults` | `pmcs_sbs_faults_pkey` | `user_pmcs_faults_pkey` |
  | `user_pmcs_faults` | `fk_pmcs_sbs_faults_pmcs_id` | `fk_user_pmcs_faults_pmcs_id` |
  | `user_pmcs_faults` | `pmcs_sbs_faults_item_index_check` | `user_pmcs_faults_item_index_check` |
  | `user_pmcs_faults` | `pmcs_sbs_faults_nonblank_fields_check` | `user_pmcs_faults_nonblank_fields_check` |
  | `user_pmcs_faults` | `pmcs_sbs_faults_status_check` | `user_pmcs_faults_status_check` |
  | `user_pmcs_inspection_comments` | `pmcs_sbs_inspection_comments_pkey` | `user_pmcs_inspection_comments_pkey` |
  | `user_pmcs_inspection_comments` | `fk_pmcs_sbs_inspection_comments_author_id` | `fk_user_pmcs_inspection_comments_author_id` |
  | `user_pmcs_inspection_comments` | `fk_pmcs_sbs_inspection_comments_pmcs_id` | `fk_user_pmcs_inspection_comments_pmcs_id` |
  | `user_pmcs_inspection_comments` | `pmcs_sbs_inspection_comments_nonblank_check` | `user_pmcs_inspection_comments_nonblank_check` |

- [ ] Run the focused test and confirm it fails only because migration `014` files do not yet exist:

  ```bash
  go test ./api/pmcs_sbs_progress -run TestTableRenameMigrationContract -count=1
  ```

### 1.2 Add the exact forward migration

- [ ] Create `migrations/014_rename_pmcs_tables.sql` with this transaction. Keep the fixed lock order and do not weaken the assertions with `IF EXISTS`:

  ```sql
  -- User PMCS persistence table naming
  -- Migration: 014_rename_pmcs_tables.sql
  -- Metadata-only rename; no rows, columns, or relationship definitions change.

  BEGIN;

  SET LOCAL lock_timeout = '2s';
  SET LOCAL statement_timeout = '30s';

  DO $$
  DECLARE
      relation_name TEXT;
  BEGIN
      FOREACH relation_name IN ARRAY ARRAY[
          'pmcs_sbs_inspections',
          'pmcs_sbs_faults',
          'pmcs_sbs_inspection_comments'
      ]
      LOOP
          IF NOT EXISTS (
              SELECT 1
              FROM pg_catalog.pg_class AS class
              JOIN pg_catalog.pg_namespace AS namespace
                ON namespace.oid = class.relnamespace
              WHERE namespace.nspname = 'public'
                AND class.relname = relation_name
                AND class.relkind = 'r'
          ) THEN
              RAISE EXCEPTION 'expected ordinary table public.% before migration 014', relation_name;
          END IF;
      END LOOP;

      FOREACH relation_name IN ARRAY ARRAY[
          'user_pmcs_inspections',
          'user_pmcs_faults',
          'user_pmcs_inspection_comments'
      ]
      LOOP
          IF pg_catalog.to_regclass(format('public.%I', relation_name)) IS NOT NULL THEN
              RAISE EXCEPTION 'destination relation public.% already exists before migration 014', relation_name;
          END IF;
      END LOOP;
  END
  $$;

  LOCK TABLE
      public.pmcs_sbs_inspections,
      public.pmcs_sbs_faults,
      public.pmcs_sbs_inspection_comments
      IN ACCESS EXCLUSIVE MODE;

  ALTER TABLE public.pmcs_sbs_inspections RENAME TO user_pmcs_inspections;
  ALTER TABLE public.pmcs_sbs_faults RENAME TO user_pmcs_faults;
  ALTER TABLE public.pmcs_sbs_inspection_comments RENAME TO user_pmcs_inspection_comments;

  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT pmcs_sbs_inspections_pkey TO user_pmcs_inspections_pkey;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_equipment_id TO fk_user_pmcs_inspections_equipment_id;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_pmcs_sbs_inspections_performed_by TO fk_user_pmcs_inspections_performed_by;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT pmcs_sbs_inspections_equipment_id_nonblank_check TO user_pmcs_inspections_equipment_id_nonblank_check;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT pmcs_sbs_inspections_source_shape_check TO user_pmcs_inspections_source_shape_check;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT pmcs_sbs_inspections_source_type_check TO user_pmcs_inspections_source_type_check;

  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT pmcs_sbs_faults_pkey TO user_pmcs_faults_pkey;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT fk_pmcs_sbs_faults_pmcs_id TO fk_user_pmcs_faults_pmcs_id;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT pmcs_sbs_faults_item_index_check TO user_pmcs_faults_item_index_check;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT pmcs_sbs_faults_nonblank_fields_check TO user_pmcs_faults_nonblank_fields_check;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT pmcs_sbs_faults_status_check TO user_pmcs_faults_status_check;

  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT pmcs_sbs_inspection_comments_pkey TO user_pmcs_inspection_comments_pkey;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_pmcs_sbs_inspection_comments_author_id TO fk_user_pmcs_inspection_comments_author_id;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_pmcs_sbs_inspection_comments_pmcs_id TO fk_user_pmcs_inspection_comments_pmcs_id;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT pmcs_sbs_inspection_comments_nonblank_check TO user_pmcs_inspection_comments_nonblank_check;

  ALTER INDEX public.idx_pmcs_sbs_inspections_equipment_performed RENAME TO user_pmcs_inspections_equipment_performed_idx;
  ALTER INDEX public.idx_pmcs_sbs_inspection_comments_pmcs_id RENAME TO user_pmcs_inspection_comments_pmcs_id_idx;

  COMMIT;
  ```

### 1.3 Add the exact rollback migration

- [ ] Create `migrations/014_rollback_rename_pmcs_tables.sql` as the exact inverse. Rename explicit indexes and constraints before restoring table names:

  ```sql
  -- Rollback: 014_rollback_rename_pmcs_tables.sql

  BEGIN;

  SET LOCAL lock_timeout = '2s';
  SET LOCAL statement_timeout = '30s';

  DO $$
  DECLARE
      relation_name TEXT;
  BEGIN
      FOREACH relation_name IN ARRAY ARRAY[
          'user_pmcs_inspections',
          'user_pmcs_faults',
          'user_pmcs_inspection_comments'
      ]
      LOOP
          IF NOT EXISTS (
              SELECT 1
              FROM pg_catalog.pg_class AS class
              JOIN pg_catalog.pg_namespace AS namespace
                ON namespace.oid = class.relnamespace
              WHERE namespace.nspname = 'public'
                AND class.relname = relation_name
                AND class.relkind = 'r'
          ) THEN
              RAISE EXCEPTION 'expected ordinary table public.% before migration 014 rollback', relation_name;
          END IF;
      END LOOP;

      FOREACH relation_name IN ARRAY ARRAY[
          'pmcs_sbs_inspections',
          'pmcs_sbs_faults',
          'pmcs_sbs_inspection_comments'
      ]
      LOOP
          IF pg_catalog.to_regclass(format('public.%I', relation_name)) IS NOT NULL THEN
              RAISE EXCEPTION 'legacy relation public.% already exists before migration 014 rollback', relation_name;
          END IF;
      END LOOP;
  END
  $$;

  LOCK TABLE
      public.user_pmcs_inspections,
      public.user_pmcs_faults,
      public.user_pmcs_inspection_comments
      IN ACCESS EXCLUSIVE MODE;

  ALTER INDEX public.user_pmcs_inspections_equipment_performed_idx RENAME TO idx_pmcs_sbs_inspections_equipment_performed;
  ALTER INDEX public.user_pmcs_inspection_comments_pmcs_id_idx RENAME TO idx_pmcs_sbs_inspection_comments_pmcs_id;

  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_pkey TO pmcs_sbs_inspections_pkey;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_user_pmcs_inspections_equipment_id TO fk_pmcs_sbs_inspections_equipment_id;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT fk_user_pmcs_inspections_performed_by TO fk_pmcs_sbs_inspections_performed_by;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_equipment_id_nonblank_check TO pmcs_sbs_inspections_equipment_id_nonblank_check;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_source_shape_check TO pmcs_sbs_inspections_source_shape_check;
  ALTER TABLE public.user_pmcs_inspections RENAME CONSTRAINT user_pmcs_inspections_source_type_check TO pmcs_sbs_inspections_source_type_check;

  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_pkey TO pmcs_sbs_faults_pkey;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT fk_user_pmcs_faults_pmcs_id TO fk_pmcs_sbs_faults_pmcs_id;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_item_index_check TO pmcs_sbs_faults_item_index_check;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_nonblank_fields_check TO pmcs_sbs_faults_nonblank_fields_check;
  ALTER TABLE public.user_pmcs_faults RENAME CONSTRAINT user_pmcs_faults_status_check TO pmcs_sbs_faults_status_check;

  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT user_pmcs_inspection_comments_pkey TO pmcs_sbs_inspection_comments_pkey;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_user_pmcs_inspection_comments_author_id TO fk_pmcs_sbs_inspection_comments_author_id;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT fk_user_pmcs_inspection_comments_pmcs_id TO fk_pmcs_sbs_inspection_comments_pmcs_id;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME CONSTRAINT user_pmcs_inspection_comments_nonblank_check TO pmcs_sbs_inspection_comments_nonblank_check;

  ALTER TABLE public.user_pmcs_inspections RENAME TO pmcs_sbs_inspections;
  ALTER TABLE public.user_pmcs_faults RENAME TO pmcs_sbs_faults;
  ALTER TABLE public.user_pmcs_inspection_comments RENAME TO pmcs_sbs_inspection_comments;

  COMMIT;
  ```

### 1.4 Verify and commit the migration contract

- [ ] Run formatting and the focused contract test:

  ```bash
  gofmt -w api/pmcs_sbs_progress/table_rename_migration_test.go
  go test ./api/pmcs_sbs_progress -run TestTableRenameMigrationContract -count=1
  ```

- [ ] Inspect the migration diff and confirm no historical migration changed:

  ```bash
  git diff --check
  git diff -- migrations api/pmcs_sbs_progress/table_rename_migration_test.go
  git status --short
  ```

- [ ] Commit only the contract test and migration pair:

  ```bash
  git add api/pmcs_sbs_progress/table_rename_migration_test.go migrations/014_rename_pmcs_tables.sql migrations/014_rollback_rename_pmcs_tables.sql
  git commit -m "refactor(pmcs): add table rename migration"
  ```

---

## Task 2: Rehearse the database migration and prove schema/data preservation

**Files:**

- Create: `tests/pmcs_sbs_progress/schema_test.go`
- Modify: `tests/pmcs_sbs_progress/migration_rollback_test.go`

**Databases:**

- Modify: `miltech_ng_test` (forward, rollback, forward)
- Modify: `miltech_ng` (forward once)

### 2.1 Establish the pre-migration baseline

- [ ] Confirm the working tree contains only Task 1 plus pre-existing unrelated state. Record `git rev-parse HEAD` and `git status --short --branch`.

- [ ] Stop the local server and ensure no other Go test process is using either database. Inspect first with `ps`; terminate only positively identified local `miltechserver`/Go test processes. Do not kill PostgreSQL itself.

- [ ] Run the current behavior baseline before either schema changes:

  ```bash
  go test ./api/pmcs_sbs_progress -count=1
  go test ./tests/pmcs_sbs_progress -count=1
  go test ./tests/shops -run 'TestGetEquipmentPmcsHistory|TestEquipmentPmcsHistory' -count=1
  ```

  Record every nonzero result exactly. Do not classify an existing failure as caused by the rename without reproducing it after the change.

- [ ] Load the local `.env` without printing it, then assert the development connection is `miltech_ng`. For the test connection, use the same host, port, username, and password but explicitly pass `-d miltech_ng_test`:

  ```bash
  set -a
  source .env
  set +a
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng -Atc "SELECT current_database();"
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng_test -Atc "SELECT current_database();"
  ```

  Stop immediately unless the outputs are exactly `miltech_ng` and `miltech_ng_test`.

- [ ] Capture row counts and deterministic fingerprints immediately before each database is migrated. Save the output under `/tmp`, not the repository. Use a per-table ordering that contains the full primary key:

  ```sql
  SELECT 'inspections', count(*),
         coalesce(md5(string_agg(md5(to_jsonb(row_data)::text), '' ORDER BY id::text)), md5(''))
  FROM public.pmcs_sbs_inspections AS row_data;

  SELECT 'faults', count(*),
         coalesce(md5(string_agg(md5(to_jsonb(row_data)::text), '' ORDER BY pmcs_id::text, section_id, item_index)), md5(''))
  FROM public.pmcs_sbs_faults AS row_data;

  SELECT 'comments', count(*),
         coalesce(md5(string_agg(md5(to_jsonb(row_data)::text), '' ORDER BY id::text)), md5(''))
  FROM public.pmcs_sbs_inspection_comments AS row_data;
  ```

  Also capture catalog definitions from `pg_attribute`, `pg_constraint`, and `pg_indexes`, including `pg_get_constraintdef(oid, true)` and `indexdef`. These snapshots are evidence, not files to commit.

### 2.2 Add a failing target-schema integrity test

- [ ] Create `tests/pmcs_sbs_progress/schema_test.go`. Reuse `requirePmcsSbsTestDatabase` before any catalog assertion. Add `TestUserPmcsPersistenceSchema` that proves:

  - `user_pmcs_inspections`, `user_pmcs_faults`, and `user_pmcs_inspection_comments` exist in `public` with `relkind = 'r'`;
  - all three legacy names return no catalog row;
  - the ordered logical columns are exactly:

    ```go
    var expectedColumns = map[string][]string{
        "user_pmcs_inspections": {
            "id", "equipment_id", "guide_manual", "performed_date", "performed_by",
            "created_at", "updated_at", "notes", "source_type", "custom_checklist_id",
            "custom_revision_id", "custom_revision_number", "custom_checklist_name",
        },
        "user_pmcs_faults": {
            "pmcs_id", "section_id", "item_index", "item_no", "status", "fault_text",
            "corrective_action", "created_at", "updated_at", "section_title",
        },
        "user_pmcs_inspection_comments": {
            "id", "pmcs_id", "author_id", "text", "created_at", "updated_at",
        },
    }
    ```

    Query `pg_attribute` ordered by `attnum`, filtering `attnum > 0 AND NOT attisdropped`. Do not assert raw `attnum` equality across databases because the test DB has historical dropped-column gaps.

  - column type, nullability, and default signatures match an explicit expected table using `format_type(atttypid, atttypmod)`, `attnotnull`, and `pg_get_expr(adbin, adrelid)`;
  - the exact 15 target constraint names exist and their normalized `pg_get_constraintdef` values preserve PK columns, FK targets/actions, and CHECK expressions;
  - the two renamed secondary indexes retain `(equipment_id, performed_date DESC)` and `(pmcs_id, created_at)` respectively;
  - the three target primary-key indexes exist under their target constraint names;
  - `to_regclass('public.pmcs_sbs_*')` is null, so no compatibility view or alias remains.

- [ ] Run only the new schema test and confirm it fails because the target tables do not yet exist:

  ```bash
  go test ./tests/pmcs_sbs_progress -run TestUserPmcsPersistenceSchema -count=1
  ```

### 2.3 Replace the obsolete rollback test with an atomic lock-timeout test

- [ ] In `tests/pmcs_sbs_progress/migration_rollback_test.go`, replace `TestInspectionSourceRollbackWaitsForConcurrentCustomInsertAndRefusesExplicitly`. Migration `011` remains historical and unchanged, but executing its rollback directly against the post-014 current schema is no longer valid.

  Add `TestTableRenameRollbackTimesOutAtomically` with this sequence:

  1. require `miltech_ng_test` on the base connection and both dedicated connections;
  2. read `../../migrations/014_rollback_rename_pmcs_tables.sql`;
  3. begin a holder transaction and run `LOCK TABLE public.user_pmcs_inspections IN ACCESS SHARE MODE`;
  4. execute the rollback file on the second connection;
  5. require a PostgreSQL lock-timeout error within the migration's two-second bound;
  6. roll back any aborted connection state;
  7. query all six relation names and prove all three target names still exist while all legacy names remain absent;
  8. roll back the holder transaction.

  The central assertion should be explicit:

  ```go
  _, rollbackErr := rollbackConnection.ExecContext(ctx, string(rollbackSQL))
  require.Error(t, rollbackErr)
  require.Contains(t, strings.ToLower(rollbackErr.Error()), "lock timeout")
  requireTargetTableNamesOnly(t, ctx, testDB)
  ```

  Retain `requirePmcsSbsTestDatabase`. Delete `requireConnectionWaitingForTableLock` only if no remaining test uses it.

### 2.4 Rehearse forward, rollback, and forward on `miltech_ng_test`

- [ ] Apply the forward migration to test only, with transaction errors fatal:

  ```bash
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng_test -f migrations/014_rename_pmcs_tables.sql
  ```

- [ ] Re-run the target-schema test. If its expected definitions expose a pre-existing catalog assumption, compare it to the pre-migration catalog snapshot before changing the assertion:

  ```bash
  go test ./tests/pmcs_sbs_progress -run TestUserPmcsPersistenceSchema -count=1
  ```

- [ ] Recompute fingerprints from the target tables by changing only the three `FROM` names. Compare byte-for-byte with the immediately preceding test snapshot.

- [ ] Apply rollback to `miltech_ng_test`, then verify the three legacy tables and original object names are restored, target names are absent, and fingerprints still match:

  ```bash
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng_test -f migrations/014_rollback_rename_pmcs_tables.sql
  ```

- [ ] Apply the forward migration a second time. Leave `miltech_ng_test` on the target schema. Run the schema test and the new lock-timeout atomicity test:

  ```bash
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng_test -f migrations/014_rename_pmcs_tables.sql
  go test ./tests/pmcs_sbs_progress -run 'TestUserPmcsPersistenceSchema|TestTableRenameRollbackTimesOutAtomically' -count=1
  ```

### 2.5 Apply forward once on `miltech_ng`

- [ ] Reconfirm `current_database()`, recapture the development fingerprint immediately before migration, and apply only the forward file:

  ```bash
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng -Atc "SELECT current_database();"
  PGPASSWORD="$DB_PASSWORD" psql -X -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d miltech_ng -f migrations/014_rename_pmcs_tables.sql
  ```

- [ ] Do not rehearse rollback on development. Query the target names and compare target-table counts/fingerprints with the immediately preceding development snapshot. At the analyzed baseline this was 8 inspections, 13 faults, and 2 comments; if legitimate local activity changed the counts, the immediate before/after fingerprints—not the old baseline—are authoritative.

- [ ] Compare the two databases' logical target column/constraint/index definitions. Ignore internal OIDs and dropped-column `attnum` gaps.

### 2.6 Verify and commit the database tests

- [ ] Format and run the new migration-sensitive tests:

  ```bash
  gofmt -w tests/pmcs_sbs_progress/schema_test.go tests/pmcs_sbs_progress/migration_rollback_test.go
  go test ./tests/pmcs_sbs_progress -run 'TestUserPmcsPersistenceSchema|TestTableRenameRollbackTimesOutAtomically' -count=1
  git diff --check
  ```

- [ ] Commit only the schema and rollback-atomicity tests:

  ```bash
  git add tests/pmcs_sbs_progress/schema_test.go tests/pmcs_sbs_progress/migration_rollback_test.go
  git commit -m "test(pmcs): verify renamed persistence schema"
  ```

---

## Task 3: Regenerate Jet and update all server references

**Files:**

- Delete: `.gen/miltech_ng/public/model/pmcs_sbs_inspections.go`
- Delete: `.gen/miltech_ng/public/model/pmcs_sbs_faults.go`
- Delete: `.gen/miltech_ng/public/table/pmcs_sbs_inspections.go`
- Delete: `.gen/miltech_ng/public/table/pmcs_sbs_faults.go`
- Add: `.gen/miltech_ng/public/model/user_pmcs_inspections.go`
- Add: `.gen/miltech_ng/public/model/user_pmcs_faults.go`
- Add: `.gen/miltech_ng/public/table/user_pmcs_inspections.go`
- Add: `.gen/miltech_ng/public/table/user_pmcs_faults.go`
- Regenerate locally but do not newly track: `.gen/miltech_ng/public/model/user_pmcs_inspection_comments.go`
- Regenerate locally but do not newly track: `.gen/miltech_ng/public/table/user_pmcs_inspection_comments.go`
- Regenerate locally but do not newly track: `.gen/miltech_ng/public/table/table_use_schema.go`
- Modify: `api/pmcs_sbs_progress/repository.go`
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl.go`
- Modify: `api/pmcs_sbs_progress/service_impl_test.go`
- Modify: `api/shops/aggregates/repository_impl.go`
- Modify: `tests/pmcs_sbs_progress/helpers_test.go`
- Modify: `tests/pmcs_sbs_progress/repository_test.go`
- Modify: `tests/shops/helpers_test.go`
- Modify: `tests/shops/shops_equipment_pmcs_history_test.go`

### 3.1 Regenerate Jet from migrated development

- [ ] Confirm `miltech_ng` contains all three target tables and no legacy table before generation.

- [ ] A fresh worktree can lack ignored generated files, while `main.go` imports packages that require them. Create a temporary, untracked `tools/jetregen/main.go` solely to run the canonical generator without compiling the server graph. Its generator configuration must exactly mirror `main.go:56-96`:

  ```go
  package main

  import (
      "fmt"
      "log"

      "miltechserver/bootstrap"

      "github.com/go-jet/jet/v2/generator/metadata"
      "github.com/go-jet/jet/v2/generator/postgres"
      "github.com/go-jet/jet/v2/generator/template"
      postgresDialect "github.com/go-jet/jet/v2/postgres"
  )

  func main() {
      env := bootstrap.NewEnv()
      if env.DBName != "miltech_ng" || env.DBSchema != "public" {
          log.Fatalf("refusing Jet generation for database %q schema %q", env.DBName, env.DBSchema)
      }

      err := postgres.Generate(
          "./.gen",
          postgres.DBConnection{
              Host: env.Host, Port: 5432, User: env.Username, Password: env.Password,
              SslMode: env.SslMode, DBName: env.DBName, SchemaName: env.DBSchema,
          },
          template.Default(postgresDialect.Dialect).UseSchema(func(schema metadata.Schema) template.Schema {
              return template.DefaultSchema(schema).UseModel(
                  template.DefaultModel().
                      UseTable(func(table metadata.Table) template.TableModel {
                          return template.DefaultTableModel(table).UseField(jsonTaggedField)
                      }).
                      UseView(func(table metadata.Table) template.TableModel {
                          return template.DefaultTableModel(table).UseField(jsonTaggedField)
                      }),
              )
          }),
      )
      if err != nil {
          log.Fatalf("error generating code: %s", err)
      }
  }

  func jsonTaggedField(column metadata.Column) template.TableModelField {
      return template.DefaultTableModelField(column).UseTags(fmt.Sprintf(`json:"%s"`, column.Name))
  }
  ```

  Set `DEBUG=true` so `bootstrap.NewEnv()` loads `.env`, run `go run ./tools/jetregen`, and delete the temporary file/directory afterward. The harness must never be committed.

- [ ] Audit the complete `.gen/miltech_ng/public` result before touching handwritten Go:

  - generated symbols are exactly `model.UserPmcsInspections`, `table.UserPmcsInspections`, `model.UserPmcsFaults`, `table.UserPmcsFaults`, `model.UserPmcsInspectionComments`, and `table.UserPmcsInspectionComments`;
  - table string literals are exactly the three target physical names;
  - JSON tags remain present;
  - columns and Go types are unchanged;
  - no unrelated table/model output drifted.

  If generation changes anything beyond the three renamed objects and schema registry entries, stop and identify the catalog drift before proceeding.

### 3.2 Make the core repository/service compile against target symbols

- [ ] First run the package test after generation but before handwritten replacements. Confirm compilation fails on legacy Jet identifiers; this is the intended red state:

  ```bash
  go test ./api/pmcs_sbs_progress -count=1
  ```

- [ ] In `api/pmcs_sbs_progress/repository.go`, replace only generated persistence types:

  - `model.PmcsSbsInspections` -> `model.UserPmcsInspections`;
  - `model.PmcsSbsFaults` -> `model.UserPmcsFaults`;
  - embedded `model.PmcsSbsInspectionComments` -> `model.UserPmcsInspectionComments`.

  Keep interface method names, `InspectionDetail`, `InspectionSummary`, `FaultKey`, and `CommentWithAuthor` unchanged.

- [ ] In `api/pmcs_sbs_progress/repository_impl.go`, mechanically replace the three Jet table variables and generated model types:

  ```go
  PmcsSbsInspections         -> UserPmcsInspections
  PmcsSbsFaults              -> UserPmcsFaults
  PmcsSbsInspectionComments  -> UserPmcsInspectionComments
  model.PmcsSbsInspections   -> model.UserPmcsInspections
  model.PmcsSbsFaults        -> model.UserPmcsFaults
  model.PmcsSbsInspectionComments -> model.UserPmcsInspectionComments
  ```

  Update anonymous/embedded struct field selectors generated from embedded type names. Preserve query structure, transactions, joins, order clauses, conflicts, authorization predicates, errors, and exported method signatures except for internal generated types.

- [ ] In `api/pmcs_sbs_progress/service_impl.go`, update:

  - `detail.PmcsSbsInspections` -> `detail.UserPmcsInspections`;
  - request-validation return types at the current `validateInspectionRequest`/fault mapping boundaries;
  - response mapping helpers accepting old inspection/fault/comment generated types.

  Do not rename response types or change JSON output.

- [ ] In `api/pmcs_sbs_progress/service_impl_test.go`, update `repoStub` signatures, fixtures, embedded literals, and embedded field selectors to the new generated types. Do not rewrite behavioral expectations.

- [ ] Format and run the core package:

  ```bash
  gofmt -w api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go
  go test ./api/pmcs_sbs_progress -count=1
  ```

### 3.3 Update Shop aggregation without changing its four-query contract

- [ ] In `api/shops/aggregates/repository_impl.go:1128-1205`, replace the embedded inspection model and all three Jet table variables with the target identifiers. Preserve:

  - the anonymous destination struct (do not turn it into a named type);
  - equipment ordering;
  - inspection ordering;
  - one inspection query, one grouped fault-count query, and one grouped comment-count query;
  - response DTO field names and source provenance mapping.

- [ ] Format and run compile/focused unit coverage that does not mutate the shared test database:

  ```bash
  gofmt -w api/shops/aggregates/repository_impl.go
  go test ./api/shops/aggregates -count=1
  ```

### 3.4 Update raw SQL and generated fixtures in integration tests

- [ ] In `tests/pmcs_sbs_progress/helpers_test.go`, update truncate names and fixture types:

  - `user_pmcs_faults`;
  - `user_pmcs_inspection_comments`;
  - `user_pmcs_inspections`;
  - `model.UserPmcsInspections`;
  - `model.UserPmcsFaults`.

  Preserve child-before-parent truncate order and `RESTART IDENTITY CASCADE`.

- [ ] In `tests/pmcs_sbs_progress/repository_test.go`, replace every raw legacy table identifier and generated model type. Preserve all existing cases for source constraints, conflict semantics, ordering, faults, comments, authorization, and cascades.

- [ ] In `tests/shops/helpers_test.go`, change only PMCS fixture insert table names to the target names.

- [ ] In `tests/shops/shops_equipment_pmcs_history_test.go`, change PMCS raw inserts and the query-counting SQL substring assertions to target names. Keep the assertion at exactly four top-level queries and preserve guide/custom/count/order behavior.

- [ ] Run formatting and the database-backed affected suites sequentially:

  ```bash
  gofmt -w tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go tests/shops/helpers_test.go tests/shops/shops_equipment_pmcs_history_test.go
  go test ./tests/pmcs_sbs_progress -count=1
  go test ./tests/shops -run 'TestGetEquipmentPmcsHistory|TestEquipmentPmcsHistory' -count=1
  ```

### 3.5 Audit legacy references and generated-file tracking

- [ ] Search all current Go and generated code. Zero legacy persistence identifiers should remain outside historical migration/documentation context:

  ```bash
  rg -n 'PmcsSbs(Inspections|Faults|InspectionComments)|pmcs_sbs_(inspections|faults|inspection_comments)' api tests .gen/miltech_ng/public
  ```

  Allowed hits at this point are limited to external/domain names that do not spell a generated persistence type or physical table. Inspect every hit; do not blanket-replace `PmcsSbs` domain/API terminology.

- [ ] Use `git status --ignored --short .gen/miltech_ng/public` and `git diff -- .gen/miltech_ng/public` to verify:

  - four tracked old inspection/fault generated files are deleted;
  - four tracked target inspection/fault replacements are added;
  - the target comment model/table and registry exist locally as ignored generated output;
  - no unrelated tracked generated file changed.

- [ ] Run compile-only coverage before committing:

  ```bash
  go test ./... -run '^$'
  git diff --check
  ```

- [ ] Stage handwritten code/tests plus only the four intended tracked generated replacements/deletions. Because `.gen/miltech_ng` is ignored, add the four new files explicitly with `git add -f`; do not force-add the directory:

  ```bash
  git add api/pmcs_sbs_progress/repository.go api/pmcs_sbs_progress/repository_impl.go api/pmcs_sbs_progress/service_impl.go api/pmcs_sbs_progress/service_impl_test.go api/shops/aggregates/repository_impl.go tests/pmcs_sbs_progress/helpers_test.go tests/pmcs_sbs_progress/repository_test.go tests/shops/helpers_test.go tests/shops/shops_equipment_pmcs_history_test.go
  git add -u .gen/miltech_ng/public/model/pmcs_sbs_inspections.go .gen/miltech_ng/public/model/pmcs_sbs_faults.go .gen/miltech_ng/public/table/pmcs_sbs_inspections.go .gen/miltech_ng/public/table/pmcs_sbs_faults.go
  git add -f .gen/miltech_ng/public/model/user_pmcs_inspections.go .gen/miltech_ng/public/model/user_pmcs_faults.go .gen/miltech_ng/public/table/user_pmcs_inspections.go .gen/miltech_ng/public/table/user_pmcs_faults.go
  git diff --cached --check
  git diff --cached --stat
  git commit -m "refactor(pmcs): adopt user PMCS table names"
  ```

---

## Task 4: Document the persistence rename

**Files:**

- Modify: `docs/api/pmcs_sbs_inspections_mobile.md`
- Modify: `docs/project_notes/decisions.md`

### 4.1 Add current API persistence documentation

- [ ] Near the top of `docs/api/pmcs_sbs_inspections_mobile.md`, add a short `## Server persistence` section stating:

  - PostgreSQL uses `user_pmcs_inspections`, `user_pmcs_faults`, and `user_pmcs_inspection_comments`;
  - these names are internal and do not change `/pmcs-sbs` routes, JSON, or `pmcs_id`;
  - server PostgreSQL `user_pmcs_inspections` is distinct from the Flutter device's SQLite/Drift table of the same name.

  Do not rewrite endpoint examples or external PMCS SBS terminology.

### 4.2 Record ADR-021 without rewriting history

- [ ] Append `ADR-021: Rename PMCS SBS Persistence Tables to User PMCS Convention (2026-08-16)` to `docs/project_notes/decisions.md` with:

  - context: the three physical names violated the established `user_pmcs_` convention;
  - decision: metadata-only transactional rename plus constraint/index rename and canonical Jet regeneration;
  - consequences: old/new binaries require matching schema; no API/data/behavior change; migration `014` is reversible and development-only execution was rehearsed;
  - alternatives rejected: compatibility views, copy/swap, and dual writes.

  Preserve ADR-017, ADR-018, ADR-020, migration comments, old plans, and old specs as period-correct history.

### 4.3 Verify and commit documentation

- [ ] Review only the current-doc diff and scan for ambiguity between server PostgreSQL and device SQLite:

  ```bash
  git diff --check
  git diff -- docs/api/pmcs_sbs_inspections_mobile.md docs/project_notes/decisions.md
  ```

- [ ] Commit the two current documentation changes:

  ```bash
  git add docs/api/pmcs_sbs_inspections_mobile.md docs/project_notes/decisions.md
  git commit -m "docs(pmcs): document renamed persistence tables"
  ```

---

## Task 5: Run final verification and independent review

**Files:** None expected. Fixes require returning to the owning task and repeating its focused verification.

### 5.1 Verify both database end states

- [ ] For each database, assert the exact name before the rest of the query and record the final catalog evidence:

  - all three target relations exist as ordinary tables;
  - all three legacy relations are absent;
  - all 15 target constraints and five indexes (three PK plus two secondary) exist;
  - constraint and index definitions match the pre-migration definitions except approved names;
  - no triggers, views, materialized views, routines, RLS policies, publications, or new grants were introduced;
  - replica identity remains default;
  - final row counts and fingerprints match each database's immediate pre-migration snapshot.

- [ ] Confirm `miltech_ng_test` and `miltech_ng` have the same logical target schema, allowing only internal OID/ordinal differences.

### 5.2 Run focused, compile, full, and race verification

- [ ] Run affected suites, keeping the two database-backed packages sequential:

  ```bash
  go test ./api/pmcs_sbs_progress -count=1
  go test ./api/shops/aggregates -count=1
  go test ./tests/pmcs_sbs_progress -count=1
  go test ./tests/shops -run 'TestGetEquipmentPmcsHistory|TestEquipmentPmcsHistory' -count=1
  ```

- [ ] Run compile-only and full repository coverage:

  ```bash
  go test ./... -run '^$'
  go test -p 1 ./... -count=1
  ```

  Report any nonzero command, package, and failing test exactly. Compare failures with the Task 2 baseline; do not call a command green when it exited nonzero.

- [ ] Run race coverage only on affected unit packages that do not destructively share `miltech_ng_test`:

  ```bash
  go test -race ./api/pmcs_sbs_progress ./api/shops/aggregates -count=1
  ```

### 5.3 Final source and scope audit

- [ ] Run a repository-wide search and classify every remaining legacy-name hit:

  ```bash
  rg -n 'pmcs_sbs_(inspections|faults|inspection_comments)|PmcsSbs(Inspections|Faults|InspectionComments)' . --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**'
  ```

  Expected snake-case hits are confined to migrations `006`-`011`, the inverse statements in migration `014`, and period-correct historical ADR text. Current Go, tests, generated target files, and current API persistence documentation must use target storage names. Domain/API singular terms such as `PmcsSbsInspectionResponse` remain valid and must not be renamed mechanically.

- [ ] Confirm external contracts did not change by reviewing route and response diffs:

  ```bash
  git diff HEAD~4..HEAD -- api/pmcs_sbs_progress api/route docs/api/pmcs_sbs_inspections_mobile.md
  ```

  There must be no route path, JSON tag, status code, authorization, ordering, or response-shape change.

- [ ] Confirm repository state:

  ```bash
  git diff --check
  git status --short --branch
  git log --oneline -5
  ```

  The only untracked pre-existing item should remain `.worktrees/`; the temporary Jet harness must be gone.

### 5.4 Request independent whole-branch review

- [ ] Use `superpowers:requesting-code-review` for an independent review of the complete range from the design/spec commit through current `HEAD`. The reviewer must check:

  - exact approved table/constraint/index map and inverse rollback;
  - migration atomicity and lock-timeout behavior;
  - before/after data and catalog evidence for both development databases;
  - complete Jet/Go/raw-SQL reference migration;
  - unchanged external contracts and Shop four-query behavior;
  - generated diff/tracking scope;
  - test evidence and baseline-failure classification;
  - no production access and no unrelated worktree changes.

- [ ] Address every confirmed finding in the task that owns it, rerun that task's focused checks, then repeat the final verification commands. Do not amend or squash unless the user explicitly asks.

### 5.5 Handoff

- [ ] Report:

  - the final commit list and exact `HEAD`;
  - `miltech_ng_test` forward/rollback/forward evidence;
  - `miltech_ng` forward-only evidence;
  - final counts/fingerprints and catalog comparison for both databases;
  - every focused/full/compile/race command with exit status;
  - any baseline failures separately and exactly;
  - independent review result;
  - final `git status --short --branch`, including untouched `.worktrees/`;
  - explicit confirmation that production, push, merge, deploy, and Flutter were not touched.
