# Shops server remediation: integrated local acceptance

Date: 2026-10-04. The independent whole-candidate review requested I1–I6/M1–M5 changes. The consolidated fix wave below is implemented and locally verified; **[scoped local re-review is complete](../reviews/2026-10-04-shops-server-remediation/final-fix-scoped-review.md)**: all I1–I6/M1–M5 are addressed, with no new Critical/Important breakage confirmed. This is not shipment approval.

## Current consolidated review fixes and evidence

Current source SHA-256 **`843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`**, 753 source input paths rehashed against the current canonical generated publication. All 32 tracked PMCS outputs equal HEAD. The full command/exits/counts/hash record, per-finding RED/GREEN, changed paths, deferred dispositions and owner gates are in [final-fix-report.md](../reviews/2026-10-04-shops-server-remediation/final-fix-report.md) and its adjacent `final-fix-verification-records.json`.

I1 legacy notifications-with-items now resolves vehicle binding, membership, headers and items in one request-context read-only repeatable-read transaction. I6 item queries bind one selected-ID array (plus aggregate per-parent limit), preserving exact authorized sets/unbounded behavior; bootstrap equipment has the same bounded query repair. I2 every worker DB phase uses the existing operation budget; job phases also cap at remaining lease, fresh acknowledgment honors shutdown, and expired work remains recoverable. I3 reverse 020 takes writer-blocking assets/reference/job locks before any emptiness check, each with a five-second lock-wait budget.

I4 log/temp exclusions and actual synthetic sentinel inventory are expanded. I5/M5 active instructions and labels now match the explicit DTO/tagged generation and exact negotiated cursor behavior. M1 vehicle deletion emits the common sanitized correlated audit-gap event while preserving best-effort deletion. M2 unused unlocked core membership upsert is removed. M3 deletion success is logged after commit; M4 sampler stop/wait is idempotent and registered in test cleanup.

Only `020_rollback_shop_message_asset_lifecycle.sql` changed among the 18 reviewed 015–023 migration files; the other 17, including all 015–018 and every forward body, remain unchanged. New reverse SHA-256 **`d8003e69aae701e8df15b0ee4ab9a7aa2b7c9ebcf2f2a91b96e35bf7039a1f6a`**; guarded SOURCE_PINS/lock-order test/runbook match. The final protected matrix regenerates after each planned action and publishes current source. Earlier statements that all 18 stayed unchanged apply only to the historical Task28 snapshot.

Final checks: safe host **619/1308** top-level/including subtests PASS (three inherited PSMag skips); unit race **147/437** PASS; guarded full DB/migrations/all-performance **303/1002** PASS; physical race **39/169** PASS; build/vet exit 0; Python isolation/runner/publication/container static **11/13/4/3** PASS. `final-fix-frozen-*` logs are current source evidence. The two test-only self-review corrections after the first final snapshot are recorded in the report; the earlier `final-fix-final-*` Go/physical snapshot is superseded. Python scripts were unchanged after their final guards.

Functional notification capacity proves all parents/items at **65534/65535/65536/67000**, across the legacy route, vehicle maintenance and Shop snapshot, with item-query bind count at most two and foreign-Shop denial. This is not a notification production latency/memory acceptance. Complete **65536 PMCS**: 25,886,989 response bytes, p95 **928.573 ms**, about **656.69 MB allocated/request** (cumulative allocation, not peak RSS). **100 Shops/25000 equipment** overview p95 **138.729 ms** plain / **142.556 ms** gzip versus one second. Warm-host measurements overlap other local verification and do not establish cold/fleet concurrency or memory budgets. No cap/index was invented.

Actual bounded Docker probe: **exit 3 in 10.281 seconds**, daemon unavailable. Static adversarial sentinels pass; context/layers/runtime/mount/read-only generation remain **UNEXECUTED**. All open owner gates below remain: late legacy commit, numeric ABA, C06 unknown budgets, separate F09/Task18 credentials, named targets/roles/fleet/writers, TMDE, parser/device/signed artifacts, live alerts, deployment/activation. Date/read flags stay false.

## Retained Task28 evidence (historical source)

The remaining Task28 commands, measurements and source digest are retained as prior evidence and are superseded for current-source provenance by the consolidated section above. The 47 F01–F37/C01–C10 dispositions remain applicable with the explicit final-fix extensions recorded below; none waives an external gate.

## Historical candidate scope before local integration

Worktree: `/Users/swisscheese/projects/miltechserver/.worktrees/shops-server-release-remediation-20261003`.
Branch: `codex/shops-server-release-remediation-20261003`; unchanged HEAD `518ef4e828abeb64c203e567287b730799222da7`. At this historical checkpoint all remediation implementation remained unstaged and uncommitted; the later user-authorized local integration is recorded in [merged-tree verification](shops-server-cleanup-integration.md). The original source review used `4abd3a19fb0efe1ac51d549e4f7a9866c0f8b8a5`; intervening baseline changes were documentation/CodeGraph work. No index/HEAD mutation, commit, merge, push, deployment, capability activation, credential rotation, named-database contact, Flutter inspection or manual generated-model edit occurred.

Go `go1.23.3 darwin/arm64`, PostgreSQL `14.18 (Homebrew)`, disposable `work_mem=4MB`, 16 logical host CPUs. Physical host memory was not measured: the sandbox refused `sysctl hw.memsize`; no memory-capacity certification follows. The fixture uses the approved baseline and current migrations, plus the existing documented materialized-view supplement. The empty TMDE compile ABI does not verify real TMDE joins, data or behavior.

## Verification commands and exact exits

Every command used the explicit worktree above and an outer `rtk proxy python3 <WS>/task-28-run.py <label> ...` recorder. The recorder unsets `TEST_DATABASE_URL`, `TEST_DATABASE_MARKER`, `TEST_DB_URL`, sets `GOCACHE=<WS>/go-cache`, `GOWORK=off`, empty `GOFLAGS`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, and `PYTHONDONTWRITEBYTECODE=1`, then runs the exact argv below. Thus host tests are network-free and cannot use inherited test targets. Database invocations use only the marker-protected loopback wrapper; packages are serialized. The three prohibited broad/dotenv integration package selections were never executed. The full `go build ./...` compiles, and does not execute TestMain.

Logs and their adjacent `.result.json` files preserve independent exits and wall times. The first safe host run, initial capacities, pre-F22 full matrix and pre-F22 races remain historical evidence; the `postfix`/`final` records below supersede them for final source acceptance.


| Record | Exit | Seconds | Exact recorded command |
| --- | ---: | ---: | --- |
| [task-28-final-host-tests](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-host-tests.log) | 0 | 27.005 | `go test -json -count=1 ./api/... ./bootstrap/... ./tools/... ./internal/... .` |
| [task-28-final-host-build](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-host-build.log) | 0 | 4.004 | `go build ./...` |
| [task-28-final-host-vet](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-host-vet.log) | 0 | 2.064 | `go vet ./api/... ./bootstrap/... ./tools/... ./internal/... .` |
| [task-28-final-unit-race](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-unit-race.log) | 0 | 16.833 | `go test -json -race -count=1 -timeout=3m ./api/shops/... ./api/equipment_services/... ./api/shared/db ./api/middleware ./internal/testsql` |
| [task-28-final-python-guards](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-python-guards.log) | 0 | 7.646 | `python3 scripts/test-shops-isolated_test.py` |
| [task-28-final-python-runner](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-python-runner.log) | 0 | 2.08 | `python3 scripts/apply-shops-remediation-migrations_test.py` |
| [task-28-final-python-publication](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-final-python-publication.log) | 0 | 1.519 | `python3 scripts/test-publish-shops-candidate-generated.py` |
| [task-28-postfix-db-migrations](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-postfix-db-migrations.log) | 0 | 140.394 | `env SHOP_AGGREGATE_PERF=1 SHOP_OVERVIEW_PERF=1 SHOP_MESSAGE_WRITE_PERF=1 bash scripts/test-shops-isolated.sh --verify-notification-migration --verify-message-sync-migration --verify-remediation-migrations --publish-candidate-generated -v -count=1 -timeout=10m ./tests/shops ./tests/equipment_services` |
| [task-28-postfix-db-race](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-postfix-db-race.log) | 0 | 26.584 | `bash scripts/test-shops-isolated.sh -race -v -count=1 -timeout=5m -run 'Test(CurrentAuthority\|AggregateSingleSnapshot\|AggregateListSnapshotMembership\|MessageAllocator\|Cleanup\|BatchRemovalRace\|InviteAdmission\|InviteRevocation\|Departure\|ServicePageSnapshot\|CompletionHistoryCurrentAuthority\|MessageMutationCancellation\|ServiceUsernameLookupCancellation)' ./tests/shops ./tests/equipment_services` |
| [task-28-docker-authorized](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-docker-authorized.log) | 3 | 10.067 | `bash scripts/test-server-container-inputs.sh /private/tmp/miltechserver-task28-container-acceptance-authorized miltechserver-container-inputs-test:task28-acceptance-authorized` |

Final host suite: **616 top-level tests / 1,305 tests including subtests PASS**, 48 packages with tests PASS. Three inherited `api/library/ps_mag` database tests skip because `TEST_DB_URL` is intentionally absent: `TestRepositorySearchSummaries_ReturnsResults`, `TestRepositorySearchSummaries_NoMatch`, `TestRepositorySearchSummaries_PageTwo`. Packages reported with no test files are not test coverage. This is not an all-repository integration claim.

Final bounded unit race suite: **144 top-level / 434 including subtests PASS**, no test skips. Python wrapper/runner/publication guards: **11 + 12 + 4 PASS**. F22 focused physical race run passes the nine new subcases plus completion/current-authority/read-cancellation preservation tests. Successful logs retain expected injected diagnostics; output is not described as pristine.


`task-28-postfix-db-migrations`: 297 top-level PASS / 974 including subtests; 0 top-level SKIP; 39 ERROR lines / 1 WARN lines. Zero failing tests is asserted only when the adjacent recorded exit is 0.

`task-28-postfix-db-race`: 32 top-level PASS / 123 including subtests; 0 top-level SKIP; 18 ERROR lines / 3 WARN lines. Zero failing tests is asserted only when the adjacent recorded exit is 0.


The initial capacity invocation failed at loopback socket allocation with `PermissionError: Operation not permitted`; this was a sandbox/setup failure, not a behavior RED. The same protected wrapper then ran with approved loopback access. Initial Docker access under the sandbox returned unresolved; the later authorized local-daemon probe also timed out after ten seconds with exit 3. The context/layer/final-image/mount/writable-generation runtime checks remain **UNEXECUTED**. Earlier synthetic guard checks do not establish actual Docker runtime success. No real credentials were mounted and no runtime configuration was changed.

## F22 regression and implementation

The final inventory found a real propagation gap despite ctx-first repository APIs. `UsernameRepository.GetUsernameByUserID` returned cancellation, but core `Create`, `GetByID`, `Update`; completion `Complete`; queries `GetByShop`, `GetByEquipment`; calendar `GetCalendarServices`; and status `GetOverdue`, `GetDueSoon` swallowed it as a fallback success. The controller explicitly expanded Task 28 for this narrow repair.

`TestServiceUsernameLookupCancellation` now runs each of those nine real services/repositories/authorization paths in a disposable database. A separate real username pool has one connection held; the test observes actual `DBStats.WaitCount` before canceling. All nine returned success before the fix (`task-28-f22-red.log`, exit 1), while persisted-state assertions passed. A cached-username cancellation regression independently failed (`task-28-f22-cache-red.log`, exit 1). These were behavioral REDs, not missing setup or compilation errors.

Services now return resolver errors with `%w`; the shared mapper returns its error to queries/calendar; status handles each lookup error; the cache checks cancellation before a hit. Missing-row/empty-name `Unknown User` behavior remains in the resolver and has a preservation test with its expected warning captured. No response keys/nullability/timestamps, service-date interpretation, authority policy or database schema changed. All nine real pool-wait regressions pass with `-race`; cached cancellation and fallback regressions pass in the final host and race suites.

For `Create`, `Update` and `Complete`, the primary transaction can commit before username enrichment blocks. The regression asserts the one persisted mutation remains, the unrelated row stays byte-identical, and no compensation or extra write occurs. The returned cancellation cannot roll back that committed transaction. A failed legacy response is therefore potentially ambiguous and **must not trigger a blind retry**; this change does not introduce idempotency or a new retry protocol.

### Final context inventory

The [exact scan](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-context-inventory.txt) covers production `api/shops`, `api/equipment_services`, shared transaction/middleware dependencies, `bootstrap`, `internal/jetgen`, and `main.go`, excluding test files; it searches `Background`, `TODO`, `WithoutCancel`, non-context `Begin/Query/QueryRow/Exec`, and legacy `WithTx` calls. Every remaining match has a disposition:

| Source / hit | Disposition |
| --- | --- |
| `main.go:29`, `:45` process root and cleanup worker child | Intentional process startup/worker lifetime; worker cancellation precedes DB close, not a request context escape. |
| `internal/jetgen/generate.go:75` 30-second background timeout | Generic tagged generation/CLI process startup, not a request handler; required before traffic. |
| `api/shared/db/transaction.go:16` legacy `WithTx` Begin | No Shops/equipment caller. Current callers are account deletion, user-saves categories, and PMCS progress; their broader context migration is outside the approved remediation. All affected Shops/equipment transactions use ctx-first helpers/BeginTx. |
| `api/shops/messages/handler.go:230,283`, `sync_handler.go:86,96`, notification changes `handler.go:64` | Gin query-string accessors, not SQL Query calls. |
| `api/user_general/repository_impl.go:51,102` incidental broader scan | Non-context writes reached only by `POST /user/general/refresh` and `/user/general/dn_change` through their service methods. Repository-wide caller search found no Shops/equipment caller. The account delete route also retains legacy `WithTx`; none is the username lookup used here. No unrelated refactor performed. |
| Equipment `shared/mappers.go` and all nine service callers | Confirmed propagation defect repaired and physically verified above. |

There are no remaining matched request SQL context escapes in the affected Shops/equipment packages. This scoped scan is not a proof that unrelated account/PMCS APIs have been migrated. Permission caches remain preflight optimization; persisted write authorization still occurs under the Shop lock.

## Migration, generator and regression provenance

The full matrix runs all five 019–023 populated forward/refusal/reverse fixtures, the actual guarded release runner, 016 receipt retention/reverse refusal, 017 enrichment upgrade, and 018 null refusal/reverse/reapply. Every planned DDL/repair stage regenerates tagged models and compiles model/table/view packages. The full application compiles at the complete candidate schema only. Intermediate prefixes are not labeled compatible with the final app. All 32 tracked PMCS files remain byte-equal to HEAD; migrations 015–018 and all 18 015–023 forward/reverse files remain unchanged relative to the Task 28 starting snapshot.

The release runner uses one backend for identity/marker checks and its migration transaction, effective-role AND privileges and rolled-back legacy writes. Missing/mismatched prior-generation proofs contact zero psql sessions. Final concrete receipts and source/catalog digests follow. These are disposable-target receipts, not live deployment proof.


```text
Candidate source SHA-256: bc5ad7ab73d9049ff7bac44910f10bc6eb337fed99b90c2ab9fa75c00834ba7e
Release runner 019 PASS: identity_session_id=78779 migration_session_id=78779; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0
Release runner 020 PASS: identity_session_id=78844 migration_session_id=78844; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0
Release runner 021 PASS: identity_session_id=78900 migration_session_id=78900; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0
Release runner 022 PASS: identity_session_id=78955 migration_session_id=78955; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0
Release runner 023 PASS: identity_session_id=79016 migration_session_id=79016; rolled-back legacy writes; generation pending; missing/mismatched checkpoint refusals connections=0
Release runner complete disposable checkpoint PASS: all five actions regenerated; 32 PMCS hashes unchanged; no named target contact.
Published untouched canonical full available candidate; all 32 tracked PMCS files unchanged.
```

```json
{
  "source_sha256": "bc5ad7ab73d9049ff7bac44910f10bc6eb337fed99b90c2ab9fa75c00834ba7e",
  "inputs_count": 749,
  "published_inputs_equal_current": true,
  "pmcs_count": 32,
  "pmcs_equal_HEAD": true,
  "migration_files_015_through_023": 18,
  "all_migration_bytes_equal_task28_start": true,
  "generated_manifest": {
    "database": "miltech_test_shops",
    "user": "postgres",
    "address": "127.0.0.1/32",
    "port": 50417,
    "server_version": "14.18 (Homebrew)",
    "schema": "public",
    "catalog_sha256": "5ff4700c1b311ecf193f0e73ee082d6283d5f7e7b6d6f5a0d25cb32da2721567",
    "generated_at": "2026-10-04T11:02:04.691547Z",
    "generator_version": "go-jet/v2 v2.13.0",
    "build_revision": "",
    "compatibility": "legacy shop_messages pre/post-018; additional release readiness gates required"
  }
}
```


The publication guard compares the exact selected Go/SQL/module input set and hashes to the source used for the candidate build, refuses changes, validates canonical paths, checks all 32 tracked PMCS hashes against HEAD, then publishes only untouched tool output. The source digest is not the older Task 17 manifest. `.gen/source-manifest.json` and `.gen/miltech_ng/public/generation-manifest.json` retain current source/schema provenance; the synthetic TMDE caveat remains explicit. No plain Jet or manual generated editing occurred.

Every planned regression prefix is reconciled in the [name inventory](../../.superpowers/sdd/2026-10-03-shops-server-release-remediation/task-28-regression-name-check.json). Three spelling/placement differences are explicit: planned `TestMessageSyncConcurrent` maps to four `TestMessageSyncConcurrency...` tests; `TestShopMessagesCursor` maps to `TestMessagesPaginatedCursorBefore`, `TestMessagesPaginatedCursorAfterIncludesAuthorUsername`, and the new drain/anchor cases; `TestMessageParentMigrationRefusesForeignRows` is the controller-approved physical shell verifier identity in the 021 fixture, not a Go-selectable test. No no-match warning counts as coverage.

Original baseline defects are retained in the historical audit: nil-env production route panic, nine-key message failures, inaccurate atomic audit expectation, and broken Python wrapper regression. Their corrected route/DTO/audit/guard cases now execute and pass. `TestAtomicNotificationItemFieldsSurviveReleasedClientSaves` now verifies actual operation kinds/payloads; production audit kinds were not changed to satisfy the old assertion. The three PS-mag skips above remain visible. The original two opt-in performance tests run in this final full integration command, as do both new opt-in performance tests; none is silently skipped.

## Capacity observations

The final measurements below use warm local fixtures with ANALYZE, one host/process, fake authentication, in-process httptest response recording, and a fresh disposable DB. They are not external HTTP/network, cold-cache, fleet, production-role, physical-device or production concurrent-memory acceptance. Default pool maximum is explicitly eight in measurement connections; monitoring uses a separate test pool. Counts include issued Query/Exec statements but exclude BEGIN/COMMIT and SQL inside triggers. Memory values are process cumulative allocation deltas and before/after live heap samples, not peak RSS or retained-response size. Monitoring/test harness allocations are included.

Overview is exactly **100 Shops / 25,000 equipment**, 100 measured requests per plain/gzip variant after one warmup each; the existing **warm p95 < 1 second** assertion applies to both. Aggregate fixtures contain 25 Shops, 250 equipment, 250 lists / 2,500 items, 250 notifications / 500 items, and 500 services. Optional message measurements additionally insert 120 messages in one Shop: omitted limits return all 120, and explicit `vehicles_limit=2&lists_limit=2&message_limit=2` returns exactly two of each. Nine message keys remain asserted. Default and optional/unlimited shapes are measured separately.

PMCS includes **all 65,536 inspections** (both guide/custom sources), complete fault/comment counts and author enrichment. Twenty measured complete HTTP responses follow a full-content validated warmup; no truncation, pagination or substitute smaller history. The existing Task 23 repository fixture independently verifies all records and four constant-parameter queries with EXPLAIN plans. See the preserved [Task 23 measured record](shops-aggregate-pmcs-history-measurements.md): its ~525 MB repository allocation and spill evidence remain relevant, not replaced by a throughput claim.

Message writes use the actual current repository/authorization/allocator with eight simultaneous workers, 25 writes each (200 measured writes), against either one Shop or eight Shops. Warmups are excluded; each message has 256 content bytes. Every result and persisted total are checked. Shop lock query timings include SQL execution/rows acquisition and repeat acquisition in the same transaction; they are **not pure lock-wait durations**. `pg_stat_activity` samples actual waiting sessions every 5 ms; zero sampled waits does not prove absence of shorter waits. Pool WaitDuration is measured independently.

Raw final percentile/resource measurements:


```text
/shops/bootstrap p50=2.118791ms p95=2.416083ms p99=2.5905ms uncompressed_bytes=36114 gzip_bytes=3189
resources bootstrap wall=235.821916ms statement_count=204 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=31278176 heap_before_bytes=3607112 heap_after_bytes=6996728 lock_samples=0/46 max_waiting_sessions=0 sampling_interval=5ms
/shops/:shop_id/snapshot p50=2.298875ms p95=2.549958ms p99=2.647375ms uncompressed_bytes=59810 gzip_bytes=3467
resources snapshot_unlimited wall=246.203584ms statement_count=714 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=49800728 heap_before_bytes=4186208 heap_after_bytes=6472800 lock_samples=0/48 max_waiting_sessions=0 sampling_interval=5ms
snapshot?include=vehicles,lists,messages,notifications,services p50=2.627ms p95=3.025709ms p99=3.084666ms uncompressed_bytes=122343 gzip_bytes=4849
resources ?include=vehicles,lists,messages,notifications,services wall=296.487708ms statement_count=816 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=86252848 heap_before_bytes=4150072 heap_after_bytes=8602648 lock_samples=0/58 max_waiting_sessions=0 sampling_interval=5ms
snapshot?include=vehicles,lists,messages&vehicles_limit=2&lists_limit=2&message_limit=2 p50=1.380541ms p95=1.428208ms p99=1.511875ms uncompressed_bytes=9807 gzip_bytes=1091
resources ?include=vehicles,lists,messages&vehicles_limit=2&lists_limit=2&message_limit=2 wall=123.030792ms statement_count=510 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=10005120 heap_before_bytes=4299432 heap_after_bytes=4800400 lock_samples=0/24 max_waiting_sessions=0 sampling_interval=5ms
resources PMCS_65536_full_history_20_requests wall=9.367444125s statement_count=80 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=13132004472 heap_before_bytes=3219128 heap_after_bytes=131062296 lock_samples=0/1864 max_waiting_sessions=0 sampling_interval=5ms
PMCS_full_history runs=20 inspections=65536 p50=466.512208ms p95=476.402542ms p99=479.050917ms uncompressed_bytes=25886989
warm-cache p50=72.967375ms p95=74.803291ms p99=82.898416ms compressed_p50=82.97075ms compressed_p95=85.6085ms compressed_p99=87.837917ms uncompressed_bytes=3051158 compressed_bytes=264693
resources overview_100_shops_25000_equipment_202_requests wall=15.865997375s statement_count=202 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=1 total_alloc_bytes=16821065232 heap_before_bytes=8691704 heap_after_bytes=17424800 lock_samples=0/3066 max_waiting_sessions=0 sampling_interval=5ms
resources current_message_writes_shops_1_workers_8 wall=183.627709ms statement_count=2200 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=8 total_alloc_bytes=18626688 heap_before_bytes=3901536 heap_after_bytes=4413472 lock_samples=36/36 max_waiting_sessions=7 sampling_interval=5ms
Shop_lock_query_duration current_message_writes_shops_1_workers_8 n=400 p50=172.333µs p95=9.589458ms p99=15.52625ms includes_execution=true
current_message_writes shops=1 workers=8 writes=200 p50=6.054375ms p95=12.907166ms p99=20.22ms request_message_bytes=256 response_bytes_total=110372
resources current_message_writes_shops_8_workers_8 wall=62.62875ms statement_count=2200 max_open=8 pool_wait_count=0 pool_wait_duration=0s pool_open=8 total_alloc_bytes=22333768 heap_before_bytes=3806400 heap_after_bytes=4813952 lock_samples=0/12 max_waiting_sessions=0 sampling_interval=5ms
Shop_lock_query_duration current_message_writes_shops_8_workers_8 n=400 p50=143.125µs p95=327.25µs p99=467.584µs includes_execution=true
current_message_writes shops=8 workers=8 writes=200 p50=2.319375ms p95=2.804833ms p99=6.928ms request_message_bytes=256 response_bytes_total=110358
full_history inspections=65536 equipment=1 payload_bytes=25886909 runtime=436.183208ms total_alloc_bytes=524965208 heap_before_bytes=3928952 heap_after_bytes=69064032
measurement_environment postgres=14.18 (Homebrew) go=go1.23.3 os=darwin arch=arm64 work_mem=4MB local_single_request=true
```


The overview target passes locally; no measured regression justified a speculative index or production cap. Full PMCS remains expensive: roughly 25.9 MB JSON and hundreds of MB allocated per complete request. Large concurrent histories can exhaust memory well before the local latency target suggests; there is no approved production SLO or concurrent-memory budget to certify. Full/optional reads and offsets keep their approved behavior and can scale linearly with data/offset. No timestamp, date mapping, index, request limit or unlimited-default policy was altered. A 20-sample PMCS p99 is its observed maximum, not a high-confidence tail estimate; 50-sample aggregate and 100-sample overview percentiles have similarly bounded inference.

## Finding and concern dispositions

“Local PASS” below means scoped implementation and listed disposable/host regression evidence. It never substitutes for an owner/fleet/device gate. Historical [audit](../audits/2026-10-02-shops-server-production-review.md), [release review](../reviews/2026-10-02-shops-release-review.md), [spec](../superpowers/specs/2026-10-03-shops-server-release-remediation-design.md), and [plan](../superpowers/plans/2026-10-03-shops-server-release-remediation.md) remain unchanged.

| Finding | Owning task(s) | Local disposition / required behavior | Remaining external or semantic gate |
| --- | --- | --- | --- |

| F01 | 2 | Local PASS: Real production group: c.Error denial/DB failure yields one correct legacy/contract-2 body and status. | Fleet/production acceptance remains separate. |
| F02 | 8 | Local PASS: A-admin/B-member cannot complete/delete B's foreign-authored service using A's Shop ID. | Fleet/production acceptance remains separate. |
| F03 | 12–14 | Local PASS: Copied/foreign/external marker never calls Delete on that target. | Historical asset ownership inventory; unknown assets protected. |
| F04 | 12–14 | Local PASS: Uploader/admin discard only zero-ref registered assets; other member/published asset denied. | Actual storage/worker deployment. |
| F05 | 8–10 | Local PASS: Removal/demotion-first blocks every uncovered writer; mutation-first serializes correctly. | Fleet/production acceptance remains separate. |
| F06 | 3 | Local PASS: Disabled, issued-before-revocation, nil/deleted, invalid and transient failure matrix; handler never runs on failure. | Real Firebase credentials/current-session deployment. |
| F07 | 9 | Local PASS: Create true persists; rename preserves true; both explicit false setting routes persist false. | Fleet/production acceptance remains separate. |
| F08 | 4 | Local PASS: Canonical regeneration plus pre/post-018 reads/send/sync/aggregate JSON has exactly nine message keys. | Actual source/schema/generated fleet proof. |
| F09 | 1,27 | Local PASS: Unit package has no real DSN and cannot contact a network; owner rotation evidence recorded separately. | Original credential-owner rotation confirmation remains OPEN. |
| F10 | 5,27 | Source repair; runtime gate OPEN: Required sentinel/context/layer/mount/startup runtime evidence remains unexecuted. | Docker runtime/layers/mount/generation UNEXECUTED; real mounted credentials require owner acceptance. |
| F11 | 15 | Local PASS: Cross-Shop parent rejected; same-Shop cascade/ref cleanup works; corrupt historical relationship refuses migration. | Existing-target data/FK inventory and migration authorization. |
| F12 | 16 | Local PASS: Creator/admin/member usage-only requests preserve metadata/admin/base values. | Fleet/production acceptance remains separate. |
| F13 | 10 | Local PASS: Revocation committed before delayed claim prevents membership insertion. | Fleet/production acceptance remains separate. |
| F14 | 10 | Local PASS: Duplicate/delayed claim after promotion cannot demote; already-member response is stable. | Fleet/production acceptance remains separate. |
| F15 | 9 | Local PASS: Inject first-membership failure: no Shop/member, no success response. | Fleet/production acceptance remains separate. |
| F16 | 11,14 | Local PASS: Concurrent leaves/removals; last-admin successor rule; creator-first/noncreator-final exit without orphan. | Fleet/production acceptance remains separate. |
| F17 | 9 | Local PASS: Current promoted admin renames; removed/demoted actor denied; explicit deletion stays creator-only. | Fleet/production acceptance remains separate. |
| F18 | 17 | Local PASS: DELETE/re-add different UUID recovers only enrichment; alternating writers/replay/conflicting NIIN cases. | Existing-target conflicting metadata owner resolution. |
| F19 | 16 | Local PASS: New negative base mileage/hours rejected; omitted defaults/nil tracked work; existing bad rows require repair. | Existing negative-value inventory/repair authorization. |
| F20 | 16 | Local PASS: Authorized metadata Admin edit appears in DB and GET; usage-only cannot rename. | Fleet/production acceptance remains separate. |
| F21 | 18 | Local PASS: Invalid second nested item rejects whole batch; negative/blank/overlong/type/target contradictions cause no row/audit changes. | Fleet/production acceptance remains separate. |
| F22 | 1,8–11,16–19,21–26,28 | Local PASS: Canceled pool acquisition/Shop lock/vehicle lock/pre-commit releases resources and returns failure. | Legacy post-commit response may be ambiguous; no blind retry. |
| F23 | 21, final I1 | Local PASS: Aggregate and legacy notifications-with-items writer barriers cannot tear headers/items; membership is in the same read snapshot. | Fleet/production acceptance remains separate. |
| F24 | 22 | Local PASS: Equal timestamps and larger-than-limit newer bursts drain exactly once; deleted anchors signal reload. | OPEN late legacy timestamp commit can land behind observed cursor; no lossless live catch-up claim. |
| F25 | 13 | Local PASS: Oversized file/field/multiple parts rejected within total budget before Azure; temp files cleaned. | Actual storage limits/mount/deployment behavior. |
| F26 | 24 | Local PASS: Metadata edit, repeated/concurrent completion, explicit date and reopen preserve intended historical time. | Fleet/production acceptance remains separate. |
| F27 | 25 | Local PASS: Both listing routes enforce each status/type/completion/date filter and count; deterministic ties/nulls; safe malformed date. | Fleet/production acceptance remains separate. |
| F28 | 10 | Local PASS: Non-null expiry/max-use values reject; omitted/null unrestricted invitation works. | Fleet/production acceptance remains separate. |
| F29 | 7,27 | Local PASS: Partial-grant roles fail AND gate; full role performs actual legacy insert/allocator under invoker privileges. | Actual app role/external writers/target first-test migration. |
| F30 | 4,27 | Local PASS: 017 guide uses tagged generator and identity-checked test-first sequence; no plain-jet instructions. | Real target/source/schema generation; no live execution. |
| F31 | 2 | Local PASS: route.Setup with nil config registers safely with flags off; entire route package completes. | Fleet/production acceptance remains separate. |
| F32 | 20 | Local PASS: Correct stored audit kind/payload and two-change/default-only assertions pass without production kind change. | Fleet/production acceptance remains separate. |
| F33 | 6 | Local PASS: Python wrapper tests import/run and exercise current helper/refusal arguments; real wrapper retains cleanup. | Fleet/production acceptance remains separate. |
| F34 | 20 | Local PASS: Single/bulk add/remove snapshots contain identity/unit/nickname, including deletion and nullable values. | Fleet/production acceptance remains separate. |
| F35 | 23, final I6 | Local PASS: Complete 65536 PMCS and 67000 notification parents/items without oversized binds; selected bootstrap equipment uses two bindings. | Large concurrent history memory/payload budget remains unknown. |
| F36 | 19 | Local PASS: Missing first/non-first/all IDs succeed idempotently; foreign surviving row rejects all; current policy race checked. | Fleet/production acceptance remains separate. |
| F37 | 19 | Local PASS: Duplicates/missing/concurrent delete report actual unique affected count. | Fleet/production acceptance remains separate. |

| Concern | Disposition and remaining gate |
| --- | --- |
| C01 | Local allocator bridge/physical mixed-writer PASS; actual external writers/role/fleet cutover inventory OPEN. |
| C02 | Stronger flag-and-schema/access probe and startup validator PASS; every deployed instance/source/schema/role and real writes unknown. |
| C03 | Ahead-watermark reset handling PASS; numeric in-range restore ABA remains OPEN, requires owner/client recovery policy. |
| C04 | Common numbering integrity guard PASS including NULL outside selected chunk, absent/behind counters and valid deleted gaps. Live schema/writers remain gated. |
| C05 | Required overview, complete PMCS, same/multi-Shop writes, optional/unlimited payloads measured here; no production concurrent/cold/memory SLO inferred. |
| C06 | Removal is not a ban; eight-hex code contract retained. Active invite population and effective authenticated UID/edge budget across instances remain UNKNOWN/BLOCKED. No owner reply/waiver inferred, no arbitrary limiter or code-length change. |
| C07 | Managed assets/references, durable cleanup, late PUT/leases/cascades/crash plus bounded DB phases/recovery and reverse writer-blocking fixtures PASS; historical ownership and real storage/worker rollout OPEN. |
| C08 | Event-time enriched best-effort legacy audit behavior PASS; measured unlimited payload costs recorded; alerting and historical null/meaning acceptance OPEN. |
| C09 | Timestamp contract preserved; `service_dates` and `service_reads` remain false. Live data mapping/released parser/device evidence and separate activation approval OPEN. |
| C10 | Disposable migration/refusal/reverse and actual runner/generation guards PASS. Both named target pin records remain UNPINNED; test-target-first/production separate authorization is UNEXECUTED. |

Two separate credential gates remain open: original F09 credential-owner rotation confirmation, and the unrelated Task 18 Docker `crystaldba/postgres-mcp` diagnostic incident (known role `postgres`, database `miltech_ng`). Neither value was rediscovered/reproduced and neither target was contacted or rotated. Existing logs/config/history were not edited. These incidents are not merged into one resolved item.

OPEN Task 22 late legacy timestamp commit semantics and numeric restore ABA are retained without an owner waiver. No answers to pending policy/exposure questions were inferred.

## Deferred minor triage and remaining release state

| Item from reviewed task ledger | Final disposition |
| --- | --- |
| Task 4 negative catalog type/nullability and namespace/output/backup symlink regressions | Retained test-depth follow-up; no demonstrated new defect, no generator change. |
| Task 5 absent-image cleanup warning | Retained diagnostic follow-up; actual Docker probe unresolved, no fabricated runtime pass. |
| Task 6 failed-marker-query redaction regression | Retained negative coverage follow-up; existing source guards/physical marker checks pass. |
| Task 8 generic-500-compatible denial assertions and equipment pre-commit Delete log | Denial-envelope precision remains follow-up. Delete success logging is fixed by final M3 and its physical deferred-commit failure regression; no false success event on failed commit. |
| Task 11 unused core AddMemberToShop helper | Closed by final M2: no production callers; obsolete interface/implementation and cancellation-only test case removed. Actual membership mutation tests retained. |
| Task 12 generated scripts/__pycache__ artifact | Only `shops_candidate_inputs.cpython-314.pyc` removed; unrelated files/checkpoints preserved. |
| Tasks 13/18/24/26 intentional compensation/access/readiness diagnostics | Existing ERROR/WARN counts recorded, not silenced or called pristine. New fallback test captures/asserts its intentional warning. |
| Task 14 shared two-minute multi-job lease clock regression | Closed by final I2 clock regression: after first operation exhausts lease, second job is not prepared/deleted and expired acknowledgment is left recoverable. Throughput/production budget remains external. |
| Task 18 direct invalid-second-element service/repository no-touch depth | Retained deeper negative coverage follow-up; HTTP/equivalent writer and transactional rollback tests pass. |
| Task 22 exact NULL-anchor and legacy reset envelope assertions | Retained targeted contract-depth follow-up; drain/anchor cases and known late-commit limitation stay explicit. |
| Task 24 focused no-match warnings and repeated prior no-matchs | No-match contributes no coverage; final full equipment package executes the owning cases. |
| Task 27 missing/mismatched successful psql receipt negative regression | Retained runner coverage follow-up; branches inspected previously, real matching backend receipts and exit failures verified. |
| Task 28 sampler failure cleanup | Closed by final M4: idempotent stop/wait registered with t.Cleanup; repeated finish and cleanup regression plus races/performance pass. |
| Vehicle deletion audit-gap collector contract | Closed by final M1 locally: exactly one sanitized correlated common event and committed deletion on injected audit failure. Live collector activation remains unexecuted. |

No speculative minor refactor was added to Task 28. The real F22 service defect found during final inventory was repaired with RED/GREEN evidence instead of being parked as noise.

The owner gate sheet remains [shops-server-remediation-release.md](shops-server-remediation-release.md). Required but unexecuted evidence includes named `miltech_ng_test` then separately authorized `miltech_ng` identity/schema/data/migration/regen; actual DB_USERNAME role privileges and legacy writes; every deployed source/schema/flag/instance and external writer fence; original and unrelated credential confirmations; Docker layers/runtime/Firebase mounts/writable generated output; real TMDE view semantics; owner-approved service-date mapping; owner-provided released request fixtures and parser replays, physical-device recovery and signed artifacts; warning/cleanup/reset observability and alerting; and separate push/deploy/activation authorization. Synthetic Go legacy-request fixtures exercise server contracts only: owner-sanitized released requests were not provided, so no real parser replay or client inspection is claimed.

Code is locally verified, and the [scoped independent re-review is complete](../reviews/2026-10-04-shops-server-remediation/final-fix-scoped-review.md): all I1–I6/M1–M5 are addressed, with no new Critical/Important breakage confirmed. This is not shipment approval. Migrations are disposable-tested only. Credentials, fleet, real clients/devices, signed artifacts, deployment, activation and push remain separate unexecuted gates. Local merge was subsequently authorized; see [integration verification](shops-server-cleanup-integration.md). Original checkpoint/report evidence is preserved in the remediation worktree and current final reports are archived under `docs/reviews/2026-10-04-shops-server-remediation/`.
