# Shops released-client compatibility evidence

## Status: acceptance blocked (2026-09-26)

No approved release artifacts or captured fixtures were supplied. The current
mobile checkout is not evidence of a deployed release contract.

- Currently deployed Android version/build: **unknown, owner evidence required**.
- Currently deployed iOS version/build: **unknown, owner evidence required**.
- Oldest affected supported Android and iOS version/build: **unknown**.
- Candidate release source commit: `62b5af5390f541d6b365696b898c526c6d50dedd`; signed artifact identity/checksum and provenance: **unavailable**.
- Actual signed-build parser behavior: **unverified**; candidate source traced below.
- Captured device request/response fixtures: **unavailable**; synthetic source-derived notification fixtures are tested below.

For each deployed and oldest affected released client, record platform,
version/build, source commit and signed artifact provenance. Trace its actual
request serialization and response parsing. Supply sanitized fixtures covering
notification updates, equipment updates/usage, audit history (empty and populated),
null versus missing fields, successful no-op updates, and request failures.
Record endpoint, method, status, envelope, parser result and artifact origin for
each fixture; use synthetic identifiers and omit credentials and personal data.

Compatibility acceptance requires running the affected released parsers against
those fixtures. Current-source unit tests cannot close this gate. This document
makes no claim that released clients accept any remediation response changes.

## E3 verification checkpoint — 2026-09-26

Tested server source: `d50a691fe59174f4e3e5d495a83746adafcc7d9e`;
mobile source: `97f2b5317b6d8bfe62f346de90f7c4af6323c2fd`. These are
local remediation commits, not deployed release identities.

- Safe Go unit and race commands over `./api/shops/...`,
  `./api/equipment_services/...`, `./api/response/...`, `./api/middleware/...`,
  and `./tests/testutil/...` with `-count=1` exited 0. Vet over the same packages
  exited 0. Actual service logging direct-file race test exited 0.
- `go test ./... -run '^$' -exec /usr/bin/true` exited 0 **compile only**;
  no test binary or database TestMain executed.
- `scripts/test-shops-isolated.sh ./... -count=1` exited 1 on its package
  allowlist. `scripts/test-shops-isolated.sh ./tests/shops ./tests/equipment_services
  -race -count=1` exited 1 on the missing approved schema/migration boundary.
  Both refused before provisioning; neither is passing integration evidence.
- All commands used `rtk proxy`; Go used
  `GOCACHE=/private/tmp/miltechserver-f1-go-cache`. Five wrapper shell-block
  regressions passed; full disposable provisioning remains unverified.
- Flutter focused acceptance passed 1,222 tests; full suite passed 3,716,
  both exit 0. Full analyzer exited 1 with the 19 unrelated existing diagnostics.
  These current-source host tests are not released-client fixtures.
- `TestContractCapabilitiesDisabled` passes: `typed_errors`,
  `atomic_notification_save`, `service_dates`, `service_reads`, `message_sync`
  remain false. New message production reads explicitly fail closed.

Still **BLOCKED**: no-header released-parser checks for shapes, defaults, date
reads/writes, errors and pagination; old notification separate writes; alternate
old/new writes on the same notification/service/message/list; old server binary
against expanded schema; new-server rollback with capabilities disabled while
retaining data. No server schema expansion was fabricated or applied. Unit SQL
drivers cannot prove physical durability, FK behavior, transaction ordering or
old-writer compatibility. S2/S3 complete service datasets and consumers remain
deferred behind S1 mapping approval.

The mobile report `docs/audits/2026-09-26-shops-remediation-verification.md`
contains the finding-by-finding map, command results, native limits and rollout
checklist. F1 schema approval, released artifacts, owner-coordinated credential
rotation, every-serving-writer evidence and native acceptance remain prerequisites.
No capability activation, live/shared DB access, merge, push or deployment occurred.

## Notification save source contract — 2026-09-27

The only identified source candidate for 3.7.0+41 is mobile commit
`62b5af5390f541d6b365696b898c526c6d50dedd`. This commit is the origin of
the synthetic, credential-free fixtures in
`tests/shops/shops_released_notification_contract_test.go`; it is **not** a
verified signed Android or iOS artifact. Signed build identifiers, checksums,
artifact provenance, and captured device traffic were not supplied. The same
Dart source implements both platforms, but platform build differences cannot
be ruled out. Signed-artifact compatibility and device staging remain open.

The source trace is:

- `lib/_bloc/shops/cubits/shop_add_notification_cubit.dart` sends the create
  request before direct items, uses the server-generated notification ID for
  those items, and edits details before applying item differences. The form can
  send `completed: true` at creation.
- `lib/_data/repository/remote_shop_repository.dart` serializes create with
  `shop_id`, `vehicle_id`, `title`, `description`, `type`, and `completed`;
  `attached_shop_list` is included only when non-null. Edit sends
  `notification_id`, present detail fields, and `attached_shop_list: null` to
  clear the attachment. Bulk item writes send `notification_id` plus `items`
  containing `shop_id`, `notification_id`, `niin`, `nomenclature`, `quantity`,
  and `save_time`. None adds `X-MilTech-Shops-Contract` or an operation ID.
- `lib/_data/api/shop_api.dart` uses legacy POST
  `/shops/vehicles/notifications`, PUT `/shops/vehicles/notifications`, and
  POST `/shops/notifications/items/bulk`. It unwraps `MilResponse.data` for
  creates and bulk items; update accepts a successful status without parsing
  the data. The model files `shop_vehicle_notification_rm.dart` and
  `shop_notification_item_rm.dart` require IDs, references, names, and time
  strings; notification `description`, `completed`, and `attached_shop_list`
  are nullable, while item `quantity` is nullable. Their domain conversions
  parse `save_time` and `last_updated` and default absent item quantity to 1.
- `lib/_data/services/dio_service.dart` accepts only 2xx statuses by default.
  `ShopApi._handleError` converts a denied Dio error into
  `RemoteUserSavesException`; it does not parse the error body into either
  notification remote model. `MilResponse` itself has optional `status`,
  `message`, and `data` fields.

The new Go integration tests send those source-derived request fields without
the contract header. They check the standard envelope and the response fields
and types consumed by the released models, including timestamps and nullable
attachment; they inspect the bulk response directly before a later GET. They
also check denied/nonmember responses remain HTTP 500 standard envelopes with
null data, which the traced Dio path rejects, and check alternating legacy and
atomic writes on one notification. A private PostgreSQL advisory-lock trigger
holds an atomic transaction at its operation-ledger insert while unrelated
legacy create and item writes finish; the test verifies the lock wait before
releasing it and checking the atomic receipt. The completed-flag regression
checks `true`, `false`, and omitted create fields in both response and GET.

For an additional exact-source parser check, a disposable detached checkout at
`62b5af53` regenerated its ignored Dart serializer parts. A disposable server
route test captured actual create, bulk-item, and nonmember-denial HTTP
responses into `/private/tmp/miltech-37041-parser-fixtures.json` (SHA-256
`5df2b78ece371a89e4cca038333d8f423de68e741b24a57e02f399902d11416a`).
An ephemeral Flutter test in that checkout injected those bodies into the
historical `ShopApi` and `DioService` through an in-memory Dio adapter. It
parsed notification and item models, including `completed: true`, and raised
the historical `RemoteUserSavesException` for the 500 denial. The focused
`flutter test` exited 0. The temporary capture test, checkout, and fixture
file were removed after verification and were not committed. This remains
source and host-test evidence, not signed Android or iOS binary evidence, and
it does not prove independently staged platform
builds accept these bytes.

## Notification save activation gate — 2026-09-27

The Task 7 migration runner, two-target procedure, and live application record
are documented in `shops-database.md`. The independently reviewed 2026-09-27 pre-migration fingerprints
are `9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70`
for `miltech_ng_test` and
`184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda`
for `miltech_ng`; both were at `192.168.20.70:5432` under role `postgres`.
They are separate physical snapshots, not evidence that migrations 001–015
ran in sequence. Migration 016 was applied to `miltech_ng_test` first and
`miltech_ng` second after explicit user authorization. Read-only post checks
confirmed the receipt ledger, its constraints, and unchanged existing Shops
rows and constraints. The post-schema SHA-256 values are
`893858c29ece15ec8ad7abf448e9869a0f1c80ca335cc1ab3cccceac0d752b58`
and `5e396f10f2793e72c802c827713701903f881fd0e1c70acdd2b6497452c3d7b0`,
respectively. Deployment identities have not been verified.

Release remains gated on instance-to-database mapping checks and a successful
old-binary and mixed-binary rehearsal with the capability flag false. Every serving new
instance must accept atomic POSTs against its intended migrated database before
`SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED=true` is set uniformly. A capability
GET from every instance must report the intended body; aggregate HTTP 200 is
insufficient. Keep the legacy routes for 3.7.0+41 and stage signed Android and
iOS 3.7.0+41 artifacts plus the future mobile build on physical devices before
release. Artifact IDs, checksums, captured device flows, fleet versions, and
those stage results are still unavailable.

If the server rollout is reverted after atomic saves, retain migration 016 and
all operation receipts on both databases. Disabling advertisement blocks fresh
saves and uncertain retries in the future client; keep a compatible endpoint
available or resolve frozen operation IDs through an operator-led receipt
lookup before a replacement action. No deployment, signed-artifact, or device
acceptance is claimed by the schema-only application.

## Message sync activation — 2026-09-29

Server branch `feature/shops-message-sync`. Schema, runner and measurements are
in [shops-database.md](shops-database.md) ("Message sync migration 018") and
[shops-message-sync-measurements.md](shops-message-sync-measurements.md). No
live database was contacted, nothing was deployed, and no flag was enabled by
this work; the released-client, signed-artifact and device gates above remain
open.

### Capability semantics

- `message_sync` in `GET /shops/capabilities` is true only when
  `SHOPS_MESSAGE_SYNC_ENABLED=true` **and** a catalog readiness probe finds the
  counter table and an enabled allocator trigger. Otherwise it is false and the
  four reads (`/shops/:shop_id/messages-v2/{initial,history,catch-up,reconcile}`)
  answer 503.
- The probe checks that the schema objects exist. It is **not** proof that the
  backfill completed, that every row is numbered, or that every serving instance
  runs the new binary and points at the migrated database. Those are separate
  gates below. A probe failure is logged as `message sync readiness probe
  failed`.
- Order matters: the migration must be applied **before** the flag is turned on.
  With the flag on and 018 not applied, `/capabilities` still reports false
  (the probe fails) but the sync read endpoints return 500 if called directly.
- Timestamps in sync responses, and the cursor's `created_at`, carry the
  database session's UTC offset (for example `-07:00`), not necessarily `Z`.
  Clients must parse offsets.

### Legacy-shape guarantee

The legacy message endpoints keep paths, request shapes, envelopes and field
sets. Their JSON keys stay exactly `id, shop_id, user_id, message, created_at,
updated_at, is_edited, parent_id, author_username`; `insertion_number` must
never appear in a legacy response. `TestMessageSyncLegacyCompatibility` pins the
key set. Because `response.ShopMessageResponse` embeds the jet model
`model.ShopMessages` and is marshalled directly, the `.gen/` model for
`shop_messages` is intentionally **not regenerated**: the reader uses raw SQL
and the legacy insert lists explicit columns. Regenerate only with
`tools/jetregen`, never the plain `jet` CLI; if the model is ever regenerated,
`InsertionNumber` must carry `json:"-"` or that test fails.

### Rolling-deployment rule

Enable the flag only when every serving instance for the target database runs
the new binary and the migration is applied. A mixed fleet answers
`/capabilities` per instance, so clients would flip between the sync and legacy
paths. Older binaries and released clients still get numbered messages from the
trigger, so the migration itself does not require a uniform fleet; the flag
does. Flag-off is the first-line rollback.

### Deployment sequence and gates (each step needs the operator's approval)

1. Pin the pre-018 schema checksums (see `shops-database.md`); the runner refuses
   until then.
2. Apply 018 to `miltech_ng_test` with the runner and record post-conditions and
   the database-privilege check (app role can write `shop_message_counters`).
3. Deploy the new binary to the test environment with
   `SHOPS_MESSAGE_SYNC_ENABLED=false`. Verify `/capabilities` reports
   `message_sync:false`, legacy flows are unchanged, and a message posted from a
   released client build (3.7.0+41 or the oldest supported) is numbered.
4. Inventory the fleet: every instance, its `DB_NAME`, host and port, and its
   binary version. Uniform new binary required.
5. Set the flag on and restart all instances; verify `message_sync:true`, the
   app shows no "Full message refresh is unavailable" banner, and edits and
   deletes from a second account appear within a poll.
6. Repeat for `miltech_ng` only after test-environment acceptance and a separate
   approval, with its own pre/post hashes.

Migration downtime is real (message reads and writes block; see the measurements
and the `shops-database.md` section); production row counts are unknown.

### Known failures on the base branch

`TestAtomicNotificationItemFieldsSurviveReleasedClientSaves` (tests/shops) and
`scripts/test-shops-isolated_test.py` (broken since wrapper refactor `f459eb4`)
fail independently of this work and are not fixed here.
