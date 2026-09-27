# Disposable integration databases

## Approved physical baseline (2026-09-27)

swisscheese approved the exact sanitized `miltech_ng_test` public-schema export
as the disposable baseline, accepting the documented differences from
`miltech_ng`. Both exports were produced on 2026-09-27 UTC from independently
verified database connections on PostgreSQL 14.18 at `192.168.20.70:5432`,
with `pg_dump 14.18 --schema-only --no-owner --no-acl --schema=public`.
The source revision is the point-in-time physical schema captured by that
export; there is no separately recorded database migration revision.

- `miltech_ng_test`: `tests/testutil/shops_schema_baseline.sql`, 146,693 bytes,
  SHA-256 `9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70`.
- `miltech_ng`: separately reviewed export SHA-256
  `184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda`,
  147,798 bytes. It is comparison evidence, not the fixture.

The exports contain schema statements only; review found no `COPY`, row
`INSERT`, role creation, ownership changes, grants, database URI, or
`postgres://` string. No connection credentials are stored here. The full
comparison and export provenance are in the separately reviewed
`/private/tmp/shops-notification-schema-pf8s1xpl/review.md` and `schema.diff`.
For Shops tables plus `users`, both sources have 91 columns, 52 indexes, and
41 constraints with matching index and constraint definitions. Eleven Shops
timestamp columns differ by explicit `(6)` precision versus omitted precision;
defaults and nullability match. Elsewhere, `miltech_ng` has two additional
materialized views and two indexes, a differently spelled material-images vote
check, other timestamp precision differences, and `pgcrypto` (absent in
`miltech_ng_test`). Both sources have `pg_trgm` and `plpgsql`.

The approved snapshot has `public.users(uid)` as a `text NOT NULL` primary key,
both validated migration-015 `shop_vehicle` nonnegative checks, and no
`shop_notification_operations` table. A read-only check on each source found
no `shop_members.user_id` without a matching `users.uid`.

## Migration 016 operator procedure (prepared, not executed)

`scripts/apply-shops-notification-migration.sh` accepts exactly one of the two
reviewed database names and the five explicit arguments shown below. The
operator must provision `PGSERVICE` in a libpq service file and authentication
outside this repository. Use PostgreSQL 14.18 client tools so the schema-only
dump can reproduce the reviewed bytes. Do not pass a URI, password, SQL path,
or a third database name. The runner uses only the fixed
`migrations/016_create_shop_notification_operations.sql` and never invokes the
reverse migration.

| Target | Reviewed server address | Port | Role | Pre-migration public-schema SHA-256 |
| --- | --- | --- | --- | --- |
| `miltech_ng_test` | `192.168.20.70` | `5432` | `postgres` | `9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70` |
| `miltech_ng` | `192.168.20.70` | `5432` | `postgres` | `184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda` |

The operator must independently verify the service mapping and obtain an
approved application window before running either command. Complete the
`miltech_ng_test` application and its post-migration verification before
considering `miltech_ng`; the latter needs its own approval at the application
window. These are procedure examples, **not** a record of execution:

```sh
export PGSERVICE=<operator-provisioned-test-service>
scripts/apply-shops-notification-migration.sh miltech_ng_test 192.168.20.70 5432 postgres 9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70

export PGSERVICE=<operator-provisioned-main-service>
scripts/apply-shops-notification-migration.sh miltech_ng 192.168.20.70 5432 postgres 184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda
```

Before any migration write, the runner rejects an unexpected argument,
connection identity, PostgreSQL version, missing or unvalidated migration-015
checks, existing operations relation, or schema fingerprint. It first checks
the live identity, then compares `pg_dump --schema-only --no-owner --no-acl
--schema=public` SHA-256 with the target's own pinned checksum, then rechecks
identity and catalog in the **same `psql -X -v ON_ERROR_STOP=1` session**
immediately before including migration 016. A refused or interrupted run is
not proof of either migrated or untouched state; inspect the catalog before
retrying. The SQL file has its own transaction. Keep other schema changes out
of the application window between fingerprint and commit.

For each target, record before and after: `current_database()`,
`inet_server_addr()`, `inet_server_port()`, `current_user`, server version,
schema SHA-256, presence and validation state of both migration-015 checks,
operations-table primary key, user cascade and fingerprint check, and Shops
row counts. Record timestamp, runner exit, service mapping, and the server
instances' effective database/host/port. The after checksum will differ from
the pre-migration value and must be captured independently. Leave operations
rows and migration 016 in place if a server release is reverted. Rehearse a
guarded reverse only on disposable databases.

There is no applied-migration history table. These pins attest to physical
public-schema state on 2026-09-27, not a sequential 001–015 application. The
source-to-physical differences below remain release considerations.

### Migration boundary and limitations

Neither source has a migration/version/flyway/goose table outside system
schemas, and swisscheese confirmed no applied-migration history exists.
Migration files in `migrations/` are source definitions, not a deployment log.
The latest observable marker in the approved physical snapshot is the pair of
validated migration-015 checks. The wrapper therefore replays no 001–015
migrations and sets `later_migrations=()` until a later migration is added.
This is a physical baseline decision, **not proof** that files 001–015 were
applied sequentially.

Catalog markers from a restore of the pinned export in disposable PostgreSQL
14.18 were checked against all relevant 001–015 source files:

- 001: four material-image tables and nine indexes present; 002: equipment
  services table, check and nine indexes present; 003: only 12 of 41 named
  indexes present; 004: both suggestion tables and indexes present;
  005: `shop_messages.parent_id` present.
- 006–008: successor `user_pmcs_inspections`, `user_pmcs_faults`, and
  `user_pmcs_inspection_comments` tables, `performed_by`, and `notes` present;
  009: sync-state, checklist, subscription tables and eleven indexes present;
  010: content UUID reservation table present; 011: successor `source_type`
  and source-shape check present.
- 012: trigram index and `pg_trgm` in `public` present; 013: community votes
  table and voter index present; 014: renamed `user_pmcs_*` tables and indexes
  present, with old `pmcs_sbs_inspections` absent; 015: both tracked-usage
  checks present and validated.

The missing 29 migration-003 index names are not dropped or renamed by any
004–015 migration. The source file was therefore not verified as fully applied;
the export is the approved physical state rather than a reconstructed migration
chain. Also, the physical `shop_messages.parent_id` foreign key uses
`ON DELETE CASCADE` whereas migration 005 specifies `ON DELETE SET NULL`, and
the physical `equipment_services` table lacks migration 002's named list
foreign key. These are source-to-physical discrepancies, not changes made by
the disposable wrapper.

### Task 1 verification

On 2026-09-27, `rtk proxy scripts/test-shops-isolated.sh ./tests/shops
./tests/equipment_services -count=1` restored the pinned export into a fresh
disposable PostgreSQL 14.18 cluster and passed both packages (`shops` 3.054s,
`equipment_services` 0.655s). The wrapper rejected supplied
`TEST_DATABASE_URL`, supplied `TEST_DATABASE_MARKER`, an altered fixture
checksum, and `./...` with exit status 1 before opening a database. The fixture
was restored byte-for-byte after the checksum probe and compared with the
approved export. Two existing test blockers were repaired in separate commits
before this pass: the Shops error sanitizer now preserves three fixed vehicle
usage messages, and a PMCS history query test now requires the disposable
database name instead of `miltech_ng_test`.

### Restore prerequisites

The schema-filtered export has `CREATE SCHEMA public` and indexes using
`public.gin_trgm_ops`, but no `CREATE EXTENSION`. The disposable cluster starts
with PostgreSQL's default `public` schema. The wrapper first installs and
verifies `pg_trgm` in that schema, checks that the pinned fixture has exactly
one `CREATE SCHEMA public;` statement, and omits only that redundant statement
from the restore stream. The fixture bytes and checksum remain unchanged;
`psql -X -v ON_ERROR_STOP=1` rejects all other restore errors. PostgreSQL
documents that schema-filtered dumps can omit dependencies outside the selected
schema: [pg_dump 14](https://www.postgresql.org/docs/14/app-pgdump.html).

## Invocation

Unset `TEST_DATABASE_URL` and `TEST_DATABASE_MARKER` first, then run:

```sh
scripts/test-shops-isolated.sh ./tests/shops ./tests/equipment_services -count=1
```

The wrapper requires local `initdb`, `pg_ctl`, `psql`, `createdb`, Python 3, Go,
`shasum`, `awk`, and `sed`. It creates an ephemeral local cluster, restores the
approved baseline, applies its explicit later migrations, writes and verifies a
random 256-bit marker, passes the exact DSN to tests, and removes the cluster
on exit or signals. The provisioning path is confined to the approved fixture
and disposable PostgreSQL.
Local trust authentication is confined to this disposable cluster on loopback and
a private socket; run it on a trusted development host. SIGKILL cannot run traps
and requires manual cleanup of the temporary cluster.

The helper rejects implicit targets, non-loopback hosts, missing ports, DSN query
overrides, missing markers and configured/effective DSN conflicts. A name prefix
alone grants no access. The first application query checks
`test_infrastructure.disposable_instance`; a mismatch or missing table closes the
pool before a test can truncate. Connection/verification failures omit driver
messages to avoid logging credentials. Shops and equipment services still hold
session advisory lock **70020** on a dedicated connection while their suites run.

All 12 prior shared-DSN callers now require explicit environment configuration:
`docs_equipment`, `eic`, `equipment_services`, `item_comments`, `item_lookup`,
`material_images`, `pmcs_sbs_progress`, `shops`, `tmde`, `user_saves`,
`user_suggestions`, `user_vehicles`. Their automatic `.env` search was removed.
The independent `user_pmcs` and `sb_700_20` setups retain their existing behavior;
broad runtime tests must not run with inherited environment or dotenv configuration.
Compile those packages without executing their TestMain until separately isolated.
The wrapper accepts only `./tests/shops` and `./tests/equipment_services` and
refuses `./...`, other package patterns, and unsupported flags. Supported flags
are `-race`, `-v`, `-failfast`, `-short`, `-count`, `-run`, `-timeout`, `-parallel`
and `-shuffle`; flags taking a value accept either Go spelling.

## Further acceptance evidence

1. Restore a separate approved baseline, seed owner-approved representative rows,
   apply only subsequent migrations, compare schema and preserved data, then run
   the suites. Existing suite truncation is not evidence of populated upgrades.
2. Prove actual marker mismatch/missing-table refusals against disposable Postgres;
   current marker regression tests use an in-process SQL driver with no network.
3. Record release compatibility evidence in `shops-release-contracts.md`.
4. Coordinate rotation of the previously committed credential with the owner.
   Removing the source constant does not rotate it or remove it from Git history.

References: [Go database/sql](https://pkg.go.dev/database/sql),
[PostgreSQL advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS).
