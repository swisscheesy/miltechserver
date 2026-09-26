# Disposable integration databases

## Status: baseline acceptance blocked (2026-09-26)

The repository has incremental migrations but no verified initial schema. The
baseline SQL deliberately raises an error, and the wrapper refuses before starting
PostgreSQL. No schema was inferred from generated Jet models. Fresh restore and
upgrade against a populated baseline have **not passed**.

The schema owner must supply a sanitized schema-only artifact from an approved
source. Record source identifier (no connection secrets), approval, source revision,
export date, PostgreSQL version, SHA-256, applied migration state, and the comparison
of types, defaults, sequences, constraints, foreign keys, indexes and extensions.
Then replace `tests/testutil/shops_schema_baseline.sql`, set the reviewed checksum in
`scripts/test-shops-isolated.sh`, and list only migrations after that boundary in
`later_migrations`. There is deliberately no environment bypass for this gate.

## Invocation

Unset `TEST_DATABASE_URL` and `TEST_DATABASE_MARKER` first, then run:

```sh
scripts/test-shops-isolated.sh ./tests/shops ./tests/equipment_services -count=1
```

The wrapper requires local `initdb`, `pg_ctl`, `psql`, `createdb`, Python 3, Go and
`shasum`. It creates an ephemeral local cluster, restores the approved baseline,
applies its explicit later migrations, writes and verifies a random 256-bit marker,
passes the exact DSN to tests, and removes the cluster on exit or signals. The
provisioning path is prepared but cannot be validated until the baseline is approved.
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

## Acceptance evidence still required

1. Restore a fresh approved schema and run the two suites above successfully.
2. Restore a separate approved baseline, seed owner-approved representative rows,
   apply only subsequent migrations, compare schema and preserved data, then run
   the suites. Existing suite truncation is not evidence of populated upgrades.
3. Prove actual marker mismatch/missing-table refusals against disposable Postgres;
   current marker regression tests use an in-process SQL driver with no network.
4. Record release compatibility evidence in `shops-release-contracts.md`.
5. Coordinate rotation of the previously committed credential with the owner.
   Removing the source constant does not rotate it or remove it from Git history.

References: [Go database/sql](https://pkg.go.dev/database/sql),
[PostgreSQL advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS).
