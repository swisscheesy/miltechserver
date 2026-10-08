# Final consolidated fix wave — implementer report

Status: **DONE — authorized fix wave implemented and locally verified; scoped independent re-review pending**. This report does not grant review approval, integration, or release approval.

## Scope, authority and baseline

Work only in `/Users/swisscheese/projects/miltechserver/.worktrees/shops-server-release-remediation-20261003`. Baseline is `task-0-final-fix/manifest.json` plus the controller's final brief, constraints and full independent review. One implementer wave, no delegated agents. Root rulings authorize I1–I6 and M1–M5, existing operation-budget reuse, new 020 reverse repair/pin, and minimal audit-helper relocation. No product cap, timestamp allocation change, ownership/admission policy change, mandatory audit or new invite budget was introduced.

No commits/index/HEAD changes, merge/push/deploy, client edits/inspection, named-target contact, credential reads/rotation, configuration/flag change, plain Jet or manual generated edits. Commands are prefixed with `rtk` in the explicit worktree. Physical SQL is exclusively through the marker-protected disposable-loopback wrapper. All 32 tracked PMCS outputs match HEAD. Original F09 and Task 18 MCP incidents remain separate owner-confirmation obligations.

## Per-finding implementation and regression evidence

| Finding | Result / implementation | Meaningful RED → focused/owning GREEN |
| --- | --- | --- |
| I1 | Repository-owned caller-context read-only repeatable-read transaction includes persisted vehicle/Shop binding, membership, headers, exact selected items; service delegates the combined authorization boundary. Empty arrays and DTO shape preserved; next request rechecks authority. | `final-fix-snapshot-red-corrected.log`: physical writer between headers/items produced old title with quantity 2 rather than 1. Same barrier, revocation-next-request, canceled/pool/blocked SQL, and single transaction options now pass in final physical suites. |
| I2 | Every Reconcile/Claim/Prepare/Finish phase has the existing OperationTimeout child budget. Job Prepare/cloud/Finish also cap at remaining lease; already expired jobs skip. Finish uses a fresh bounded child of the live worker context; shutdown or expiry leaves durable work recoverable. Cloud calls remain outside database transactions. No new configuration. | `final-fix-worker-red.log`: no DB deadline and two preparations after first job exhausted lease. Unit deadline/clock/fresh-ack tests pass. Physical pool blocking for reconcile/claim, asset lock for Prepare and job lock for Finish time out before the outer watchdog, release resources, then both jobs complete after recovery. |
| I3 | Only new 020 reverse changed: 5-second per-lock wait, ACCESS EXCLUSIVE locks on assets/uploads → references → jobs before any emptiness read. Runner SOURCE_PINS and explicit order/budget/hash guard updated. | `final-fix-reverse-audit-red.log`: in-flight reservation passed the old check without waiting. `TestCleanup020ReverseSerializesWriters` now observes physical blocking before committing reservation/job; reverse refuses with retained proof. Exact prefix is tested without dropping the shared test fixture; full reverse/reapply and populated refusal are separately exercised by the complete migration matrix with tagged regeneration. |
| I4 | Root/nested log, logs, tmp/temp/scratch directories and temporary-file extensions excluded; real synthetic context/layer sentinel inventory extended. Required Go/frontend/gen/runtime inputs remain included by guard. | `final-fix-container-red.log`: 12 log/temp paths not excluded and missing runtime sentinels. Three static tests now pass, including intentionally removing the log pattern. The guard is explicitly conservative static pattern inspection, not a Docker parser/runtime certification. Actual bounded Docker attempt remains daemon unavailable. |
| I5 | Active release directions use explicit nine-field DTO, mandatory tagged generation, current numeric integrity/reset/fleet fences, and exact negotiated legacy anchor behavior. Obsolete Jet embedding/no generation/manual tags/silent NULL-number omission instructions are replaced. Historical ADR/audit context retained. | Manual cross-check against `sync_repository.go`, `sync_contract.go`, legacy service+handler and `shared/contract.go`; no source-policy change. Specific review wording correction below. |
| I6 | Aggregate items and bootstrap equipment use one text-array parameter plus limit; legacy items use one array through Jet RawArgs. Exact authorized selected parent sets, unbounded defaults, per-parent limits and ordering preserved. No cap/index added. Legacy ties now have stable ID ordering consistent with aggregate reads. | `final-fix-capacity-red.log`: aggregate 65535 and legacy 65536 route failure, including 67000 case. Final routes pass at 65534/65535/65536/67000 with all parents/items; private Shop denial checked; measured item-query max parameters 2. Bootstrap physical test proves exact four authorized Shops, per-Shop latest/limit/unbounded rows, exclusion of foreign Shop and constant two-parameter query. |
| M1 | Common sanitized `legacy_notification_audit_failed` event reused for savepoint-protected vehicle deletion. Actor/Shop/vehicle/server correlation/category and `vehicle_deleted` operation; one event; empty notification ID. Shared helper relocation preserves existing item wrappers/callers and one context correlation. | Audit-trigger fault RED showed unmatched raw-error warning. Final test asserts exactly one monitored sanitized event, UUID correlation, no raw error/sentinel and persisted deletion. All existing notification/item failure cases still pass. |
| M2 | Removed obsolete `core.Repository.AddMemberToShop`, its unlocked role-upsert implementation and its cancellation-only test branch. CodeGraph/current caller search found no production callers. Supported membership/admission/departure tests retained. | Removed only obsolete case; final host/physical member suites compile and pass. This is removal of an unused unsafe future entry point, not a claimed active exploit. |
| M3 | Equipment Delete success log occurs only after successful transaction return. | `final-fix-observability-red.log`: deferred constraint-trigger commit failure rolled back deletion but logged success. Final test proves rollback/no success event, then successful delete/event. |
| M4 | Sampler stop/wait is idempotent via sync.Once, registered with t.Cleanup and reused by finish. Assertion/FailNow paths now stop and join. | Same RED log: second finish panicked closing closed channel. Final test runs finish twice and cleanup without panic; race/performance suites pass. |
| M5 | Durable notes corrected: F29 privilege gates; F30 unsafe017 directions; F32 old failure explicitly historical and its corrected audit assertions; allocator writer correctly `WithTxContext`. | Manual source/evidence label review, historical diagnostic context retained. |

### I5 source-truth correction approved by controller

The review/brief used an imprecise “legacy cursor 503” description. Actual `messages-v2` readers share the universal numbering/timestamp integrity guard and fail with 503 `unsupported_contract`; ahead-of-counter numeric cursors return 409 `message_sync_reset_required`. Legacy timestamp pagination keeps pre-018 compatibility and no numbering dependency. Its deleted/NULL-created_at anchor yields a typed reset: contract version 2 receives 409 `reset_required`; unnegotiated legacy preserves HTTP 500, `data:null`, omitted `code`, and the safe reload message through handler `c.Error` → `HandleContractError` → `WriteFailure`. The controller confirmed this precision. No gate/behavior was added. Legacy late-commit timestamp semantics remain OPEN.

## TDD and setup accounting

Actual intended failures are the RED rows above, not compilation/setup errors. Initial snapshot invocation failed to bind sandbox loopback; the authorized loopback retry then found a missing NewService test constructor argument. `final-fix-snapshot-red-corrected` is the behavioral RED. Reverse reservation and vehicle audit REDs are valid; the first job subcase lacked its required reason fixture and was corrected before GREEN. The first focused read run passed all 12 threshold routes but its parameter counter accidentally counted unrelated 4-argument aggregate queries; it was scoped to the target item/bootstrap queries. The first combined GREEN build found an unused import after audit-helper relocation, then the corrected run passed all affected cases except bootstrap's duplicate-admin fixture; the corrected bootstrap fixture passes. These intermediate nonzero logs are retained.

Two test-only self-review corrections after the initial final snapshot restored the prior exact aggregate nil-user sentinel assertion and added blocked-test resource cleanup on early assertion failure. Consequently `final-fix-final-*` source evidence is superseded by `final-fix-frozen-*` for Go/physical verification and generated publication. Python guard/container scripts did not change after their final checks. No failed suite is called green; no-test selections are not counted as coverage.

## Compatibility and no-touch assessment

Explicit legacy nine message keys/nulls and raw valid unit fields remain owned by the existing DTO. Current creator/member/last-admin/retained-metadata and missing/empty username fallback policies remain. Post-commit username failure remains ambiguous with saved business rows, no compensation or blind retry. No date/timestamp interpretation, automatic startup tagged generation, generic intermediate tagged CLI, service flags, upload retention/adoption or invitation policy changed.

Only 020 reverse changed among 18 reviewed migration files (015–023 pairs); the other 17, including all eight historical 015–018 files and all reviewed forward bodies, remain byte-identical to the final-fix baseline. New reverse SHA-256: `d8003e69aae701e8df15b0ee4ab9a7aa2b7c9ebcf2f2a91b96e35bf7039a1f6a` (old `14f36dfe247e5695ea0d60dee44713a2e3f1cc0857c1ecb4e9a3be6935dcb241`). No named target pin was filled or guessed.

## Deferred and declined dispositions

The independent `whole-branch-review.md` deferred/declined sections remain authoritative and preserved. Specific minor defects M1–M5 and the ten-job lease-clock scenario are addressed in this wave. The following assessed follow-ups remain explicit, not represented as implemented: Task4 generator/path negative permutations; Task5 never-created-image cleanup warning; Task6 failed-marker-query redaction negative case; Task8 denial-envelope precision; general expected-diagnostic capture refinements; Task18 direct invalid-second-element no-touch regression; Task22 exact NULL-anchor/serialized legacy-versus-contract2 envelope precision; Task26 warning capture; Task27 zero-exit/no-receipt/mismatched-PID branches. Prior no-match package selections remain zero coverage, with fresh full owning suites recorded separately. No blanket logging/test refactor was authorized or performed.

Still OPEN/UNKNOWN/UNEXECUTED: owner choice on current-writer legacy late commit; numeric in-range restore ABA/epoch/client recovery; C06 active code population/fleet/UID-edge budgets; original F09 and separate Task18 MCP credential confirmations; named `miltech_ng_test` then separately approved `miltech_ng` identity/schema/source/data repairs; real app role/grants/writes; fleet/external writer inventories/fences; real TMDE view; Docker context/layers/runtime/mount/read-only generation; released parser/request fixtures, physical devices/signed artifacts; alert routing; deployment/activation. Service date/read flags remain false. No historic-asset adoption, ready-draft expiry, mandatory legacy audit/outbox, new invite limits or production history caps were inferred. Full 67000 notification and 65536 PMCS proofs are functional capacity evidence, not production memory/cold/concurrent/SLO approval.

## Self-review

Read own incremental candidate snapshot delta for every production edit and owning tests; verified current caller/lock/transaction path and document codes against source. Restored strict prior nil-user assertions rather than weakening unrelated coverage. Test-held connections/transactions and sampler now have cleanup on early failures. Physical rollback test runs the exact pre-drop lock/check prefix and the full DDL matrix independently validates drop/reapply; this is the stated boundary, not a simulated full race drop. Static Docker checks explicitly cannot certify Docker semantics/runtime. All final review fixes still require controller-owned scoped re-review.

## Final verification, capacity, provenance and changed paths

All required runnable local checks passed for the frozen source; Docker remains explicitly unresolved. **DONE; independent scoped re-review pending.**

Published source SHA-256: `843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`, **753 input paths** independently rehashed against `.gen/source-manifest.json`. All **32** tracked PMCS models equal HEAD. The migration delta is exactly one reverse file as above. `bc5ad7...` is pre-fix Task28 source; `7b03efa...` is the intermediate fix snapshot, neither is final.

Commands use `rtk proxy python3 .superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-run.py <label> <argv>`, explicit worktree, safe unset test targets and the recorder's local/offline Go settings. Exact argv/exits/durations/log SHA-256 and pass counts for every completed attempt are in `final-fix-verification-records.json`; no raw source diff is used as a verification artifact.

| Record | Exit | Seconds | Exact argv (recorder prefix below) |
| --- | ---: | ---: | --- |
| [final-fix-frozen-host-tests](final-fix-frozen-host-tests.log) | 0 | 52.532 | `go test -json -count=1 ./api/... ./bootstrap/... ./tools/... ./internal/... .` |
| [final-fix-frozen-build](final-fix-frozen-build.log) | 0 | 15.435 | `go build ./...` |
| [final-fix-frozen-vet](final-fix-frozen-vet.log) | 0 | 6.741 | `go vet ./api/... ./bootstrap/... ./tools/... ./internal/... .` |
| [final-fix-frozen-unit-race](final-fix-frozen-unit-race.log) | 0 | 30.224 | `go test -json -race -count=1 -timeout=3m ./api/shops/... ./api/equipment_services/... ./api/shared/db ./api/middleware ./internal/testsql` |
| [final-fix-final-python-guards](final-fix-final-python-guards.log) | 0 | 28.367 | `python3 scripts/test-shops-isolated_test.py` |
| [final-fix-final-python-runner](final-fix-final-python-runner.log) | 0 | 22.317 | `python3 scripts/apply-shops-remediation-migrations_test.py` |
| [final-fix-final-python-publication](final-fix-final-python-publication.log) | 0 | 4.568 | `python3 scripts/test-publish-shops-candidate-generated.py` |
| [final-fix-final-container-static](final-fix-final-container-static.log) | 0 | 0.086 | `python3 scripts/test-server-container-inputs-static.py` |
| [final-fix-frozen-db-migrations](final-fix-frozen-db-migrations.log) | 0 | 229.537 | `env SHOP_AGGREGATE_PERF=1 SHOP_OVERVIEW_PERF=1 SHOP_MESSAGE_WRITE_PERF=1 bash scripts/test-shops-isolated.sh --verify-notification-migration --verify-message-sync-migration --verify-remediation-migrations --publish-candidate-generated -v -count=1 -timeout=10m ./tests/shops ./tests/equipment_services` |
| [final-fix-frozen-db-race](final-fix-frozen-db-race.log) | 0 | 63.373 | `bash scripts/test-shops-isolated.sh -race -v -count=1 -timeout=5m -run 'Test(CurrentAuthority\|AggregateSingleSnapshot\|AggregateListSnapshotMembership\|AggregateContext\|MessageAllocator\|Cleanup\|BatchRemovalRace\|InviteAdmission\|InviteRevocation\|Departure\|ServicePageSnapshot\|CompletionHistoryCurrentAuthority\|MessageMutationCancellation\|ServiceUsernameLookupCancellation\|LegacyAuditFailureKeepsBusinessSuccess\|EquipmentDeleteCommitFailureDoesNotLogSuccess\|PerformanceSamplerFinishIsIdempotent\|BootstrapEquipmentBoundedParameters)' ./tests/shops ./tests/equipment_services` |
| [final-fix-docker](final-fix-docker.log) | 3 | 10.281 | `bash scripts/test-server-container-inputs.sh /private/tmp/miltechserver-final-fix-container-acceptance miltechserver-container-inputs-test:final-fix-acceptance` |

Counts: host **619 top-level / 1308 including subtests PASS**, three inherited PSMag DB skips; unit race **147 / 437 PASS**, zero skips; full guarded physical matrix/performance **303 / 1002 PASS**, zero skips; physical race **39 / 169 PASS**, zero skips. Python isolation/runner/publication/container-static **11 / 13 / 4 / 3 PASS**. Build/vet exit 0. Full matrix has 27 ERROR and 3 WARN token-bearing lines from negative/denial/injected scenarios; output is not claimed pristine. The physical race has 17 ERROR / 2 WARN token-bearing lines. Host JSON output repeats diagnostics in Output events; token counts there are not independent incident counts.

The full matrix applied all required protected migration/refusal/reverse/data-repair actions, regenerated tagged outputs after stages, compiled intermediate generated packages, ran the guarded release runner for 019–023, then compiled/published the untouched full candidate and ran both owning physical packages with all performance flags enabled. No named target was contacted. Current artifact hashes were checked again after publication.

Capacity: the 12 public notification-route cases retain all headers/items at **65534, 65535, 65536, 67000** with maximum **two** item-query bindings (legacy one). Bootstrap retains exact four authorized parent sets, per-Shop limit/order and unbounded shape at two bindings. These are functional scope/protocol proofs. Complete PMCS: **65536 inspections**, **25,886,989 bytes** per response; 20 measured requests p50 **697.967 ms**, p95 **928.573 ms**, p99 **1059.120 ms**; 13,133,738,784 cumulative allocated bytes across 20 requests (about 656.69 MB/request), not peak RSS. Overview **100 Shops/25000 equipment**, 100 measured plain and gzip requests: p95 **138.729 ms** plain / **142.556 ms** gzip versus the one-second target; response **3,051,158 bytes** plain / **264,693 bytes** gzip. Pool waits zero in these measurements; measured 5ms lock sampling showed zero sampled waiting sessions, not absence of all contention. Same-Shop eight-worker message writes p95 **13.568 ms**, eight Shops p95 **3.135 ms**. Host/race verification ran concurrently on this local host during portions of final measurement; these warm-host timings do not certify cold/concurrent fleet behavior or approved production memory budgets. No speculative index/cap was added.

Docker: actual local Unix-socket daemon probe ran for **10.281 seconds**, exit **3**. Context/layer/final image/runtime/credential mount/read-only generated tree acceptance remains **UNEXECUTED**. Static sentinels and a deliberate omitted-log-pattern negative check pass, but are not Docker runtime PASS.

Self-review and final `git diff --check` are clean within the known candidate. Root owns incremental safe-diff generation and scoped re-review; this report does not say review approved.


Changed paths relative to the controller final-fix baseline (hash inventory in `final-fix-source-freeze.json`; generated publication is tool-only):

- `.dockerignore`
- `.gen/source-manifest.json`
- `api/equipment_services/core/repository_impl.go`
- `api/shops/aggregates/repository_notifications.go`
- `api/shops/aggregates/repository_snapshot.go`
- `api/shops/core/repository.go`
- `api/shops/core/repository_impl.go`
- `api/shops/messages/cleanup_worker.go`
- `api/shops/messages/cleanup_worker_test.go`
- `api/shops/shared/legacy_audit.go`
- `api/shops/vehicles/notifications/items/service_impl.go`
- `api/shops/vehicles/notifications/repository_impl.go`
- `api/shops/vehicles/notifications/service_impl.go`
- `api/shops/vehicles/repository_impl.go`
- `docs/migrations/shop_message_allocator_lock_order.md`
- `docs/migrations/shop_message_asset_lifecycle.md`
- `docs/project_notes/bugs.md`
- `docs/project_notes/decisions.md`
- `docs/project_notes/issues.md`
- `docs/testing/shops-database.md`
- `docs/testing/shops-release-contracts.md`
- `docs/testing/shops-server-remediation-acceptance.md`
- `migrations/020_rollback_shop_message_asset_lifecycle.sql`
- `scripts/apply-shops-remediation-migrations.sh`
- `scripts/apply-shops-remediation-migrations_test.py`
- `scripts/test-server-container-inputs-static.py`
- `scripts/test-server-container-inputs.sh`
- `tests/shops/shops_aggregate_performance_test.go`
- `tests/shops/shops_aggregate_snapshot_consistency_test.go`
- `tests/shops/shops_cleanup_budget_test.go`
- `tests/shops/shops_core_lifecycle_regression_test.go`
- `tests/shops/shops_final_observability_test.go`
- `tests/shops/shops_message_cleanup_test.go`
- `tests/shops/shops_notification_audit_snapshot_test.go`
- `tests/shops/shops_notification_capacity_test.go`
