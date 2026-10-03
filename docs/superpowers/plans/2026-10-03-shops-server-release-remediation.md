# Shops Server Release Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair the Shops server review findings and establish verifiable production migration, security, compatibility, and release gates.

**Architecture:** Preserve the existing bounded route/handler/service/repository packages. Reuse context-bound transactions and coordinating Shop locks; add narrowly scoped asset and enrichment persistence, fixed message DTOs, and consistent read snapshots. Generated models remain Jet-owned, including generation at startup.

**Tech Stack:** Go module 1.23.0/toolchain 1.23.4, Gin 1.10.1, PostgreSQL, Jet 2.13.0, lib/pq 1.10.9, Firebase Go 4.15.2, Azure azblob 1.6.1, testify 1.10.0. No dependency/version upgrade is part of this plan.

**Spec:** [Approved server remediation design](../specs/2026-10-03-shops-server-release-remediation-design.md). Read it together with the [37-finding audit](../../audits/2026-10-02-shops-server-production-review.md).

## Global Constraints

- Server only: Go server/tests/docs, required migrations, generator/build inputs. No Flutter/client source review or edits.
- Keep automatic Jet generation at application startup, before traffic acceptance.
- After every planned PostgreSQL migration or explicitly approved data-repair step, regenerate through the tagged Jet workflow; never use plain `jet` or manually edit `.gen/**`.
- Verify tracked `user_pmcs_*` generated files remain unchanged; unexpected changes stop the generation sequence.
- Disposable marker-protected loopback clusters for integration/destructive rehearsal. Existing `miltech_ng_test` first, then `miltech_ng`, require separate explicit authorization and same-session identity checks.
- Preserve valid legacy envelopes, nine message keys, nullable keys, raw valid unit codes, timestamp interpretation, and omitted-unbounded reads. Reject unsafe/invalid operations explicitly.
- Explicit Shop deletion remains original-creator-only while currently a member; last admin with other members must first promote a successor; sole-member exit uses internal cleanup.
- Uploader/current admin may discard registered unattached uploads. Published assets require authorized message changes and zero references.
- Invitation expiry/max-use controls are rejected when non-null. Membership removal is not a ban; active codes may readmit.
- Retain notification/NIIN nickname/unit until notification deletion; never restore quantity or an item row merely to recover enrichment.
- Legacy audits remain best effort with monitored warnings; atomic audits/receipts remain mandatory and transactional.
- Keep service_dates/service_reads false until a separate approved mapping/activation gate.
- Prefix shell commands with `rtk`. Do not log credentials, run shared-credential tests, contact existing targets, push, deploy, or commit without the applicable explicit authorization.
- Preserve unrelated dirty/untracked files. Check recent history before editing each target. At execution start use the worktree skill; do not create the worktree during plan review.

## Review Focus

- An upstream response already written must remain a single valid body, even with queued handler errors — Task 2.
- Membership removed after a successful Azure PUT must prevent finalization and leave recoverable cleanup — Task 13.
- Azure PUT completing after cancellation/404 cleanup must be found by trusted tombstone reconciliation — Task 14.
- Conflicting duplicate NIIN metadata must remain ambiguous after deletion makes the active set look unique — Task 17.
- A counter greater than MAX(insertion_number) after deletion is healthy, not corruption — Task 26.

---

## Execution order and shared verification contract

The reviewed source is `4abd3a19fb0efe1ac51d549e4f7a9866c0f8b8a5` on `cleanup`. Recheck HEAD/worktree drift at execution; regenerate ignored Jet inputs from a disposable candidate schema in the isolated workspace before source compilation needs new tables. A copied dirty checkout is not a clean implementation base.

Run Tasks 1–6, then allocator bridge Task 7, then lifecycle Tasks 8–11, assets Tasks 12–15, mutations Tasks 16–20, reads Tasks 21–26, and acceptance Tasks 27–28. Task 27's operator actions are gated, not implicitly executed in numeric order. No new asset/retention protocol is activated while unsafe old writers remain publicly callable.

### Commands used below

`UNIT(<packages>, <regex>)` means:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  go test -count=1 -run '<regex>' <packages>
```

`DB(<regex>)` means:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh -v -count=1 -run '<regex>' \
  ./tests/shops ./tests/equipment_services
```

Before Task 6, use the current wrapper with its approved fixture. After Task 6, its default fixture must include the new migration files present at that task and its isolated generated build workspace. New migration verification is `DB-MIGRATIONS`:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  --verify-remediation-migrations -v -count=1 \
  ./tests/shops ./tests/equipment_services
```

The last flag is produced by Task 6. The wrapper continues accepting only its two approved integration packages and controls package serialization. Do not replace it with a direct TEST_DATABASE_URL to a named database. Test names below are new unless explicitly called existing. Each test assertion snippet describes the decisive assertions after the named fixture/actions, not a complete copied test implementation.

In abbreviated signatures below, `ctx` is `context.Context`, `tx` is `*sql.Tx`, and `user` is `*bootstrap.User`; every untyped ID/text/code parameter is string. Signature changes to existing methods preserve all unspecified payload/result types. New producer types/functions are defined in their owning task, not left for another task to invent.

For every code task, write the regression first, run its listed command and record the intended pre-fix failure, implement the minimal change, rerun to PASS, then `rtk proxy git diff --check` and review its scope. A failure due to missing setup is not the intended red test. Existing unrelated baseline failures remain reported; focused passes do not imply the full suite is green. Commit only when the user authorizes commits; checkpoint records are required regardless.

### New migration files and data gates

| Task | Forward / reverse pair | Data gate |
| --- | --- | --- |
| 7 | `migrations/019_fix_shop_message_allocator_lock_order.sql` / `019_rollback_fix_shop_message_allocator_lock_order.sql` | Verify actual 018 allocator and numbering/roles. |
| 12 | `migrations/020_create_shop_message_asset_lifecycle.sql` / `020_rollback_shop_message_asset_lifecycle.sql` | Preserve unknown historical ownership and actual cascades. |
| 15 | `migrations/021_enforce_shop_message_parent_ownership.sql` / `021_rollback_enforce_shop_message_parent_ownership.sql` | Refuse foreign parents/unapproved FK-action conversion. |
| 16 | `migrations/022_add_shop_vehicle_base_usage_constraints.sql` / `022_rollback_shop_vehicle_base_usage_constraints.sql` | Refuse invalid historical values pending explicit repair. |
| 17 | `migrations/023_create_shop_notification_item_metadata.sql` / `023_rollback_shop_notification_item_metadata.sql` | Refuse conflicting notification/exact-NIIN backfill. |

These new slots refine the originally tentative four-pair grouping: split the two independent integrity gates and number in implementation order; the spec now records the same five pairs. Reconfirm slots before execution; renumber only new pairs consistently if occupied. Keep 015–018 and their pinned checksums unchanged. Each migration task includes its write-up, disposable forward/refusal/reverse checks, and tagged generation. Compile generated packages after each intermediate schema step; compile the full application at the complete candidate schema or matching task stage. Do not call an intermediate schema prefix compatible with final source that imports tables not yet created. Pre-018 legacy SQL compatibility is tested using expanded compiled models and explicit projections, independently of later asset/retention schema.

## Phase 1: HTTP, authentication, context, build and test foundation

### Task 1: Context-bound transaction helpers and network-free unit tests

**Finding:** F09, F22. **Dependencies:** none.

**Files:** Modify `api/shared/db/transaction.go`, `api/shared/db/transaction_test.go`. Create `internal/testsql/driver.go`, `internal/testsql/driver_test.go`, `api/shared/db/transaction_context_test.go`.

**Interfaces:** Preserve `WithTx[T](conn *sql.DB, fn func(*sql.Tx) (T,error)) (T,error)` and existing `WithTxContext(ctx context.Context, conn *sql.DB, fn func(*sql.Tx) error) error`. Produce `WithTxOptions(ctx context.Context, conn *sql.DB, opts *sql.TxOptions, fn func(*sql.Tx) error) error`; context helper delegates with READ COMMITTED. Test-only `testsql.Open(t testing.TB, behavior *Behavior) *sql.DB`; Behavior carries BeginErr/QueryErr/ExecErr/CommitErr/RollbackErr errors, Begins/Queries/Execs/Commits/Rollbacks atomic.Int64 counts, Columns []string and Rows [][]driver.Value. Implement database/sql driver interfaces locally; add no mocking dependency.

- [ ] **Step 1:** Replace the real-DSN tests before running this package. Add `TestWithTxOptionsCancellation`, `TestWithTxOptionsCallbackAndRollbackErrors`, `TestWithTxOptionsCommitError`, and pool-wait cancellation using one occupied connection. Assertions:

```go
require.ErrorIs(t, err, context.Canceled)
require.Zero(t, behavior.Commits.Load())
require.ErrorIs(t, joined, callbackError)
require.ErrorIs(t, joined, rollbackError)
```

- [ ] **Step 2:** Run `UNIT(./api/shared/db ./internal/testsql, TestWithTx)`. Expected red: lost context/options or rollback cause; no socket/DDL is attempted. Never execute the old credential tests to get a red result.
- [ ] **Step 3:** Implement WithTxOptions using BeginTx, callback, commit, and deferred rollback without losing the primary error. Preserve ambiguous commit failure as failure, not a replay invitation. Remove the credential literal without displaying it.
- [ ] **Step 4:** Rerun the command and `rtk proxy go test -race -count=1 ./api/shared/db ./internal/testsql`; expect PASS. Record credential-owner rotation as Task 27's external gate.
- [ ] **Step 5:** Review transaction behavior and API stability; record focused outputs and scope checkpoint.

### Task 2: Complete scoped error handling and nil-config registration

**Findings:** F01, F31. **Dependencies:** Task 1.

**Files:** Modify `api/shops/shared/contract.go`, `api/shops/shared/contract_test.go`, `api/shops/route.go`, `api/route/route.go`. Create `api/route/shops_error_contract_test.go`; update existing route setup tests.

**Interfaces:** Preserve `ContractMiddleware(c *gin.Context)` and public `route.Setup` signature. Add private `setupRoutes(db *sql.DB, router *gin.Engine, authenticate gin.HandlerFunc, env *bootstrap.Env, blob *azblob.Client)`; public Setup supplies the real authentication middleware, package tests supply controlled identity. Reuse `HandleContractError(c, err) bool`. Nil Env produces all optional flags false.

- [ ] **Step 1:** Add `TestProductionShopsQueuedErrorWritesOnce` with real setup, testsql returning an internal sentinel error, and `GET /api/v1/auth/shops/<uuid>`; test both contract selectors, nil registration, two queued errors, and a prewritten body. Decisive assertions:

```go
require.Equal(t, http.StatusInternalServerError, recorder.Code)
require.NotContains(t, recorder.Body.String(), "private-error-sentinel")
require.JSONEq(t, originalBody, prewritten.Body.String())
require.NotPanics(t, func() { Setup(nil, router, nil, nil, nil) })
```

- [ ] **Step 2:** Run `UNIT(./api/shops/shared ./api/route, TestProductionShops|TestContract|TestSetup)`; expect missing body/nil-config failure.
- [ ] **Step 3:** After Next, handle the first unwritten error through the existing classifier/writer; skip written responses. Default nil Env flags off without DB/bootstrap side effects. Keep unrelated public middleware layout intact.
- [ ] **Step 4:** Run full `UNIT(./api/shops/shared ./api/route, .)`; expect PASS, including later tests formerly aborted by panic. Verify global ErrorHandler plus scoped middleware writes once.
- [ ] **Step 5:** Review real route chain and redaction/status mappings; checkpoint.

### Task 3: Enforce current Firebase identity and fail startup on auth errors

**Finding:** F06. **Dependencies:** Task 2.

**Files:** Modify `api/middleware/authentication.go`, `api/middleware/optional_auth.go` only for shared ProcessToken call compatibility, `bootstrap/authentication.go`, `bootstrap/app.go`, `main.go`. Create `api/middleware/authentication_test.go`, `bootstrap/authentication_test.go`.

**Interfaces:** Keep public `AuthenticationMiddleware(*auth.Client) gin.HandlerFunc` and `ProcessToken(*gin.Context,*auth.Client,*auth.Token)` adapters. Private `identityClient` exposes VerifyIDToken(context.Context,string)(*auth.Token,error) and GetUser(context.Context,string)(*auth.UserRecord,error). Produce `processToken(ctx context.Context, client identityClient, token *auth.Token) (*bootstrap.User,error)` and `authenticationMiddleware(client identityClient) gin.HandlerFunc`. Change `NewFireAuth(ctx) (*auth.Client,error)` and `App(ctx context.Context,env *Env) (Application,error)`; update callers in the same task. Preserve SetupEngine's public signature and stop safely on returned initialization error.

- [ ] **Step 1:** Add `TestAuthenticationCurrentIdentity` with IssuedAt=100, valid-after=100000/100001, disabled/nil/deleted user, malformed Bearer, invalid verification, GetUser outage and cancellation. Use a fake implementing the exact SDK interface; include valid token email. Obtain SDK-classified failures through its response parser with a fake HTTP transport, without a live endpoint. Assert:

```go
require.Equal(t, 401, issuedBeforeRevocation.Code)
require.Equal(t, 503, dependencyOutage.Code)
require.Equal(t, 1, fake.GetUserCalls)
require.Zero(t, deniedHandlerCalls)
```

- [ ] **Step 2:** Run `UNIT(./api/middleware ./bootstrap, TestAuthentication|TestFirebase)`; expect disabled/revoked acceptance or nil dereference. Confirm comparison against installed SDK v4.15.2, not a different AuthTime rule.
- [ ] **Step 3:** Pass HTTP context, reject Disabled and IssuedAt*1000 < TokensValidAfterMillis, classify SDK failures, sanitize logging, and abort before user/handler state on failure. Retain matched-route contract-2 auth envelopes, including controlled unavailable classification. Propagate bootstrap errors; do not return a usable nil client.
- [ ] **Step 4:** Run full middleware/bootstrap unit tests and the auth/route subset with `-race`; expect PASS and one lookup. No real Firebase call.
- [ ] **Step 5:** Review effects on all authenticated routes and optional-auth adapter behavior; checkpoint.

### Task 4: Fix message DTOs and share required tagged Jet generation

**Findings:** F08, generator portion of F30. **Dependencies:** Tasks 1–3. Task 6 later owns the complete migration-prefix generation matrix.

**Files:** Modify `api/response/user_shops_response.go`, `api/shops/messages/repository.go`, `repository_impl.go`, `sync_repository.go`, `main.go`, `tools/jetregen/main.go`, `bootstrap/database.go`. Create `main_test.go`, `internal/jetgen/generate.go`, `template.go`, `generate_test.go`, `bootstrap/database_test.go`, `api/response/shop_message_response_test.go`, `tests/shops/shops_message_generation_compatibility_test.go`.

**Interfaces:** Preserve the name `response.ShopMessageResponse`, with nine explicit legacy fields and matching current generated types/nullability. Produce `response.NewShopMessageResponse(message model.ShopMessages,authorUsername *string) ShopMessageResponse`; SQL reads either map explicit column aliases correctly or read the explicitly selected generated row and call this constructor. Produce `jetgen.Config{Schema, OutputDirectory string}`, `jetgen.Generate(db *sql.DB, cfg Config) error`, `jetgen.JSONTaggedTemplate() template.Template`, and `bootstrap.DatabaseDSN(env *Env)(string,error)`. GenerateDB writes canonical OutputDirectory/miltech_ng/schema. CLI accepts JET_DSN and optional JET_OUTPUT_DIR/JET_SCHEMA with defaults .gen/public. Private `setupEngine(ctx context.Context,env *bootstrap.Env,generate func(*sql.DB,jetgen.Config) error,buildApplication func(context.Context,*bootstrap.Env)(bootstrap.Application,error))(*gin.Engine,error)` enables startup tests without live dependencies; public SetupEngine supplies real functions and preserves mandatory generation before traffic using the application's verified database/config.

- [ ] **Step 1:** Add `TestShopMessageNineKeysAfterGeneration` for send/read/cursor/sync/aggregate and `TestStartupJetUsesApplicationPort`. Assert:

```go
require.ElementsMatch(t, []string{"id","shop_id","user_id","message","created_at","updated_at","is_edited","parent_id","author_username"}, keys)
require.Equal(t, persistedID, decodedMessage["id"])
require.Equal(t, persistedText, decodedMessage["message"])
require.Nil(t, decodedMessage["parent_id"])
require.Equal(t, applicationPort, generatorPort)
require.Equal(t, 1, startupGenerationCalls)
```

- [ ] **Step 2:** Run `UNIT(./api/response ./internal/jetgen ./bootstrap ., TestShopMessage|TestStartupJet|TestDatabaseDSN)` and `DB(TestMessageSyncLegacyCompatibility|TestShopMessageNineKeysAfterGeneration)`; expect generated insertion_number leakage/port mismatch. The new disposable test generates tagged models to t.TempDir through jetgen.Generate and compiles the generated package/probe there, never the shared output tree.
- [ ] **Step 3:** Remove model embedding, use explicit legacy SQL projections/readback and the same DTO in sync/aggregates. Consolidate tagged table/view template. Validate output/config/catalog identity; stage and safely publish serialized generation output; sanitize errors. Keep startup generation mandatory. Allow compatible extra columns; record source/schema/build provenance, recognizing regeneration does not rebuild a running binary.
- [ ] **Step 4:** Rerun the commands and build with current expanded models; expect nine keys, canonical tagged generation and port/startup tests PASS. Compare tracked user_pmcs outputs. Task 6 additionally runs the fresh pre/post-018 and complete migration-prefix matrix in isolated workspaces. No manual .gen changes.
- [ ] **Step 5:** Update the stale regeneration workaround in `docs/project_notes/decisions.md` and `key_facts.md`; checkpoint compatible schema/build boundaries.

### Task 5: Exclude secrets from container inputs and support runtime generation

**Finding:** F10. **Dependencies:** Tasks 3–4.

**Files:** Create `.dockerignore`, `scripts/test-server-container-inputs.sh`. Modify `Dockerfile`, `bootstrap/authentication.go`, `docs/testing/shops-release-contracts.md`.

**Interfaces:** Keep FIREBASE_AUTH_KEY as the supported credentials-file location; deployment supplies the file/identity without image embedding. Preserve tagged generated build inputs and a writable generator output location. Script takes a disposable build workspace and image tag, never an existing secret-bearing checkout context.

- [ ] **Step 1:** Add sentinel `.env`/Firebase/VCS/tool files in a temporary build fixture. Expected assertions:

```sh
test "$sentinel_matches_in_context_and_layers" -eq 0
test "$startup_generation_calls" -eq 1
test "$missing_secret_startup_exit" -ne 0
```

- [ ] **Step 2:** Run the sentinel script against the current packaging fixture; expect leaked sentinel/final key COPY. Do not build with real secrets.
- [ ] **Step 3:** Add exclusions, remove secret COPY, provide mounted credentials and writable generation output; retain .gen inputs needed to compile. No startup-generation bypass.
- [ ] **Step 4:** `rtk proxy bash scripts/test-server-container-inputs.sh <disposable-workspace> <local-test-tag>`; expect zero sentinels, working mounted-secret generation, controlled failure for missing credentials/unwritable output. Report unavailable Docker as an unresolved check, never PASS.
- [ ] **Step 5:** Review final filesystem/layers and operational mount instructions; checkpoint. Deployment stays gated.

### Task 6: Repair isolated test wrapper and add migration/generation rehearsal

**Finding:** F33; test infrastructure for C10. **Dependencies:** Task 4's generator CLI.

**Files:** Modify `scripts/test-shops-isolated.sh`, `scripts/test-shops-isolated_test.py`, `docs/testing/shops-database.md`. Create `scripts/verify-shops-remediation-migrations.sh`. Preserve existing TestMain identity/marker validation unchanged.

**Interfaces:** Preserve accepted package/loopback/marker guards and inherited-target refusal. Add `--verify-remediation-migrations`. The wrapper applies available new forward pairs in order, creates an isolated source/generation workspace, and passes only its validated disposable URL/marker to tests. Verification helper consumes the wrapper's guarded connection; it cannot select named targets itself.

- [ ] **Step 1:** Repair/add `test_current_psql_arguments`, `test_inherited_target_refused`, `test_unsupported_package_refused`, `test_marker_required`, `test_cleanup_on_failure`; assertions:

```python
self.assertNotEqual(result.returncode, 0)
self.assertEqual(existing_database_connection_attempts, 0)
self.assertFalse(disposable_cluster_directory.exists())
```

- [ ] **Step 2:** `rtk proxy python3 scripts/test-shops-isolated_test.py`; expect the original import-anchor failure or targeted refusal mismatch.
- [ ] **Step 3:** Exercise current helper arguments instead of obsolete slicing. Preserve cleanup/serialization. Add sorted migration verification and tagged isolated generation after each migration/repair step; use corresponding source stages or generated-package compilation at intermediate prefixes, then full candidate build. Never generate into the shared checkout during fixture comparisons.
- [ ] **Step 4:** Run Python tests and DB-MIGRATIONS with new pairs added by later tasks. Expect refusal/cleanup checks PASS and migration records for every present pair; retain known application failures until their tasks fix them.
- [ ] **Step 5:** Review target safety and generated workspace cleanup; checkpoint.

## Phase 2: Allocator ordering and current authority/lifecycle

### Task 7: Bridge old/current message allocator lock ordering

**Concern:** C01; privilege evidence for F29. **Dependencies:** Tasks 4, 6.

**Files:** Create the 019 pair from the migration table, `docs/migrations/shop_message_allocator_lock_order.md`, `tests/shops/shops_message_allocator_bridge_test.go`. Modify `scripts/verify-shops-remediation-migrations.sh`, `tests/shops/shops_message_sync_concurrency_test.go`.

**Interfaces:** Trigger remains sole insertion-number allocator. New trigger revision acquires persisted Shop FOR KEY SHARE before counter INSERT/UPDATE; invoker privileges remain. No application-side allocation or weaker authorization lock. A reverse restores the pinned 018 function only in a guarded disposable rehearsal/safe fenced deployment.

- [ ] **Step 1:** Add `TestMessageAllocatorMixedWriters`, with one current Shop-FOR-UPDATE writer and one legacy-shaped INSERT, deterministic physical sessions, plus rollback and restricted roles. Assert:

```go
require.NoError(t, currentWriterError)
require.NoError(t, legacyWriterError)
require.Less(t, firstCommittedNumber, secondCommittedNumber)
require.Error(t, selectOnlyRoleInsertError)
```

- [ ] **Step 2:** `DB(TestMessageAllocatorMixedWriters)` against original 018; expect a deliberately orchestrated deadlock, not a wall-clock-only timeout.
- [ ] **Step 3:** Add the new function revision/trigger migration, verifying source definition/counter invariants before changes. Keep app Shop lock/counter commit ordering. Enumerate SELECT/row-lock and counter privileges; no broad grants or SECURITY DEFINER shortcut.
- [ ] **Step 4:** `DB-MIGRATIONS` and `DB(TestMessageAllocator|TestMessageSyncConcurrent)`; expect deterministic mixed paths, different Shops, deletion/rollback, reverse/reapply and partial-role cases PASS. Regenerate after each change.
- [ ] **Step 5:** Record supported writer inventory and fenced unsupported inversion; checkpoint. Physical named-role proof remains Task 27.

### Task 8: Carry context/current actor into resource authorization

**Findings:** F02, F05, F22. **Dependencies:** Tasks 1–3, 7.

**Files:** Modify `api/shops/shared/authorization.go`, `cached_authorization.go`; vehicle `repository.go`, `repository_impl.go`, `service_impl.go`; `api/equipment_services/shared/authorization.go`, core/completion `repository.go`, `repository_impl.go`, `service_impl.go`. Modify calling shared-auth adapters/fakes in Shops and equipment-service packages as identified by the interface compiler. Create `tests/shops/shops_current_authority_test.go`, `tests/equipment_services/equipment_services_authority_test.go`; update existing authorization regression tests.

**Interfaces:** Preserve `LockShopMutation(ctx,tx,shopID,userID)(admin,adminOnly bool,err error)` and owned-resource policies. Add ctx as first parameter to shared authorization interface methods/implementations and their direct callers; keep return types unchanged. Vehicle writer repositories now consume `(ctx,user,existingPayload)`; `AdjustShopVehicleUsage(ctx,user,adjustment)(*model.ShopVehicle,error)` adds the missing actor. Equipment repository methods likewise receive ctx; completion/delete resolve actual Shop and apply persisted existing author/admin policy in their transaction.

- [ ] **Step 1:** Add `TestCurrentAuthorityWriterInterleavings` for vehicle create/metadata/absolute/PATCH and equipment complete/delete; A-admin/B-member nonauthor case; pool/Shop/vehicle lock cancellation. Use channels plus physical locks. Assert:

```go
require.Error(t, writeAfterRemoval)
require.Error(t, writeAfterDemotion)
require.Equal(t, before, persistedAfterRejectedWrite)
require.NoError(t, writeAuthorizedBeforeRemoval)
```

- [ ] **Step 2:** `DB(TestCurrentAuthority|TestEquipmentServiceActualShop)`; expect current PATCH/demotion/wrong-Shop defects. Compile all affected auth consumers to expose interface gaps.
- [ ] **Step 3:** Resolve persisted target → lock Shop/member → ordered references/resource → recheck → write through WithTxContext/QueryContext/ExecContext. Pass context to shared auth/cache/username queries; caches never authorize commit. Retain existing ordinary usage/author/delete permissions; no policy expansion.
- [ ] **Step 4:** Rerun DB command and shared-auth/vehicle/equipment unit packages; expect PASS for every listed writer and cancellation boundary. Remaining core/invite races are covered in Tasks 9–10, not implicitly claimed fixed here.
- [ ] **Step 5:** Review typed identity and lock order end-to-end; record compiler-adapted file list and checkpoint.

### Task 9: Atomic Shop creation, promoted-admin rename, and explicit settings

**Findings:** F07, F15, F17; core portion of F05/F22. **Dependencies:** Task 8.

**Files:** Modify `api/shops/core/handler.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; `api/shops/settings/handler.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; `api/request/shops_request.go`. Create `tests/shops/shops_core_lifecycle_regression_test.go`; update core/settings unit tests and fakes.

**Interfaces:** Add ctx to core/settings repository methods, preserving existing result types. Core create owns Shop+creator-admin insert in one transaction. Metadata update uses presence fields and excludes AdminOnlyLists. Settings request uses `*bool` or equivalent presence representation for the dedicated value; wire name unchanged. Rename persistence admits current admin rather than immutable creator predicate.

- [ ] **Step 1:** `TestShopCreateMembershipAtomic`, `TestShopRestrictionIntent`, `TestPromotedAdminRenameCurrentAuthority`; inject second-insert failure with a disposable DB constraint/controlled repository seam. Assert:

```go
require.Zero(t, shopsAfterFailedCreate)
require.Zero(t, membersAfterFailedCreate)
require.True(t, restrictionAfterRename)
require.False(t, restrictionAfterExplicitFalse)
```

- [ ] **Step 2:** `DB(TestShopCreateMembershipAtomic|TestShopRestrictionIntent|TestPromotedAdminRename)`; expect orphan/create flag/reset/promoted failure.
- [ ] **Step 3:** Implement atomic creation and current-admin rename; make dedicated setting omission/false explicit on both endpoints; preserve omitted metadata fields. Do not move deletion policy into rename.
- [ ] **Step 4:** Rerun DB command and core/settings units; expect PASS, including removal-first rename/settings interleavings and no successful create after membership error.
- [ ] **Step 5:** Review response statuses and no accidental policy columns; checkpoint.

### Task 10: Atomic invite admission/revocation and unsupported-control rejection

**Findings:** F13, F14, F28; invite portion of F05/F22. **Dependencies:** Tasks 8–9.

**Files:** Modify `api/shops/members/handler.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; invites `handler.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; `api/request/shops_request.go`. Create `tests/shops/shops_invite_admission_race_test.go`; update invites/member tests.

**Interfaces:** Produce `members.Repository.JoinViaInvite(ctx context.Context,user *bootstrap.User,code string) error`. It resolves Shop, coordinates lifecycle lock, rechecks active invite and inserts without updating roles. Invite repository methods receive ctx; actual create/revoke/delete checks occur in the same write transaction. Stable already-member behavior remains the existing one. Non-null restrictions reject before writes.

- [ ] **Step 1:** Add `TestInviteRevocationWinsClaim`, `TestDuplicateClaimPreservesPromotion`, `TestUnsupportedInviteControls`, and remove-then-valid-rejoin. Assert:

```go
require.Zero(t, membershipAfterRevokedClaim)
require.Equal(t, "admin", roleAfterDelayedDuplicate)
require.Equal(t, 400, restrictedInviteResponse.Code)
require.NoError(t, validRejoinError)
```

- [ ] **Step 2:** `DB(TestInviteRevocationWinsClaim|TestDuplicateClaim|TestUnsupportedInvite|TestRemovalAllowsRejoin)`; expect stale admission/demotion/ignored controls.
- [ ] **Step 3:** Implement JoinViaInvite and coordinated invite writers. Admission locks the Shop directly before invite/current-member lookup; it cannot call LockShopMutation's RequireShopMember for a not-yet-admitted user. Existing-member detection preserves role; insert never upserts it. Preserve member generation/admin revocation policy, unrestricted defaults/code format, and removal-only semantics. Reject supplied zero/empty restriction values as well as meaningful limits.
- [ ] **Step 4:** Rerun command and relevant unit tests; expect PASS for revoke/delete/duplicate/promotion, null/omitted controls, and removal-first invite creation.
- [ ] **Step 5:** Document supported invite semantics; checkpoint. Practical guessing exposure remains the measured Task 27 gate.

### Task 11: Serialize final departure, successor handoff, and explicit deletion

**Finding:** F16. **Dependencies:** Tasks 9–10; cleanup enqueue integration completed in Task 14.

**Files:** Modify `api/shops/members/repository.go`, `repository_impl.go`, `service_impl.go`; `api/shops/core/repository.go`, `repository_impl.go`, `service_impl.go`. Create `tests/shops/shops_departure_race_test.go`.

**Interfaces:** Produce `LeaveShop(ctx context.Context,user *bootstrap.User,shopID string) error` and `RemoveMemberFromShop(ctx,user,shopID,targetUserID) error`, making count/admin/final cleanup decisions inside one transaction. Private `deleteFinalMemberShop(ctx,tx,shopID,userID) error` is separate from explicit creator-only DeleteShop. Final cleanup later consumes Task 14's transaction-scoped enqueue interface; no request flag.

- [ ] **Step 1:** Add `TestConcurrentFinalDepartures`, `TestLastAdminRequiresSuccessor`, `TestNoncreatorFinalMemberMayLeave`, `TestExplicitDeleteCreatorPolicy`. Assert:

```go
require.Zero(t, memberlessSurvivingShops)
require.Error(t, lastAdminExitWithOthers)
require.NoError(t, soleNoncreatorExit)
require.Error(t, removedCreatorDelete)
```

- [ ] **Step 2:** `DB(TestConcurrentFinalDepartures|TestLastAdmin|TestNoncreatorFinal|TestExplicitDeleteCreator)`; expect count-before-lock/original-creator cleanup defects.
- [ ] **Step 3:** Lock and re-evaluate count/admin state; successor rule precedes removal; sole-member cleanup bypass is internal and guarded by exact current state. Remove standalone blob calls in preparation for durable enqueue; never substitute unsafe URL deletion.
- [ ] **Step 4:** Rerun command for leave/leave, leave/remove, creator-first, promoted successor, explicit deletion. Expect PASS; asset cleanup acceptance waits for Task 14 integration.
- [ ] **Step 5:** Review no automatic owner/promotion guessing and record orphan/adminless live-data inventory gate; checkpoint.

## Phase 3: Managed message assets and reply isolation

### Task 12: Persist owned assets/references and protect discard

**Findings:** F03, F04; schema for C07. **Dependencies:** Tasks 7–11.

**Files:** Create `api/shops/messages/assets.go`, `asset_repository.go`, `asset_repository_impl.go`, `blob_store.go`, the 020 pair, `docs/migrations/shop_message_asset_lifecycle.md`, `tests/shops/shops_message_assets_test.go`. Modify messages `repository.go`, `repository_impl.go`, `service_impl.go`, `api/shops/route.go`; fixture cleanup in `tests/shops/helpers_test.go`, `tests/equipment_services/helpers_test.go`; migration verifier.

**Interfaces:** `Asset{ID,OperationID,ShopID,UploaderID,Account,Container,BlobKey,URL,Extension,State string; LeaseUntil time.Time}`; upload states exactly uploading/ready/cleanup_pending/deleting/deleted. `AssetStorage{Account,Container string}` is constructor configuration, with container shop-message-images and account from verified Env. `NewAssetRepository(db *sql.DB,storage AssetStorage) AssetRepository` produces `Reserve(ctx,tx,user,shopID,extension string)(Asset,error)`, `Finalize(ctx,tx,user,assetID string) error`, `FailReserved(ctx,tx,reserved Asset,reason string) error`, `ReplaceReferences(ctx,tx,user,messageID,text string) error`, `Discard(ctx,tx,user,shopID,assetID string) error`, `EnqueueShopCleanup(ctx,tx,shopID string) error`. Each consumes a coordinated transaction, checks its invariant, and makes no Azure call. FailReserved is server-only compensation using the minted operation/target proof; it changes only still-uploading reservations, never a ready/published asset after uncertain finalization. Define narrow BlobStore Upload/Delete and NewAzureBlobStore from Task 13 here so later consumers compile. IDs map to generated column types internally; targets are immutable. Shared upload defaults are 30s operation timeout/2m lease, reused by Task 14.

- [ ] **Step 1:** Add `TestMessageAssetDeletionAuthority`: foreign/account/copied URLs, trusted URL with query/escaped-path variants, two same-Shop messages sharing one asset, multiple markers, uploader/admin/other member discard and published references. Assert:

```go
require.Zero(t, foreignTargetDeleteCalls)
require.Error(t, otherMemberDiscard)
require.Error(t, publishedAssetDiscard)
require.Equal(t, 2, managedReferenceCount)
```

- [ ] **Step 2:** `DB(TestMessageAssetDeletionAuthority)` with a fake cloud adapter; expect old text/orphan authority failures, not a real Azure operation.
- [ ] **Step 3:** Add registry/reference/job tables, composite `(id,shop_id)` keys/FKs, durable target snapshots, state checks and candidate AFTER DELETE hook. Resolve managed markers by verified account/container/exact blob-key identity, not raw URL equality; query strings cannot hide a known same-Shop reference, and untrusted hosts/path normalization cannot create authority. Preserve display text. Adapter configuration must match the actual SDK storage endpoint, not just an Env label. Wire all-marker reference changes; remove DeleteBlobByURL authority. Discard validates current member plus uploader/admin and zero refs. Unknown legacy/external text remains unmanaged/readable. Update fixture cleanup for independent jobs/registry.
- [ ] **Step 4:** Regenerate after 020 and run DB-MIGRATIONS plus the asset tests; expect empty/populated/reverse refusal, references and authorization PASS. Historical uploader backfill remains gated, never guessed.
- [ ] **Step 5:** Review exact Azure targets, cascade durability and generated diff; checkpoint. Actual deletion execution is Task 14.

### Task 13: Bound multipart uploads and finalize with current authority

**Finding:** F25; Review Focus upload revocation. **Dependencies:** Task 12.

**Files:** Modify `api/shops/messages/handler.go`, `service.go`, `service_impl.go`, `repository.go`, `repository_impl.go`, `assets.go`; create `api/shops/messages/upload_test.go`, `tests/shops/shops_message_upload_authority_test.go`.

**Interfaces:** Consume Task 12's `BlobStore.Upload(ctx context.Context,asset Asset,data []byte,contentType string) error`, `Delete(ctx,asset) error`, and `NewAzureBlobStore(client *azblob.Client,storage AssetStorage) BlobStore`. Produce `ImageUpload{MessageID,ShopID,ImageURL,FileExtension string}` with the four existing snake_case JSON tags. Service `UploadMessageImage(ctx,user,shopID string,imageData []byte,contentType string)(ImageUpload,error)` reserves then uploads outside SQL transaction, then finalizes under fresh authority. Wire response remains message_id/shop_id/image_url/file_extension.

- [ ] **Step 1:** Add `TestUploadBodyBoundBeforeParse`, `TestUploadRevokedAfterPut`, `TestUploadLegacyResponseKeys`. Test 6 MiB total, 5 MiB actual file, oversized fields/multiple parts/zero Shop/nonmember, and remove membership at the fake PUT barrier. Assert:

```go
require.LessOrEqual(t, bytesConsumed, int64(6*1024*1024+1))
require.Zero(t, rejectedUploadCloudCalls)
require.Equal(t, "cleanup_pending", revokedAsset.State)
require.ElementsMatch(t, []string{"message_id","shop_id","image_url","file_extension"}, keys)
```

- [ ] **Step 2:** Run upload units and `DB(TestUploadRevokedAfterPut|TestUploadLegacyResponseKeys)`; expect late bound/missing reservation authority failures.
- [ ] **Step 3:** MaxBytesReader before multipart access; actual bounded read and temporary-file cleanup; preserve supported formats. Reserve unique target, commit, perform context-bound PUT, then transactionally finalize. On denied/failed finalization while context is live, use FailReserved in a separate trusted compensation transaction; on canceled/unavailable context, leave the lease for worker recovery rather than detach business work from HTTP context. Conditional state/operation proof protects a ready row after an ambiguous commit. Do not hold DB locks during PUT or auto-expire ready drafts.
- [ ] **Step 4:** Rerun commands; expect PASS including canceled/truncated read, PUT failure, finalize failure and revoked member. Add `TestUploadAmbiguousFinalizeProtectsReady` with a fake that stores ready then returns a commit error; assert no cleanup transition/deletion of that ready asset. Assert no active transaction while fake PUT runs.
- [ ] **Step 5:** Review request budget and safe failure/response shape; checkpoint.

### Task 14: Durable cleanup worker, cascades, and late PUT recovery

**Concern:** C07, cleanup completion for F03/F04. **Dependencies:** Tasks 11–13.

**Files:** Create `api/shops/messages/cleanup_worker.go`, `cleanup_worker_test.go`; modify `blob_store.go`, `asset_repository_impl.go`, message `repository_impl.go`/`service_impl.go`, core/member `repository_impl.go`/`service_impl.go`, `api/shops/route.go`, `main.go`, `main_test.go`; create `tests/shops/shops_message_cleanup_test.go`. Wire feature workers in main/route, never import messages from bootstrap because messages already imports bootstrap.User.

**Interfaces:** Produce `CleanupJob{ID,AssetID,Scope,Account,Container,BlobKey,ShopID,State string; LeaseUntil time.Time}`, `CleanupRepository.Claim(ctx context.Context,limit int,leaseUntil time.Time)([]CleanupJob,error)`, `Prepare(ctx,job CleanupJob)(Asset,bool,error)`, `Finish(ctx,jobID string,deleteErr error) error`, `Reconcile(ctx context.Context) error`. `CleanupConfig{BatchSize,MaxAttempts int; PollInterval,OperationTimeout,LeaseDuration,MaxBackoff time.Duration}` comes from `DefaultCleanupConfig() CleanupConfig`; `NewCleanupWorker(repo CleanupRepository,blobs BlobStore,cfg CleanupConfig)(*CleanupWorker,error)` validates it. `CleanupWorker.Run(ctx context.Context) error` polls bounded batches with a package-private single-iteration helper and injected now function for tests. Defaults: batch 10, one worker, poll 5s, cloud operation timeout 30s, lease 2m, retry backoff 5s doubling to 5m, manual review after 10 failed attempts; validate positive settings and lease > operation timeout. Upload attempts use the same bounded operation/lease budget. Extend BlobStore with `ListPrefix(ctx context.Context,account,container,prefix,cursor string)(keys []string,next string,err error)` and verified immutable single-key Delete. Claim commits before Prepare; Prepare releases resource locks before cloud I/O; Finish does not hold resource locks. New mutation transactions consume Task 12 EnqueueShopCleanup. Main cancels/joins worker before closing DB; bootstrap remains feature-independent.

- [ ] **Step 1:** Add `TestCleanupReferenceAndCrashRecovery`, `TestCleanupLatePutTombstone`, `TestCleanupAccountMismatch`, parent/Shop/account cascade cases. Fake 404, retryable error, crash-after-delete, expired lease and late PUT after tombstone. Assert:

```go
require.Zero(t, deleteCallsWithSurvivingReferences)
require.Equal(t, trustedTarget, deletedTarget)
require.Equal(t, "deleted", finalAsset.State)
require.Zero(t, cloudCallsUnderDatabaseLock)
```

- [ ] **Step 2:** Run worker unit tests and `DB(TestCleanup)`; expect absent durable retry/cascade/ref protection.
- [ ] **Step 3:** Implement bounded lease/retry worker, immutable account matching, zero-ref/state freeze, 404 success and shutdown. Reserve/PUT leases delay cleanup; terminal tombstones reconcile late writes without reactivation/key reuse. Claim/candidate enqueue cannot lose final removal through deduplication. Exact Shop prefix enumeration is durable/paged; job proof survives parent deletion.
- [ ] **Step 4:** Run worker units with `-race`, DB cleanup/departure tests and 020 populated reverse refusal. Expect PASS; prove cloud calls happen outside DB locks. Use fake clock/barriers, not timing sleeps.
- [ ] **Step 5:** Document backlog/lease/manual-review alerts and historical ownership inventory; checkpoint.

### Task 15: Enforce same-Shop reply relationships

**Finding:** F11. **Dependencies:** Tasks 12–14.

**Files:** Modify `api/shops/messages/repository_impl.go`, `service_impl.go`; create the 021 pair, `docs/migrations/shop_message_parent_ownership.md`, `tests/shops/shops_message_parent_ownership_test.go`; update migration verifier.

**Interfaces:** Message create with ctx/user resolves and locks parent in its coordinated transaction. Reuse Task 12 message composite key; `(parent_id,shop_id)` relationship enforces actual ownership. Preserve verified intended cascade action; unknown live action requires an explicit conversion manifest.

- [ ] **Step 1:** Add `TestMessageParentSameShop`, `TestMessageParentMigrationRefusesForeignRows`, `TestMessageParentCascadeAssets`. Assert:

```go
require.Error(t, crossShopReplyError)
require.Equal(t, beforeSchema, schemaAfterRefusal)
require.Zero(t, survivingDeletedReplyReferences)
```

- [ ] **Step 2:** `DB(TestMessageParent)`; expect accepted foreign parent and missing constraint refusal.
- [ ] **Step 3:** Validate parent after locking, add new composite FK through preflighted migration, and preserve pinned same-Shop cascade/managed cleanup. Do not silently rewrite 005/018 or convert a different live FK action.
- [ ] **Step 4:** Regenerate after each 021 rehearsal; DB-MIGRATIONS and parent/cleanup tests PASS for valid, corrupt-refusal and cascade cases.
- [ ] **Step 5:** Review historical relationship gates and no cross-Shop cleanup; checkpoint.

## Phase 4: Mutation intent, retained enrichment, validation, and audits

### Task 16: Preserve equipment intent, persist Admin, and validate base usage

**Findings:** F12, F19, F20. **Dependencies:** Task 8; migration prefix through Task 15.

**Files:** Modify `api/shops/vehicles/handler.go`, `service.go`, `service_impl.go`, `repository.go`, `repository_impl.go`, `usage.go`; `api/request/shops_request.go`. Create `api/shops/vehicles/mutation_intent.go`, `mutation_intent_test.go`, the 022 pair, `docs/migrations/shop_vehicle_base_usage.md`, `tests/shops/shops_vehicle_mutation_intent_test.go`.

**Interfaces:** `VehicleMetadataUpdate{VehicleID string; Admin,Niin,Model,Serial,Uoc,Comment *string; Mileage,Hours *int32}` carries presence. `VehicleUpdateInput{Metadata VehicleMetadataUpdate; TrackedMileage,TrackedHours *int32}` is the domain input to `Service.UpdateShopVehicle(ctx,user,input VehicleUpdateInput) error`; handler binds presence and maps it before generated-model defaults erase intent. Produce repository `UpdateShopVehicleMetadata(ctx,user,update VehicleMetadataUpdate) error`; keep existing mutually exclusive metadata/usage branch meaning, usage arithmetic and model response types from Task 8. Existing usage-only profile is evaluated before role branches; metadata-only requests omit tracked readings. DB constraints enforce nonnegative base domain after preflight/repair.

- [ ] **Step 1:** Add `TestUsageOnlyPreservesMetadataForEveryRole`, `TestAdminEditPersists`, `TestBaseUsageNonnegative`, and negative-value migration refusal. Include admin/base snapshot labels ignored by usage-only intent. Assert:

```go
require.Equal(t, beforeMetadata, afterUsageMetadata)
require.Equal(t, "NEW-ADMIN", fetched.Admin)
require.Error(t, newNegativeBaseError)
require.Equal(t, oldValue, valueAfterMigrationRefusal)
```

- [ ] **Step 2:** `DB(TestUsageOnlyPreserves|TestAdminEditPersists|TestBaseUsage)` plus intent units; expect privileged metadata wipe/Admin omission/negative acceptance.
- [ ] **Step 3:** Bind presence separately from generated models. Route every usage-only role to preserving tracked writer; other authorized metadata writes SET only present fields including Admin. Validate new base inputs; add preflighted 022 without zeroing historical data. Preserve effective-reading arithmetic and legacy nil/default meaning.
- [ ] **Step 4:** Regenerate after 022; DB-MIGRATIONS and full vehicle usage/intent tests PASS, including nil tracked, explicit permitted clears and ordinary-member rename denial. Do not claim this fixes stale absolute PUT versus newer PATCH.
- [ ] **Step 5:** Review field-presence matrix and historical repair gate; checkpoint.

### Task 17: Retain logical notification-item enrichment across deletion/re-add

**Finding:** F18; Review Focus sticky ambiguity. **Dependencies:** Tasks 1, 8, 16.

**Files:** Create `api/shops/vehicles/notifications/items/retained_metadata.go`, `retained_metadata_test.go`, the 023 pair, `docs/migrations/shop_notification_item_metadata_retention.md`, `tests/shops/shops_notification_metadata_retention_test.go`. Modify item `legacy_mutation.go`, `repository.go`, `repository_impl.go`, `service_impl.go`; notification `atomic_save.go`, `atomic_save_repository.go`; fixture cleanup and migration verifier.

**Interfaces:** `MetadataIntent{ItemID,NotificationID,Niin string; Nickname,UnitOfMeasure *string}`; `ResolvedMetadata{Nickname,UnitOfMeasure *string}`. Produce `ResolveRetainedMetadata(ctx context.Context,tx *sql.Tx,intent MetadataIntent)(ResolvedMetadata,error)` and `RetainItemMetadata(ctx,tx,item model.ShopNotificationItems) error` in the item package. Atomic caller imports the item helper without a reverse parent import. Logical key is notification UUID + exact raw NIIN; resolved/ambiguous state retains raw candidate source IDs and no quantity.

- [ ] **Step 1:** Add `TestMetadataDeleteReadd`, `TestMetadataReplayAfterLaterEdit`, `TestMetadataAmbiguitySurvivesDeletion`, `TestMetadataNotificationIsolation`, and populated backfill/refusal. Assert:

```go
require.Equal(t, "Front hub", *replacement.Nickname)
require.Equal(t, "KT", *replacement.UnitOfMeasure)
require.EqualValues(t, 3, replacement.Quantity)
require.NotEqual(t, removedID, replacement.ID)
require.Equal(t, "ambiguous", stateAfterDeletingConflictingSibling)
```

- [ ] **Step 2:** `DB(TestMetadata)`; expect loss after legacy DELETE/re-add. Same-ID atomic preservation already passes and is not the red proof.
- [ ] **Step 3:** Add 023 and helper under existing notification lock. Preserve omission from existing physical row; retained fallback only for unambiguous logical identity. Capture before deletion, resolve new UUID re-add, handle explicit empty nickname/valid raw unit, sticky ambiguity and explicit resolution per spec. Original request fingerprint precedes resolution; receipt replay does not reapply resolved/current values. No active-item uniqueness/merge rule.
- [ ] **Step 4:** Regenerate after 023; DB-MIGRATIONS and tests PASS for alternating legacy/atomic writes, null/default/unknown raw unit, delayed re-add, conflicting backfill refusal and populated reverse refusal. Check no quantity resurrection.
- [ ] **Step 5:** Review transaction/fingerprint/version/collision boundaries and import cycles; checkpoint.

### Task 18: Validate every equivalent item/notification writer

**Finding:** F21. **Dependencies:** Tasks 16–17.

**Files:** Modify `api/request/shops_request.go`; `api/shops/lists/items/handler.go`, `service_impl.go`; notification/item handlers, services and `legacy_mutation.go`; `api/shops/shared/notification_item_fields.go`. Create `api/shops/shared/item_validation.go`, `item_validation_test.go`, `tests/shops/shops_writer_validation_test.go`.

**Interfaces:** Produce `ValidateItemFields(niin,nomenclature string,quantity int32) error` for trimmed-nonempty required content and positive quantity; `ValidateNotificationType(value string) error` permits only M1/PM/MW. The pinned fixture has TEXT NIIN/nomenclature/title/description/type: do not invent length caps for them. Reuse ValidNotificationItemFields's 50-rune nickname/unit and nonempty raw-unit rules; verified narrower named-target columns are a schema compatibility gate, not a guessed constant. Add bulk-only request `BulkNotificationItemInput{NotificationID *string; Niin,Nomenclature string; Quantity int32; Nickname,UnitOfMeasure *string}` with unchanged JSON keys. Normalize omitted nested ID from the outer target before validation; retain the single-item required-ID contract. Validate bulk before a transaction writes. Same-Shop multi-list batches remain supported.

- [ ] **Step 1:** Add `TestEquivalentWriterValidation` table covering create/update/single/bulk, invalid second element, negative/zero quantity, whitespace, nickname/unit 51 runes, valid long TEXT content, unsupported type, omitted nested-ID inheritance, and nested-target contradiction. Assert:

```go
require.Equal(t, 400, invalidResponse.Code)
require.Equal(t, beforeRows, afterRows)
require.Equal(t, beforeAudits, afterAudits)
require.NoError(t, validUnknownRawUnitError)
```

- [ ] **Step 2:** `DB(TestEquivalentWriterValidation)` and shared validator units; expect invalid bulk/type/quantity persistence.
- [ ] **Step 3:** Apply shared validation in service boundaries plus nested binding; map known validation failures through existing WriteValidationError so invalid input is 400 with the negotiated envelope. Keep TEXT fields uncapped and established 50-rune enrichment limits, without narrower NIIN/NSN formats. Inherit omitted nested notification ID before nested validation, reject supplied mismatch. Validate all elements before writes/audits and preserve multi-list policy.
- [ ] **Step 4:** Rerun command; expect PASS for all variants and both contract envelopes; valid raw unit/no-op default behavior remains.
- [ ] **Step 5:** Review validation-before-write and target-consistency matrices; checkpoint.

### Task 19: Make batch removal idempotent and return actual counts

**Findings:** F36, F37. **Dependencies:** Tasks 8, 17–18.

**Files:** Modify `api/shops/shared/authorization.go`; list-item `handler.go`, `service.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; notification-item corresponding files and `legacy_mutation.go`. Create `tests/shops/shops_batch_removal_contract_test.go`.

**Interfaces:** Removal service/repository methods return `(int64,error)` for actual affected count, adding ctx first where absent. Add `AuthorizeExistingListItems(ctx,tx,userID string,itemIDs []string)([]string,error)` for deletion only; existing strict AuthorizeListMutation behavior remains for nondeletion operations. Produce sorted unique surviving IDs after persisted ownership/current policy checks; all-missing authenticated returns empty/count zero.

- [ ] **Step 1:** Add `TestBatchRemovalActualCountAndStaleIDs`: first/non-first/all missing, duplicates, mixed Shops, foreign survivor, removal race. Assert:

```go
require.EqualValues(t, 1, countForDuplicateAndMissingIDs)
require.Zero(t, allMissingCount)
require.Equal(t, beforeRows, rowsAfterForeignSurvivorDenial)
```

- [ ] **Step 2:** `DB(TestBatchRemoval)`; expect first-ID anchoring/new strict missing-row failure or overstated count.
- [ ] **Step 3:** Deduplicate/find all survivors, lock/recheck every persisted target, skip stale missing rows safely, remove first-ID service inference, perform one authorized deletion and return RowsAffected/count. Keep foreign-survivor whole-batch rejection and retained metadata capture.
- [ ] **Step 4:** Rerun command plus list/notification permissions suites; expect PASS and actual wire counts under concurrent removal.
- [ ] **Step 5:** Review no weakened nondelete authorization or missing-row data exposure; checkpoint.

### Task 20: Enriched event-time history and accurate audit assertions

**Findings:** F32, F34; C08. **Dependencies:** Tasks 17–19.

**Files:** Modify `api/shops/vehicles/notifications/items/service_impl.go`, `legacy_mutation.go`; notification `legacy_mutation.go`; `changes/repository_impl.go`; `tests/shops/shops_notification_item_fields_test.go`, `shops_logging_test.go`. Create `tests/shops/shops_notification_audit_snapshot_test.go`.

**Interfaces:** Add explicit snapshot DTO `ItemAuditSnapshot{ItemID,Niin,Nomenclature string; Quantity int32; Nickname,UnitOfMeasure *string}` to existing payload structures. Atomic item-save events keep stored update kind/items_updated payload; legacy items_added/items_removed kinds remain. Legacy logging includes operation/resource/actor/correlation and failure category; no mandatory legacy outbox. History uses captured title/admin first and deterministic event-ID ties.

- [ ] **Step 1:** Add `TestLegacyAuditEnrichedRemovalSnapshot`, `TestHistoryPrefersEventLabels`, `TestLegacyAuditFailureKeepsBusinessSuccess`; correct existing enrichment test by asserting both real operations and default-only no-change. Assert:

```go
require.Equal(t, "update", storedChangeType)
require.Equal(t, "DZ", *removedSnapshot.UnitOfMeasure)
require.Equal(t, eventTitle, historyTitle)
require.True(t, businessCommittedDespiteLegacyAuditError)
require.Equal(t, 1, structuredAuditWarnings)
```

- [ ] **Step 2:** `DB(TestLegacyAudit|TestHistoryPrefers|TestAtomicNotificationItemFieldsSurviveReleasedClientSaves)`; expect missing enriched snapshots/incorrect audit assertion.
- [ ] **Step 3:** Capture before deletion, include enrichment/identity, prefer event-time labels with old-row fallback, and log best-effort legacy failure. Preserve atomic rollback-on-audit-failure, receipt timestamps, null/empty envelopes and existing limits.
- [ ] **Step 4:** Rerun command plus atomic failure-injection suite; expect PASS and no production audit-kind change to satisfy tests.
- [ ] **Step 5:** Add log alert/runbook coverage; checkpoint.

## Phase 5: Consistent reads, complete pages/history, service lifecycle, sync

### Task 21: Compose aggregates and authors in one snapshot

**Findings:** F23, read portion of F22; username query count. **Dependencies:** Tasks 1, 8, 20.

**Files:** Modify `api/shops/aggregates/repository.go`, `repository_impl.go`, `repository_snapshot.go`, `repository_notifications.go`, `repository_services.go`, `service_impl.go`; create `api/shops/aggregates/snapshot_read.go`, `tests/shops/shops_aggregate_snapshot_consistency_test.go`.

**Interfaces:** Produce package-private `withSnapshot(ctx context.Context,fn func(*sql.Tx) error) error` using WithTxOptions(ReadOnly=true, Isolation=RepeatableRead). Component helpers receive the same `*sql.Tx` and ctx. Add repository `GetVehicleMaintenanceSnapshot(ctx context.Context,user *bootstrap.User,vehicleID string,limits SnapshotLimits)(*response.VehicleMaintenanceSnapshotResponse,error)` so service cannot orchestrate separate pool reads. Current GetShopSnapshot/GetBootstrap/GetListsWithItems signatures remain.

- [ ] **Step 1:** Add `TestAggregateSingleSnapshot` with writer commit after header/before item/count read; `TestAggregateAuthorsBatched` with multiple creators and context failure. Assert:

```go
require.True(t, allComponentsDescribeOneCommittedVersion)
require.Equal(t, expectedScopedCounts, response.Counts)
require.LessOrEqual(t, authorQueryCount, 1)
require.Error(t, canceledReadError)
```

- [ ] **Step 2:** `DB(TestAggregateSingleSnapshot|TestAggregateAuthorsBatched)`; expect torn response or N+1 lookup.
- [ ] **Step 3:** Own snapshot around membership and every component; sequential shared-connection reads/batched joins, no pool/nested transaction escape. Preserve limits/unlimited/count domains and Unknown User fallback; DB failure is not a fallback username.
- [ ] **Step 4:** Rerun command and all aggregate compatibility tests; expect PASS, including read membership-as-of-snapshot and later-request revocation. Record query-count delta.
- [ ] **Step 5:** Review tx/rows lifetime and contract preservation; checkpoint.

### Task 22: Drain legacy message cursor ties and newer bursts

**Finding:** F24; remaining message context paths. **Dependencies:** Tasks 4, 12, 21.

**Files:** Modify `api/shops/messages/repository.go`, `repository_impl.go`, `service.go`, `service_impl.go`; `tests/shops/shops_messages_cursor_test.go`. Create `tests/shops/shops_messages_cursor_drain_test.go`.

**Interfaces:** Replace time-only repository cursor with `GetShopMessagesByCursor(ctx,user,shopID,cursorID string,cursorTime time.Time,isBefore bool,limit int)([]response.ShopMessageResponse,error)`. Other message repository calls receive ctx. Service resolves ID anchors with matching Shop and maps missing legacy anchors to reload error; v2 tuple cursor unchanged.

- [ ] **Step 1:** Add `TestLegacyCursorDrainsTiesAndBurst` using 10 messages and limit 2, equal timestamps, both directions/deleted anchors. Assert:

```go
require.Len(t, distinctCollectedIDs, 10)
require.ElementsMatch(t, expectedIDs, collectedIDs)
require.Error(t, deletedLegacyAnchorError)
```

- [ ] **Step 2:** `DB(TestLegacyCursorDrains|TestShopMessagesCursor)`; expect timestamp omissions or newest-burst repetition.
- [ ] **Step 3:** Use tuple predicate/order, nearest-newer selection and draining continuation. Preserve legacy presentation envelope; continuation follows drain boundary rather than accidental last displayed newest ID. No schema numbering dependency in legacy projection.
- [ ] **Step 4:** Rerun command plus v2 deleted-anchor history tests; expect complete one-time draining and preserved v2 behavior.
- [ ] **Step 5:** Document deleted legacy anchor reload and server fixture handoff; checkpoint.

### Task 23: Remove PMCS history bind-parameter ceiling

**Finding:** F35. **Dependencies:** Task 21.

**Files:** Modify `api/shops/aggregates/repository_pmcs_history.go`; create `tests/shops/shops_pmcs_history_capacity_test.go`; extend `tests/shops/shops_equipment_pmcs_history_test.go` and `docs/testing/shops-message-sync-measurements.md` only to link a separate aggregate measurement record.

**Interfaces:** Preserve GetEquipmentPmcsHistory(ctx,user) result/source union; consume the same snapshot helper. Relational membership/equipment/inspection joins replace variable-length UUID parameter arrays for rows and fault/comment counts.

- [ ] **Step 1:** Add `TestPmcsHistoryBeyondParameterLimit` with 65,536 accessible inspections plus foreign/guide/custom controls and `TestPmcsHistoryQueryParameterCount`. Assert:

```go
require.Len(t, accessibleInspections, 65536)
require.Equal(t, expectedFaultAndCommentCounts, counts)
require.Less(t, boundParameterCount, 65535)
```

- [ ] **Step 2:** `DB(TestPmcsHistoryBeyondParameterLimit|TestPmcsHistoryQueryParameterCount)`; expect old driver ceiling, not a reduced fixture/cap.
- [ ] **Step 3:** Implement bounded-parameter joins/subqueries preserving complete access/source/provenance/privacy/count semantics. No new response cap or speculative index.
- [ ] **Step 4:** Rerun command; measure representative EXPLAIN/query count/payload/memory in disposable DB. Expect PASS and all records, with plans recorded.
- [ ] **Step 5:** Review full history rather than only query-construction success; checkpoint.

### Task 24: Preserve historical completion date on edits/retries

**Finding:** F26. **Dependencies:** Task 8.

**Files:** Modify `api/equipment_services/core/service_impl.go`, `repository_impl.go`, `completion/service_impl.go`, `repository_impl.go`; create `tests/equipment_services/completion_history_test.go`.

**Interfaces:** Keep Complete(ctx,user,serviceID string,completionDate *time.Time)(*model.EquipmentServices,error) and metadata result types. Current persisted completed/date state determines defaulting under the same authority transaction; omitted is_completed binding/default contract is preserved.

- [ ] **Step 1:** Add `TestCompletionDateStableOnRetryAndMetadataEdit`: first completion, repeat omission/null, explicit date, reopen and concurrent repeat. Assert:

```go
require.Equal(t, firstCompletionDate, retry.CompletionDate)
require.Equal(t, firstCompletionDate, metadataEdit.CompletionDate)
require.Equal(t, explicitDate, explicitEdit.CompletionDate)
```

- [ ] **Step 2:** `DB(TestCompletionDateStable)`; expect historical time rewritten.
- [ ] **Step 3:** Default operation time only on first completion; preserve already-completed date on omission; explicit value and reopen retain existing semantics. Resolve state inside transaction, not a stale service preflight.
- [ ] **Step 4:** Rerun command and service core/completion/edge suites; expect PASS under actual-Shop author/admin policy.
- [ ] **Step 5:** Review no accidental date-only/flag/default activation; checkpoint.

### Task 25: Implement service filters and consistent deterministic pages

**Finding:** F27. **Dependencies:** Tasks 1, 8, 24.

**Files:** Modify `api/equipment_services/queries/route.go`, `service.go`, `service_impl.go`, `repository.go`, `repository_impl.go`; calendar `service_impl.go`, `repository.go`, `repository_impl.go`; status repositories/services for shared predicate/context use; `shared/errors.go`. Create `api/equipment_services/shared/service_filters.go`, `service_filters_test.go`, `tests/equipment_services/service_filter_contract_test.go`.

**Interfaces:** Keep public query request fields. Replace service equipment query with `GetByEquipment(ctx,user,equipmentID string,req request.GetEquipmentServicesRequest)(*response.PaginatedEquipmentServicesResponse,error)` and repository counterpart returning `([]model.EquipmentServices,int64,error)`; GetByShop preserves request/result types with ctx added to repository. Produce `ServiceFilters{Status,ServiceType string; IsCompleted *bool; EquipmentID string; From,To *time.Time}` and `ServiceFilterPredicate(filters ServiceFilters,evaluationTime time.Time)(postgres.BoolExpression,error)`. Both count/rows share read-only RR options from Task 1; use existing sort direction plus service ID/null order.

- [ ] **Step 1:** Add `TestServiceFiltersConjunctive` on both routes with completed/overdue/future/null date records, type/completion contradictions, status blank/unknown, seven-day boundary, tied IDs and malformed timestamp. Assert:

```go
require.ElementsMatch(t, expectedDueSoonIDs, dueSoonIDs)
require.Subset(t, scheduledIDs, dueSoonIDs)
require.EqualValues(t, len(expectedIDs), filteredTotal)
require.Equal(t, "invalid", contract2MalformedDate.Code)
```

- [ ] **Step 2:** `DB(TestServiceFiltersConjunctive|TestServicePageTies)` plus predicate units; expect ignored equipment/status filters, count mismatch or internal date error.
- [ ] **Step 3:** Apply spec timestamp predicates exactly: overdue < now; due soon > now through now+7 days; scheduled > now includes due soon. Conjoin supplied filters; blank status means none, unknown rejects. Dedicated days_ahead remains 1–30. Known timestamp parse failures map through existing invalid classifier; preserve legacy envelope and gated mapping.
- [ ] **Step 4:** Rerun command and query/calendar/status suites; expect PASS for count/rows/ties/nulls. Document offset shifts between requests instead of claiming keyset behavior.
- [ ] **Step 5:** Review boundaries/timezone interpretation and unchanged false date capabilities; checkpoint.

### Task 26: Guard all sync reads and strengthen readiness discovery

**Concerns:** C02, C03, C04; Review Focus deleted-number gap. **Dependencies:** Tasks 4, 7, 21–22.

**Files:** Modify `api/shops/messages/sync_repository.go`, `sync_contract.go`, `sync_readiness.go`, `sync_handler.go`, `api/shops/capabilities/route.go`, `api/shops/route.go`, `api/shops/vehicles/notifications/handler.go`, `route.go`; create `api/shops/capabilities/readiness.go`, `readiness_test.go`, `tests/shops/shops_message_sync_integrity_test.go`; extend sync route/schema tests.

**Interfaces:** `ValidateShopMessageNumbering(ctx context.Context,tx *sql.Tx,shopID string)(int64,error)` returns valid counter (empty missing counter=0) or typed unavailable. `syncResetRequired() *shared.Failure` has Status 409/Code message_sync_reset_required. Flags gain AtomicNotificationReady func(context.Context) bool; advertisement is flag AND readiness. Produce `capabilities.AtomicReady(ctx context.Context,db *sql.DB) bool`; notifications RegisterRoutes receives its func(context.Context) bool closure for the atomic HTTP entry only. Entry validates user/contract/body, then checks schema/access independently of advertisement flag and returns controlled 503 when unavailable. Probe methods are bounded read-only current-role checks; pure atomic service tests remain independent of catalog I/O.

- [ ] **Step 1:** Add `TestSyncCommonIntegrityGuard`, `TestSyncWatermarkResetRequired`, `TestCapabilitiesFlagAndReadiness`: NULL outside selected chunk, NULL created_at, missing/behind counter, counter above max after deletion, wrong schema/access, flag off/ready. Assert:

```go
require.Equal(t, 503, corruptCatchUp.Code)
require.Equal(t, "message_sync_reset_required", aheadResponse.Code)
require.NoError(t, counterAboveDeletedMaximumError)
require.False(t, capabilityWithMissingRequiredGrant)
```

- [ ] **Step 2:** `DB(TestSyncCommonIntegrity|TestSyncWatermarkReset|TestCapabilitiesFlagAndReadiness)` and readiness units; expect filtered NULL escape/raw flag readiness/undifferentiated reset.
- [ ] **Step 3:** Guard every per-Shop sync snapshot before advancement; counter >= max, never equality. Add explicit ahead reset vs malformed through<after classification. Retain held bounds/gaps/member rechecks, unsigned cursor authorization, and sole trigger allocator. Strengthen required schema/access probes; date flags remain false.
- [ ] **Step 4:** Rerun command and all sync/atomic capability suites; expect PASS including member removed between chunks and callable-before-advertisement. No numeric in-range restore ABA claim.
- [ ] **Step 5:** Document client/operator reset handoff and per-instance/fleet gates; checkpoint.

## Phase 6: Runbooks, data gates, and integrated acceptance

### Task 27: Correct operational instructions and define target-specific release gates

**Findings:** F29, F30; gates C05, C06, C09, C10 and external parts of F09/F10. **Dependencies:** Tasks 4–26.

**Files:** Modify `docs/testing/shops-database.md`, `shops-release-contracts.md`, `shops-service-date-mapping.md`, `shops-message-sync-measurements.md`; `docs/migrations/shop_notification_item_nickname_uom_migration.md`; existing notification/message migration runners only where a guarded new release procedure requires it. Create `scripts/apply-shops-remediation-migrations.sh`, `scripts/apply-shops-remediation-migrations_test.py`, `docs/testing/shops-server-remediation-release.md`; update project decisions/key facts/bugs/issues with actual implemented status.

**Interfaces:** New runner accepts one explicit authorized target and approved forward action, refuses UNPINNED/wrong identity/inherited selectors before contact, then verifies db/address/port/role/version/checksums in the same migration session. Privilege checks are independent AND checks under the actual application role from DB_USERNAME. Reverse actions require a separate explicit resolution authorization; no convenience destructive default. Conditional UID-limiter work, if exposure evidence requires it, belongs to `api/shops/shared/invite_rate_limiter.go`, `invite_rate_limiter_test.go`, member/invite route files and Env config/tests; owner-approved numeric budget is required before that dependent work.

- [ ] **Step 1:** Add `test_runner_refuses_unpinned_and_wrong_target`, `test_runner_uses_same_identity_session`; disposable `TestRequiredPrivilegeIntersection` for SELECT-only/INSERT-only/UPDATE-only/full roles. Assertions:

```python
self.assertEqual(connection_attempts_for_unpinned, 0)
self.assertEqual(identity_session_id, migration_session_id)
```

```go
require.False(t, partialPrivilegeGate)
require.NoError(t, fullApplicationRoleLegacyInsert)
```

- [ ] **Step 2:** Run runner Python tests and `DB(TestRequiredPrivilegeIntersection)`; expect original comma-list false positive. Do not contact miltech_ng_test or miltech_ng while obtaining red evidence.
- [ ] **Step 3:** Implement guarded/pinned forward runner, real-role checks, tagged regeneration checkpoint after each action and corrected 017 test-first instructions. Record manifests for orphan/adminless Shops, parent actions/foreign parents, negative bases, ambiguous metadata and protected historical assets; unresolved cases refuse activation rather than auto-repair.
- [ ] **Step 4:** Run Python refusal tests and DB-MIGRATIONS; expect PASS. Prepare these separate, unexecuted owner gates: named test-target authorization/identity/application-role write; then production authorization; credential rotation confirmation; uniform build/flags and supported external writers; mounted secrets/writable generation; service-date mapping; sanitized released-parser/device recovery evidence.
- [ ] **Step 5:** Evaluate C06 from active invite population and effective authenticated/edge fleet budget. If insufficient, obtain the owner-approved limits and add a bounded UID limiter for claim/create with expiry/eviction plus tests for burst/cross-instance budget/many identities; do not silently change code length or reuse an unbounded IP map on all reads. Record unresolved exposure as a release blocker. Checkpoint the complete gate sheet.

### Task 28: Verify the integrated candidate and report exact remaining release state

**Coverage:** Every F01–F37 and C01–C10. **Dependencies:** Tasks 1–27.

**Files:** Create `docs/testing/shops-server-remediation-acceptance.md`; add representative performance fixtures in `tests/shops/shops_aggregate_performance_test.go`, `shops_equipment_overview_performance_test.go`, `shops_message_sync_plan_test.go`; preserve existing audit and link dispositions. No test runs against a named database are implied by this task.

**Interfaces:** Acceptance record contains integrated HEAD/worktree state, actual Go version, generated-source/schema manifest, exact commands/results, 37 findings/10 concerns, capacity observations, and separate code/migration/credential/fleet/client/push/deploy status. It consumes all earlier task evidence, not just a passing focused test.

- [ ] **Step 1:** Check all task regression names execute, migration/refusal/reverse cases are present, and all literal scope/Jet policies survive the integrated diff. No unresolved baseline is relabeled a skip. Prepare sanitized released-request replay fixtures with owner-provided values; no Flutter inspection.
- [ ] **Step 2:** Run fresh network-free `rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL go test -count=1 ./api/... ./bootstrap/... ./tools/... ./internal/... .`; then `rtk proxy go build ./...`, `rtk proxy go vet ./api/... ./bootstrap/... ./tools/... ./internal/... .`, both Python scripts and DB-MIGRATIONS. Expect exit 0 for each; capture each exit separately.
- [ ] **Step 3:** Run unit race checks and physical-session DB race variants through the wrapper (`-race -count=1`), including current-authority, snapshot, allocator and cleanup scenarios. Container sentinel check must pass or remain explicitly unresolved. Do not run `go test ./tests/...` because other packages can use existing dotenv targets.
- [ ] **Step 4:** Measure overview 100 Shops/25,000 equipment at the established warm-cache p95 <1 second target; complete PMCS 65,536 fixture; representative same-Shop/multi-Shop writes and optional/unlimited aggregate payloads. Record p50/p95/p99, pool/lock wait, query count, bytes and memory. Add optimization/index only when measured evidence justifies it, then rerun its affected tests/measurement.
- [ ] **Step 5:** Perform the selected execution method's whole-branch independent review, resolve valid findings, and rerun checks affected by any changes. Confirm exact source/generated state. The final report distinguishes local code PASS from unperformed existing-target migration, rotation, fleet, client/device, push and deployment gates; do not declare shipment ready without their evidence.

## Coverage index

| Findings / concerns | Owning tasks |
| --- | --- |
| F01, F31 | 2 |
| F02, F05 | 8; core/invite authority also 9–10 |
| F03, F04, F25 | 12–14 |
| F06 | 3 |
| F07, F15, F17 | 9 |
| F08 | 4 |
| F09 | 1; rotation gate 27 |
| F10 | 5; deployment/rotation gate 27 |
| F11 | 15 |
| F12, F19, F20 | 16 |
| F13, F14, F28 | 10 |
| F16 | 11, 14 |
| F18 | 17 |
| F21 | 18 |
| F22 | 1, 8–11, 16–19, 21–26; full context closure below |
| F23 | 21 |
| F24 | 22 |
| F26 | 24 |
| F27 | 25 |
| F29 | 7, 27 |
| F30 | 4, 27 |
| F32, F34 | 20 |
| F33 | 6 |
| F35 | 23 |
| F36, F37 | 19 |
| C01 | 7, 27 |
| C02, C03, C04 | 26, 27 |
| C05 | 21, 23, 28 |
| C06 | 10, 27 |
| C07 | 12–15 |
| C08 | 20, 28 |
| C09 | 25–27 |
| C10 | 6, 27–28 |

### Context closure and verification ownership

Each domain task migrates every service→repository read/write boundary in its touched package to ctx-first and adapts its direct callers/fakes, retaining other parameter/result types. Lists/list items are owned by Tasks 18–19; notification/item/changes by Tasks 17–20; messages by Tasks 12–14/22/26; aggregates by Tasks 21/23; core/member/settings/invites by Tasks 8–11; equipment services by Tasks 8/24–25. Do not leave an untouched read method calling Query without context simply because its write regression passes.

Task 28 searches the affected production packages for context.Background/Begin/noncontext Query/Exec, inspects each remaining hit, and records a justification or fixes it through the owning task. Genuine process startup/worker lifetime contexts remain intentional; request boundaries cannot use them. Canceled username lookup must propagate error. Permission cache hits remain preflight optimization, not commit authorization.

### Review and commit boundaries

The chosen execution method controls task/whole-branch review; this planning work dispatches no agents. Keep security, lifecycle, asset, retention and read changes independently reviewable at the task checkpoints. Commit only if the user explicitly authorizes commits; use Conventional Commits and stage only the reviewed task's files. Do not include unrelated CodeGraph/report changes. No push, merge, deploy, named-database migration or capability activation is authorized by plan approval alone.

## Plan review and execution handoff

This plan is documentation-only and awaits review. After review, the user selects **Subagent-driven** (fresh implementer/reviewer per task, plus whole-branch review) or **Native** (one implementer in this session, then whole-branch review). Subagent-driven is recommended because asset/retention protocols, authorization races and irreversible cleanup cross several package interfaces; task review gates reduce the chance of shipping a valid-looking but unsafe partial repair. Either method preserves the separate existing-target/data/deployment gates.
