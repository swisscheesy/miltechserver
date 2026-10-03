# Shops server production remediation design

- Date: 2026-10-03
- Status: written specification approved on 2026-10-03; implementation plan under preparation
- Repository: `miltechserver`, branch `cleanup`
- Source baseline: `4abd3a19fb0efe1ac51d549e4f7a9866c0f8b8a5`
- Primary evidence: [Shops server production review](../../audits/2026-10-02-shops-server-production-review.md)

## 1. Purpose, scope, and success criteria

Prepare the Shops server for production shipment by repairing every confirmed finding F01–F37 and addressing each conditional concern C01–C10 in the review. Repairs must protect current resource authority, avoid data loss, return accurate errors, preserve valid server contracts, and produce reproducible migration and release evidence.

This is a design for Go server code, server tests, PostgreSQL migrations, build inputs, and server documentation. It includes directly required changes to equipment services, Firebase middleware, bootstrap, and transaction helpers. It does not authorize implementation, migration execution, deployment, credential rotation, or access to an existing database. No Flutter/client source is part of this work.

The report's reproduced defects and source-confirmed paths are the evidence baseline. Conditional concerns depend on additional data, privileges, concurrency, capacity, or fleet conditions. This specification does not claim that any exploit occurred or that a deployed database has a particular state. Historical review results are not fresh verification of future fixes.

Success requires:

- Every F finding has a regression acceptance check and a disposition in the final implementation report.
- Every C concern has a concrete repair, measurement, or release gate; none disappears because a focused suite passes.
- Current permissions are checked in the transaction that performs each mutation, using persisted resource ownership.
- Generated schema changes do not alter the legacy message JSON contract.
- Supported notification-item replacement preserves nickname/unit without restoring a deleted item or its quantity.
- Authorized image cleanup deletes only proven targets and survives process/database-parent deletion failures.
- Full server verification completes on the integrated source and generated inputs, with known failures reported explicitly.
- Named-database, credential, fleet, and released-client gates have their own evidence before shipment.

## 2. Approved decisions and literal boundaries

| Decision | Required behavior |
| --- | --- |
| Architecture | Targeted, coordinated repairs within the existing route → handler → service → repository structure. No Shops rewrite or new general framework. |
| PostgreSQL changes | Necessary additive tables and new integrity/trigger migrations may be designed. Existing data repair requires explicit target-specific authorization. |
| Jet at startup | Automatic Jet generation remains part of application startup and completes before traffic is accepted. |
| Jet after database changes | After every authorized planned PostgreSQL schema or data change, regenerate models through the tagged Jet workflow and verify the result. |
| Generated files | `.gen/**` is Jet-owned. Never edit generated files by hand or use plain `jet`. Changes produced by authorized canonical regeneration are expected. No generation is run during design. |
| Explicit Shop deletion | Only the original creator, while still a current member, may explicitly delete the Shop. No ownership transfer or request-controlled bypass. |
| Last admin | With other members remaining, the last admin must promote a successor through the existing operation before leaving or being removed. |
| Final member | A sole current member may leave; internal final-member cleanup is distinct from creator-only explicit deletion. |
| Unattached upload discard | The uploader or a current Shop admin may discard a registered, unattached upload. Current membership is required. |
| Published images | Authorized message edits/deletion release references; cleanup is possible only after the final managed reference disappears. |
| Invite restrictions | Reject supplied non-null expiry/max-use restrictions. Keep unrestricted, non-expiring invitations and their existing default request behavior. |
| Membership removal | Removal revokes current access, but is not a ban. A valid active invitation can readmit the user. Admins revoke invitation codes separately. |
| Retained item metadata | Retain nickname/unit by notification and exact stored NIIN until notification deletion. Recover metadata after DELETE/re-add without resurrecting rows or quantities. |
| Legacy audits | Best effort: business success survives audit-delivery failure; emit structured warnings and monitor them. Atomic-save audits remain mandatory in the write transaction. |
| Service status | Implement completed, overdue, due_soon, and scheduled filters using the existing timestamp domain. Date-only activation remains gated. |
| Existing targets | Disposable rehearsal first; separately authorized `miltech_ng_test` before `miltech_ng`, with identity/schema checks in the same session. |

Valid legacy routes, envelopes, request defaults, and nullable fields remain supported. Security fixes may reject previously accepted unsafe, malformed, contradictory, or unauthorized requests. This specification makes those changes explicit in section 13. A server-only implementation cannot establish that a released client handles every new recovery signal or preserves every unknown raw unit code; that requires a separate owner handoff.

Do not modify Flutter source, manually modify `.gen`, silently rewrite checksum-pinned SQL, print credentials, run application startup against an existing target during development checks, or treat this document's approval as database/deployment authorization. Preserve unrelated workspace changes and the original audit.

## 3. Architecture and implementation boundaries

Retain the current bounded packages: `api/shops/{core,members,settings,lists,vehicles,messages,aggregates,capabilities}` and `api/equipment_services/{core,completion,queries,calendar,status,shared}`. Existing atomic-save receipts, fingerprinting, deterministic notification locking, list dependency checks, usage arithmetic, and healthy sync-v2 pagination are protections to preserve.

New responsibilities are deliberately narrow:

| Component | Responsibility and interface boundary |
| --- | --- |
| Shops contract middleware | Serialize an unwritten handler error once, using existing classification/redaction. |
| Shared transaction helper | Context-bound transaction creation, error return, commit/rollback handling; repositories return their own typed results. |
| Shared authorization | Coordinate current Shop membership/policy with persisted resource ownership inside write transactions. |
| Message response DTO | Own the public legacy message fields independently of Jet's generated column set. |
| Message asset repository | Reserve immutable upload targets, resolve managed references, and enqueue trusted cleanup jobs transactionally. |
| Message cleanup worker | Lease/retry durable jobs and perform Azure operations outside database locks. |
| Notification metadata repository | Resolve and retain logical enrichment across physical item deletion/recreation. |
| Aggregate read transaction | Supply one queryable snapshot to every component and its membership check. |
| Shared Jet generator | Reuse the tagged template, verified connection configuration, and canonical output namespace for startup and the CLI. |
| Readiness probes/runbooks | Separate runtime schema/access availability from operator-verified data, fleet, and release readiness. |

Proposed new source files are `api/shops/messages/assets.go`, `asset_repository.go`, `asset_repository_impl.go`, `cleanup_worker.go`, and `api/shops/vehicles/notifications/items/retained_metadata.go`. Add a small shared generator package under `internal/jetgen/`. Keep HTTP adaptation in existing handlers and Azure calls behind a narrow testable interface. Names can be adjusted to existing conventions in the implementation plan; the responsibilities and boundaries must remain explicit.

### 3.1 Context and transaction contract — F22

Pass `context.Context` through every Shops/equipment-service service, repository, authorization, and username-lookup boundary. The HTTP request supplies it. A worker uses its bounded service context. Do not replace either with `context.Background()` in these paths.

Reuse the existing `WithTxContext(ctx, database, callback) error` in `api/shared/db/transaction.go`. Add `WithTxOptions(ctx, database, options, callback) error` for explicit transaction options and let the existing context helper retain its signature and READ COMMITTED behavior. Use `BeginTx(ctx, options)` and context-bound statements. Preserve unrelated consumers of the generic `WithTx` helper until deliberately migrated. Callback failure rolls back; commit failure is returned; deferred cleanup must not hide the primary error. Repository interfaces return actual affected rows/results rather than infer success from request length. This source-verified naming correction does not change the approved context/transaction behavior.

Cancellation before transaction acquisition, while waiting for locks, or before commit must release work and prevent a successful response for uncommitted work. A response lost after commit remains an uncertain outcome. Do not automatically replay legacy non-idempotent creates after cancellation or an ambiguous commit. Existing atomic operation IDs and receipts remain the mechanism for resolving atomic-save uncertainty.

Retries are allowed only for a known rolled-back transaction, such as PostgreSQL `40P01` or `40001`, with a bounded attempt count and the same live context. An ownership change found during lock acquisition requires a fresh resolution/recheck, not reuse of cached authority. Never retry only the final statement of a transaction.

## 4. HTTP, authentication, and generated-model boundaries

### 4.1 Complete production error handling — F01, F31

Extend `api/shops/shared/contract.go` so the already registered Shops/equipment-service middleware calls `c.Next()`, then serializes the first relevant `c.Error` only if the response is still unwritten. Use the existing scoped writer and failure classifier; unknown internal details remain redacted. A response already written by a handler or another middleware is never written again. Tests that also install the global `ErrorHandler` must observe one response.

Keep this repair scoped rather than reorganizing every public route. Test real `api/route/route.go` assembly using controlled identity/dependencies: legacy and contract-2 requests, missing/forbidden resources, validation, internal failure, already-written responses, and multiple queued errors. A failed authenticated Shops operation cannot finish as an empty HTTP 200.

In `api/shops/route.go`, absent `deps.Env` means optional capabilities are off. Route registration must not connect to PostgreSQL/Firebase/Azure, regenerate models, or panic to infer configuration. Bootstrap validates mandatory production configuration separately. Run the entire route package after fixing the nil-config panic, including tests previously aborted by it.

### 4.2 Firebase current-session enforcement — F06

Use the HTTP context for `VerifyIDToken` and the existing `GetUser` request in `api/middleware/authentication.go`. Strictly parse the supported Bearer form; reject missing/invalid tokens and nil/deleted records without executing the handler.

For the pinned Firebase Go SDK v4.15.2, enforce the same disablement/revocation decision as its checked verifier using the already fetched user record:

- Reject `UserRecord.Disabled`.
- Reject when `token.IssuedAt * 1000 < user.TokensValidAfterMillis`.
- Use the SDK's error classifications to distinguish invalid/deleted/revoked identity from a transient identity-service failure.

The revocation comparison is **IssuedAt**, not AuthTime. Pin parity tests to the installed SDK behavior, including equality and time units. Do not add an unnecessary second account lookup. Invalid/deleted/disabled/revoked identity returns 401; a transient dependency outage returns a controlled 503 and never falls through as authenticated. Logs may identify the failure category and stable actor ID, but never tokens or private credentials.

In `bootstrap/authentication.go` and `bootstrap/app.go`, initialization errors stop startup before route acceptance; a nil authentication client is not a usable production dependency. The shared authenticated middleware change affects other authenticated server features, so cover their existing envelope behavior as well. Shops negotiation must not be falsely promised on authentication failures that occur before the Shops contract middleware.

Primary reference: [Firebase session revocation](https://firebase.google.com/docs/auth/admin/manage-sessions#detect_id_token_revocation), with the installed SDK source as the implementation authority.

### 4.3 Stable message wire contract — F08

Replace generated-model embedding in `api/response/user_shops_response.go` with a handwritten message DTO containing exactly:

`id`, `shop_id`, `user_id`, `message`, `created_at`, `updated_at`, `is_edited`, `parent_id`, `author_username`.

Preserve current types, nullability, timestamp formatting, and key presence; do not add `omitempty` to nullable message fields. `insertion_number` belongs to the sync envelope, never the nested legacy message object.

Replace `ShopMessages.AllColumns` on legacy reads, GetByID, cursor reads, aggregates, and create readback with the explicit legacy column projection. Map sync's raw-SQL reader into the same DTO. Legacy requests must work both before 018 and with the expanded generated schema; sync stays unavailable before its schema exists. A post-018 generated model is allowed to include insertion number normally: no generated tag override or manual edit is required.

Schema changes to unrelated generated models must not leak new fields through new Shops aggregate DTOs. Preserve existing non-message wire contracts and verify their affected projections after generation.

### 4.4 Required startup generation — F08, F30

Keep generation before accepting traffic. Share the snake_case table/view template between `main.go` and `tools/jetregen/main.go`; use `bootstrap.Env.Port` rather than startup's hardcoded 5432. Both generator and application must use the same verified host, port, database, schema, role, and TLS configuration.

Use the installed Jet `GenerateDB` entry point where necessary to generate into canonical `.gen/miltech_ng/<schema>/` from a verified test/disposable source. Application imports remain canonical even when the source database has a different name. Record the real source identity separately; do not disguise it as `miltech_ng`, alter generated files afterward, or copy an arbitrary alternate namespace over production models.

Preflight output writability and required catalog access. Serialize generation for a shared output directory; generate to staging and publish a complete output tree so failure cannot leave partial inputs. Stop startup with a sanitized diagnostic on configuration, access, or generation failure. The deployment must provide a writable output location. Do not silently skip generation because the binary already exists.

Runtime source generation does not rebuild the running Go binary. Release compilation must therefore use models generated from the verified intended schema, with a recorded schema/generation manifest. Startup confirms compatibility and regenerates source; it cannot make an older compiled binary support new required columns. Validate required columns/types/functions and the build's supported schema range, allowing explicitly compatible additive columns. Do not require byte-identical schema fingerprints where the pre/post-018 legacy projection is deliberately compatible. An incompatible schema/build combination blocks acceptance of traffic rather than implying regeneration upgraded the executable.

This deliberately supersedes ADR-022's message-sync workaround “do not regenerate” and the equivalent `docs/project_notes/key_facts.md` note. Update those documents during implementation to explain the fixed DTO/projection boundary, mandatory regeneration, and retained trigger allocator. Do not change the accepted unbounded-read decisions in ADR-015/ADR-016 incidentally.

## 5. Current resource authority and Shop lifecycle

### 5.1 Coordinated write authorization — F02, F05

Resolve the target's Shop from persisted rows, not a supplied URL/body Shop ID. A supplied identity must agree. For equipment services, determine the service's actual Shop and equipment before evaluating its existing author/admin policy. Admin authority in another Shop is never usable.

All writers use a deterministic lock order: Shop → current membership/policy → referenced lists, ordered by ID → owned resource → children/assets, ordered by ID. The exact existing helper can be reused; do not weaken its coordinating Shop lock. Lookup before locking is for resolution only. Recheck ownership and current authority after locking. For multi-resource operations, order all relevant locks consistently and reject foreign targets before writes.

Cover vehicle create/metadata PUT/absolute usage PUT/usage PATCH, Shop rename/settings/delete, invite create/revoke/delete/claim, service create/update/complete/delete, message create/edit/delete, and existing list/notification writers. Typed actor identity reaches the repository; cached role/membership and preflight success cannot authorize persistence.

A write that holds the coordinating lock and authorizes before removal may commit first. When removal or demotion commits first, later authorization must reject the write. Use deterministic barriers to prove both orderings, without sleep-based tests. Check every F05 operation, not only PATCH.

Service completion/deletion retains the existing service-author/admin policy, evaluated against the persisted service and its actual Shop inside the transaction. Do not expand ordinary-member authority, grant original-Shop-creator privileges after removal, or return success when no authorized row changed.

### 5.2 Atomic creation, metadata, settings — F07, F15, F17

Create the Shop and its initial creator-admin membership in one transaction. Persist the requested `admin_only_lists` value. Failure of membership creation rolls back both; no 201 or inaccessible Shop remains.

Any current Shop admin may rename through the same service and persistence rule. Remove the contradictory creator-only rename predicate, while keeping explicit deletion creator-only. A metadata update must not touch `admin_only_lists` unless the dedicated setting operation is requested. Both setting endpoints distinguish omitted input from explicit false; false is valid and persists. Pin omission behavior with tests rather than relying on a required binding tag on a bool.

### 5.3 Admission and invitation lifecycle — F13, F14, F28

Claim an invitation and insert membership under the Shop lifecycle lock. Re-read/lock the invite and its active state inside the transaction. Revocation/delete participates in the same ordering; revocation committed first prevents admission. Invite lookup outside the transaction is only for finding the Shop.

Admission inserts a new member and never updates an existing member's role. Duplicate joins preserve the stable already-member outcome and cannot demote a concurrently promoted admin. Promotion alone owns role changes.

Keep existing current-member invite generation and current-admin revocation/deletion policy. Reject non-null `max_uses` or `expires_at`, including a supplied zero or empty restriction value, instead of accepting false security assurances. Omitted/null values retain unrestricted defaults. No expiry/use-count schema is introduced.

Removal is membership revocation, not a ban. A later valid active invitation can readmit that user. Document this result; do not create a hidden ban/readmission table.

### 5.4 Departure and deletion — F16

Under the Shop lifecycle lock, evaluate the current members/admins and make exactly one transition:

| State/action | Outcome |
| --- | --- |
| Last admin attempts exit/removal while other members remain | Reject with existing conflict semantics; a successor must first be promoted. |
| More than one member, successor/admin requirement satisfied | Remove the target membership. |
| Sole current member leaves | Internally delete the Shop and schedule trusted asset cleanup in that transaction. |
| Explicit deletion by current original creator | Delete through the explicit creator-only operation and schedule cleanup. |
| Explicit deletion by anyone else or removed creator | Reject. |

Keep final-member cleanup separate from explicit deletion in repository methods; do not expose a `force`/bypass field. Concurrent leaves/removals re-evaluate after acquiring the Shop lock, so they cannot leave a memberless Shop. A noncreator final member is not trapped.

Existing orphan/adminless Shops require an inventory and an owner-approved repair manifest. Do not automatically promote a member or guess ownership in a migration.

## 6. Message images and reply integrity

### 6.1 Ownership model — F03, F04, C07

Text is display content, never deletion authority. Stop deriving Azure delete targets from marker suffixes or arbitrary hosts. Preserve readable legacy text/marker syntax, including unverified/external images, but only registry-proven same-Shop assets acquire managed references or cleanup authority.

Add these narrowly scoped tables, generated by Jet after authorized migration:

| Table | Required data/invariants |
| --- | --- |
| `shop_message_uploads` | Server-minted upload UUID; immutable original Shop UUID, uploader UID, storage account/container/blob key, extension, canonical URL; state, operation lease/deadline, timestamps and classified failure. Unique immutable blob target. |
| `shop_message_asset_references` | Message UUID, upload UUID, and immutable Shop UUID; unique message/upload pair. Composite FKs match both message and upload `(id, shop_id)` keys. Message deletion removes references; upload deletion cannot discard active references. |
| `shop_message_blob_cleanup_jobs` | Job UUID, trusted immutable target or exact Shop prefix, related upload ID when applicable, reason, state, attempts, next attempt, lease owner/deadline and sanitized failure. Target survives deletion of the Shop/message/upload parent. |

The registry's original Shop ID and cleanup target are immutable snapshots, not cascading-delete authorities. Use a nullable live-parent link if useful, but do not cascade away the proof or pending cleanup. The asset migration supplies the composite keys needed by references; the later reply constraint reuses the message key. Queue state and indexes support bounded due-job claiming; add query-driven indexes, not speculative ones. References and state changes use generated Jet tables; no user-supplied raw SQL.

Upload states are `uploading → ready → cleanup_pending → deleting → deleted`, with interrupted/rejected finalization also taking `uploading → cleanup_pending`. Attachment/discard decisions inspect these states under the asset lock. Only ready assets attach; cleanup/deletion is irreversible for that unique key. Cleanup jobs are pending, leased, retryable, completed, or manual-review; expired leases return to pending and manual-review retains the target until resolved. Do not cascade away terminal tombstones needed for late-PUT reconciliation.

### 6.2 Bounded upload and reservation — F25

Before `PostForm`/`FormFile`, install a whole-request byte bound. Use a 6 MiB total request budget, retaining the 5 MiB actual file limit and allowing 1 MiB for multipart overhead/fields. Reject excessive parts/fields within that bounded body. Remove multipart temporary files. Read actual file bytes with a limit-plus-one check; do not trust only the multipart header size. Preserve current supported image formats rather than inventing an unrelated MIME restriction.

Authenticate, parse the bounded request, validate Shop identity, and reserve the unique upload under current membership. Commit that reservation before Azure I/O. Upload with the HTTP context outside database locks. Finalize under fresh current authority: success becomes ready; loss of authority or failed finalization becomes cleanup pending. Rejected/oversized/nonmember requests make no Azure call.

The response retains exactly `message_id` (the upload UUID), `shop_id`, `image_url`, and `file_extension`. The upload UUID is not a message-row UUID. The server mints the blob key once; extension/input cannot select an arbitrary target.

### 6.3 Reference updates and discard

Parse every marker on message create/edit. Resolve known registry targets by immutable canonical identity. Within the authorized message transaction, lock affected upload rows in sorted order and insert/remove reference pairs. Known same-Shop assets in cleanup/deleting/deleted states cannot gain a managed reference. A foreign, copied, external, or unverified marker cannot grant ownership or authorize deletion of its target.

The existing orphan route may discard only a registered same-Shop upload with zero references, by its uploader or a current Shop admin. Other members cannot discard it. Lock and recheck membership, asset state, and all references. Never guess orphan status from absence of a message with the upload UUID. Published assets remain protected even from their uploader until authorized message changes remove the final reference.

Ready unattached uploads gain no automatic expiry policy. That would change the supported draft lifecycle and is outside the approved design.

### 6.4 Durable cleanup and recovery

Authorized edits/delete, same-Shop reply cascades, and whole-Shop deletion identify affected managed references before deletion, release them, and enqueue final-reference cleanup in the same transaction. Add a narrow AFTER DELETE hook on managed reference rows that copies the registry's immutable target into a cleanup-candidate job. It covers FK/account cascades and supported external row deletion without taking an asset/Shop lock or making Azure calls. The worker rechecks zero references before deletion; a candidate is not deletion authority by itself. Duplicate candidate jobs are safe, and enqueue must not deduplicate against an already claimed/completing job in a way that loses the final removal. Verify hook invoker privileges for every supported deletion role. Trusted registry reconciliation remains a backstop for interrupted uploads and inventory anomalies. Tests must cover every supported deletion path, including account-related cascades if present in the verified schema.

Worker flow:

1. Claim a bounded batch with row locking and a lease, then commit the claim. No queue lock is held while acquiring Shop/asset locks.
2. Recheck the registry and zero references, freeze the asset in deleting state, and commit. New managed references are now forbidden.
3. Verify the job's immutable account/container matches the configured authorized storage target, then delete outside database locks with a bounded context. Configuration mismatch requires manual review rather than deleting the same key in another account. A 404 is successful deletion.
4. Persist success or retry with bounded backoff and a sanitized error. Lease expiry allows crash recovery; deletion is idempotent.

Use the existing Azure timeout/retry conventions, with validated worker limits, one bounded batch per poll, and clean shutdown. Monitor backlog age, attempts, permanent/manual-review failures, and lease recovery. Never solve retry failure by dropping the durable job.

An upload attempt has a bounded operation lease. Cleanup waits for a live uploading lease rather than racing an active PUT. Expired/interrupted reservations become cleanup candidates. Because cancellation cannot prove a remote PUT did not complete late, preserve terminal registry tombstones and reconcile trusted failed/deleted targets for late objects. Unique keys are never reused. Finalization cannot reactivate a tombstoned upload. This provides eventual cleanup without interpreting client text as a cloud target.

Whole-Shop deletion can enqueue a trusted exact `<persisted-shop-uuid>/` prefix job. Enumerate/delete that exact prefix with durable paging/retry and coordinate in-flight reservations through the registry. Never accept a prefix from a request or broaden deletion to a container.

Unknown historical uploader ownership is protected. Inventory canonical storage targets and historical relationships; only an explicit owner-approved mapping can register them for individual cleanup. Do not infer uploader from a message author. Historical readable messages remain readable while this gate is unresolved.

### 6.5 Same-Shop replies — F11

Resolve/lock a reply parent inside the authorized write transaction and require the parent's persisted Shop to match. Plan a composite relationship using a unique `(id, shop_id)` key and a `(parent_id, shop_id)` foreign key after historical preflight. Preserve pinned same-Shop cascade behavior and test its managed-asset cleanup.

Do not silently edit 005 or 018. The pinned fixture and later migration have differing historical deletion-action evidence; inspect each authorized target's actual FK action. If it differs from the intended cascade policy, require a reviewed conversion/repair manifest before replacement. Existing cross-Shop parents or missing parent identities cause migration refusal until explicitly resolved, never silent orphaning or cross-Shop deletion.

## 7. Equipment intent, validation, deletion, and audit semantics

### 7.1 Role-independent equipment mutation intent — F12, F19, F20

Classify the existing legacy usage-only profile before role branching: at least one non-null tracked reading is provided, and NIIN/model/serial/UOC/comment are absent or empty. For every role, this updates only tracked values and ignores snapshot admin/base readings. A privileged caller must not be routed into metadata erasure.

For other authorized metadata updates, use field presence rather than default empty strings. Preserve omission; permit explicit clearing only where the existing domain allows it; persist an actual Admin edit. Maintain existing absolute tracked-reading semantics and nullable/omission behavior. Do not introduce a required intent field. A full metadata/admin-only edit can omit tracked readings; an ambiguous payload matching the established usage-only profile retains usage-only meaning.

Validate new base mileage/hours as nonnegative on create and metadata writes. Ignored usage-only snapshot labels are not validated as a request to change base values. Preserve established effective-reading arithmetic and bounds. Inventory historical invalid base values; do not zero them automatically. Database hardening follows approved data repair and must not unexpectedly prevent unrelated writes to an unrepaired historical row.

This does not introduce versioned conflict resolution for stale absolute PUT versus a newer PATCH. Pin the current arithmetic and expose uncertainty honestly; do not claim presence handling solves a different concurrency contract.

### 7.2 Equivalent input validation — F21

Validate every nested bulk element before any row/audit writes, in addition to handler binding. Single, update, bulk, and atomic-compatible paths share positive quantity, trimmed-nonempty required content, established rune/column limits, and supported notification types `M1`, `PM`, `MW`. Preserve supported NIIN/NSN formats; no new numeric-only pattern or arbitrary cap.

The notification bulk outer ID is authoritative. A supplied nested ID must agree; an omitted nested ID inherits the outer target if the supported request shape permits omission. Contradictory targets reject the entire request. Keep existing same-Shop multi-list bulk behavior; do not reinterpret its outer list ID as a new single-list restriction. Current persisted authorization still checks each actual target.

Nickname may be explicitly empty to clear and is limited to 50 runes. Unit must be nonempty after trimming and at most 50 runes; retain the raw valid code without an enum allowlist or conversion. Omitted/null enrichment preserves existing semantics. A newer valid code must not be silently converted to EA by server validation.

### 7.3 Actual, idempotent batch deletion — F36, F37

Deduplicate requested IDs. Find and authorize every surviving item; skip missing IDs safely. Replace first-ID service anchoring as well as repository authorization. An authenticated all-missing request is a no-op with count zero; it does not reveal foreign ownership. A foreign surviving item rejects the whole batch before deletion. Recheck current restrictions under coordinating locks.

Return actual affected counts from persistence through service to the established success envelope. Duplicates, absent rows, and concurrent removal cannot increase the reported count. Apply the same count rule to list and notification item bulk removal.

### 7.4 Event-time history and best-effort legacy delivery — F34, C08

Capture physical item ID, exact NIIN, nomenclature, quantity, nickname, raw unit, notification title, and equipment admin at event time, including removal before deleting the row. Prefer captured historical labels; use current labels only when old snapshots lack them. Retain current null/empty-history envelope behavior and documented limits. Give equal event timestamps an event-ID tie-breaker without claiming offset pages are cross-request snapshots.

Legacy business commits remain successful when their later audit delivery fails. Emit a structured warning with operation/resource/actor identifiers, failure category, and correlation ID. Define log-based alerts for audit failures; do not swallow them silently or claim gap-free legacy history. This approval does not introduce a mandatory legacy audit outbox or roll back legacy business success.

Atomic saves retain mandatory transactional audit/receipt behavior. Preserve stored `change_type=update` with `items_updated` payload semantics. Fix `tests/shops/shops_notification_item_fields_test.go` to inspect each operation's actual kind/payload and count both real item changes; assert default-only resaves produce no change. Do not alter production audit kinds just to satisfy F32's incorrect expectation. Resource timestamps and receipt committed_at represent different events; equality is not required.

## 8. Durable notification-item enrichment — F18

Add `shop_notification_item_metadata`, keyed by notification UUID and the exact stored NIIN. Store nullable nickname/unit, a resolved/ambiguous state, and provenance/version information needed to identify the latest accepted metadata edit. On conflict, preserve raw candidate values with their source item UUIDs rather than overwrite the evidence. The notification owns the record and its deletion ends retention. The record holds no quantity and is not an active item row.

All legacy single/bulk add/update/delete and atomic item writers participate under the existing notification coordinating lock. Resolve an existing physical item's omitted values from that row; update retained metadata when explicit fields change. Before physical deletion, retain its metadata. Re-add with a new UUID and omitted fields recovers the retained values. Explicit empty nickname clears; valid explicit unit replaces; absence/null preserves the approved existing interpretation. No metadata crosses notification boundaries or normalized NIIN spellings.

Compute atomic request fingerprints from the original request before retention lookup or resolution. Replays return the original receipt and must not reapply old metadata over intervening edits. New accepted operations resolve against current retained metadata. Keep user-scoped operation identity, replay reauthorization, deterministic create UUID, and receipt-only replay unchanged.

### 8.1 Identity collisions and rollout refusal

Physical UUID uniqueness does not prove one logical NIIN identity. Preflight duplicate NIINs and compare raw metadata. Do not aggregate quantities, normalize NIINs, convert units, choose an arbitrary winner, or silently add active-item uniqueness as part of backfill.

Backfill resolved metadata only where the notification/NIIN identity is unambiguous. Conflicting historical duplicates block activation until the owner supplies an explicit resolution manifest. Keep same-ID preservation for existing rows; never borrow a sibling's fields merely because its NIIN matches.

At runtime, lock the logical key and inspect active rows before an operation relies on retained fallback. Compare raw nullable metadata; do not pick a winner using physical UUID or query order. If candidates disagree, retain ambiguous state and return a conflict when fallback is needed. An explicit edit of one physical row remains attributable to that row; it cannot silently redefine a conflicting sibling. Deletion alone cannot clear ambiguity, even when only one candidate remains. Resolution requires an owner-approved repair or explicit metadata operations establishing a consistent logical value. With no active conflicting siblings, a re-add supplying both nickname and valid unit may deliberately establish that value; an omitted field still cannot borrow an ambiguous value. No new active-item uniqueness/merge policy is assumed. Any later uniqueness proposal must be separately reviewed with its data migration.

Deletion followed by re-add is two committed operations, so this repair guarantees enrichment retention, not resurrection of a removed quantity or automatic idempotence of a separate legacy INSERT. Test the complete sequence and alternating legacy/atomic operations, not only same-ID atomic omission.

## 9. Consistent reads, pagination, and service behavior

### 9.1 One aggregate snapshot — F23

Use a context-bound, read-only REPEATABLE READ transaction for each aggregate. Perform membership evaluation, counts, summary, selected sections, notification headers/items, services, PMCS inspections, and fault/comment counts through the same queryable connection. Membership is evaluated as of that request snapshot; each later request/chunk rechecks it. Do not start nested component transactions or fall back to the pool for username enrichment. Sequence component reads on the transaction connection, closing rows before the next query; replace former pool-parallel composition with justified batched/joined queries where measurement calls for it.

Preserve documented distinctions between section-limited and Shop-wide totals and every omitted-unbounded contract. Joined/batched username lookup uses the same snapshot and preserves existing `Unknown User`/nullable fallback. A query/context failure is an error, not an unknown username.

Test with a deterministic writer commit between component reads: the response must represent the old or new committed version consistently. An isolated default READ COMMITTED transaction is insufficient because its statements can observe different versions. Reference: [PostgreSQL transaction isolation](https://www.postgresql.org/docs/14/transaction-iso.html).

### 9.2 Legacy message cursors — F24

Order and bound by `(created_at, id)` everywhere, with matching comparison/order direction. Older paging drains older tuples. `after_id` paging selects the nearest newer tuples first so repeated continuation drains a burst instead of repeatedly returning the newest rows. Compute continuation from the draining boundary, even if a preserved presentation order differs. Retain legacy envelope and limit semantics.

Test ties spanning multiple pages, a burst larger than the limit, both directions, and boundary IDs. A deleted legacy ID anchor cannot supply its lost timestamp: return a deliberate reload/reset outcome through the supported legacy error envelope. Do not claim the ID-only cursor survives deletion. Sync-v2 opaque tuple history cursors continue across deleted anchors and retain their existing authorization.

### 9.3 Complete PMCS history — F35

Replace growing equipment/inspection UUID `IN` parameter lists in `api/shops/aggregates/repository_pmcs_history.go` with membership-scoped joins/subqueries for equipment, inspections, faults, and comments. Preserve guide/custom source union, nullable provenance, authored-tree privacy, all accessible records, and count semantics. Do not fix the 65,535 parameter ceiling by truncating history.

Use the same read snapshot. Verify query construction beyond 65,535 accessible inspections and run a populated disposable capacity fixture. Record query count, payload size, memory, and representative `EXPLAIN (ANALYZE, BUFFERS)` results before adding indexes.

### 9.4 Completion history — F26

When the resulting service is already completed and the request omits/nulls completion date, preserve the existing date. First completion defaults to the operation time; an explicit date is an intentional update. Reopen follows existing semantics. Repeated no-date completion does not change the original completion time, including concurrent retries.

Apply this in both metadata update and completion repositories under actual-Shop authority. Do not incidentally change the current `is_completed` request omission/default contract; pin it in tests. This date-preservation fix does not activate date-only storage or parsing.

### 9.5 Service filters and stable pages — F27

Share predicates between Shop/equipment listing and count queries. Evaluate time once per read snapshot:

| Status | Predicate |
| --- | --- |
| completed | `is_completed = true` |
| overdue | Incomplete; non-null service_date strictly before evaluation time. |
| due_soon | Incomplete; service_date strictly after evaluation time and at/before seven days later. |
| scheduled | Incomplete; service_date strictly after evaluation time, including due_soon. |

These are timestamp/date predicates, not a newly invented mileage/hour forecast. Existing dedicated due-soon retains its `days_ahead` 1–30 control. Blank status means no status predicate; unknown nonempty status is invalid. Apply status, completion, type, equipment, and date bounds conjunctively on both routes; counts use identical predicates.

Add service ID tie-breakers and intentional null order to the existing sort directions. Count and rows share a read snapshot. Retain offset pagination; inserts/deletions between requests can still shift pages, which a tie-breaker cannot prevent.

Classify malformed supported timestamp inputs as known validation failures using existing negotiated error mappings, including calendar. Do not expose `time.Parse` internals as internal errors or change date interpretation by connecting the gated date-only parser.

## 10. Message-sync invariants and mixed writers

### 10.1 Common integrity guard — C02, C03, C04

Within each per-Shop sync read snapshot, validate the common invariants before advancing a watermark: no message has a NULL insertion number or created_at; a nonempty Shop has a counter; and the counter is at least the maximum assigned number. A missing counter in an empty Shop legitimately yields zero. A counter above the maximum is valid after deleting the highest-numbered messages; equality is not required.

Broken numbering/readiness fails closed with a sanitized 503 and monitoring signal. Catch-up must not filter NULL numbers away and advance silently. Keep held upper bounds, deleted gaps, reconciliation scope, member checks, and the trigger as the sole allocator. Do not assign insertion numbers in application code.

Malformed `through < after` remains invalid input. A previously usable `after`/held watermark ahead of the current counter returns HTTP 409 with explicit code `message_sync_reset_required`, requesting a new initial snapshot. Keep this distinguishable from corruption/unavailable and document caller recovery. Do not silently lower watermarks.

Existing numeric watermarks cannot detect a restore/reset that reuses numbers within the current range. Do not claim transparent recovery from that ABA case. Gate numbering rollback/reapply, restore, and wrong-target cutovers operationally; clients must discard affected sync state through the documented owner handoff. No Flutter fix or mandatory client-update flow is included.

### 10.2 Allocator bridge — C01

In a new migration, revise the insertion trigger to acquire the relevant Shop row `FOR KEY SHARE` before counter allocation. Current writers already acquire the stronger Shop mutation lock first; the bridge gives old-style inserts the same Shop-before-counter order. Keep the counter lock/update as the sole commit-ordered allocator and preserve rollback/gap semantics.

Do not weaken the current `FOR UPDATE` authorization lock or silently change pinned 018. PostgreSQL key-share conflicts with a Shop `FOR UPDATE` lock, so validate the intended order with physical sessions. Arbitrary external transactions that lock counter/message rows before Shops still require a cutover fence or reviewed retry policy; the bridge is not a proof about every possible writer.

Test current versus old legacy-shaped INSERT, same-Shop and different-Shop inserts, deletion, rollback, and restricted writer roles with deterministic barriers. Verify actual SELECT and row-lock prerequisites as well as counter INSERT/UPDATE/SELECT. Do not introduce blanket grants or SECURITY DEFINER to hide privilege failures.

References: [PostgreSQL row locks](https://www.postgresql.org/docs/14/explicit-locking.html), [SELECT locking privileges](https://www.postgresql.org/docs/14/sql-select.html).

### 10.3 Capability discovery

Extend atomic readiness to required schema and application-role access, with advertisement equal to flag AND readiness. Retain callable-before-advertisement atomic-save intent: the route remains callable once its required schema exists, even while advertisement is false. Routes still fail safely when required schema/access is absent.

Strengthen message schema checks to required columns/functions/trigger/indexes and effective access. Runtime probes are read-only, bounded, and context-aware; an unexpected probe failure advertises false and logs the category. Per-request integrity checks protect data that a catalog probe cannot prove. Fleet and real write verification remain operator gates. Keep `service_dates` and `service_reads` false until the mapping gate is separately resolved.

## 11. Migrations, generation, and existing-data gates

Proposed new migration slots, based on the reviewed maximum 018, are below. Reconfirm availability when implementation starts; renumber new pairs together if another branch has used a slot. Each forward migration has a matching rollback and a write-up; existing pinned SQL remains unchanged.

| Proposed pair | Purpose | Mandatory populated-data preflight |
| --- | --- | --- |
| 019 message allocator lock-order bridge | Replace allocator function/trigger ordering while retaining numbering. | Actual 018 definition/checksum, counter invariants, writer roles, FK/lock behavior. |
| 020 message asset lifecycle | Registry, references, independent cleanup jobs, narrow deletion hooks. | Actual deletion cascades; existing asset inventory remains protected unless mappings are approved. |
| 021 message parent ownership | Same-Shop reply constraint, reusing the asset migration's composite message key. | Cross-Shop/missing parents and actual FK action; explicit repair/conversion manifest first. |
| 022 equipment base usage integrity | Nonnegative base-reading checks. | Invalid base readings; explicit repair manifest first. |
| 023 notification item metadata retention | Logical enrichment table and unambiguous seed. | Duplicate exact NIIN identities/conflicting metadata; no automatic merge. |

The implementation plan splits relationship and usage integrity because their data gates differ, and numbers the new pairs in implementation order. Schema additions are not permission to transform historical data without review. Row-count/checksum evidence must show refusals leave the original state intact.

### 11.1 Disposable rehearsal

Use the marker-protected loopback database provisioned by `scripts/test-shops-isolated.sh`. Preserve its target/package/marker refusals. Test empty and populated forward paths, malformed-data refusal, repeated runner invocation, interrupted migration rollback, generated types/JSON, and permitted reverse/reapply. Unsupported destructive reversals must refuse before dropping data.

After **each** rehearsed migration or explicit data-repair step, run tagged Jet into an isolated output workspace and verify compilation and exact API keys. These are the planned database-change steps referred to in this document. This avoids rewriting the shared checkout's ignored generated tree while comparing pre/post schema fixtures. Do not edit generated models to make a fixture compile.

### 11.2 Existing targets and runner safeguards — F29, F30, C10

No existing target is contacted during this design. Later migration work requires separate explicit authorization naming the target/action. Order: `miltech_ng_test`, then `miltech_ng` only after test-target acceptance.

In the same migration connection/session, verify actual database name, server address/port, role, PostgreSQL version, schema fingerprint/migration checksums, and intended transaction. Distinguish operator and application roles using real `DB_USERNAME` configuration, not an assumed `postgres` role or an incorrect environment-variable name. Reject inherited target selectors and unapproved identities. Deliberately `UNPINNED` schema checksums stay a refusal until authorized evidence supplies their values.

Replace comma-list `has_table_privilege` gates with independent checks joined by AND; PostgreSQL treats a list as any listed privilege. Verify relevant schema usage, SELECT, INSERT, UPDATE, sequence/function/locking access individually and any needed defaults for newly created objects. Rehearse SELECT-only, INSERT-only, UPDATE-only, and fully authorized roles. A real legacy-shaped send under the application role remains mandatory before production, even with sync disabled, because the trigger still runs.

After every approved migration **and every approved data repair**, regenerate with `go run ./tools/jetregen` using the verified source identity and canonical output, then verify generated manifest, build, API keys, and `git status .gen`. Tracked `user_pmcs_*` outputs must be unchanged; unexpected changes stop the sequence. Never use plain `jet`. Update the 017 guide to this test-first workflow.

Primary privilege reference: [PostgreSQL privilege inquiry functions](https://www.postgresql.org/docs/14/functions-info.html).

### 11.3 Cutover and rollback

Additive schema may precede capability advertisement, but new ownership/retention writers require a uniform repaired fleet or a write fence. Old binaries bypass the new lifecycle protocol and still contain authorization defects; they are not safe concurrent mutation participants merely because their INSERT works. Rehearse any explicitly permitted old/new overlap for the allocator bridge separately.

Prefer a roll-forward or a rollback artifact that retains security and ownership protections. Disable advertised optional capabilities when appropriate, preserve additive data, and stop unsafe writes if no safe rollback binary exists. Flags do not disable always-on database triggers or repair old authentication.

Reverse migrations are disposable rehearsals by default. A populated retention/job/receipt reverse refuses data loss unless the owner separately authorizes and supplies an explicit resolution. Never reverse 017 against a named target merely because a rollback file exists. Record any restore/numbering reset and required sync-state reset before resuming traffic.

## 12. Credential, packaging, and verification repairs

### 12.1 Credential-bearing transaction tests — F09

Remove the fixed credential-bearing URI from `api/shared/db/transaction_test.go` without printing/copying its value. Replace nominal unit tests with a network-free SQL driver/fake covering begin/callback/commit/rollback/cancellation error propagation. Real transactions belong in marker-validated disposable integration tests.

The credential owner must rotate the exposed credential, verify affected access, and provide non-secret confirmation. Source removal, a passing test, or a rewritten Dockerfile does not prove rotation. History cleanup, if desired, is a separately authorized repository operation.

### 12.2 Container inputs — F10

Add `.dockerignore` excluding `.env` variants, Firebase private keys, VCS, CodeGraph/tool metadata, logs, and local temporary/test artifacts. Preserve generated build inputs needed by Go; do not exclude `.gen` wholesale. Remove the Firebase key from final-image COPY and provide credentials through deployment-managed secret mounting or supported identity. Adapt bootstrap's credential location without breaking explicitly supported local development.

Keep a writable runtime generation path and required generation/runtime inputs. Verify a clean build with sentinel secret files and inspect the resulting context/layers/final filesystem for absence of sentinels. Do not inspect real secret contents or build an image containing them to demonstrate the defect. Verify generation failures and secret-mount absence stop startup safely. An immutable container root needs a mounted writable generation location rather than skipping Jet.

### 12.3 Regression-test accuracy — F31, F32, F33

Fix route nil-config registration, correct atomic enrichment audit expectations, and remove the Python wrapper test's obsolete substring anchor. Prefer exercising current helper arguments and target/package/marker refusal behavior over slicing brittle source text. Retain the real wrapper's cleanup, advisory serialization, and refusal protection.

Run all relevant unit packages after F09 becomes network-free. Do not keep excluding it as a permanent workaround. Full isolated Shops and equipment-service verification must pass with every migration flag and the corrected assertions; nine exact-key failures and the incorrect audit test from the audit remain visible until repaired.

## 13. Compatibility changes and server/client handoff

| Contract area | Preserved | Deliberate change or limitation |
| --- | --- | --- |
| Shops envelopes | Existing successful legacy shapes and negotiated error writer. | Previously unwritten failures become actual errors; unsafe requests no longer report success. |
| Authentication | Bearer authentication and existing identity enrichment. | Disabled/revoked/deleted users receive 401; transient identity failures receive controlled 503. |
| Messages | Nine nested fields, nullable keys, routes, marker text, upload response keys. | Generated insertion_number excluded; deterministic forward draining; deleted legacy anchors require reload. |
| Images | Valid same-Shop upload/display/message operations. | Arbitrary/copy-derived deletion forbidden; discard requires registered unattached ownership/admin authority. Historical unknown ownership stays protected. |
| Settings | False/default values and metadata routes. | Metadata cannot clear policy accidentally; explicit false settings work. |
| Equipment | Legacy usage profile, absolute semantics, raw metadata contracts. | Usage-only no longer erases metadata; Admin edits persist; new negative base values rejected. |
| Items | Supported NIIN/NSN, raw valid units, null/omission semantics, same-Shop multi-list bulk. | Positive/complete validation, contradictory notification targets rejected, actual delete count and stale-delete no-op. Ambiguous enrichment fallback is a conflict. |
| Invites | Current unrestricted codes/defaults and removal/rejoin semantics. | Supplied unsupported restrictions rejected instead of ignored. |
| Services | Timestamp interpretation, existing route/envelope and completion-flag defaults. | Accepted filters now apply; repeated completion preserves date; malformed timestamps classify as input failures. Offset cross-request shifts remain. |
| Sync | Healthy commit order, held bounds, gaps, scoped reconcile and opaque history cursor. | Common corrupt-state 503; ahead-of-counter reset-required 409. Numeric ABA reset remains an operational gate. |
| History | Existing audit kinds, null/empty envelope and limits. | Enrichment/event-time labels captured; legacy delivery remains best effort. |

Use sanitized released-request fixtures supplied by the owner to replay server compatibility sequences without opening Flutter source. Hand off exact reset/unavailable behavior, changed filter results, protected orphan cleanup, and raw-unit retention. Owner/client-team parser or device confirmation is separate evidence; do not claim a server suite proves UX on a released binary.

## 14. Staged implementation boundaries

This is the agreed design sequence, not the execution plan. The plan written after specification approval must name exact edits, test commands, dependencies, and review boundaries. Each phase ends with its relevant regressions passing; the final gate runs the integrated full suite.

| Phase | Findings | Primary existing files and planned additions | Required acceptance |
| --- | --- | --- | --- |
| 1. HTTP/auth/context/API/build foundation | F01, F06, F08–F10, F22, F31, F33 | `api/shops/shared/{contract,context,failures}.go`; `api/middleware/authentication.go`; `api/route/route.go`; `api/shared/db/transaction*.go`; `api/response/user_shops_response.go`; affected repository/service interfaces; `bootstrap/{app,authentication,database,env}.go`; `main.go`; `tools/jetregen/main.go`; new `internal/jetgen/`; `Dockerfile`; new `.dockerignore`; wrapper tests. | Production route errors; SDK-parity identity cases; pool/lock/commit cancellation; pre/post-018 nine keys; no-network unit helper; clean container/generator checks. |
| 2. Current authority, lifecycle, admission | F02, F05, F07, F13–F17, F28 | `api/shops/shared/authorization.go`; core/members/invites/settings handler/service/repository files; vehicles writer interfaces; `api/equipment_services/shared/authorization.go`, core/completion repositories/services; invitation request presence. | Two-Shop role matrix; every removal/demotion barrier; atomic creation; false settings; promoted rename; claim/revoke/promotion races; final-member/last-admin cases. |
| 3. Owned images and reply integrity | F03, F04, F11, F25 | `api/shops/messages/{handler,service,service_impl,repository,repository_impl}.go`; planned asset/worker files; main/route worker wiring using bootstrap clients; 020/relationship migrations and runners/tests. | Foreign/copy/published-target protection; body bound before parsing; shared refs/all markers; cascade/Shop deletion; crash/late-upload/lease recovery with fake Azure. |
| 4. Mutation intent, retention, validation, audits | F12, F18–F21, F32, F34, F36, F37 | `api/request/shops_request.go`; `api/shops/vehicles/{handler,service_impl,repository_impl,usage}.go`; list item writers; notification/item legacy and atomic files; changes repository; retention migration/helper. | Role-independent usage; Admin GET persistence; every nested validator; actual counts/stale retries; DELETE/re-add and alternating atomic/legacy; conflict/replay/audit failure semantics. |
| 5. Snapshot reads, complete pages/history, service lifecycle | F23, F24, F26, F27, F35 | `api/shops/aggregates/repository*.go`; message read/sync files; `api/equipment_services/{queries,calendar,status,core,completion}` interfaces/services/repositories; shared date/error mapping. | Deterministic snapshot writer; ties/burst drains; status/count agreement; completion retries; >65,535 history; common numbering guards and reset code. |
| 6. Migration/runbook/fleet acceptance | F29, F30, shared conditional gates | New numbered pairs/runners/write-ups; `docs/testing/shops-{database,release-contracts,message-sync-measurements,service-date-mapping}.md`; 017 guide; project decisions/key facts/bug/work records during implementation. | AND privilege roles; bridge physical sessions; refusal/reverse/regen; integrated tests; measured capacity; owner-confirmed credential/fleet/named-target gates. |

Bridge/readiness work can be prepared early where another phase depends on it. Avoid declaring the security fixes complete while old unsafe binaries still serve the affected writes. Retain bounded changes instead of merging unrelated refactors into these phases.

## 15. Finding-to-acceptance ledger

Each row is a required implementation acceptance case. The audit retains detailed source provenance and severity; this table specifies the intended disposition.

| ID | Fix section | Minimum regression evidence |
| --- | --- | --- |
| F01 | 4.1 | Real production group: c.Error denial/DB failure yields one correct legacy/contract-2 body and status. |
| F02 | 5.1 | A-admin/B-member cannot complete/delete B's foreign-authored service using A's Shop ID. |
| F03 | 6.1–6.4 | Copied/foreign/external marker never calls Delete on that target. |
| F04 | 6.3 | Uploader/admin discard only zero-ref registered assets; other member/published asset denied. |
| F05 | 5.1 | Removal/demotion-first blocks every uncovered writer; mutation-first serializes correctly. |
| F06 | 4.2 | Disabled, issued-before-revocation, nil/deleted, invalid and transient failure matrix; handler never runs on failure. |
| F07 | 5.2 | Create true persists; rename preserves true; both explicit false setting routes persist false. |
| F08 | 4.3–4.4 | Canonical regeneration plus pre/post-018 reads/send/sync/aggregate JSON has exactly nine message keys. |
| F09 | 12.1 | Unit package has no real DSN and cannot contact a network; owner rotation evidence recorded separately. |
| F10 | 12.2 | Sentinel credentials absent from context/layers/runtime; mounted-secret and writable-generation startup works. |
| F11 | 6.5 | Cross-Shop parent rejected; same-Shop cascade/ref cleanup works; corrupt historical relationship refuses migration. |
| F12 | 7.1 | Creator/admin/member usage-only requests preserve metadata/admin/base values. |
| F13 | 5.3 | Revocation committed before delayed claim prevents membership insertion. |
| F14 | 5.3 | Duplicate/delayed claim after promotion cannot demote; already-member response is stable. |
| F15 | 5.2 | Inject first-membership failure: no Shop/member, no success response. |
| F16 | 5.4 | Concurrent leaves/removals; last-admin successor rule; creator-first/noncreator-final exit without orphan. |
| F17 | 5.2 | Current promoted admin renames; removed/demoted actor denied; explicit deletion stays creator-only. |
| F18 | 8 | DELETE/re-add different UUID recovers only enrichment; alternating writers/replay/conflicting NIIN cases. |
| F19 | 7.1, 11 | New negative base mileage/hours rejected; omitted defaults/nil tracked work; existing bad rows require repair. |
| F20 | 7.1 | Authorized metadata Admin edit appears in DB and GET; usage-only cannot rename. |
| F21 | 7.2 | Invalid second nested item rejects whole batch; negative/blank/overlong/type/target contradictions cause no row/audit changes. |
| F22 | 3.1 | Canceled pool acquisition/Shop lock/vehicle lock/pre-commit releases resources and returns failure. |
| F23 | 9.1 | Writer between aggregate sections cannot tear headers/items/totals/PMCS counts. |
| F24 | 9.2 | Equal timestamps and larger-than-limit newer bursts drain exactly once; deleted anchors signal reload. |
| F25 | 6.2 | Oversized file/field/multiple parts rejected within total budget before Azure; temp files cleaned. |
| F26 | 9.4 | Metadata edit, repeated/concurrent completion, explicit date and reopen preserve intended historical time. |
| F27 | 9.5 | Both listing routes enforce each status/type/completion/date filter and count; deterministic ties/nulls; safe malformed date. |
| F28 | 5.3 | Non-null expiry/max-use values reject; omitted/null unrestricted invitation works. |
| F29 | 11.2 | Partial-grant roles fail AND gate; full role performs actual legacy insert/allocator under invoker privileges. |
| F30 | 4.4, 11.2 | 017 guide uses tagged generator and identity-checked test-first sequence; no plain-jet instructions. |
| F31 | 4.1, 12.3 | route.Setup with nil config registers safely with flags off; entire route package completes. |
| F32 | 7.4, 12.3 | Correct stored audit kind/payload and two-change/default-only assertions pass without production kind change. |
| F33 | 12.3 | Python wrapper tests import/run and exercise current helper/refusal arguments; real wrapper retains cleanup. |
| F34 | 7.4 | Single/bulk add/remove snapshots contain identity/unit/nickname, including deletion and nullable values. |
| F35 | 9.3 | Relational query/fixture beyond 65,535 inspections returns full history/counts without oversized binds. |
| F36 | 7.3 | Missing first/non-first/all IDs succeed idempotently; foreign surviving row rejects all; current policy race checked. |
| F37 | 7.3 | Duplicates/missing/concurrent delete report actual unique affected count. |

## 16. Conditional concern ledger

| ID | Disposition | Closure evidence / remaining gate |
| --- | --- | --- |
| C01 mixed-writer deadlock | New Shop-before-counter allocator bridge; safe overlap fence. | Deterministic physical mixed-writer/role tests; inventory external writers before cutover. |
| C02 capability readiness | Flag AND stronger schema/access probe; retain operator gates. | Every instance/build/schema/role verified, plus real app-role write; GET alone insufficient. |
| C03 counter reset recovery | Explicit ahead-watermark reset code; restore/reapply fence. | Corrupt vs reset tests; owner/client recovery evidence; numeric in-range ABA limitation documented. |
| C04 NULL numbering | Common per-Shop guard before all sync results. | Catch-up with a NULL row outside selected chunk fails closed; missing/behind counters and valid deleted gap tested. |
| C05 contention/capacity | Context cancellation, relational reads, measured budgets. | Representative same-Shop concurrency/payload/query/pool measurements; no arbitrary truncation or altered unlimited defaults. |
| C06 invite guessing/rejoin | Removal-only policy explicit; throttling/exposure gate. | Verify active-code population and effective authenticated/edge limits across the fleet; decide acceptable exposure from measured budget. |
| C07 blob leaks | Managed references, durable retry/reconciliation and cascade coverage. | All-marker/shared-reference/edit/parent/Shop deletion and late PUT/crash recovery tests; historical ownership inventory gate. |
| C08 history limits/meaning | Event-time enriched snapshots, stable ties, explicit best-effort policy. | Audit warnings monitored; historical null/limits retained; no timestamp-equality claim; capacity evidence for unbounded routes. |
| C09 service dates | Keep capabilities false and timestamp contract. | Owner-approved live mapping/parser/device handoff before separate activation. |
| C10 live migration preparation | Pinned new runners, same-session identity, safe reverse refusal. | Authorized test-target preparation first; production separately approved; UNPINNED remains a refusal. |

For C06, 32-bit random codes retain the existing wire format. Against N active uniformly random codes, expected random attempts to hit any code are approximately `2^32 / N`. The current middleware is per-IP and does not establish a per-UID/fleet-wide budget. Verify edge controls; if insufficient, use a bounded authenticated-UID limiter for claim/creation routes with explicit expiry/eviction and a deployment-wide budget, rather than attaching an unbounded global IP map to every Shops read. Select limits from active-code exposure and legitimate request evidence; absent sufficient proof, admission rollout remains blocked. Do not claim source review alone resolves guessing exposure or silently lengthen codes.

For C05, preserve ADR-015's accepted overview workload of 100 Shops/25,000 equipment records and warm-cache p95 below one second as its existing benchmark. Establish observed baselines for other routes; do not invent a production SLO. Record p50/p95/p99, query count, pool wait, Shop lock wait, payload size and memory at representative single-Shop and multi-Shop contention. Add indexes/batching only when evidence justifies them and rerun the affected measurement.

## 17. Verification and shipment evidence

The future implementation plan must use these checks, with fresh outputs tied to the integrated HEAD and generation manifest:

1. Network-free `go test -count=1 ./api/... ./bootstrap/... ./tools/... ./internal/...`, including the repaired transaction package, production route tests, and shared generator tests; focused race tests for shared state/auth/worker/transaction boundaries.
2. `go build ./...` and `go vet ./api/... ./bootstrap/... ./tools/... ./internal/...` against freshly generated intended-schema inputs. Record actual Go toolchain version.
3. `python3 scripts/test-shops-isolated_test.py` and full `scripts/test-shops-isolated.sh --verify-notification-migration --verify-message-sync-migration -v -count=1 ./tests/shops ./tests/equipment_services`, extended with new migration rehearsals. Use only disposable loopback targets and the marker guard. Keep package serialization where tables are shared.
4. Physical-session transaction tests for every authority/lifecycle race, snapshot tearing, and mixed allocator writers. A mocked call order is not sufficient evidence of PostgreSQL lock behavior. Run appropriate race-enabled disposable suites as well.
5. Populated upgrade/refusal/reverse/reapply checks for each new pair; regenerate after each database change in the isolated generation workspace; exact legacy JSON and raw sync DTO checks. Verify failures do not partially mutate data/schema.
6. Fake-Azure ownership/worker fault tests and container sentinel-secret/writable-generation checks. Real Azure/Firebase acceptance, if needed, is separately authorized and cannot be inferred from fakes.
7. Representative query/capacity/contention measurements and sanitized released-request fixtures, then owner-confirmed named-target, role, credential, fleet and client/device gates.

The audit baseline was not green: safe API tests exited nonzero due to route panic; isolated integration had 138 top-level passes, 10 failures, and 2 skips; wrapper tests failed at import. Nine message failures exposed the generated field and one audit test was incorrect. Compilation/vet and six disposable migration checks passed. Those historical outcomes explain needed acceptance work; this specification runs no product tests and repairs none of those failures.

The final implementation report must separate code/test state, generated-model provenance, named-database migration state, credential rotation, fleet flags/builds, push/deploy status, and released-client/device evidence. Passing local tests alone is not production shipment approval.

## 18. Required document updates and next authorization stage

During implementation update the 017 migration guide, database/contract/measurement runbooks, generator instructions, message-sync ADR/key-fact workaround, and project bug/work records. Describe approved policy and limitations; do not mark a planned fix as already applied. Keep the original report intact and link each finding's final verification disposition rather than rewriting its historical evidence.

The user approved the written specification on 2026-10-03. Create the detailed implementation plan with exact files, tests, phase dependencies, isolated-workspace approach, and review boundaries. The user then reviews that plan and selects its execution method before implementation. Existing-database actions and deployment retain their separate authorization/identity gates.

No product code, migrations, generated files, capability settings, or existing database state was changed to produce this document. The document remains uncommitted under the repository instruction to commit only when asked.
