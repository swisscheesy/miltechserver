# Shops server remediation: target-specific release gates

Checkpoint: 2026-10-04. **NOT RELEASE READY.** This records local implementation
and a procedure for future authorization, not permission to connect, migrate,
rotate credentials, publish, deploy, or activate a flag. Work is server-only.

## Target ledger

| Gate | Existing `miltech_ng_test` | Existing `miltech_ng` |
| --- | --- | --- |
| Separate owner forward authorization | UNEXECUTED | UNEXECUTED; only after accepted test evidence |
| Current address/port/database/migration role/version | UNPINNED | UNPINNED |
| Actual application role from `DB_USERNAME` | UNKNOWN | UNKNOWN |
| Current 015–018 physical stage and schema catalog | UNPINNED | UNPINNED |
| 019/020/021/022/023 per-stage reviewed source/target/data pins | UNPINNED | UNPINNED |
| Legacy write as actual application role | UNEXECUTED | UNEXECUTED |
| Intended-schema tagged generation and 32 PMCS hashes | UNEXECUTED | UNEXECUTED |
| Exact TMDE materialized-view definition and behavior | UNKNOWN | UNKNOWN |

Do not infer these values from dotenv, historical exports, migration filenames,
or disposable fixtures. The old 016 application evidence remains historical.
An explicitly authorized **read-only inventory** must first establish the target
record; then the owner approves the exact forward action. Test-target approval
never authorizes the production target. The production record must link accepted
complete test-target evidence through `test_target_evidence`.

## One-action guarded runner

`scripts/apply-shops-remediation-migrations.sh` takes:

```text
TARGET forward-NNN AUTHORIZATION ADDRESS PORT MIGRATION_ROLE CATALOG_SHA256 PRECEDING_GENERATION_JSON
```

Only the two named targets and forward 019–023 are allowed. The checked-in
`TARGET_PINS` entries deliberately contain `UNPINNED`. No command in this guide
fills a live value. A reviewed source change must add **one target/stage record**
with these fields before the runner can connect:

- `address`, integer `port`, `role`, integer `version` (server_version_num),
  `application_role` (the actual `DB_USERNAME`), operator `service` name;
- exact `catalog_sha256`, `data_sha256`, `authorization` scoped to this target and
  action; `probe_shop_id` and `probe_user_id` identify owner-approved existing
  fixtures for a rolled-back legacy message write;
- `generation_evidence_sha256`, `source_sha256`, `pmcs_sha256` describing the
  preceding checkpoint; `test_target_evidence` links the accepted test record
  for production (for test, link its baseline inventory).

The runner refuses wrong arguments, unknown/unpinned actions, incomplete pins,
inherited selectors (even empty values), a mismatched `PGSERVICE`/`DB_USERNAME`,
altered/redirected 015–023 forward or reverse source files, and missing/mismatched
preceding generation proof **before connection**. Only `PGSERVICE`,
`PGSERVICEFILE`, and `PGPASSFILE` are allowed libpq environment inputs. The owner
provisions service/authentication files outside the repository. No dotenv,
credential/configuration dump, generic DSN, or arbitrary SQL action is supported.

One `psql -X -w` session checks current database, server address/port, effective
migration role, version and catalog checksum, takes Shop-first locks, checks
reviewed data-manifest hashes and unresolved cases, and switches to the actual
application role for independent privileges and a real numbered legacy-shaped
message insert. That proof is rolled back. The reviewed migration's single
outer BEGIN/COMMIT envelope is replaced by the enclosing guarded transaction;
its remaining SQL body is preserved verbatim and source-hashed before contact.
The actual-role proof is repeated before commit. Identity and migration backend
PIDs must match. A failed gate stops execution and rolls the transaction back;
a connection loss around COMMIT may be uncertain and **requires owner inspection**.
No automatic retry or automatic data repair is provided.

Separate SELECT, INSERT and UPDATE checks are conjunctive. A comma-separated
`has_table_privilege` list means **ANY** privilege and is invalid for this gate.
The Shop lock requires SELECT plus UPDATE. Table-level grants are conservatively
required by this runner; a deployment intentionally using only column grants
needs a separately reviewed equivalent procedure, not a bypass. Runtime 020/023
schema/privilege checks and real application smoke tests remain additional gates.

The catalog hash uses the same canonical public-catalog text query as the tagged
Jet generator, SHA-256 encoded as UTF-8. It is **not a pg_dump checksum**. The query
covers relations/columns/defaults/constraints/indexes/functions/triggers/enums/
sequences, not every possible extension, RLS policy or grant. Privilege checks
are separate; exact live schema review and writer fencing remain mandatory.
Unrelated schema writers must also be fenced: table locks alone cannot prevent
all concurrent CREATE/ALTER operations elsewhere in `public`.

## Generation checkpoint after every action

Exit 0 means **SQL applied; generation PENDING**, never release completion.
Stop immediately after each migration or approved repair. Do not retry an
already-applied action. Run only the canonical `go run ./tools/jetregen` with a
separately identity-verified explicit intended source, secure operator-provided
credentials, `public` schema, and a controlled output directory. The runner does
not bridge libpq `PGSERVICE` to the generator's lib/pq `JET_DSN`, and rejects an
inherited `JET_DSN`. Never use plain Jet or edit generated Go files manually.

Retain the generator's `generation-manifest.json`, the current candidate source
manifest, before/after hashes of all 32 tracked `user_pmcs_*` files, and exact
model/table/view build results. Unexpected PMCS changes stop the sequence.
Compile generated packages at each prefix; build the full application only at
the complete 023 candidate. Runtime generation does not rebuild the running
binary. Generic generation supports earlier stages; startup of the final binary
requires mandatory 020/023 schema validation before generation and routes.

The operator assembles a protected JSON checkpoint with these exact keys:

```json
{
  "stage": "018",
  "source_sha256": "reviewed candidate source manifest SHA-256",
  "generation_manifest": {"database": "verified target", "address": "verified IP", "port": 5432, "user": "actual DB_USERNAME", "schema": "public", "catalog_sha256": "verified catalog digest"},
  "pmcs_before": {"each of the 32 tracked generated paths": "SHA-256"},
  "pmcs_after": {"the same 32 paths": "the same SHA-256"},
  "generated_packages_compile": "PASS"
}
```

Include the actual full generated manifest, not the illustrative subset above.
For action 019 the previous stage is 018; for action 020 it is 019, and so on.
The evidence file's SHA-256 is pinned in the next stage record. `pmcs_sha256` is
SHA-256 of Python `json.dumps(pmcs_before, sort_keys=True).encode()`; the runner
requires 32 equal before/after entries and compares that digest to the reviewed
baseline pin. Source digest, intended target/schema/role/port/address and catalog
must match the next record. This mechanically checks the reviewed evidence; it
does not manufacture or independently attest owner build execution. Fabricated
records are invalid. Final 023 has the same mandatory proof and a separate
release-owner acceptance checkpoint even though no next migration consumes it.
The manual gap after SQL commit remains an operational risk; fence deployment
and subsequent actions until the proof is reviewed.

## Data inventory and refusal gates

The runner's `DATA_SQL` constructs an in-session JSON manifest and checks its
SHA-256 against `data_sha256`. It hashes ordered affected IDs, exact parent FK
actions, and historical message text containing `shop-message-images`; it does
not print authored data. Store full owner-reviewed manifests securely, with
row counts, target/time/source identity, disposition and evidence links. Historical
image candidates remain protected regardless of whether this text scan is
exhaustive; no ownership or deletion authorization is inferred from text.

| Manifest | Required disposition |
| --- | --- |
| Orphan Shops / Shops without an administrator | Refuse; owner resolves retention/successor policy explicitly |
| Parent FK actions / foreign or missing parents | Review actual CASCADE versus historical SET NULL; refuse incompatible migration or foreign parents; no silent conversion |
| Negative base mileage/hours | Refuse until separately approved exact-row repair; never clamp |
| Conflicting notification + exact-NIIN nickname/unit | Refuse until explicit resolution; preserve null versus empty and raw valid units |
| Historical asset candidates / unknown ownership | Preserve; no adoption or destructive cleanup without ownership evidence; account/container/blob and reference manifest required |

After each separately authorized repair, regenerate tagged models and repeat
inventory/identity/source checks before considering the next action. Do not restore
quantity or active item rows to recover metadata. Migration reverses are not
runner actions: each needs a separate explicit owner resolution/retention export,
compatible binary, target identity/checksum plan, rehearsal and immediate tagged
regeneration. Preserve all five 019–023 pairs and pinned 015–018 SQL unchanged.

## Remaining release gates

| Gate | Current evidence/status | Required owner evidence |
| --- | --- | --- |
| C06 invitations | BLOCKED: population, instance count and effective per-user/edge budgets UNKNOWN | Active code population, authenticated create/claim limits across fleet, edge enforcement and approved numeric budget; if insufficient, separately approve bounded UID limiter with expiry/eviction, burst/cross-instance/many-identity tests |
| Task 22 legacy cursor | OPEN: late current-writer commit may fall behind observed timestamp cursor | Owner decision; timestamps preserved, no waiver or lossless live catch-up claim |
| Numeric cursor restore ABA | OPEN | Explicit epoch/recovery policy; local reset detection does not solve all restore cases |
| F09 original shared-test credential | Owner rotation/cleanup UNEXECUTED | Separate owner confirmation for the original removed test credential |
| Task 18 diagnostic credential incident | Separate owner review/rotation confirmation UNEXECUTED | Unrelated known Docker `crystaldba/postgres-mcp`, role `postgres`, database `miltech_ng`; never rediscover/copy credential or assume it is F09 |
| Fleet and external writers | UNKNOWN | Uniform intended build/artifact hashes and flags, supported writers, fence counter-first writers for 019, explicit authorization before activation |
| Container runtime | UNEXECUTED; synthetic build-context sentinel checks passed | Actual daemon/image layers/runtime, nonroot readable mounted Firebase credentials, writable startup `.gen`, startup failure/recovery evidence |
| Target generated model provenance | UNKNOWN | Intended target schema and actual TMDE view; disposable empty TMDE compile ABI does not establish semantics |
| Dates | BLOCKED; `service_dates=false`, `service_reads=false` | Approved two-way mapping and ambiguity policy from real schema/timezone/writer samples |
| Released contracts/devices | UNEXECUTED | Sanitized exact released request/parser evidence, response-loss recovery, nullable keys, signed artifact and physical device acceptance |
| Observability | Owner collector/alert verification UNEXECUTED | Mandatory atomic receipts/audits, best-effort legacy audit warnings, cleanup-worker failures and invalid readiness alerting |
| Push/deploy/activation | NOT AUTHORIZED / NOT EXECUTED | Separate explicit owner authorization after the complete gate sheet is accepted |

No owner numeric invite budget was supplied, so this task adds no conditional UID
limiter and changes no code length. Do not equate an IP map on read endpoints with
an authenticated create/claim budget. Local tests, source fixes and documented
gates cannot establish exposure safety or release readiness.


## Disposable verification boundary

`--verify-remediation-migrations` in the existing protected wrapper includes a
separate `miltech_test_release_runner` fixture for this runner. The narrower
`--verify-release-runner` flag selects that rehearsal for debugging without
changing allowed test packages. Both retain the wrapper's loopback/private
cluster, marker, cleanup and no-inherited-target guards. Temporary fixture pins
exist only in the private test copy. The internal `--disposable-rehearsal` mode
refuses either named target and requires `miltech_test_*`, a loopback address and
a same-transaction marker check. Named mode refuses fixture targets entirely.
This is not a route for authorizing named databases with disposable pins.

The physical rehearsal verifies equal backend IDs, zero persisted probe messages,
missing/mismatched generation evidence with zero `psql` calls, SQL-applied versus
generation-pending receipts, and tagged regeneration after all five actions.
The independent five-pair forward/refusal/reverse matrix is retained. Restricted
actual-role tests separately prove each partial privilege set fails and the full
role succeeds; a fixture superuser does not establish live application grants.
