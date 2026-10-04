# Shops released-client compatibility evidence

## Container credentials and generation gate — 2026-10-03

The container recipe excludes environment files, Firebase credential files,
VCS/agent state, worktrees and local tool artifacts from the build context.
It retains canonical tagged `.gen` inputs for compilation and copies no
credential into the runtime image. `FIREBASE_AUTH_KEY` remains the supported
runtime credentials-file location; an absent/unreadable mount fails before
database or storage initialization. Credential read errors do not expose the
mount path. No startup-generation bypass is supported.

The runtime user is UID/GID `10001:10001`. Startup generates under
`/app/.gen` before routes register; that directory and its schema descendants
must be writable for lock, staging, backup and publication. The image's
generated inputs have that ownership. A read-only root filesystem requires
a writable volume or tmpfs at `/app/.gen`. Provision a persistent bind volume
with that owner when generation manifests need to survive container removal.
Do not mount generated build inputs read-only or run concurrent unlocked builds
against a directory being published. Runtime generation does not rebuild the
running binary or establish schema/function/grant readiness.

An operator supplies runtime environment and a read-only credential mount,
both outside the build context. For example, after separately approved build
and deployment gates:

```sh
docker run --read-only --tmpfs /tmp \
  --tmpfs /app/.gen:uid=10001,gid=10001,mode=0700 \
  --env-file /operator/runtime.env \
  --mount type=bind,source=/operator/firebase.json,target=/run/secrets/firebase.json,readonly \
  -e FIREBASE_AUTH_KEY=/run/secrets/firebase.json \
  -p 8080:8080 <approved-server-image>
```

The credential file must be readable by UID 10001 inside the container.
Provision permissions through the secret manager or an appropriate group;
never loosen production credential access merely to match a fixture.
Operator runtime configuration must point at the separately approved intended
database and storage targets. This example is documentation, not deployment
authorization.

Run `rtk proxy bash scripts/test-server-container-inputs.sh
/private/tmp/<new-empty-workspace> miltechserver-container-inputs-test:<unique-tag>`
for disposable packaging verification. The script refuses checkout/nonempty
contexts, nonlocal Docker endpoints and existing image tags. It creates only
synthetic sentinel files, a minimal Go/Node packaging fixture and a fake JSON
credential outside the context. It checks the actual `.dockerignore` through
a Docker context image, all saved backend/final layers, the final filesystem,
preserved generated build input, and fixture startup with a mounted credential
and writable output. Missing credentials and read-only output must fail.
The images/containers are removed on exit; temporary workspace artifacts remain
for inspection and can then be deleted by the operator.

Fixture runtime results demonstrate packaging, UID, mount and write access;
they do not execute Firebase/Azure SDKs or PostgreSQL/Jet. The actual server's
injected startup tests separately check exactly one required generation call
before route registration and failure before routes. Full freshly generated
application compilation still requires the Task 6 schema/function gate.
Local Docker validation is currently **unresolved**: the verified local Unix
endpoint did not answer its bounded availability probe. Context, layer and
container runtime checks have not passed. Actual SDK/Postgres startup,
intended-schema build, release artifact, fleet and deployment gates remain open.

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

The message-sync flag gates the four reads
(`/shops/:shop_id/messages-v2/{initial,history,catch-up,reconcile}`). Flag-off
requests return 503 `unsupported_contract`. Each enabled read rechecks current
membership and validates **all** messages in that Shop's read-only repeatable-read
snapshot, including messages outside the selected page or reconcile IDs. NULL
numbers/timestamps, missing nonempty counters, negative counters and counters
behind surviving numbers return sanitized 503 without advancing state. Missing
counters on empty Shops yield zero; counters above the surviving maximum remain
valid after deletion. Query/access/cancellation failures also fail closed with 503.

`message_sync` advertisement additionally checks required column types, the
supported 019 allocator body/execution settings, exact trigger timing, valid
018 indexes and effective current-role read/allocator privileges. The read-only
catalog statement has a two-second budget including pool wait. Probe failures
and incompatible schema/access log a category and advertise false.

`atomic_notification_save` is the configured flag AND bounded current-role
readiness for receipt, notification, item, metadata, audit and locking access.
Atomic HTTP entry validates authentication, contract and body first, then uses
the same readiness independently of the flag; valid saves remain callable before
advertisement. Unavailable schema/access returns 503 `unsupported_contract`
before any receipt or business write. Pure atomic services do no catalog I/O.
Its two-second budget covers one read-only repeatable-read snapshot and all probes.
`service_dates` and `service_reads` remain false.

Startup additionally validates mandatory 020 asset and 023 retained-metadata
schema (including resolution_version), lifecycle functions, keys, checks,
defaults and application access before automatic tagged generation and route
registration. Compatible additive columns are allowed. Optional sync/atomic
schema is still a capability gate, not a startup prerequisite. Generic tagged
generation remains available for earlier disposable migration stages. Neither
startup generation nor a passing probe recompiles or certifies the running binary.

For catch-up, `through < after` remains 400 `invalid`. An `after` or held `through`
ahead of the current counter returns 409 `message_sync_reset_required`. The
caller must discard the affected Shop's numeric sync state and obtain a new
initial snapshot, then resume from that snapshot's watermark. Never lower a
stored watermark silently or treat 503 as an empty successful page.

Numeric watermarks cannot detect a restore/cutover that reuses in-range numbers
(the ABA case). Before numbering rollback/reapply, restore or target cutover,
operators must fence affected writers, identify the affected Shops/clients, and
coordinate an explicit state-discard/new-initial-snapshot handoff with the client
owner. This server change includes no client implementation or automatic repair.
The current-writer late-commit legacy timestamp omission remains unresolved.

Catalog readiness does not prove backfill, every pool/session setting, every
serving instance's build/target, or successful writes under the deployed role.
Verify migration/schema/build identity and real authorized writes per instance
before enabling flags fleet-wide; preserve the separate signed-artifact/device
and deployment gates. No capability GET replaces those checks.

- Timestamps in sync responses, and the cursor's `created_at`, carry the
  database session's UTC offset (for example `-07:00`), not necessarily `Z`.
  Clients must parse offsets.

### Legacy-shape guarantee

The legacy message endpoints keep paths, request shapes, envelopes and field
sets. Their JSON keys stay exactly `id, shop_id, user_id, message, created_at,
updated_at, is_edited, parent_id, author_username`; `insertion_number` must
never appear in a legacy response. `TestMessageSyncLegacyCompatibility` pins the
key set. `response.ShopMessageResponse` explicitly owns the nine fields and
preserves nullable keys; it does not embed the generated database model.
Canonical tagged generation through `tools/jetregen` is mandatory after every
authorized migration or data repair and at startup before routes accept traffic.
The generated `insertion_number` retains its normal tag; the explicit DTO keeps
it out of legacy responses. Never edit generated tags or skip generation.

The four messages-v2 reads fail closed with HTTP 503 `unsupported_contract`
when their universal Shop integrity check finds unready numbering or NULL message
timestamps. Numeric cursors ahead of the persisted counter require HTTP 409
`message_sync_reset_required` and recovery from a new baseline. Legacy timestamp
pagination keeps pre-018 compatibility: a deleted or NULL-created_at anchor
produces the safe reload message. Negotiated contract version 2 exposes HTTP 409
`reset_required`; an unnegotiated legacy request preserves HTTP 500, `data:null`
and omitted `code`. Legacy pagination does not depend on numeric numbering readiness. This does not detect in-range restore ABA. A schema restore or
reverse/reapply requires the separately approved recovery/epoch/client gate.
Apply the current 018/019 migration, role, identity and generation runbooks with
all serving instances and external writers inventoried/fenced; flags alone are
not fleet readiness. The current-writer late-commit timestamp omission remains
OPEN. These directions supersede the former embedded-Jet/skip-generation/tag-edit
instructions, retained only as historical review evidence.

### Legacy message cursor fixture handoff — 2026-10-04

Legacy `/shops/:shop_id/messages/paginated` keeps the existing envelope,
limit defaults/range, nullable fields and nine message keys. ID anchors retain
exact stored TEXT identity and must belong to the requested Shop. Membership is
checked before resolving an anchor. Ordering and bounds use `(created_at, id)`.
`before_id` selects older tuples descending; `after_id` selects the nearest newer
tuples ascending and then presents that page descending. To drain a burst,
clients must follow `next_cursor` in the same direction: it is the last tuple
selected, which is the **first displayed row** for an `after_id` page. It is
present only when another row was observed beyond the limit. There is no
pagination metadata in cursor responses and no numbering dependency in their
SQL projection.

A missing/deleted legacy ID anchor cannot recover its stored timestamp. Reload
messages when the existing legacy error envelope returns HTTP/status 500,
`message: "Message cursor is unavailable; reload messages"`, `data: null`, and
no `code`. The public constant is `messages.LegacyCursorReloadMessage`. Requests
opting into Shops contract 2 receive the existing typed mapping: HTTP/status 409
and `code: "reset_required"`. Query failures and cancellations remain errors;
they do not become reload responses or successful empty pages. The separate
sync-v2 opaque tuple history cursor still continues after its anchor is deleted.

Server fixtures: `TestLegacyCursorDrainsTiesAndBurst` drains ten fixed-timestamp
rows at limit two in both directions, with ties and a larger-than-limit newer
burst, checking uniqueness, full union, DESC presentation, nullable keys, author,
continuation boundary, membership denial and deleted-anchor reload.
`TestLegacyCursorAnchorScopeAndExactText` covers foreign-Shop and case-distinct
TEXT anchors. `TestMessageSyncRoutesHistoryEqualTimestampsAndDeletedAnchor`
continues to cover v2 behavior. These are server fixtures for client handoff,
not released-client or device acceptance evidence.

**Confirmed live catch-up limitation:** tuple draining covers rows on the
requested side of the anchor; it is not commit-order catch-up. The current
service selects `created_at` before obtaining the repository Shop lock. A
paused current writer can therefore commit an older timestamp after another
message has committed and been observed as an `after_id` anchor. That late row
is outside the tuple predicate. `TestLegacyCursorCurrentWriterCommitOrderLimitation`
physically reproduces this with two current-service writes and characterizes
the omission; it does not fix it. Existing timestamps and writer behavior remain
unchanged. Old/external writers and clock ordering are additional fleet gates;
no monotonic clock guarantee is asserted. Lossless live catch-up requires a
separately accepted writer policy or the separately gated v2 insertion-number
protocol. Finite legacy tuple-drain success does not close those gates.

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
2. Run the case-sensitive ID preflight and the privilege pre-check from
   `shops-database.md` (all ID counts must be 0). Apply 018 to `miltech_ng_test`
   with the runner and record post-conditions and the post-migration privilege
   check (app role can write `shop_message_counters`; a real message insert
   through the application succeeds there).
3. Deploy the new binary to the test environment with
   `SHOPS_MESSAGE_SYNC_ENABLED=false`. Verify `/capabilities` reports
   `message_sync:false`, legacy flows are unchanged, and a message posted from a
   released client build (3.7.0+41 or the oldest supported) is numbered.
4. Inventory the fleet: every instance, its `DB_NAME`, host and port, and its
   binary version. Uniform new binary required.
5. Set the flag on and restart **all** instances together in a low-traffic
   window; verify `message_sync:true`, the app shows no "Full message refresh is
   unavailable" banner, and edits and deletes from a second account appear
   within a poll. Expect already-open client sessions to show the refresh-failed
   banner while instances restart (see the next section); a brief mixed fleet
   answers per instance.
6. Repeat for `miltech_ng` only after test-environment acceptance and a separate
   approval, with its own pre/post hashes.

### Runbook: capability true but a sync endpoint answers 503 or 403

Once a client has seen `message_sync:true` it has **no legacy fallback**
(`lib/_bloc/shops/cubits/shop_messages_cubit.dart`, lines 106 to 151 in the
mobile repository, `loadInitialMessages`; mobile path by reference only).
Client behaviour:

- **Initial load answers 503** (or any error): the whole Messages tab shows
  `Failed to load messages.` until the tab is re-activated.
- **Refresh (catch-up or reconcile) answers 503 or 403**: the client shows
  `Message refresh failed. Unverified messages have been retained.`, keeps the
  rows it has, and new messages stop appearing until the user leaves and
  re-enters the shop. It keeps polling (about every 10 s), so a flag turned off
  produces a 503 per open session per poll until sessions are reopened.
- A possible later client change, not made here: treat `unsupported_contract`
  from the initial read as a signal to fall back to the legacy read.

Operator actions: confirm every instance has the flag, binary and database you
expect; for 503 on reads check the current schema, numbering/counter integrity,
capability configuration and required 018/019 migration state on that instance's
database; for 403 on every read of a shop check the ID preflight in
`shops-database.md` (non-canonical shop ID); for `Incomplete reconciliation`
loops check message IDs the same way. If unnumbered rows exist (`SELECT shop_id,
count(*) FROM public.shop_messages WHERE insertion_number IS NULL GROUP BY 1`),
fence serving writers and use the separately authorized, identity-checked repair
in `shops-database.md`, followed by tagged generation. All messages-v2 reads, including catch-up,
return 503 for unready numbering rather than silently omitting NULL-numbered
rows. Legacy timestamp pagination instead signals deleted/NULL-time anchor reload
with negotiated 409 `reset_required` or unnegotiated legacy 500, as described above. Ahead-of-counter numeric cursors
return 409 `message_sync_reset_required`; in-range
restore ABA remains an owner/client recovery gate. Turning the flag off does not
repair cursor integrity. Never disable the allocator trigger while serving writes.

Migration downtime is real (message reads and writes block; see the measurements
and the `shops-database.md` section); production row counts are unknown.

## Notification history and legacy audit delivery — 2026-10-03

Legacy notification and item mutations commit business rows before audit
delivery. Audit failure still returns business success. Each failed delivery
emits one structured `legacy_notification_audit_failed` warning with
`operation`, `shop_id`, `vehicle_id`, `notification_id`, `actor_id`,
`correlation_id`, and `failure_category`. Correlation is a server-minted UUID
carried by the request context, including direct repository mutations; it is
never read from a client header. Categories are `request_canceled`,
`request_deadline`, or `database_failure`. No database error text, credentials,
item values, title or admin label is logged in this warning. Post-commit
delivery uses the original request context, so cancellation can leave a gap.
There is no legacy outbox or detached retry worker. Vehicle deletion uses a
savepoint-protected best-effort audit in the deletion transaction. A failed audit
preserves the deletion and emits this same event once with operation
`vehicle_deleted`, the actor/Shop/vehicle/server correlation, and empty
`notification_id`; no raw database error is logged. The collector rule below
covers that operation too.

Configure the log collector to count WARN records with
`msg=legacy_notification_audit_failed`, grouped by instance, operation and
failure category. Alert on any nonzero count in a five-minute window; route
the alert to the server operator and retain the correlation/resource fields
for investigation. This is the required alert definition; collector routing
and live alert activation have not been verified or configured by this work.

On an alert, use the correlation/resource identifiers to inspect the request
and intended instance/database mapping under the operator's normal access
controls. Confirm business state before considering any action. Check
database availability and audit INSERT permissions for `database_failure`,
and request deadlines/cancellations for the request categories. Do not repeat
the business mutation solely to fill a history gap, fabricate an event-time
snapshot from current values, or claim legacy history is gap-free. A repair
needs separate owner authorization and original event evidence.

Single/bulk item add/remove events keep `items_added`/`items_removed` kinds.
Their snapshots include `item_id`, raw NIIN/nomenclature, quantity and nullable
nickname/unit fields captured from the actual physical row before removal.
History prefers captured notification title/vehicle admin; old NULL snapshots
fall back to current labels. Equal event times sort by event ID ascending.
Atomic audits and receipts remain mandatory inside the business transaction:
audit failure rolls back business rows and operation claims. Atomic item edits
keep stored kind `update` and the `items_updated` payload. The corrected F32
test verifies both quantity and enrichment edits and a separate default-only
resave with no item mutation; mandatory notification audits may still exist.

### Historical failures on the base branch

`TestAtomicNotificationItemFieldsSurviveReleasedClientSaves` (tests/shops) and
`scripts/test-shops-isolated_test.py` (broken since wrapper refactor `f459eb4`)
failed on the earlier base branch. The current candidate corrects the atomic
kind/count assertion in Task 20 and the obsolete wrapper assertion in Task 6;
their current verification is recorded in the task reports. Released-client,
fleet, signed-artifact, device, alert-routing and deployment gates remain open.

## Remediation release gate refresh — 2026-10-04

The [target-specific release sheet](shops-server-remediation-release.md) governs
019–023. Local source and disposable tests are distinct from release approval.
Named targets, actual roles, uniformly deployed builds/flags, external writers,
released parser/request/device recovery, signed artifacts, actual Docker
daemon/layers/runtime, mounted Firebase credentials and writable startup `.gen`
remain unexecuted gates. Synthetic container-context sentinels passed only.
Task 26 now validates mandatory 020 assets and 023 retained metadata, including
resolution versions and required privileges, before tagged startup generation
and route registration. The generic generator still supports intermediate schemas.
Its model manifest does not certify the running binary or the actual TMDE view.

The current writer can commit behind an already-observed legacy timestamp
cursor. This Task 22 limitation is OPEN; timestamp semantics remain unchanged,
with no owner waiver and no lossless live legacy catch-up claim. Numeric cursor
restore ABA also remains open without an epoch protocol.

Original F09 shared-test credential rotation and the unrelated Task 18 MCP
credential incident require **separate** credential-owner confirmations. Neither
was rotated or inspected for this task. No secret belongs in evidence artifacts.
C06 remains blocked on active invite population and effective authenticated-user/
edge fleet budgets; no numeric limits, UID limiter, or code-length change was
invented. `service_dates` and `service_reads` remain false.
