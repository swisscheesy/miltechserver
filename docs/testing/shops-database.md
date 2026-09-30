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

## Migration 016 operator procedure and 2026-09-27 application

`scripts/apply-shops-notification-migration.sh` accepts exactly one of the two
reviewed database names and the five explicit arguments shown below. The
operator must provision `PGSERVICE` in a libpq service file and authentication
outside this repository. Use PostgreSQL 14.18 client tools so the schema-only
dump can reproduce the reviewed bytes. Do not pass a URI, password, SQL path,
or a third database name. The runner uses only the fixed
`migrations/016_create_shop_notification_operations.sql` and never invokes the
reverse migration.
Unset `PGDATABASE`, `PGHOST`, `PGHOSTADDR`, `PGPORT`, and `PGUSER` before
invocation; the runner refuses even empty values for these inherited selectors
so the service file must provide the target mapping. `PGSERVICEFILE` and the
operator's authentication channel remain available.

| Target | Reviewed server address | Port | Role | Pre-migration public-schema SHA-256 |
| --- | --- | --- | --- | --- |
| `miltech_ng_test` | `192.168.20.70` | `5432` | `postgres` | `9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70` |
| `miltech_ng` | `192.168.20.70` | `5432` | `postgres` | `184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda` |

The operator must independently verify the service mapping and obtain an
approved application window before running either command. Complete the
`miltech_ng_test` application and its post-migration verification before
considering `miltech_ng`; the latter needs its own approval at the application
window. These are reusable procedure examples; the execution record follows:

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

The runner resolves its physical script path through symlinks before locating
the fixed SQL source. Before any database contact it copies that source to a
private mode-0600 file and checks its pinned SHA-256
`32b7a07f383aa1c6be028df865871bb0034e8df42d6f8e8474bff3aa139087de`.
It rechecks the copy immediately before the final session includes it. This
private verified `\i` path intentionally replaces the plan's literal relative
`\i migrations/016_create_shop_notification_operations.sql`: a symlinked
runner or source-file replacement cannot redirect the include to different
SQL. The private copy is removed on exit.

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

### Live application record

swisscheese approved application to both named databases on 2026-09-27. A
private mode-0600 `PGSERVICEFILE` mapped the previously verified PostgreSQL
MCP host, port, and role to each target; the test entry changed only the
database name. Authentication stayed outside the repository and command line.
The runner used reviewed SQL SHA-256
`32b7a07f383aa1c6be028df865871bb0034e8df42d6f8e8474bff3aa139087de`.
Its address guard was corrected in `76c44f9` to compare
`host(inet_server_addr())`: the `inet` text rendering includes `/32` on this
server. A read-only live check and independent review passed before either
application.

| Target | Runner completion (UTC) | Pre-schema SHA-256 | Post-schema SHA-256 | Result |
| --- | --- | --- | --- | --- |
| `miltech_ng_test` | 2026-09-27 13:56:48 | `9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70` | `893858c29ece15ec8ad7abf448e9869a0f1c80ca335cc1ab3cccceac0d752b58` | Exit 0; migration 016 present |
| `miltech_ng` | 2026-09-27 13:57:58 | `184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda` | `5e396f10f2793e72c802c827713701903f881fd0e1c70acdd2b6497452c3d7b0` | Exit 0; migration 016 present |

Immediately before each run, an independent read-only `pg_dump 14.18`
reproduced its pinned pre-schema checksum. Both connections reported
`192.168.20.70:5432`, role `postgres`, PostgreSQL 14.18, both validated
migration-015 checks, and no operation table. The runner independently
rechecked identity, prerequisites, and fingerprint before applying SQL.
Read-only post snapshots show each target's ten existing Shops table counts,
39 constraints, and 50 indexes unchanged. The test database retained one
Shop, one vehicle, two members, and one invite code. The main database retained
four Shops, nine vehicles, nine notifications, 18 direct items, 57 messages,
55 notification changes, 13 list items, 12 lists, six members, and five invite
codes. Each new operations table has zero rows, the same five non-null column
definitions, a `(user_id, operation_id)` primary key, a cascading `users(uid)`
foreign key, and a 32-byte fingerprint check. Its three constraints and one
index are logically identical across the two targets. The post-schema hashes
differ because the previously approved source schemas differ.
An immediate second invocation for each target was refused by the initial
guard because the operations table already exists, before schema dump or SQL
include.

This application did not run destructive Go tests, reverse migration, server
deployment, or capability activation on either named database. Signed
3.7.0+41 artifacts and staging/device execution remain unavailable; the user
authorized this additive schema-only application with those release gates
still open. Server instance `DB_NAME`/host/port mappings and fleet flag state
have not been verified, so this record does not authorize rollout.

### Migration boundary and limitations

Neither source has a migration/version/flyway/goose table outside system
schemas, and swisscheese confirmed no applied-migration history exists.
Migration files in `migrations/` are source definitions, not a deployment log.
The latest observable marker in the approved physical snapshot is the pair of
validated migration-015 checks. The wrapper therefore replays no 001–015
migrations; `later_migrations` now contains only migration 016.
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

## Message sync migration 018 (2026-09-29)

Migration `migrations/018_add_shop_message_insertion_numbers.sql` is the schema
half of Shops message synchronization. It is **final and checksum-pinned**; any
edit requires a new review and new pins. It is additive and does not change a
legacy response shape.

What it does, in one transaction (`SET LOCAL lock_timeout = '5s'`):

- Locks `public.shops` (`SHARE ROW EXCLUSIVE`), then `public.shop_messages`
  (`ACCESS EXCLUSIVE`). It refuses (raises, rolls back) if any message has a
  NULL `created_at`.
- Adds nullable `shop_messages.insertion_number bigint` and creates
  `shop_message_counters (shop_id text PRIMARY KEY, last_number bigint NOT NULL
  DEFAULT 0 CHECK (last_number >= 0))`.
- Backfills per-shop numbers 1..n ordered by `(created_at, id)`, seeds one
  counter per existing shop (including shops with no messages), then adds the
  counter's foreign key to `shops(id) ON DELETE CASCADE` after seeding.
- Installs `assign_shop_message_insertion_number()` and a `BEFORE INSERT ... FOR
  EACH ROW` trigger as the **sole allocator**: it overwrites any supplied value,
  so released clients and older server binaries that never mention the column
  are numbered too.
- Creates unique index `shop_messages_shop_insertion_number_key (shop_id,
  insertion_number)` and `idx_shop_messages_shop_created_id (shop_id, created_at
  DESC, id DESC)`.

Its rollback, `migrations/018_rollback_shop_message_insertion_numbers.sql`,
drops all of the above. It is never invoked by the runner.

Pinned SHA-256 values (recomputed by the runner; it refuses on any difference):

- add: `b0e8809dca1d8e109cc3072c13c9033f38f40bbe4a086bffd8322f57617777d7`
- rollback: `775a51afebf059dbc5d7a733a47fdd4fc598a03a6100443e9e31a60077371ecf`

### Disposable rehearsal (wrapper mode)

The wrapper appends 018 to its `later_migrations` (after 016 and 017). The mode
`scripts/test-shops-isolated.sh --verify-message-sync-migration ./tests/shops
-run 'MessageSyncSchema' -count=1` additionally rehearses, on fresh disposable
databases: a populated upgrade (rows that exist before 018 are backfilled in
`(created_at, id)` order per shop and counters are seeded), the NULL
`created_at` refusal (nothing changes), and rollback followed by re-apply. It
prints `Populated message sync upgrade passed.`, `Migration 018 refused NULL
created_at and changed nothing.` and `Message sync reverse and re-apply
passed.`. Measurements (query plans, lock hold, contention probes) are in
[shops-message-sync-measurements.md](shops-message-sync-measurements.md); they
are not restated here.

### Preflight evidence

The read-only preflight queries (NULL `created_at` count, non-UUID IDs, per-shop
volume, existing counter/column/trigger, whether migration 017 columns exist)
were to be run by the operator on both databases. **Their results have not been
supplied and are not recorded here.** The only live facts on record are from
2026-09-27: `miltech_ng` had 57 messages across four shops; `miltech_ng_test`
had one shop. Migration 018 itself refuses a NULL `created_at`; it does **not**
check that message IDs are UUIDs, which the sync reader requires. Production row
counts and message sizes are unknown.

### Operator procedure and pre-018 schema pins

`scripts/apply-shops-message-sync-migration.sh` is a copy-adaptation of the 016
runner: five explicit arguments, an operator-provisioned `PGSERVICE`, refusal of
inherited `PGDATABASE`, `PGHOST`, `PGHOSTADDR`, `PGPORT` and `PGUSER` (even
empty), a private mode-0600 copy of the fixed SQL checked against the pinned
SHA-256 twice, an identity guard before the schema dump, and the same guard
repeated in the one `psql -X -v ON_ERROR_STOP=1` session that includes the file.
It applies only the add migration. Use PostgreSQL 14.18 client tools.

The guard requires: database name, `host(inet_server_addr())`, port and role to
match the arguments; server version 14.18; the migration-015 checks (kept from
016); `shop_notification_operations` **present** (016 applied);
`shop_message_counters` absent; no `insertion_number` column; and row-level
security **not** enabled on `public.shops`. The RLS check is a conservative
guard: with RLS the counter foreign key validation was assumed to take per-row
`FOR KEY SHARE` locks on `shops`, the deadlock the migration was designed to
avoid. PostgreSQL's `CREATE POLICY` documentation says referential integrity
checks bypass row security, so the guard may be stricter than necessary; it was
not probed.

**Pins are not set.** The runner's two `approved_checksum` values are the
sentinel `UNPINNED`, and the runner exits 1 with `Refused: pre-018 schema
checksum for <target> has not been pinned; see docs/testing/shops-database.md`
before any tool lookup or database contact, even when a fifth argument is
given. The pre-016 hashes above must not be reused (the databases now carry
016), and whether migration 017 is applied live is unrecorded. To pin:

1. With PostgreSQL 14.18 client tools and a read-only session against the
   target (`PGSERVICE` only), run `pg_dump --schema-only --no-owner --no-acl
   --schema=public | shasum -a 256` for each of `miltech_ng_test` and
   `miltech_ng`, and record whether the 017 columns (`nickname`,
   `unit_of_measure` on `shop_notification_items`) exist.
2. Record both hashes and the date in this section.
3. Replace the two `approved_checksum=$unpinned_checksum` values in the runner
   in one reviewed commit.

The post-016 hashes recorded on 2026-09-27 are
`893858c29ece15ec8ad7abf448e9869a0f1c80ca335cc1ab3cccceac0d752b58`
(`miltech_ng_test`) and
`5e396f10f2793e72c802c827713701903f881fd0e1c70acdd2b6497452c3d7b0`
(`miltech_ng`). They are valid pre-018 values **only if migration 017 is not
applied** on that database and nothing else changed since; they are quoted for
comparison and are not pins.

Once pinned, the procedure (each database needs its own approval;
`miltech_ng_test` first, `miltech_ng` only after test-environment acceptance):

```sh
export PGSERVICE=<operator-provisioned-test-service>
scripts/apply-shops-message-sync-migration.sh miltech_ng_test 192.168.20.70 5432 postgres <pre-018 test schema SHA-256>

export PGSERVICE=<operator-provisioned-main-service>
scripts/apply-shops-message-sync-migration.sh miltech_ng 192.168.20.70 5432 postgres <pre-018 main schema SHA-256>
```

Negative probes run on 2026-09-29 with no database reachable (bogus
`PGSERVICE`): wrong argument count, unknown target, both targets with the
sentinel (with matching and non-matching fifth arguments) refused with `Refused:`
and exit 1. On a temporary copy with a fake pin under `/private/tmp` (not
committed), a mismatching fifth argument, wrong address, port or role, missing
or malformed `PGSERVICE`, each inherited `PG*` selector, a missing migration file
and a migration file whose content differs from the pinned SHA-256 were all
refused with exit 1 before a database was contacted. No live database was
contacted and the runner has never applied anything.

### After applying: verification

Record for each database, before and after: identity, server version, schema
SHA-256, and the following. All must hold before deploying the binary or
enabling the flag:

- `shop_message_counters` row count equals the `shops` row count.
- `SELECT count(*) FROM shop_messages WHERE insertion_number IS NULL` is 0.
- `shop_messages_assign_insertion_number` exists in `pg_trigger` and is enabled
  (`tgenabled = 'O'`).
- Both new indexes are valid (`pg_index.indisvalid`).
- `max(insertion_number)` per shop equals its `last_number`.
- The `shop_messages` row count is unchanged.
- A message posted through the application (or by a released client) is
  numbered.
- **Database privilege gate.** The trigger is `SECURITY INVOKER`. After 018,
  every message insert by the application role also inserts into and updates
  `shop_message_counters`, so that role must own the new table or hold
  privileges on it (for example through default privileges of the role that
  runs the migration). The runner's role is `postgres`; the application role
  is not on record. Verify on `miltech_ng_test`, with the real
  application role, that a message insert succeeds **before** production. If
  privileges are missing, every message insert fails, in the legacy path too.

### Lock behaviour, downtime and retry

This is **not** a zero-downtime change. The migration blocks message reads and
writes (both legacy and sync) from acquiring `shop_messages` until commit, and
writes to `shops` from acquiring its lock. Measured on a 2026-09-29 Apple-silicon
laptop with a disposable PostgreSQL 14.18: about 1.1 s at 100,000 rows and about
6.7 s at 450,000 rows (one run), against concurrent probes that were blocked for
essentially the whole hold. **Production row counts and hardware are unknown**,
so these numbers are a shape, not a promise. Apply in a low-traffic window.

Lock order is `shops` first, then `shop_messages`, matching a cascading `DELETE
FROM shops`; the counter's foreign key is added after seeding, so the seed does
not take per-row `FOR KEY SHARE` locks on `shops` that deadlocked against
`shared.LockShopMutation` transactions. Both defects were reproduced by probes
and fixed (details in the measurements document).

**Accepted residual cycle.** Any transaction that touches `shop_messages` (even
a `SELECT`) and then writes `shops` deadlocks with the migration. The migration
is the deadlock victim and rolls back. Current application code has no such path
(checked writers: settings, core and members repositories; message create, edit
and delete). If a run fails with `deadlock detected` or a lock timeout, it is
atomic, nothing changed, and it is safe to re-run.

### Rollback

First-line rollback is `SHOPS_MESSAGE_SYNC_ENABLED=false` on every instance: no
schema change is needed, and the trigger keeps numbering messages. The 018
rollback file is for a full schema revert, run only with the flag off on every
instance; it locks `shops` in `ACCESS EXCLUSIVE` mode, blocking reads and writes
of `shops` for the few milliseconds it runs (its contention behaviour was not
probed). Watermarks already given to clients become meaningless; clients
re-initialize when the capability reads false. Rehearse only on disposable
databases.

### Known failures unrelated to 018

Both pre-date this work on the base branch and are not fixed here:

- `tests/shops` `TestAtomicNotificationItemFieldsSurviveReleasedClientSaves`
  fails (expected 1, actual 0).
- `scripts/test-shops-isolated_test.py` fails at import since the wrapper
  refactor `f459eb4` moved the anchors it slices.

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
