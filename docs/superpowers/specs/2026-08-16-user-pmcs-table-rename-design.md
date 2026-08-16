# User PMCS Server Table Rename Design

Date: 2026-08-16
Status: Approved for implementation planning

## Problem

Three PostgreSQL tables that persist Shop PMCS inspections use the legacy
`pmcs_sbs_` prefix while the related server-side PMCS tables use the
`user_pmcs_` prefix:

- `pmcs_sbs_inspections`;
- `pmcs_sbs_faults`; and
- `pmcs_sbs_inspection_comments`.

The inconsistency extends into generated Jet identifiers, handwritten Go
repositories, raw SQL in integration tests, constraint names, and index
names. The database names are internal implementation details; changing them
must not alter the PMCS SBS HTTP API, JSON representations, authorization,
inspection identity, history ordering, fault behavior, comments, or Shop
equipment-history aggregation.

## Goal

Rename the three tables and their table-derived database objects without
copying or discarding rows:

| Current table | Target table |
|---|---|
| `pmcs_sbs_inspections` | `user_pmcs_inspections` |
| `pmcs_sbs_faults` | `user_pmcs_faults` |
| `pmcs_sbs_inspection_comments` | `user_pmcs_inspection_comments` |

Update Jet-generated and handwritten server references so all existing
functionality continues to use the same columns, relationships, queries, and
external contracts through the new physical names.

## Scope

This design covers:

- the non-production PostgreSQL databases `miltech_ng_test` and `miltech_ng`;
- a forward and rollback migration;
- constraint and index names belonging to the three renamed tables;
- canonical Jet regeneration;
- affected Go repositories and tests;
- current schema and API documentation; and
- migration rehearsal and data-preservation verification.

Both databases may be stopped and freely migrated. No production database,
production deployment, or zero-downtime compatibility layer is in scope.

## Non-goals

The change does not rename or alter:

- `/api/v1/auth/pmcs-sbs/...` routes;
- request or response JSON;
- `pmcs_id` fields or columns;
- `pmcs_sbs/` guide-manual paths;
- the `api/pmcs_sbs_progress` Go package;
- PMCS domain terminology presented to users;
- Flutter code or its independent SQLite/Drift tables;
- table columns, data types, nullability, defaults, or data;
- primary-key, foreign-key, check-constraint, or cascade semantics; or
- historical migrations `006` through `013`.

Historical documents may continue to describe the names that existed at the
time of the recorded decision. Current documentation must identify the new
names without rewriting history.

## Current-state evidence

The live catalog was inspected read-only on 2026-08-16.

Both databases run PostgreSQL 14.18 and contain the three legacy table names.
None of the three target names exists. The logical table shapes, 15
constraints, and five indexes are equivalent in both databases. Differences
in internal column ordinal numbers in `miltech_ng_test` are remnants of prior
add/drop migration rehearsals and do not represent a logical schema
difference.

Current row counts are:

| Database | Inspections | Faults | Comments |
|---|---:|---:|---:|
| `miltech_ng_test` | 0 | 0 | 0 |
| `miltech_ng` | 8 | 13 | 2 |

Neither database has triggers, views, materialized views, stored routines,
row-level security policies, publications, or non-owner grants attached to
these tables. All three use default replica identity. The only inbound table
relationships are the fault and comment foreign keys to the inspection
table. PostgreSQL owns these dependencies by object identifier and retains
them across a table rename.

The server accesses the tables in two runtime areas:

- `api/pmcs_sbs_progress` performs inspection, fault, and comment CRUD,
  authorization, transactional upserts, history queries, and cascaded
  deletion; and
- `api/shops/aggregates` batches inspections, fault counts, and comment counts
  into Equipment Details history.

Jet-generated model and table identifiers are used throughout those
repositories. Integration tests also use raw table names for fixture inserts,
cleanup, catalog assertions, row counts, and fixed-query-count assertions.

## Chosen approach

Use one metadata-only, transactional PostgreSQL rename migration followed by
canonical Jet regeneration and a mechanical handwritten-code migration.

`ALTER TABLE ... RENAME TO` changes catalog metadata and does not rewrite the
table or its rows. PostgreSQL-maintained dependencies continue to point to
the same object identifiers. The migration therefore preserves data and
relationships more safely than creating replacement tables and copying rows.

The operation requires `ACCESS EXCLUSIVE` locks. Because only the two
non-production databases are in scope, local application and test processes
will be stopped before migration. A bounded `lock_timeout` will make an
unexpected active session fail the entire transaction cleanly instead of
waiting indefinitely.

### Rejected alternatives

#### Compatibility views

Renaming the physical tables and exposing the legacy names as views would add
two schemas for one concept and would not safely support the repository's
`INSERT ... ON CONFLICT` write paths. Compatibility is unnecessary when both
database and application can be changed together.

#### Copy and swap

Creating new tables and copying data would require recreating and validating
constraints, indexes, privileges, and sequences, then proving the copied data
is complete. It creates more failure modes than an in-place catalog rename.

#### Dual-write rollout

A dual-write or trigger-based rollout is appropriate only when old and new
application versions must run concurrently. That requirement does not exist
for these development databases.

## Migration design

Add the next migration pair:

- `migrations/014_rename_pmcs_tables.sql`; and
- `migrations/014_rollback_rename_pmcs_tables.sql`.

The repository has no migration ledger or automatic migration runner. The
files are applied manually with `psql` after verifying `current_database()`.
Migration `014` remains the current-schema transition for fresh databases:
migrations `006` through `011` create and evolve the historical names, and
`014` renames the completed structures.

### Forward transaction

The forward file must:

1. start a transaction;
2. set a local lock timeout;
3. assert that each legacy relation exists as an ordinary table;
4. assert that each target relation is absent;
5. acquire `ACCESS EXCLUSIVE` locks in the fixed order inspections, faults,
   comments;
6. rename the inspection table, fault table, and comment table;
7. rename all table-derived constraints;
8. rename the two explicit secondary indexes; and
9. commit.

No `IF EXISTS`, `DROP`, `CREATE TABLE`, data-copy statement, or `CASCADE`
belongs in the forward migration. A partially applicable schema must fail
rather than be silently accepted. The transaction makes all three table
renames and dependent-name changes atomic.

### Constraint rename map

The constraint definitions remain unchanged. Only their names change.

| Current constraint | Target constraint |
|---|---|
| `pmcs_sbs_inspections_pkey` | `user_pmcs_inspections_pkey` |
| `fk_pmcs_sbs_inspections_equipment_id` | `fk_user_pmcs_inspections_equipment_id` |
| `fk_pmcs_sbs_inspections_performed_by` | `fk_user_pmcs_inspections_performed_by` |
| `pmcs_sbs_inspections_equipment_id_nonblank_check` | `user_pmcs_inspections_equipment_id_nonblank_check` |
| `pmcs_sbs_inspections_source_shape_check` | `user_pmcs_inspections_source_shape_check` |
| `pmcs_sbs_inspections_source_type_check` | `user_pmcs_inspections_source_type_check` |
| `pmcs_sbs_faults_pkey` | `user_pmcs_faults_pkey` |
| `fk_pmcs_sbs_faults_pmcs_id` | `fk_user_pmcs_faults_pmcs_id` |
| `pmcs_sbs_faults_item_index_check` | `user_pmcs_faults_item_index_check` |
| `pmcs_sbs_faults_nonblank_fields_check` | `user_pmcs_faults_nonblank_fields_check` |
| `pmcs_sbs_faults_status_check` | `user_pmcs_faults_status_check` |
| `pmcs_sbs_inspection_comments_pkey` | `user_pmcs_inspection_comments_pkey` |
| `fk_pmcs_sbs_inspection_comments_author_id` | `fk_user_pmcs_inspection_comments_author_id` |
| `fk_pmcs_sbs_inspection_comments_pmcs_id` | `fk_user_pmcs_inspection_comments_pmcs_id` |
| `pmcs_sbs_inspection_comments_nonblank_check` | `user_pmcs_inspection_comments_nonblank_check` |

Renaming a primary-key constraint also renames its backing index. It must not
be followed by a second `ALTER INDEX` for the primary key.

### Explicit index rename map

The new names follow the existing `user_pmcs_<purpose>_idx` convention.

| Current index | Target index |
|---|---|
| `idx_pmcs_sbs_inspections_equipment_performed` | `user_pmcs_inspections_equipment_performed_idx` |
| `idx_pmcs_sbs_inspection_comments_pmcs_id` | `user_pmcs_inspection_comments_pmcs_id_idx` |

Index definitions and sort directions do not change:

- inspections remain indexed by `(equipment_id, performed_date DESC)`; and
- comments remain indexed by `(pmcs_id, created_at ASC)`.

### Rollback transaction

The rollback performs the inverse operation in one transaction:

1. assert that all target tables exist and legacy tables are absent;
2. acquire the same locks in a deterministic order;
3. restore explicit index names;
4. restore constraint names; and
5. restore legacy table names.

The rollback is data-preserving. Application rollback and database rollback
must be coordinated because code generated for one set of table names cannot
query the other set.

## Jet generation and Go impact

Jet must be regenerated from migrated `miltech_ng`, never hand-edited. The
canonical programmatic generator and JSON-tag template in `main.go` are the
source of truth. A raw Jet CLI invocation that omits the tracked template is
not acceptable because it can remove JSON tags from unrelated generated
models.

The regeneration will delete the legacy generated files, create target-name
files, and update the generated schema registry. Expected generated symbols
include:

- `model.UserPmcsInspections` and `table.UserPmcsInspections`;
- `model.UserPmcsFaults` and `table.UserPmcsFaults`; and
- `model.UserPmcsInspectionComments` and
  `table.UserPmcsInspectionComments`.

The full `.gen/miltech_ng/public` diff must be audited. Expected changes are
limited to filenames, Go identifiers, table-name string literals, and schema
registration for the three renamed tables. Any unrelated generated schema
drift stops the task for investigation.

Handwritten Go changes are mechanical type/table-symbol replacements. Public
domain response types retain their existing names and JSON tags. Raw SQL in
integration tests changes only physical table identifiers.

## Database rehearsal sequence

### `miltech_ng_test`

1. Verify the target database name and stop other database-backed test suites.
2. Capture schema definitions, row counts, and deterministic row fingerprints
   for the three legacy tables.
3. Add a lock-contention test that proves the bounded timeout leaves every
   legacy name intact when a conflicting transaction holds a lock.
4. Apply the forward migration.
5. Verify target names, constraint/index names and definitions, preserved
   foreign-key actions, unchanged row fingerprints, and absence of all legacy
   names.
6. Run focused schema and PMCS behavior tests.
7. Apply the rollback migration.
8. Verify the complete legacy schema and unchanged fingerprints.
9. Apply the forward migration again and leave the test database migrated.

### `miltech_ng`

1. Verify the database name and stop the local server.
2. Capture the same pre-migration schema and data evidence.
3. Apply the forward migration once; do not use development for rollback
   rehearsal.
4. Verify the target schema and unchanged data evidence.
5. Run canonical Jet generation from this migrated schema.

The two databases must end in the same logical target schema. Internal column
ordinal numbers are not an equality requirement.

## Test design

### Schema integrity

Integration coverage must prove:

- all three target tables exist as ordinary tables;
- all three legacy names are absent;
- expected columns, data types, nullability, and defaults are unchanged;
- all 15 target constraints exist with the original definitions;
- both explicit indexes and three primary-key indexes exist with the original
  column order and direction;
- no unexpected view or alias preserves a legacy table name; and
- row counts and deterministic row fingerprints match pre-migration values.

The schema test must verify `current_database() = 'miltech_ng_test'` before it
performs migration-sensitive work.

### Functional regression

Existing tests, updated to the target names and generated types, must continue
to cover:

- guide and custom inspection creation;
- inspection UUID/source conflict behavior;
- inspection reads and date/id history ordering;
- transactional fault upsert and fault-list ordering;
- individual and bulk fault deletion;
- inspection notes;
- comment creation, author-only editing, and deleted-text replacement;
- clean inspections with no faults;
- vehicle authorization and non-disclosing cross-vehicle lookups;
- vehicle deletion cascading through inspections to faults and comments;
- inspection deletion cascading to faults and comments;
- performer deletion setting `performed_by` to null;
- comment-author foreign-key behavior;
- Shop Equipment Details history, fault counts, comment counts, guide/custom
  provenance, and fixed four-query behavior; and
- unchanged HTTP route and JSON contract tests.

### Verification commands

Implementation verification includes:

- focused `api/pmcs_sbs_progress` unit and route tests;
- `tests/pmcs_sbs_progress` integration tests;
- focused Shop PMCS history tests;
- compile-only `go test ./... -run '^$'`;
- full `go test ./... -count=1` with any pre-existing failures reported
  separately and exactly; and
- race tests for affected unit-test packages that do not destructively share
  the integration database.

Database-backed packages that share `miltech_ng_test` must run sequentially
or under the repository's existing advisory-lock discipline.

## Documentation

Current schema/API documentation must state the new physical server table
names. Historical ADRs, superseded contracts, and migrations retain their
period-correct names. No mobile application change is required because the
physical PostgreSQL names are not part of the HTTP contract.

The independent Flutter/Drift table named `user_pmcs_inspections` has a
different local schema and lifecycle. Documentation must qualify server
PostgreSQL versus device SQLite when both are discussed so the shared name is
not mistaken for shared storage.

## Risks and controls

### Old and new binaries are mutually schema-incompatible

The current binary emits SQL for `pmcs_sbs_*`; the regenerated binary emits
SQL for `user_pmcs_*`. Stop local server processes before migration and only
restart a binary compiled from the migrated generated files. Roll back both
code and schema together if backout is required.

### Lock acquisition can block

`ALTER TABLE ... RENAME` takes `ACCESS EXCLUSIVE`. Stop database users first,
use a local lock timeout, and test atomic failure under deliberate contention.

### Generated-code blast radius

Jet regeneration scans the complete schema. Use only the canonical tracked
template and reject unrelated generated diffs.

### Test database cross-package interference

Multiple integration packages truncate shared tables. Run migration rehearsal
with other suites stopped, then run destructive database suites sequentially.

### Documentation ambiguity with device-local storage

Server `user_pmcs_inspections` and Flutter's local table have different
schemas. Always identify the database when documenting physical storage.

## Acceptance criteria

The change is complete when:

- `miltech_ng_test` and `miltech_ng` contain only the three target names;
- development's 8 inspection, 13 fault, and 2 comment rows are preserved,
  subject to updated counts if legitimate development activity occurs before
  implementation;
- row fingerprints match immediately before and after each rename;
- every column, constraint definition, index definition, and foreign-key
  action is unchanged except for approved object names;
- rollback and re-forward rehearsal pass on `miltech_ng_test`;
- Jet is canonically regenerated with no unrelated schema drift;
- focused behavior, compile, full-suite, and applicable race verification are
  reported accurately;
- no API, JSON, authorization, route, or Flutter behavior changes; and
- the working tree contains only the approved implementation and pre-existing
  unrelated state.

## References

- [PostgreSQL `ALTER TABLE`](https://www.postgresql.org/docs/current/sql-altertable.html)
- [PostgreSQL explicit locking](https://www.postgresql.org/docs/current/explicit-locking.html)
- [PostgreSQL dependency tracking](https://www.postgresql.org/docs/current/ddl-depend.html)
- [Jet code generation](https://github.com/go-jet/jet)
