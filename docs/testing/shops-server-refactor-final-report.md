# Shops server refactor: final change and merge report

Date: 2026-10-04. Target branch: **`cleanup`**.

## Result and scope

The reviewed server remediation is fully merged into `cleanup`. The local merge was a fast-forward from `518ef4e828abeb64c203e567287b730799222da7` to **`2e58cd8522f424a88193814530dbf190d0fad794`**. Commit `02716774c9700987ca1cac486de800ac4f0f2902` records the integration evidence and client handoff. This report and the finalized handoff are documentation updates on the same branch.

The final merge confirmation returned `Already up to date.` The remediation branch is an ancestor of `cleanup`; all subsequent tracked differences from that branch were documentation only. All **753 reviewed Go/SQL/build input hashes** matched their recorded source manifest. Source SHA-256 remains **`843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`**. All **32 tracked generated PMCS files** matched HEAD.

This work covers the Go server, PostgreSQL migration definitions, server tests, build configuration and documentation. **No Flutter/client source was reviewed or modified. No connection or change to `miltech_ng` or `miltech_ng_test` occurred during this work.** Database rehearsals used disposable `miltech_test_*` databases in temporary local clusters. No push, deployment, credential rotation or capability activation occurred.

At the user's instruction, **no new server tests, builds or database checks were run during this final documentation pass**. The verification results below are the earlier recorded merged-tree results, with their limitations retained.

## Application changes

### Authentication and failure responses

Files: `api/middleware/authentication.go`, `optional_auth.go`, `api/route/route.go`, `api/shops/shared/contract.go`, `failures.go`, and related bootstrap authentication wiring.

- Firebase authentication validates the current account as well as the token. Disabled/deleted accounts and tokens issued before revocation are rejected. A missing/invalid user cannot reach an authenticated handler.
- The Authorization header must use Bearer syntax. Authentication dependency failures return an unavailable response rather than admitting the request or presenting empty data as success.
- Production Shops/equipment routes install the error handler so a queued `c.Error` produces a controlled failure response instead of an empty HTTP 200.
- Unknown internal errors are sanitized. Legacy response compatibility is retained; clients opting into Shops contract 2 receive negotiated codes/statuses on participating handlers. Several legacy validation/bulk-delete handlers intentionally retain flat responses.
- Nil configuration is handled safely during route setup with optional feature flags off.

These changes close authentication bypass and false-success risks. They do not enable a forced client upgrade or globally change every legacy error status.

### Current resource permissions and cancellation

Files: `api/shops/shared/authorization.go`, `cached_authorization.go`, Shops repositories, `api/equipment_services/shared/authorization.go`, and `api/shared/db/transaction.go`.

- Mutations resolve the Shop and parent resource from persisted IDs and recheck current membership/role inside the transaction. Cached checks remain a preflight optimization.
- Shop-first locking serializes participating mutations with membership, role and settings changes. A removed/demoted user cannot rely on a permission check made before the change committed.
- Equipment-service operations verify that the service belongs to the actual path Shop. Being an administrator in another Shop cannot authorize completion, update or deletion.
- Affected SQL and transaction operations carry the caller context through pool acquisition, lock waits, queries and commit handling.
- Equipment username enrichment propagates cancellation/failure instead of returning fallback success. The cache checks cancellation even on a cache hit.

A legacy mutation can commit before later username enrichment or response delivery fails. It is therefore unsafe to automatically replay a legacy create, message send or usage adjustment merely because its response was lost. The atomic notification save receipt is the separately supported replay mechanism.

### Shops, settings, membership and invitations

Files: `api/shops/core/`, `settings/`, `members/`, and `members/invites/`.

- Shop creation and initial creator-admin membership are atomic; a failed membership insert cannot leave a partially created Shop.
- Creation persists an explicit `admin_only_lists:true`. Rename preserves the stored setting. Both settings update routes accept explicit false.
- Shop rename is permitted to a current Shop administrator. Explicit Shop deletion is restricted to the original creator while that creator remains a member.
- Rename preserves omitted/null details; an explicit empty string clears them.
- Before the last administrator leaves a Shop containing other members, a successor must be promoted. The final member can leave and trigger cleanup, including a noncreator who became the sole remaining member.
- Join and invitation revocation are serialized. Duplicate admission preserves an existing membership role instead of demoting an administrator.
- Unimplemented invitation expiry/max-use controls are rejected when non-null, including zero/empty values. Omitted/null controls remain unrestricted invitations. Removal remains removal, not a ban; code format is unchanged.
- The unused unlocked core membership-upsert API was removed to prevent future reuse outside the supported locking boundary.

### Lists and list items

Files: `api/shops/lists/`, `lists/items/`, `shared/list_dependencies.go`, and `shared/item_validation.go`.

- List mutations enforce the current `admin_only_lists` policy. List deletion retains its creator/admin policy and refuses deletion of a referenced list.
- NIIN and nomenclature must be nonblank; quantity must be positive. Nickname and unit validation also runs in services/repositories so an alternate writer cannot bypass handler validation.
- Bulk additions validate all items before effects and authorize all persisted target lists. A cross-Shop set is rejected together.
- Bulk deletion deduplicates IDs and tolerates already missing rows. The returned count is the actual number of unique deleted rows, not the input length. All missing IDs return zero after authentication; foreign surviving rows reject the mutation.
- Single-item deletion remains strict. Bulk deletion's stale-ID tolerance does not make other mutations replay-safe.

### Vehicle metadata and tracked usage

Files: `api/shops/vehicles/`, including `mutation_intent.go`.

- Legacy tracked-usage payloads are classified before metadata privilege decisions. Stale `admin`, base mileage and base hours snapshots carried by usage-only requests cannot overwrite vehicle metadata.
- Metadata updates preserve omitted/null pointer fields, apply a supplied nonempty admin, and support explicit empty detail values. Empty UOC retains the established `UNK` interpretation.
- New/supplied negative base mileage or hours is rejected. Existing invalid rows are not silently clamped; a live-data repair requires a separate decision.
- Creator/admin metadata/delete authority and member usage authority are rechecked on the actual persisted vehicle.
- The existing relative-usage PATCH retains overflow/result validation and its existing request/response contract. It does not acquire idempotent retry semantics.

### Notifications, direct items, attachment intent and receipts

Files: `api/shops/vehicles/notifications/`, `notifications/items/`, `notifications/changes/`, and shared notification mutation/validation helpers.

- Legacy updates distinguish an omitted attachment from explicit null: omission preserves the linked list, null detaches it, and a supplied valid same-Shop list replaces it.
- Notification types are restricted to `M1`, `PM` or `MW`; titles and item fields receive equivalent validation across legacy, bulk and atomic writers.
- Nested notification IDs inherit the outer ID when omitted/null and reject a contradictory target. Invalid later elements cannot leave a partial batch or partial audit.
- Nickname/unit retention is scoped to the same logical notification and exact NIIN. Delete/re-add with a new UUID recovers omitted enrichment without restoring deleted item rows or quantities. Ambiguous retained values refuse rather than choose silently; explicit values can resolve the respective metadata.
- Notification-item bulk deletion returns actual unique counts and tolerates stale IDs, while requiring all surviving items to belong to one notification.
- Atomic notification save independently checks readiness before writing. Capability advertisement requires both the deployment flag and readiness.
- Atomic save retains stable operation identity, semantic fingerprints and receipt replay. Omitted enrichment preserves old fingerprints across deployment. It replaces the complete direct-item set; linked-list items remain represented through attachment.

The atomic route already existed. No activation occurred. Legacy success shapes and nullable metadata remain compatible.

### Messages, pagination and synchronization

Files: `api/shops/messages/`, `api/response/user_shops_response.go`, and message-sync readiness/repository code.

- Legacy send/read/aggregate responses use an explicit nine-field message DTO. Storage-only `insertion_number` does not leak into JSON after schema generation.
- A supplied parent must belong to the same Shop. The matching database ownership constraint is part of migration 021.
- Legacy pagination uses the timestamp/ID tuple, preserves equal-timestamp rows and drains the nearest newer batch before presenting it descending. Clients advance with the returned `next_cursor`.
- Deleted/missing/null-time anchors return a safe reload signal: negotiated HTTP 409 `reset_required`, or the preserved legacy HTTP 500 envelope with the same reload message.
- Numeric message readers validate numbering, timestamps and counter integrity, including corruption outside the selected chunk. Invalid integrity returns HTTP 503 `unsupported_contract`; a cursor ahead of the counter returns HTTP 409 `message_sync_reset_required`.
- The allocator bridge enforces compatible ordering for participating legacy/numeric writers. History cursors remain opaque, and watermarks remain decimal strings.

Two limitations remain explicit: a late timestamped legacy commit can land behind an observed cursor; an in-range numeric watermark reused after restore is not detected by the current protocol. Neither is represented as fixed or waived.

### Managed image uploads and durable cleanup

Files: `api/shops/messages/assets.go`, asset repositories, `blob_store.go`, upload handler/service and `cleanup_worker.go`.

- Managed uploads have server-side ownership/storage registration. Message marker text selects registered candidates; it cannot authorize deletion of an arbitrary cloud target.
- Only an upload's owner or current Shop administrator, while a current member, can discard a registered unattached upload. Published/referenced assets are protected.
- Multipart requests are bounded before parsing: one file part, only the supported optional Shop field, 5 MiB file limit and 6 MiB total-body limit. Temporary multipart files are cleaned.
- Message edits/deletes, parent cascades and Shop/final-member deletion maintain registered references and durable cleanup proof. Historical assets of unknown ownership are protected.
- Cleanup leases, retries and tombstones cover late upload completion, interruption and recovery. Every worker database/cloud phase uses its bounded operation budget; job phases respect remaining lease time. A cloud timeout does not poison fresh acknowledgment, while shutdown and expired leases preserve recoverability.
- Reverse migration 020 locks affected writer tables before checking emptiness/drop eligibility, preventing a writer from committing proof between the check and drop.

A successful discard acknowledges durable scheduling; it does not prove that Azure bytes are already gone. Automatic destructive expiry/adoption of unknown historical uploads was not introduced.

### Aggregate reads, service reads and large result sets

Files: `api/shops/aggregates/`, notification read repositories, core overview/history code and `api/equipment_services/`.

- Combined Shop/list/maintenance/bootstrap responses and legacy notifications-with-items bind membership, headers, items and counts to one caller-context read-only repeatable-read snapshot.
- Large selected parent sets use bounded array bindings rather than one parameter per ID. Functional tests covered 65,536 complete PMCS inspections and up to 67,000 notification parents/items.
- Existing optional limits, unlimited defaults and primary wire schemas remain. No arbitrary production cap or speculative index was added.
- Equipment-service completion preserves historical completion dates on repeated omitted/null-date completion, including historical null. Explicit completion dates update it; reopening clears it. First completion defaults inside the locked transaction.
- Shop/equipment listing routes apply status, type, completion and timestamp range filters conjunctively. Counts match the filtered snapshot, ties are deterministic, malformed RFC3339 dates fail safely, and due-soon bounds are validated.
- Service timestamp semantics remain unchanged. `service_dates` and `service_reads` remain false pending separate live-schema/client mapping acceptance.

Earlier local measurements found about 25.9 MB JSON and 656.7 MB cumulative allocation for a complete 65,536-inspection PMCS response. These are historical fixture measurements, not peak-RSS or concurrent production-capacity approval.

### Audit behavior, startup and generated models

Files: `api/shops/shared/legacy_audit.go`, notification/item/vehicle audit callers, `bootstrap/`, `main.go`, `internal/jetgen/`, `tools/jetregen/`, `Dockerfile` and `.dockerignore`.

- Event-time audit snapshots retain item identity/nickname/unit and deleted-resource metadata. Legacy audit remains best effort, with a sanitized correlated audit-gap event; atomic save retains its required transaction/receipt behavior.
- Equipment deletion success is logged after commit. An injected commit failure cannot emit false deletion success. Performance sampler cleanup stops and joins its goroutine even on early test failure.
- Startup validates required final-schema dependencies and keeps **mandatory automatic tagged Jet generation before routes**. Generation failure prevents serving an incompatible application.
- The supported generator produces snake_case JSON tags and provenance. Plain Jet is not used, and generated Go models are not hand-edited.
- Canonical tagged output from the full disposable candidate was transferred unchanged into the merged checkout for build inputs. All 32 tracked PMCS outputs remained byte-identical.
- After any future PostgreSQL schema change or approved repair, tagged Jet models must be regenerated against the verified intended target. Runtime generation does not rebuild the already running binary.
- Build input exclusions cover credentials, environment files, worktrees, agent/cache data, logs and temporary artifacts. Firebase credentials are supplied through a runtime file mount; nonroot startup needs readable credentials and writable generated output.

Actual container/runtime acceptance remains incomplete for the reasons recorded below.

## PostgreSQL migration definitions

Five additive forward/reverse pairs are committed under `migrations/`; their write-ups are under `docs/migrations/`.

| Migration | Purpose |
| --- | --- |
| 019 `fix_shop_message_allocator_lock_order` | Shop-first allocator ordering and compatible legacy message allocation |
| 020 `create_shop_message_asset_lifecycle` | Registered uploads, references and durable cleanup jobs; reverse refuses destructive loss of retained proof |
| 021 `enforce_shop_message_parent_ownership` | Same-Shop parent ownership; incompatible historical relationships refuse migration |
| 022 `add_shop_vehicle_base_usage_constraints` | Nonnegative base mileage/hours; existing invalid values require explicit resolution |
| 023 `create_shop_notification_item_metadata` | Notification/exact-NIIN enrichment retention independent of active item rows |

Migrations 015–018 were retained. The final reviewed reverse-020 hash is `d8003e69aae701e8df15b0ee4ab9a7aa2b7c9ebcf2f2a91b96e35bf7039a1f6a`.

**These are repository changes, not existing-database changes.** Neither named database was contacted or migrated. Future application must use separately authorized target identity/schema/data/application-role checks, `miltech_ng_test` first and then a separately authorized `miltech_ng`, with tagged regeneration after each action. Both target pin records remain unpinned. See [target-specific release gates](shops-server-remediation-release.md).

## Test and verification evidence

The following results were recorded earlier on the merged source. They were **not rerun after the user's instruction to skip server checks**. Exact commands/exits/log hashes are in [verification JSON](shops-server-cleanup-verification.json); the [integration record](shops-server-cleanup-integration.md) explains the execution boundary and preserved logs.

| Recorded check | Result and limit |
| --- | --- |
| Safe host Go suite | Exit 0; 619 top-level / 1308 including subtests PASS; three inherited PSMag database skips |
| Shops/equipment/shared middleware unit race suite | Exit 0; 147 / 437 PASS; no skips |
| Full protected integration/migration/release-runner/performance matrix | Final exit 0; 303 / 1002 PASS; no skips; disposable targets only |
| Physical concurrency/race selection | Exit 0; 34 / 132 PASS; no skips; this is the recorded selection, not every database test under race |
| Build and scoped vet | Both exit 0 |
| Isolation / release-runner / publication / container-static Python guards | 11 / 13 / 4 / 3 PASS |
| Endpoint handoff JSON examples | 73 examples previously validated as JSON |

An earlier full integration run failed `TestCleanupCascadeAndFinalMember/reply` on a database-wide active-transaction assertion. Twenty repeats, 200 diagnostic repeats, a full diagnostic sequence and the final unmodified matrix passed. Temporary diagnostics were restored exactly; no assertion was weakened. Its cause remains **intermittent/unresolved**, so rerun success is not a reliability claim.

The Docker fixture invocation reached image builds but its merged scanner recursively expanded embedded Go TAR fixtures, including a 60 GB logical sparse member. The scanner was terminated after 1111.451 seconds with approximately 3 GB RSS/one CPU; invocation exit was 143. Container/runtime acceptance is **incomplete**. A separate scanner repair exists only as uncommitted work in the isolated worktree and was not added to `cleanup` after the user questioned that tooling expansion.

## Shell/Python tooling included in the reviewed merge

The merge includes 10 script changes: eight additions and two modifications. These are migration/verification tools, not HTTP application code or normal startup scripts.

| Purpose | Files under `scripts/` |
| --- | --- |
| Guarded named-target procedure and disposable rehearsal | `apply-shops-remediation-migrations.sh`, `verify-shops-remediation-migrations.sh`, existing `test-shops-isolated.sh` |
| Refusal/isolation regression tests | `apply-shops-remediation-migrations_test.py`, existing `test-shops-isolated_test.py` |
| Exact source and tagged output checks | `shops_candidate_inputs.py`, `publish-shops-candidate-generated.py`, `test-publish-shops-candidate-generated.py` |
| Synthetic container packaging checks | `test-server-container-inputs.sh`, `test-server-container-inputs-static.py` |

The additional uncommitted scanner changes in the isolated worktree (`test-server-container-inputs.sh` and new `test-server-container-archive-scan.py`) are excluded. No further tooling work or server checks were performed in this final documentation pass.

## Client requirements

No existing endpoint URL or HTTP method was renamed or removed. A client may still need request, response or UX handling changes for the corrected server behavior. The separate [client endpoint handoff](shops-server-client-api-changes.md) lists every affected endpoint group, why it changed, the resulting behavior, conditional client requirements, and JSON request/response examples.

The main actions are strict Bearer/error handling, explicit false and omission/null intent, creator/successor permissions, supported invitation fields, minimal tracked-usage payloads, positive complete item batches and actual bulk-delete counts, safe upload identity/multipart handling, returned cursor/reload behavior, and completion/filter preservation. Atomic/numeric sync adoption remains optional and capability-gated. No client implementation or deployed-parser acceptance is claimed.

## Remaining implementation and release actions

The approved local server remediation and final review fixes are merged. Remaining items are explicit:

1. **Legacy late-commit cursor behavior:** an owner decision is needed on retained timestamp limitations versus a monotonic-write/synchronization implementation.
2. **Numeric restore recovery:** an epoch/recovery design and corresponding client work may be needed for in-range watermark reuse; ahead-of-counter reset detection does not solve it.
3. **Invitation abuse controls:** active invite population, fleet/UID/edge budgets and acceptable exposure must be supplied. A bounded limiter may require implementation if current enforcement is insufficient.
4. **Client work:** apply the handoff where the released client relies on changed behavior; validate request/parser/retry behavior and devices before feature activation.
5. **Date/read activation:** a separately approved live-schema/timezone mapping and client activation implementation is required if those capabilities are to be enabled.
6. **Verification follow-ups:** the intermittent cleanup assertion, incomplete container acceptance, excluded scanner repair and retained negative-test-depth follow-ups remain documented.

Operator release work remains separate: named-target inventory/data resolution and migrations, actual-role write/privilege proof, Jet regeneration, real TMDE/view/build provenance, external-writer/fleet fencing, both separate credential-owner confirmations, real container/SDK/PostgreSQL startup and mounts, observability/alerts, production capacity budgets, released/signed artifacts, device acceptance, deployment and capability activation. This report grants no shipment approval.

## Preserved workspace state and references

The preexisting dirty `.codegraph/codegraph.db` and `.codegraph/daemon.log` are excluded from commits. Local generated lock/source/PMCS provenance files remain untracked; no tracked generated model was hand-edited. The isolated worktree is retained because it holds unique review/test evidence and the explicitly excluded uncommitted scanner work. No unrelated worktree was removed.

- [Client endpoint handoff with JSON examples](shops-server-client-api-changes.md)
- [Merged-tree integration evidence](shops-server-cleanup-integration.md)
- [Reviewed finding dispositions and local acceptance](shops-server-remediation-acceptance.md)
- [Target-specific release gates](shops-server-remediation-release.md)
- [Archived whole-branch and final scoped review evidence](../reviews/2026-10-04-shops-server-remediation/README.md)
