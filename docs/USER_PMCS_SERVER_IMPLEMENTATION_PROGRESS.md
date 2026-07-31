# User-Created PMCS Server Implementation Progress

## Execution baseline

- Started: 2026-07-29 (America/Phoenix)
- Branch: `user_generated_pmcs`
- Expected handoff HEAD: `1125dfac32e17f2560ac4b3a2982be7ff37274f9`
- Actual baseline HEAD: `1125dfac32e17f2560ac4b3a2982be7ff37274f9`
- Initial worktree: clean (`git status --short --branch` returned only
  `## user_generated_pmcs`)
- Migration-number verification: migrations `001` through `008` are tracked;
  no migration `009` exists, so `009` remains the correct number.
- Database-target preflight:
  - `TEST_DATABASE_URL` resolved to `miltech_ng_test`.
  - Standard `DB_HOST`, `DB_PORT`, `DB_USERNAME`, and `DB_PASSWORD`
    variables resolved with `--dbname=miltech_ng` to `miltech_ng`.
  - No migration was applied during this read-only preflight.
- Authoritative documents re-read:
  - `docs/superpowers/plans/2026-07-29-user-pmcs-server-implementation.md`
  - `docs/superpowers/specs/2026-07-28-user-pmcs-server-schema-api-design.md`
  - `docs/superpowers/specs/2026-07-28-user-pmcs-server-schema-api-design-REVIEW.md`
  - `docs/client/2026-07-27-user-pmcs-server-sync-schema-design.md`
  - all four ordered implementation plans
- Required research completed before implementation:
  - Gin transaction handling
  - conditional HTTP request semantics
  - Postgres FK, partial-index, composite-index, primary-key, and data-type
    guidance
  - sequential implementation and migration-risk analysis
- Production migration, push, merge, deployment, and mobile implementation:
  not authorized and not performed.

## Baseline verification

- Command: `go test ./... -count=1`
- Sandboxed attempt: failed because 13 integration packages could not connect
  to `192.168.20.70:5432` (`connect: operation not permitted`); this was an
  environment denial, so the command was rerun with database network access.
- Database-enabled result: 635 passed, 3 failed, 14 skipped across 90
  packages.
- Accepted baseline failures:
  - `api/route`:
    `TestSetupRegistersPmcsSbsFaultRoutesUnderAuth` — route not registered:
    `GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`.
  - `tests/pmcs_sbs_progress`:
    `TestRepositoryEnsureInspectionCreatesRecord` —
    `pq: insert or update on table "shop_vehicle" violates foreign key
    constraint "user_vehicle_shop_id_fkey"`.
  - `tests/shops`:
    `TestShopsBootstrapReturnsAllEquipmentWhenLimitOmitted` —
    `pq: insert or update on table "shop_vehicle" violates foreign key
    constraint "user_vehicle_shop_id_fkey"`.
- Status: baseline established with three unrelated pre-existing failures; no
  full-green claim.

## Task progress

### Foundation Task 1: Migration, integrity tests, and Jet generation

- Status: complete; independent review passed after fix round 1
- Base commit: `1125dfac32e17f2560ac4b3a2982be7ff37274f9`
- Implementer: `/root/foundation_task1_impl` (fresh)
- Reviewer dispatched: `/root/foundation_task1_review` (fresh; historical checkpoint)
- Focused tests:
  - `go test ./tests/user_pmcs -run TestUserPmcsSchemaIntegrity -count=1`
    passed after canonical generation (`ok`, 0.525s final post-commit run).
  - `go test ./... -run '^$'` passed: 91 packages compiled (40 package pass
    events, 51 packages with no tests).
- Database migration results:
  - `miltech_ng_test`: verified target; forward passed; schema test passed;
    rollback passed; absence check returned `t`; second forward passed; schema
    test passed.
  - `miltech_ng`: verified target; one forward passed; existence check returned
    `user_pmcs_sync_state`; no rollback was run.
  - Production: not targeted.
- Review findings:
  - Important: partial-index tests did not prove the exact
    `(checklist_id)` key and exact `state = 'draft'` / `state = 'published'`
    predicate.
  - Important: `TestMain` discarded `loadEnv()` errors and could hide the
    original `.env` read/parse failure.
  - Minor (deferred to final whole-branch review): restrictive-FK catalog
    lookup does not explicitly require the `public` namespace.
  - Reviewer warnings resolved by controller:
    - RED/GREEN chronology, migration rehearsal, compile, and canonical Jet
      provenance are recorded with exact evidence in the implementer report.
    - Grapheme/count enforcement and server-side SHA-256 computation are
      explicitly Foundation Task 3 scope; Task 1 correctly supplies storage
      constraints and generated types only.
- Task commits:
  - `2d179b3ec72edbb79c4776a53830095485e26184`
    `feat(user-pmcs): add sync and sharing schema`
  - `7508b29991331835683d798c75d69be6a82ed771`
    `test(user-pmcs): strengthen schema integrity checks`
- Full-suite comparison:
  - `go test -count=1 -json ./...`: 653 passed, 3 accepted failures, 14
    skipped.
  - Failures are the same three baseline failures recorded above; Task 1 added
    18 passing schema-test events.
- Jet generation:
  - Canonical tracked generator configuration from `main.go:56-96` ran against
    verified `miltech_ng`.
  - Generated 12 model and 12 table files; UUID columns have no generated
    defaults; `content_hash` is `[]byte`.
  - A raw-generator JSON-tag mismatch was detected before commit, fully
    recovered with the canonical generator, and the affected existing unit
    test passed.
- Fix round 1:
  - Both reviewed test files amended; migrations and generated code unchanged.
  - Wrong-index mutation (`UNIQUE(id)` with the production predicates) caused
    the strengthened partial-index subtests to fail, then the canonical
    indexes were restored and catalog-verified on `miltech_ng_test`.
  - `TestMainReportsLoadEnvError` failed against the prior suppressed-error
    behavior, then passed after `TestMain` began failing immediately with the
    original loader error.
  - Final `go test ./tests/user_pmcs -count=1`: passed (`ok`, 0.423s).
  - Scoped re-review by the original reviewer: both Important findings
    ADDRESSED; no new Critical/Important breakage; no out-of-scope
    observations.
- Historical checkpoint HEAD: `7508b29991331835683d798c75d69be6a82ed771`
- Worktree: only this untracked progress ledger remains
- Next task: Foundation Task 2 — shared configuration, domain contracts, and
  HTTP primitives

### Foundation Task 2: Shared configuration, domain contracts, and HTTP primitives

- Status: complete; independent review passed after fix round 1
- Base commit: `7508b29991331835683d798c75d69be6a82ed771`
- Implementer: `/root/foundation_task2_impl` (fresh)
- Reviewer dispatched: `/root/foundation_task2_review` (fresh; historical checkpoint)
- Focused tests:
  - `go test ./api/user_pmcs/shared -count=1`: passed (`ok`, 0.491s).
  - `go test ./bootstrap ./api/route -run '^$'`: passed; bootstrap compiled
    with no tests and route compiled with no matching tests (`ok`, 0.459s).
- Accepted baseline failures: the three failures recorded in Baseline
  verification
- Task commits:
  - `4f309f30da60a5fc472d986bba64574009d97a6d`
    `feat(user-pmcs): add shared API contracts`
  - `96eff4f`
    `fix(user-pmcs): harden shared API primitives`
- Full-suite evidence:
  - Counted `go test -json ./...`: exit 1; 628 passed, 5 failed, 12 skipped.
  - Failures: the accepted missing PMCS SBS route; the accepted Shops
    `user_vehicle_shop_id_fkey`; and three different PMCS SBS integration
    failures (`pmcs sbs equipment not found`, `shop_members_shop_id_fkey`,
    and `fk_pmcs_sbs_inspections_equipment_id`).
  - The unexpected equipment-services failure from an earlier uncounted run
    passed alone (1/1) and passed in the counted rerun.
  - All three unexpected PMCS SBS failures passed together in isolated focused
    execution (3/3).
  - These full-suite failures remain recorded as failures; the isolated
    evidence shows order-sensitive/shared-database noise outside Task 2, not a
    Task 2 regression. No full-green claim.
- Review findings:
  - Important: malformed/overflowing `USER_PMCS_*` integers silently fell
    back to defaults rather than failing startup.
  - Important: missing and malformed mutation preconditions returned generic
    errors rather than preserving `428 precondition_required` versus
    `400 invalid_precondition`.
  - Important: cursor decoders accepted version-only payloads with zero
    timestamp/UUID anchors.
  - Minor (deferred to final whole-branch review): tests do not pin every
    stable error constructor and current config tests construct
    `bootstrap.Env` directly.
  - Minor (deferred to final whole-branch review): cursor negative tests could
    also cover unknown fields, invalid UTF-8, zero anchors, and unsupported
    versions during encoding.
  - Reviewer warnings resolved by controller: authenticated identity sourcing,
    parent ETag enforcement, stale-precondition handling, and final delta
    wrapper composition are explicitly later handler/service tasks; Task 2
    correctly provides shared primitives only.
- Fix round 1:
  - Added actual-environment subprocess coverage for malformed and overflowing
    User-PMCS settings; existing DB-pool integer parsing behavior remains
    unchanged.
  - Missing preconditions now return typed 428 errors and malformed
    preconditions typed 400 errors.
  - Community and subscription cursors now reject/decline encoding zero
    anchors.
  - `go test ./bootstrap ./api/user_pmcs/shared -count=1`: both packages
    passed.
  - `go test ./bootstrap ./api/route -run '^$'`: compile checks passed.
  - Scoped re-review: all three Important findings ADDRESSED; no new
    Critical/Important breakage.
  - Minor (deferred to final whole-branch review): subprocess config tests
    retain unrelated ambient `USER_PMCS_*` variables and could fail for the
    wrong key in a contaminated environment.
- Historical checkpoint HEAD: `96eff4f48c713a9eee34da57ccb5db105cfd4a0b`
- Worktree: only this untracked progress ledger
- Next task: Foundation Task 3 — Unicode normalization, validation, and
  canonical hashing

### Foundation Task 3: Unicode normalization, validation, and canonical hashing

- Status: complete
- Base commit: `96eff4f48c713a9eee34da57ccb5db105cfd4a0b`
- Implementer: `/root/foundation_task3_impl` (fresh)
- Reviewer dispatched: `/root/foundation_task3_review` (fresh; historical checkpoint)
- Focused/race/dependency tests:
  - Required RED failed only on the intentionally absent Task 3 APIs.
  - A second focused RED caught draft blank-model handling before commit.
  - Required focused command passed (`ok`, 0.530s).
  - Complete shared-package run passed: 172 test/subtest PASS events,
    0 FAIL events, and 1 package PASS event.
  - Race run passed (`ok`, 1.579s).
  - `go vet ./api/user_pmcs/shared` passed.
  - Repository-wide compile-only check passed with zero package failures.
  - Module resolved as `github.com/clipperhouse/uax29/v2 v2.4.0`; package
    resolved as `github.com/clipperhouse/uax29/v2/graphemes`.
- Accepted baseline/environment failures: recorded above; exact Task 3 suite
  was not run because the task brief required the repository-wide compile-only
  check, not a full-green claim
- Review findings:
  - Important: preparation re-enforced the raw HTTP body ceiling by
    `json.Marshal`-encoding the decoded value. HTML-sensitive authored text can
    expand during re-encoding and falsely produce `413 content_too_large`
    after the actual wire body already passed `DecodeStrictJSON`.
  - Minor (deferred to final whole-branch review): canonical hash tests have no
    fixed independently derived golden vector.
  - Minor (deferred to final whole-branch review): the Unicode fixture lacks a
    Unicode-16-specific differentiator, although the pinned dependency itself
    declares Unicode 16 data.
- Fix round 1:
  - Regression RED reproduced the false post-decode
    `413 content_too_large`.
  - Preparation no longer infers request size by re-encoding; the authoritative
    raw-byte ceiling remains at `DecodeStrictJSON`.
  - Focused regression, required selection, full shared package, race, vet,
    dependency proof, and repository-wide compile-only check all passed.
  - Fix commit: `1fed1689c8e727a64751ecc31d4996b9aea5277a`
    `fix(user-pmcs): enforce actual mutation body size`
  - Scoped re-review: Important finding ADDRESSED; no new
    Critical/Important breakage
- Task commits:
  - `0c6adcbae27f4c82b93ea158125387832f407633`
    `feat(user-pmcs): validate and hash revision trees`
  - `1fed1689c8e727a64751ecc31d4996b9aea5277a`
    `fix(user-pmcs): enforce actual mutation body size`
- Historical checkpoint HEAD: `1fed1689c8e727a64751ecc31d4996b9aea5277a`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Foundation Task 4 — transactional repository and change-version
  primitives

### Foundation Task 4: Transaction retry, account locking, and batched tree persistence

- Status: complete
- Base commit: `1fed1689c8e727a64751ecc31d4996b9aea5277a`
- Implementer: `/root/foundation_task4_impl` (fresh)
- Reviewer dispatched: `/root/foundation_task4_review` (fresh; historical checkpoint)
- Required documentation research: complete
- Required sequential analysis: complete
- Database skills: Go, database-optimizer, and Supabase Postgres guidance
  applied to the task brief
- Focused/race/integration/plan results:
  - Persistence unit tests passed (`0.455s`).
  - Real `miltech_ng_test` tree integration passed (`3.140s`).
  - Race passed (`1.495s`); vet passed with no output.
  - Repository-wide compile-only check passed.
  - Ceiling-sized EXPLAIN fixture plus nine decoys completed inside a rolled
    back transaction; exact plans are in the task report.
- Accepted baseline/environment failures: recorded above; Task 4 result pending
- Review findings:
  - No Critical, Important, or Minor findings.
  - Non-blocking coverage gaps deferred to the whole-branch review: ownership
    integration does not hit every notice/step branch; ceiling writes and
    deletion batches above 1,000 IDs are proven by batching logic/plan
    fixtures rather than direct end-to-end mutations; synthetic row
    iteration/close failures are not injected.
- Task commits:
  - `f760093ebf43bdafc840117de01604a22076ef38`
    `feat(user-pmcs): add transactional tree persistence`
- Historical checkpoint HEAD: `f760093ebf43bdafc840117de01604a22076ef38`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Foundation completion gate

### Foundation completion gate

- Status: complete
- `go test ./api/user_pmcs/shared ./api/user_pmcs/persistence -count=1`:
  187 passed in 2 packages
- Sandboxed integration attempt: failed exactly because network access to
  `192.168.20.70:5432` was denied; not counted as a skip or product failure
- Approved rerun
  `go test ./tests/user_pmcs -run 'Test(UserPmcsSchema|Tree)' -count=1`:
  26 passed in 1 package
- Approved `go test ./... -run '^$'`: passed; no tests selected
- HEAD: `f760093ebf43bdafc840117de01604a22076ef38`
- Worktree: only this untracked progress ledger
- Next plan: Owned sync, Task 5

### Owned Sync Task 5: Owned create, current read, and draft lifecycle

- Status: complete
- Base commit: `f760093ebf43bdafc840117de01604a22076ef38`
- Implementer: `/root/owned_task5_impl` (fresh)
- Reviewer dispatched: `/root/owned_task5_review` (fresh; historical checkpoint)
- Required documentation research and sequential analysis: complete
- Focused/integration/race/plan results:
  - Owned unit package passed (`0.476s`).
  - Real `miltech_ng_test` integration passed (`5.926s`).
  - Race passed (`1.507s`); vet passed.
  - Repository-wide compile-only gate passed.
  - Active-owned EXPLAIN used the approved owner-delta index via bitmap scan;
    `deleted_at` remained a heap filter; execution `0.149 ms`; fixtures rolled
    back.
- Accepted baseline/environment failures: recorded above; Task 5 result pending
- Review findings:
  - Important: current GET reads root/revisions/trees/community through
    separate `READ COMMITTED` statements on `*sql.DB`, so a concurrent draft
    commit can produce a torn aggregate with ETag/version N and tree N+1.
  - Important: global content UUID ownership uses check-then-insert without a
    common serialization point across different owners/checklists; concurrent
    same-table reuse can leak SQL errors and cross-table reuse can both commit.
  - Minor (deferred to whole-branch review): GET uses the strict single-strong
    mutation parser for `If-None-Match`, rejecting wildcard, lists, and weak
    validators allowed by RFC 9110.
- Fix round 1:
  - Deterministic temporary-trigger RED proved GET could return an old
    root/draft with children missing from the concurrently replaced tree.
  - Concurrent UUID RED proved same-table reuse leaked a raw primary-key error
    and cross-table reuse allowed both owners to commit.
  - GET now uses one read-only `REPEATABLE READ` transaction.
  - Create/PutDraft acquire deterministic sorted transaction advisory locks
    for submitted revision/content UUIDs before ownership preflight.
  - UUID race stress passed 20 repetitions (`15.334s`); torn-read barrier
    stress passed 5 repetitions (`17.414s`) with cleanup each run.
  - Owned unit (`0.484s`), integration (`8.578s`), race (`1.509s`), vet, and
    repository compile-only checks passed; fresh post-commit focused checks
    also passed.
  - Fix commit: `26122f505276b7c7e3069312885a8f6a80a659e6`
    `fix(user-pmcs): serialize owned tree snapshots`
  - Scoped re-review: both original Important findings ADDRESSED
  - New Important: the fix acquires one transaction advisory lock per UUID,
    up to 16,101 locks for a valid maximum tree, risking exhaustion of the
    finite PostgreSQL shared lock pool
  - New Minor: the temporary-trigger test registers cleanup after several
    setup operations and can leak resources if setup fails
- Fix round 2:
  - Max-tree RED proved 16,101 advisory locks exceeded the required bound.
  - UUIDs now map into a fixed 32-stripe User-PMCS namespace; stripes are
    sorted/deduplicated and collisions only add serialization.
  - Barrier cleanup now registers immediately and tracks acquired resources.
  - Collision stress x20 (`15.856s`) and snapshot stress x5 (`8.043s`) passed.
  - Owned (`0.393s`), integration (`8.158s`), race (`1.513s`), vet, and
    repository compile-only checks passed; post-commit focused checks passed.
  - Test catalog after stress: zero temporary triggers and zero functions.
  - Fix commit: `d0f94dd907ccc46fd2b9c3395ccd4fb77bd5c07a`
    `fix(user-pmcs): bound UUID serialization locks`
  - Scoped re-review: bounded locks and barrier cleanup ADDRESSED; no new
    Critical/Important breakage
- Task commits:
  - `06e517308b7684d383b56586106a235723e84627`
    `feat(user-pmcs): sync owned checklist drafts`
  - `26122f505276b7c7e3069312885a8f6a80a659e6`
    `fix(user-pmcs): serialize owned tree snapshots`
  - `d0f94dd907ccc46fd2b9c3395ccd4fb77bd5c07a`
    `fix(user-pmcs): bound UUID serialization locks`
- Historical checkpoint HEAD: `d0f94dd907ccc46fd2b9c3395ccd4fb77bd5c07a`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Owned Sync Task 6 — publication, immutable history, and offline
  replay

### Owned Sync Task 6: Publication, immutable history, and offline replay

- Status: complete
- Base commit: `d0f94dd907ccc46fd2b9c3395ccd4fb77bd5c07a`
- Implementer: `/root/owned_task6_impl` (fresh)
- Reviewer dispatched: `/root/owned_task6_review` (fresh; historical checkpoint)
- Required documentation research and sequential analysis: complete
- Focused/integration/concurrency/race results:
  - Post-commit owned package passed (`0.412s`).
  - Publication/history/offline integration passed (`7.120s`).
  - Complete user-PMCS integration package passed (`17.469s`).
  - Concurrent exact-next stress passed 20/20 (`16.804s`).
  - Race and vet passed; repository compile-only gate passed.
- Accepted baseline/environment failures: recorded above; Task 6 result pending
- Review findings:
  - Important: parent ETag mismatch is rejected before exact published/
    superseded retry proof, so a lost-response retry with the original
    precondition returns 412 instead of idempotent success.
  - Important: historical JSON exposes mutable lifecycle `state` under a
    content-only immutable ETag and one-year immutable caching.
  - Important: historical ETags are recomputed with the current canonicalizer
    instead of using the stored immutable 32-byte `content_hash`.
  - Coverage gaps deferred to whole-branch review: interruption test cancels
    before publication begins; no HTTP-level complete historical DTO/privacy
    assertion; known general `If-None-Match` limitation.
- Fix round 1:
  - Immediate and post-supersession original-ETag retry REDs reproduced stale
    rejection; historical RED exposed mutable state and missing stored-hash
    result boundary.
  - Exact stale-precondition retries now pass only stored metadata/hash/number
    proof and consume no versions.
  - Owned historical DTO omits mutable lifecycle state and remains
    byte-identical after supersession.
  - ETag uses validated stored 32-byte content hash from the same
    repeatable-read snapshot.
  - Lost-response, supersession, and concurrency stress each passed 20/20
    (`57.684s` total); full owned/user-PMCS, race, vet, and compile checks
    passed.
  - Fix commit: `c02a887a1bf3c0c751f9785395630910ebd96ade`
    `fix(user-pmcs): preserve immutable revision retries`
  - Scoped re-review: all three Important findings ADDRESSED; no new
    Critical/Important breakage
- Task commits:
  - `a5b30a8bb7dec2a8bf8511d14a1d1d9f494c928d`
    `feat(user-pmcs): publish immutable checklist revisions`
  - `c02a887a1bf3c0c751f9785395630910ebd96ade`
    `fix(user-pmcs): preserve immutable revision retries`
- Historical checkpoint HEAD: `c02a887a1bf3c0c751f9785395630910ebd96ade`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Owned Sync Task 7 — checklist tombstones and deletion retention

### Owned Sync Task 7: Checklist tombstones and deletion retention

- Status: complete
- Base commit: `c02a887a1bf3c0c751f9785395630910ebd96ade`
- Implementer: `/root/owned_task7_impl` (fresh)
- Reviewer dispatched: `/root/owned_task7_review` (fresh; historical checkpoint)
- Documentation research and sequential analysis: complete
- Tests: deletion 7 passed; full user-PMCS 64 passed; deletion stress 35
  passed; concurrency/race/vet/repository compile-only passed
- Review:
  - Important: active subscription pins are selected without row locks, so a
    concurrent unsubscribe can clear a pin after discovery and leave an
    orphaned release/tree with no v1 garbage collector.
  - Minor (deferred): direct delete-route negative coverage is incomplete.
  - Minor (deferred): private tombstone test does not explicitly assert
    `owner_uid` preservation.
- Fix round 1:
  - Deterministic two-connection barrier RED proved deletion did not wait for
    a concurrent unsubscribe row update.
  - Pin discovery now uses deterministic `ORDER BY subscriber_uid FOR UPDATE`;
    READ COMMITTED re-evaluation deletes the newly unpinned release/revision.
  - Focused 8, stress 40, owned 32, full user-PMCS 65, and race suites passed;
    vet and compile gates passed.
  - Fix commit: `e51d72ebd992dd9ccc41ccc88138f9bac1e7b2fd`
    `fix(user-pmcs): lock active release pins`
  - Scoped re-review: Important finding ADDRESSED; no new
    Critical/Important breakage
- Commit: `a2c81d037ec74b500b6645d8015421fc675e7edc`
  `feat(user-pmcs): retain permanent checklist tombstones`
- Fix commit: `e51d72ebd992dd9ccc41ccc88138f9bac1e7b2fd`
  `fix(user-pmcs): lock active release pins`
- Historical checkpoint HEAD: `e51d72ebd992dd9ccc41ccc88138f9bac1e7b2fd`
- Historical checkpoint HEAD: `a2c81d037ec74b500b6645d8015421fc675e7edc`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Owned Sync Task 8 — embedded owner-plus-subscription account delta

### Owned Sync Task 8: Embedded owner-plus-subscription account delta

- Status: complete
- Base commit: `e51d72ebd992dd9ccc41ccc88138f9bac1e7b2fd`
- Implementer: `/root/owned_task8_impl` (fresh)
- Reviewer dispatched: `/root/owned_task8_review` (fresh; historical checkpoint)
- Documentation research and sequential analysis: complete
- Tests:
  - Post-commit sync passed (`0.638s`); AccountDelta integration passed
    (`1.972s`).
  - Complete user-PMCS integration (`25.266s`), snapshot barrier x10
    (`3.297s`), focused integration race (`6.686s`), all User-PMCS race, vet,
    and repository compile-only passed.
  - Full repository suite was not green: accepted route baseline plus six
    unrelated PMCS SBS shared-DB/order failures; all six passed together in
    isolation (`6.331s`). A transient equipment-services FK failure on the
    first console run did not repeat.
- Review:
  - Important: byte accounting sums only independently marshaled changes and
    omits commas, array/delta framing, and the standard envelope, so a normal
    multi-entry response can exceed the configured uncompressed ceiling.
  - Minor (included in fix round): requested owner draft/publication IDs that
    are absent from the batched tree result silently become zero-value
    revisions instead of an error.
  - Deferred gaps: exact query-count assertion, injected SQL/transaction
    failures, and actual gzipped error through registered middleware.
- Fix round 1:
  - Exact RED: actual final StandardResponse was 3,963 bytes against a 3,962
    configured limit.
  - Repository now commits the snapshot then sizes each candidate with the
    same exact envelope constructor used by the handler, including final
    metadata, separators, and `has_more`; first-change exception remains.
  - Missing requested draft/publication trees now error before commit.
  - Focused sync (`0.412s`), integration (`2.309s`), full user-PMCS
    (`25.355s`), snapshot x10 (`3.033s`), race, vet, and compile passed.
  - Fix commit: `c5e763a66d4d1920daa98f5c44366b212885655f`
    `fix(user-pmcs): enforce exact delta response size`
  - Scoped re-review: Important and Minor findings ADDRESSED; no new
    Critical/Important breakage
- Commit: `e1e33892c875b68f50fa363b6776781081590704`
  `feat(user-pmcs): add embedded account delta sync`
- Fix commit: `c5e763a66d4d1920daa98f5c44366b212885655f`
  `fix(user-pmcs): enforce exact delta response size`
- Historical checkpoint HEAD: `c5e763a66d4d1920daa98f5c44366b212885655f`
- Historical checkpoint HEAD: `e1e33892c875b68f50fa363b6776781081590704`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Owned-sync completion gate

### Owned-sync completion gate

- Status: complete
- User-PMCS API packages: 237 passed in 4 packages
- Sandboxed integration attempt: failed exactly on denied connection to
  `192.168.20.70:5432`; not converted to a skip
- Approved integration rerun: 41 passed in 1 package
- User-PMCS API race gate: 237 passed in 4 packages
- Repository compile/TestMain gate: passed; no tests selected
- HEAD: `c5e763a66d4d1920daa98f5c44366b212885655f`
- Worktree: only this untracked progress ledger
- Next plan: Community and subscriptions, Task 9

### Community Task 9: Revision-specific release, retirement, and retained pins

- Status: complete
- Base commit: `c5e763a66d4d1920daa98f5c44366b212885655f`
- Implementer: `/root/community_task9_impl` (fresh)
- Reviewer: `/root/community_task9_review` (fresh)
- Documentation research and sequential analysis: complete
- Tests:
  - Post-commit community (`0.458s`) and release/retire integration
    (`13.106s`) passed.
  - Full user-PMCS (`39.428s`), concurrency 20/20 (`34.159s`), race, vet, and
    compile-only passed.
  - Full repository suite failed on accepted route baseline plus parallel
    shared-DB contamination; equipment-services, PMCS SBS progress, and Shops
    packages passed sequentially.
- Review: PASS; no Critical, Important, or Minor findings
- Commit: `29438052f2de1f3fe53e37be2af427d48df25c60`
  `feat(user-pmcs): release community checklist revisions`
- Historical checkpoint HEAD: `29438052f2de1f3fe53e37be2af427d48df25c60`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Community Task 10 — anonymous recent browse and public detail

### Community Task 10: Anonymous recent browse and current public detail

- Status: complete
- Base commit: `29438052f2de1f3fe53e37be2af427d48df25c60`
- Implementer: `/root/community_task10_impl` (fresh)
- Reviewer: `/root/community_task10_review` (fresh)
- Documentation research and sequential analysis: complete
- Tests:
  - Final focused community (`0.452s`) and integration (`12.845s`) passed.
  - Relevant full suites, keyset stress x10, race, vet, compile, and filtered/
    unfiltered EXPLAIN index proofs passed.
  - Sequential full suite retained only accepted missing PMCS-SBS route;
    parallel suite had additional shared-DB interference while Task 10 passed.
- Review:
  - Important: repeated `If-None-Match` field lines are not combined; a
    matching validator on a later line is ignored and returns 200 instead of
    bodyless 304.
  - Important verification gap: the exact filtered browse EXPLAIN used the
    model primary key; the reverse-model index was proven only by a different
    standalone diagnostic query.
  - Plan-gate resolution: on a rolled-back 20,000-row representative fixture
    with 5% model selectivity, the exact query rationally used the recent index
    plus unique `(revision_id, normalized_text)` PK to preserve ordered LIMIT;
    forcing the reverse index would add sorting. No query/schema degradation
    is authorized; the contradiction and diagnostic evidence remain recorded.
- Fix round 1:
  - RED proved later matching field lines returned 200 and split wildcard/tag
    returned 304.
  - All header values are now combined in order; later weak/strong matches
    return bodyless 304 and mixed wildcard lists return 400.
  - Header stress 500/500, focused community/integration, race, vet, and
    compile passed.
  - Fix commit: `00d68699f94913502e2a91b053a7b1361ad961bf`
    `fix(user-pmcs): combine public conditional headers`
  - Scoped re-review: header finding ADDRESSED; accepted planner contradiction
    verified honest; no new Critical/Important breakage
- Commit: `5afd82a58c1bc91a7720fa0df6343662d23e200f`
  `feat(user-pmcs): browse active community checklists`
- Fix commit: `00d68699f94913502e2a91b053a7b1361ad961bf`
  `fix(user-pmcs): combine public conditional headers`
- Historical checkpoint HEAD: `00d68699f94913502e2a91b053a7b1361ad961bf`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Community Task 11 — install, unsubscribe, resubscribe, and pinned
  release reads

### Community Task 11: Install, unsubscribe, resubscribe, and pinned release reads

- Status: in progress
- Base commit: `00d68699f94913502e2a91b053a7b1361ad961bf`
- Implementer: `/root/community_task11_impl` (fresh)
- Reviewer dispatched: `/root/community_task11_review` (fresh; historical checkpoint)
- Documentation research and sequential analysis: complete
- Implementation commit: `72c638fb61eb4844a41b19688c2772b0b2d3e4cb`
  `feat(user-pmcs): add linked checklist subscriptions`
- Tests:
  - Focused package and integration checks passed.
  - Full user-PMCS, focused race, vet, and compile-only checks passed.
  - Initial GREEN attempt encountered exact sandbox denial for the normal Go
    build cache; approved reruns passed.
- Implementer concern: the active-ceiling branch is covered with an injected
  limit of 1; production remains the approved default of 500.
- Review:
  - Important: install locks the community source before the checklist root,
    conflicting with release/retire lock order and permitting deadlocks.
  - Important: an identical active install ignores a supplied stale
    `If-Match`.
  - Important: pinned `If-None-Match` uses a strong single-tag mutation parser
    instead of weak comparison across wildcard/list/weak validators.
  - Important contract contradiction: the pinned response includes live source
    status/current creator display name while the plan mandates
    `private, max-age=31536000, immutable`; the representation can change at
    the same URL while clients are told not to revalidate for one year.
  - Verification gap: the retention test nulls only `users.username`; it must
    construct the retained tombstoned/null-owner state and delete the owner row.
  - Minor deferred: malformed `revision_id` reports `checklist_id` in error
    details.
- Fix round 1: original implementer dispatched for the non-conflicting
  Important findings and retained-owner-deletion fixture; immutable-cache
  contract awaits user direction.
- Fix commit: `9bd4bb67324af0d2925fb73ee590cef13a544dbf`
  `fix(user-pmcs): harden linked checklist subscriptions`
- Fix verification:
  - RED reproduced weak/list/wildcard/repeated-header conditional failures and
    stale active `If-Match` acceptance.
  - Amended focused integration passed in `18.078s`.
  - Subscription/community/shared packages, full user-PMCS, focused race,
    vet, and compile-only checks passed.
- Scoped re-review:
  - Root/source lock order: ADDRESSED.
  - Active create retry and matching/stale `If-Match`: ADDRESSED.
  - Full conditional GET matching semantics: ADDRESSED.
  - Retained tombstoned/null-owner/deleted-owner fixture: ADDRESSED.
  - New Important: malformed `If-None-Match` errors inherit the release ETag
    and one-year immutable cache headers.
- Fix round 2: original implementer dispatched for malformed-conditional error
  cache headers.
- Fix commit: `e8e4b2e80d210446f1ef44e1a834eae844f919e7`
  `fix(user-pmcs): avoid caching invalid subscription requests`
- Fix verification: handler RED/GREEN, focused Task 11, vet, and compile checks
  passed.
- Fix round 2 scoped re-review: malformed error cache-header finding
  ADDRESSED; no new Critical/Important breakage
- User resolution: preserve live source status/current creator attribution;
  replace the pinned response policy with `private, no-cache` and derive its
  strong ETag from the complete representation.
- Fix round 3: original implementer dispatched for the authorized contract
  correction.
- Fix commit: `6745247bc5cda3fbb2bab49881bc82307fe7667d`
  `fix(user-pmcs): revalidate pinned release metadata`
- Fix verification:
  - RED proved retirement retained the old revision-only ETag.
  - Focused Task 11 passed (`0.646s`, `17.830s`).
  - Focused race passed (`1.877s`, `19.080s`); vet and compile-only passed.
- Fix round 3 scoped re-review: all five authorized cache/ETag requirements
  ADDRESSED; no new Critical/Important breakage
- Status: complete
- Historical checkpoint HEAD: `6745247bc5cda3fbb2bab49881bc82307fe7667d`
- Worktree: only this untracked progress ledger
- Worktree after completion: only this untracked progress ledger
- Next task: Community Task 12 — paginated update discovery and explicit
  acceptance

### Community Task 12: Paginated update discovery and explicit acceptance

- Status: in progress
- Base commit: `6745247bc5cda3fbb2bab49881bc82307fe7667d`
- Implementer: `/root/community_task12_impl` (fresh)
- Reviewer dispatched: `/root/community_task12_review` (fresh; historical checkpoint)
- Documentation research and sequential analysis: complete
- Implementation commit: `cc7ec08a955ba1aa3b3cbd0430002aef85f2975d`
  `feat(user-pmcs): discover and accept subscription updates`
- Tests:
  - Focused package/integration passed (`0.436s`, `5.246s`).
  - Community/subscription completion commands passed, including `44.268s`
    integration and focused race (`1.488s`, `1.887s`).
  - User-PMCS vet and full repository compile-only passed.
- EXPLAIN:
  - Rollback-only 500-active-subscription fixture.
  - Exact query used `user_pmcs_subscriptions_active_update_idx`.
  - 51 rows; 537 total shared-buffer hits; planning `5.716ms`; execution
    `5.757ms`.
  - Source join used a sequential scan under the local fixture distribution;
    no query/schema manipulation was made.
- Historical checkpoint HEAD: `cc7ec08a955ba1aa3b3cbd0430002aef85f2975d`
- Worktree: only this untracked progress ledger
- Review:
  - Important: missing active/no-update integration coverage and stale
    subscription ETag rejection with unchanged pin/subscription/account
    versions.
  - Verification gap: the EXPLAIN helper and full raw plan were removed,
    preventing independent reproduction from the task diff.
  - Minor deferred: pagination test does not assert terminal page
    `has_more=false` and nil cursor.
- Fix round 1: original implementer dispatched for the Important coverage and
  reproducible EXPLAIN evidence.
- Fix commit: `cf50a8c870664620da27e931ca1e18018cef8323`
  `test(user-pmcs): cover subscription update acceptance`
- Fix verification:
  - Active/no-update and stale-ETag invariants passed in `3.538s`.
  - Committed 500-row EXPLAIN passed in `30.474s`; index used with 544
    shared-buffer hits, `4.858ms` planning, and `4.581ms` execution.
  - Task 12 integration (`36.758s`), focused race, vet, compile-only, and diff
    checks passed.
- Scoped re-review:
  - Active/no-update coverage: ADDRESSED.
  - Stale subscription ETag invariants: ADDRESSED.
  - Success/idempotency regression coverage: ADDRESSED.
  - Reproducible EXPLAIN evidence: ADDRESSED.
  - No new Critical/Important breakage.
- Status: complete
- Historical checkpoint HEAD: `cf50a8c870664620da27e931ca1e18018cef8323`
- Worktree after completion: only this untracked progress ledger
- Next step: community/subscription completion gate

### Community/subscription completion gate

- Status: complete
- Community/subscription packages: 42 passed in 2 packages
- Sandboxed integration attempt: failed exactly on denied connection to
  `192.168.20.70:5432`; not converted to a skip
- Approved integration rerun: 22 passed in 1 package
- Community/subscription race gate: 42 passed in 2 packages
- Sandboxed compile/TestMain attempt: failed exactly on the same denied test
  database connection; not converted to a skip
- Approved full repository compile/TestMain rerun: passed; no tests selected
- HEAD: `cf50a8c870664620da27e931ca1e18018cef8323`
- Worktree: only this untracked progress ledger
- Next plan: account lifecycle and hardening, Task 13

### Hardening Task 13: Authenticated account deletion and retained public content

- Status: in progress
- Base commit: `cf50a8c870664620da27e931ca1e18018cef8323`
- Implementer: `/root/hardening_task13_impl` (fresh)
- Reviewer dispatched: `/root/hardening_task13_review` (fresh; historical checkpoint)
- Documentation research and sequential analysis: complete
- Implementation commit: `91a5d87fe633cb2c78f98548b4cd374390b1455f`
  `fix(accounts): delete PMCS data from authenticated identity`
- Tests:
  - Exact focused authority/persistence/integration gates passed.
  - 189 scoped tests passed across 8 packages; integration took `114.369s`.
  - Focused race, scoped vet, full repository compile-only, and diff checks
    passed.
  - Initial RED attempt recorded exact sandbox Go-cache denial; approved RED
    rerun failed on expected missing cleaner/context contracts.
- Historical checkpoint HEAD: `91a5d87fe633cb2c78f98548b4cd374390b1455f`
- Worktree: only this untracked progress ledger
- Review:
  - Important: active external subscription pins are counted but not locked,
    so concurrent unsubscribe can destabilize retention classification.
  - Important: retained redownload checks only revision ID with an empty tree;
    it does not prove the complete aggregate survives, multiple active pins,
    or external tombstone exclusion.
  - Minor deferred: `NewRepository` permits a nil cleaner and would panic.
  - Minor deferred: restrictive-FK test does not isolate release-to-revision
    ordering because the source pointer is still present.
- Fix round 1: original implementer dispatched for both Important findings.
- Fix round 1 RED:
  - Full non-empty aggregate retention passes.
  - Unsubscribe-first serialization passes.
  - Account-deletion-first followed by final unsubscribe leaves one zero-pin
    retained source/root.
- Plan contradiction: the reviewer requires convergence after the final
  unsubscribe, but the approved master plan restricts release reclamation to
  checklist or account deletion and excludes unsubscribe cleanup.
- Implementer interrupted before any unsubscribe cleanup change; uncommitted
  work contains only non-conflicting account-cleanup locking/tests.
- User resolution: authorize narrow final-pin unsubscribe cleanup only for an
  already owner-null, tombstoned checklist with a retired source; do not
  garbage-collect active/owned checklist history.
- Fix round 1: original implementer resumed.
- Fix commit: `9ed053de38d08e5381f9fb5e105c7bc27a3fb413`
  `fix(accounts): serialize PMCS pin cleanup`
- Fix verification:
  - Full-tree/multi-pin/tombstone and negative eligibility cases passed.
  - Both forced serialization orders passed 20× in `22.243s`.
  - Task 11 subscription regression, focused race, vet, compile-only, and
    fresh post-commit focused checks passed.
  - First full scoped run hit existing planner-sensitive index-choice noise;
    isolated EXPLAIN and full rerun passed, with integration `105.686s`.
- Scoped re-review:
  - Both original Important findings and all six authorized correctness gates
    are ADDRESSED.
  - New Important: ordinary unsubscribe unconditionally locks all active and
    historical subscription rows for a source, causing unbounded lock
    amplification on popular sources.
- Fix round 2: original implementer dispatched to keep ordinary unsubscribe
  target-only and limit candidate/account cleanup unions to required rows.
- Fix commit: `b0b24e4cbf79e927e8c83153381374e074984e76`
  `perf(user-pmcs): narrow subscription pin locks`
- Fix verification:
  - Three strict RED/GREEN historical-tombstone lock-scope regressions passed.
  - Five contention cases passed 100/100; Task 11 passed 12/12.
  - Full scoped suite passed 389 tests; focused race, vet, compile-only, diff,
    and fresh post-commit checks passed.
  - Initial sandboxed compile/TestMain recorded exact test-database denial;
    approved identical rerun passed.
- Fix round 2 scoped re-review: lock-amplification finding ADDRESSED; no new
  Critical/Important breakage
- Status: complete
- Historical checkpoint HEAD: `b0b24e4cbf79e927e8c83153381374e074984e76`
- Worktree after completion: only this untracked progress ledger
- Next task: Hardening Task 14 — top-level routes,
  compression, rate limits, and safe observability

### Hardening Task 14: Top-level routes, compression, rate limits, and safe observability

- Status: in progress
- Base commit: `b0b24e4cbf79e927e8c83153381374e074984e76`
- Documentation research and sequential analysis: complete
- Task brief:
  `.superpowers/sdd/2026-07-29-user-pmcs-hardening-rollout/task-14-brief.md`
- Implementer: `/root/hardening_task14_impl` (fresh; in progress)
- Worktree before dispatch: only this untracked progress ledger
- Implementation commit: `d8fab627bdb3486c106337c816561d131496e205`
  `feat(user-pmcs): wire routes and operational safeguards`
- RED:
  - Route registration failed on missing authenticated user-PMCS sync route.
  - Root package failed to compile on missing limiter, configuration, and
    observation contracts.
  - Authored-query redaction failed because the legacy logger emitted the
    raw query string.
- GREEN:
  - Task 14 root tests passed.
  - All 17 route pairs and authored-query redaction regressions passed.
  - Broader `api/user_pmcs/...` plus bootstrap passed.
  - Root user-PMCS race, scoped vet, and compile-only checks passed.
- Accepted baseline failure: the exact combined
  `go test ./api/user_pmcs ./api/route -count=1` command fails only the
  pre-existing stale PMCS-SBS assertion for
  `GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`; this was not
  converted to a skip or reported green.
- Worktree after implementation: only this untracked progress ledger
- Reviewer dispatched: `/root/hardening_task14_review` (fresh; historical checkpoint)
- Review verdicts:
  - Spec compliance: FAIL
  - Task quality: CHANGES REQUIRED
- Important: production `gin.Default` logs the raw community query string, so
  authored model text can still enter the access log; the existing test uses
  `gin.New` and does not reproduce production.
- Important: live observations populate only operation, status, overall
  duration, and byte counts; API code, database/encode duration, retry count,
  and node count remain unpopulated.
- Fix round 1: original implementer dispatched for both Important findings.
- Fix commit: `36dfeb891ead282e8ed5c8cfbcbeea9b96da00f7`
  `fix(user-pmcs): sanitize logs and collect metrics`
- Fix verification:
  - Production-equivalent access-log redaction passed while unrelated route
    query logging remained intact.
  - End-to-end API code/encode, retrying transaction DB/retry, and full-tree
    node-count observations passed.
  - Task 14 focused, broader user-PMCS/bootstrap, race, vet, and compile-only
    gates passed.
  - The exact combined command retains only the accepted stale PMCS-SBS route
    assertion.
- Fix round 1 scoped re-review:
  - Original production raw-query leak: ADDRESSED.
  - Operational measurement finding: NOT ADDRESSED because sync delta
    post-commit response sizing is included in DB duration and excluded from
    encode duration.
  - New Important: invalid authored dynamic path segments can still enter
    user-PMCS access logs before UUID validation.
- Fix round 2: original implementer dispatched for the two open Important
  findings.
- Fix commit: `2a6690a0744aa9a84d5c7631009f5895c768277c`
  `fix(user-pmcs): correct observability attribution`
- Fix verification:
  - Deterministic sync repository test proves transaction time ends at
    commit/rollback and post-commit sizing contributes to encode duration.
  - Production-equivalent access log uses Gin route templates or fixed
    user-PMCS family labels, excluding invalid authored path segments while
    preserving method, status, unrelated logging, and recovery.
  - Focused/broader user-PMCS/bootstrap, root and sync race, vet, and
    compile-only gates passed.
  - Exact combined route command retains only the accepted stale PMCS-SBS
    assertion.
- Fix round 2 scoped re-review:
  - Sync DB/encode duration attribution: ADDRESSED.
  - Authored invalid dynamic path logging: ADDRESSED.
  - No new Critical/Important breakage.
- Status: complete
- Historical checkpoint HEAD: `2a6690a0744aa9a84d5c7631009f5895c768277c`
- Worktree after completion: only this untracked progress ledger
- Next task: Hardening Task 15 — focused authorization, concurrency, and
  query-plan test matrices

### Hardening Task 15: Real-Postgres concurrency, authorization, and performance gates

- Status: in progress
- Base commit: `2a6690a0744aa9a84d5c7631009f5895c768277c`
- Documentation research, Go/database/Postgres skill review, and sequential
  analysis: complete
- Task brief:
  `.superpowers/sdd/2026-07-29-user-pmcs-hardening-rollout/task-15-brief.md`
- Database preflight: controller `TEST_DATABASE_URL` was unset; direct `psql`
  fell back to `/tmp/.s.PGSQL.5432` and failed with “No such file or
  directory.” The repository test loader must resolve the environment and
  assert `current_database() = miltech_ng_test` before mutating fixtures.
- Implementer: `/root/hardening_task15_impl` (fresh; in progress)
- Database target proof: repository loader resolved
  `current_database() = miltech_ng_test`.
- Environment failure: initial focused rerun was denied access to
  `/Users/swisscheese/Library/Caches/go-build`; the identical approved rerun
  passed.
- Worktree before dispatch: only this untracked progress ledger
- Concurrency group: 8 tests characterized PASS in `6.712s`.
- Blocker: the real
  `PUT /auth/user-pmcs/subscriptions/{unknown}/installed-releases/{revision}`
  route returns `409 invalid_transition` with “community checklist is not
  available for update.” Approved specification §15.3 defines absent or
  intentionally hidden resources as `404 resource_not_found`, and Task 15
  explicitly requires safe-404 coverage.
- Cause: `AcceptUpdate` checks the missing/inactive community source before
  checking whether the authenticated user has a subscription.
- The Task 15 plan is tests-only, so changing production behavior requires
  explicit authority. The implementer was interrupted before any production
  change; its uncommitted helper/concurrency/HTTP test work is preserved.
- Historical checkpoint HEAD: `2a6690a0744aa9a84d5c7631009f5895c768277c`
- Current worktree:
  - modified `tests/user_pmcs/helpers_test.go`
  - untracked `tests/user_pmcs/concurrency_test.go`
  - untracked `tests/user_pmcs/http_contract_test.go`
  - this untracked progress ledger
- User resolution: authorize a narrow production fix that preserves existing
  lock order, returns `404 resource_not_found` for a missing/deleted
  subscription before evaluating source transition availability, and retains
  `409 invalid_transition` for an existing subscription whose source cannot
  transition.
- Commit rule: make the production correction as a separate narrow fix commit,
  then make Task 15’s specified tests-only commit.
- Original implementer resumed.
- Authorized fix RED: HTTP and repository tests reproduced `409` versus
  required `404`.
- Authorized fix GREEN: three focused subcases passed in `3.193s`; Task 11
  passed in `17.204s`; Task 12 passed in `37.794s`; API race, vet, and compile
  passed.
- Authorized production fix commit: `077a9f1`
  `fix(user-pmcs): hide missing update subscriptions`
- HTTP/auth group: GREEN in `3.876s`, covering 15 authenticated routes × two
  auth failures, 13 unknown/public cases, 12 mutation failures, cross-owner
  hiding, owner-subscribe rejection, public visibility/publication invariants,
  and current creator-name changes.
- Maximum/performance group:
  - Exact 100-section, 2,000-item, 4,000-notice, 10,000-step tree.
  - `8,388,608`-byte body accepted; `8,388,609` rejected.
  - Eight subtests passed in `25.573s`.
  - 25-root delta was `19,054,769` bytes with all aggregates intact.
  - Fixed tree loader query count: 7; 500-update discovery query count: 1.
- Query plans: all eight representative EXPLAIN gates use approved indexes
  after enlarging selective fixtures.
- Expanded cross-owner/tombstone matrix: GREEN in `4.560s`.
- Final exact gates:
  - Focused: passed in `11.055s`.
  - Performance: passed in `34.303s`.
  - Race across all `api/user_pmcs/...`: passed.
  - Vet and compile-only: passed.
- Allocation benchmark: `4,122,459 ns/op`, `6,682,184 B/op`,
  `16,564 allocs/op`.
- Tests commit: `c440a3522206e7898151a704d48d77073145123e`
  `test(user-pmcs): cover concurrency and performance limits`
- Historical checkpoint HEAD: `c440a3522206e7898151a704d48d77073145123e`
- Worktree after implementation: only this untracked progress ledger
- Reviewer: `/root/hardening_task15_review` (fresh)
- Review verdicts:
  - Spec compliance: FAIL.
  - Task quality: CHANGES REQUIRED.
- Important findings:
  - Named concurrency cases lack scoped deterministic dedicated-connection
    database barriers and bounded fixture cleanup.
  - Required lock-wait, DB/encode, and peak-allocation measurements are
    missing or mislabeled across scenarios.
  - The 50-publication case retries a canceled publish instead of exercising
    first-sync reconstruction interruption/resume.
  - EXPLAIN gates use synthetic or empty-result probes rather than
    representative production query shapes with full relation-scoped plans.
  - Route/body/conditional/tombstone coverage and exact indistinguishable
    safe-404 envelopes are incomplete.
  - Superseded-unreleased visibility and publication notice completeness are
    not exercised through service/route boundaries.
- Deferred Minor: exact boundary fixtures use nondeterministic UUIDs and
  plus-one field checks accept nonspecific database errors.
- Authorized `AcceptUpdate` production correction: approved as narrow; no
  Critical finding.
- Fix round 1: original implementer dispatched for all six Important findings.
- Fix round 1 finding 1 GREEN: identified single-connection workers with exact
  fixture-rooted waiter chains, a real repeatable-read delta/mutation
  interleaving, and bounded fixture database contexts; concurrency group passed
  in `7.454s`.
- Fix round 1 finding 2 GREEN: all seven scenarios explicitly measure and
  validate p50/p95, isolated driver DB time, measured lock wait or semantic
  zero, encode time, peak live-heap delta, raw/gzip bytes, and actual SQL
  count.
- Fix round 1 finding 3 GREEN: first sync reconstructs 50 distinct published
  roots as 25 plus durable-cursor resume 25; every ID appears exactly once with
  a complete publication tree and final account version 100.
- Fix round 1 finding 4 GREEN: driver-captured production SQL plans and
  per-large-relation no-sequential-scan assertions pass after fixture
  `ANALYZE`; only the 112-row `users` lookup uses a sequential scan.
- Fix round 1 finding 5 GREEN: 30 auth, 12 JSON-body, 20 missing/stale
  conditional, and 16 malformed-validator cases; exact safe-404 envelope
  equality plus complete owned/subscription tombstone lifecycle matrices.
- Fix round 1 finding 6 GREEN: released revision 1 remains the only anonymous
  result while superseded-unreleased revision 2, current-unreleased revision
  3, and a private draft remain hidden; real publish handler/service requests
  reject missing notice type and blank notice text with exact 422 fields and
  unchanged database state.
- Fix round 1 final gates:
  - Focused: `20.695s` GREEN.
  - Performance: `48.550s` GREEN.
  - Race across seven user-PMCS packages, vet, compile, and benchmark: GREEN.
  - Supplemental full `tests/user_pmcs`: GREEN in `177.474s`.
- Environment: initial sandboxed `git add` was denied `.git/index.lock`;
  identical approved retry succeeded.
- Fix commit: `ffe7b8295dfc7633335adceb209f0da460314217`
  `test(user-pmcs): strengthen hardening evidence`; exactly the four permitted
  Task 15 files.
- Fix round 1 scoped re-review:
  - All six original Important findings: ADDRESSED.
  - New Critical/Important findings: none.
  - Scoped result: APPROVED.
- Status: complete.
- Historical checkpoint HEAD: `ffe7b8295dfc7633335adceb209f0da460314217`.
- Worktree after completion: only this untracked progress ledger.
- Deferred Minor remains for final review: deterministic UUIDs in exact
  boundary fixtures and specific plus-one database constraint assertions.
- Task 16 exact-matrix diagnosis found that the subscription update EXPLAIN
  assertion required one physical index even though its rollback-only fixture
  is absent from PostgreSQL global statistics. The original isolated target
  passed, but the package-order and serialized repository runs selected the
  equally selective delta index and failed the assertion.
- Task 15 planner correction commit:
  `62ce505b4766bc159fc30798fc19e238ed29136e`
  `test(user-pmcs): stabilize subscription plan probe`.
- The correction preserves the no-sequential-scan gate and accepts only the
  active-update, delta, or primary-key subscription indexes.
- Independent scoped correction review: APPROVED with 0 Critical, 0 Important,
  and 0 Minor findings.
- Corrected target:
  `TestSubscriptionUpdateDiscoveryExplainUsesApprovedIndex` passed in `28.97s`
  (`29.364s` package), using
  `user_pmcs_subscriptions_active_update_idx`; plan execution was `0.450ms`.
- Corrected full `tests/user_pmcs` package: GREEN in `180.938s`.
- Next task: Hardening Task 16 — client contract, migration rehearsal, and
  complete verification matrix.

### Hardening Task 16: Mobile contract, migration rehearsal, and final matrix

- Status: review fix round 1 complete; scoped re-review pending.
- Base commit: `ffe7b8295dfc7633335adceb209f0da460314217`.
- Effective pre-Task-16 HEAD after the reviewed Task 15 planner correction:
  `62ce505b4766bc159fc30798fc19e238ed29136e`.
- Documentation research and sequential analysis: complete.
- Migration numbering: reverified; `009` remains the highest/current migration
  and both forward and rollback files exist.
- Task brief:
  `.superpowers/sdd/2026-07-29-user-pmcs-hardening-rollout/task-16-brief.md`.
- Worktree before dispatch: only this untracked progress ledger.
- Implementer: `/root/hardening_task16_impl` (fresh).
- Client contract:
  `docs/client/2026-07-29-user-pmcs-server-api-contract.md`.
- Contract audit: all 17 registered routes are documented from implemented
  route declarations and DTO JSON tags, including authentication, exact
  path/query/header parameters, strict request bodies, success/error envelopes,
  ETags and conditional behavior, cursor rules, current draft/publication
  deltas, owned/subscription tombstones, creator display name, offline first
  sync, pinned downloads, and explicit update acceptance.
- Implemented compatibility detail recorded: install and accept-update expose
  the untagged `MutationResult` wrapper as uppercase `Subscription`,
  `Installed`, `Created`, and `Idempotent` keys; unsubscribe returns the tagged
  subscription object directly.
- Decision update: ADR-019 records only the user-authorized final-pin exception:
  unsubscribe may reclaim retained content when the pin is final, owner is
  null, checklist is tombstoned, and source is retired. No general unsubscribe
  garbage collection was added.

#### Non-production migration rehearsal

- Production migration: not applied.
- Initial sandboxed `TEST_DATABASE_URL` preflight was denied with
  `dial tcp 192.168.20.70:5432: connect: operation not permitted`; the
  identical approved retry was used.
- Test target was verified before every migration step:
  `current_database()=miltech_ng_test`.
- Because migration `009` was already present at handoff, a separately
  identified setup rollback removed it before the official rehearsal;
  setup absence checks passed.
- Official test rehearsal:
  - forward migration: PASS;
  - `TestUserPmcsSchemaIntegrity`: all 18 subtests PASS in `0.654s`;
  - rollback: PASS;
  - absence audit: all 12 tables, 32 named constraints, and 11 explicit indexes
    absent; `to_regclass(...) IS NULL` true;
  - final forward migration: PASS;
  - `TestUserPmcsSchemaIntegrity`: all 18 subtests PASS in `0.601s`.
- Final test state: migration `009` applied to `miltech_ng_test`.
- Development target was separately verified as
  `current_database()=miltech_ng`.
- Migration `009` was already applied to development, so it was not reapplied
  and development was never rolled back.
- Development schema audit: 12/12 required tables, 32/32 named constraints,
  and 11/11 explicit indexes present; no missing objects.
- Jet was regenerated only from migrated `miltech_ng` using the canonical
  tracked JSON-tag template. It found 117 tables, 4 views, and 0 enums,
  regenerated `.gen/miltech_ng/public`, and produced an exact empty diff.
  Generated files were not hand-edited.

#### Verification matrix

- `go test ./api/user_pmcs/... ./api/user_general ./api/route -count=1`:
  user-PMCS and user-general packages PASS; command FAIL only on the accepted
  stale `api/route.TestSetupRegistersPmcsSbsFaultRoutesUnderAuth` assertion for
  `GET /api/v1/auth/pmcs-sbs/equipment/:equipment_id/faults`.
- `go test ./tests/user_pmcs -count=1`: first run exposed the now-corrected
  planner assertion; after Task 15 correction, PASS in `180.938s`.
- `go test -race ./api/user_pmcs/... -count=1`: all seven user-PMCS packages
  PASS.
- `go test ./... -count=1`: FAIL. The exact concurrent repository suite
  reproduced the accepted stale route assertion and shared-test-database
  interference: deadlocks and consequent fixture FK/not-found fallout in
  `tests/equipment_services`, `tests/pmcs_sbs_progress`, and `tests/shops`,
  plus an account-deletion deadlock in `tests/user_pmcs`. The user-PMCS package
  took `220.695s`. This result is not reported as green and the additional
  failures are not accepted as baseline.
- Isolation evidence before the correction: equipment services PASS in
  `11.193s`, PMCS-SBS progress PASS in `19.859s`, shops PASS in `81.882s`, and
  account-deletion lock scope PASS in `0.67s` (`1.215s` package).
- Supplemental `go test -p 1 ./... -count=1`: FAIL only on the accepted stale
  route assertion. All integration packages passed, including equipment
  services `13.088s`, PMCS-SBS progress `21.428s`, shops `78.000s`, and
  user-PMCS `181.637s`. This classifies the additional exact-suite failures as
  cross-package shared-database interference without relabeling the exact
  matrix command.
- The two previously documented `user_vehicle_shop_id_fkey` failures are not
  independently reproducible: the shops package passes alone and serialized.
- `git diff --check`: PASS with no output immediately before commit.
- Pre-commit `git status --short` contained only the three authorized staged
  paths:
  - `A  docs/USER_PMCS_SERVER_IMPLEMENTATION_PROGRESS.md`
  - `A  docs/client/2026-07-29-user-pmcs-server-api-contract.md`
  - `M  docs/project_notes/decisions.md`

#### Delivery

- Task 16 report:
  `.superpowers/sdd/2026-07-29-user-pmcs-hardening-rollout/task-16-report.md`
  (ignored SDD evidence).
- Initial Task 16 commit:
  `09962fd8a8aba3531841a3cd5e24234bca25d16a`
  `docs(user-pmcs): publish server synchronization contract`.
- Mobile sync implementation: not started; this task publishes the server
  contract only.
- Deferred v1 limitations remain unchanged: no collaborator editing, no
  background update install, no public revisions beyond released snapshots,
  and the Task 15 deferred Minor test-fixture refinements.

#### Independent review and fix round 1

- Initial Task 16 post-commit state: HEAD
  `09962fd8a8aba3531841a3cd5e24234bca25d16a`; `git status --short` produced
  no output.
- Reviewer: `/root/hardening_task16_review`.
- Initial verdict: FAIL with 0 Critical, 3 Important, and 1 Minor findings.
- Important: subscription update discovery and pinned-release reads did not
  return `409 account_not_initialized` when Firebase authentication succeeded
  but the UID was absent from `users`.
- Important: this ledger lacked Task 16 reviewer, commit, HEAD, worktree, and
  next-gate evidence and labeled historical HEAD/reviewer checkpoints as
  current.
- Important: the ignored Task 16 report named a nonexistent migration rollback
  file instead of `migrations/009_rollback_user_pmcs_sync.sql`.
- Minor: the client contract overstated UUID input canonicalization even though
  `github.com/google/uuid.Parse` accepts additional textual forms.
- RED:
  `TestHTTPContractSubscriptionReadsRequireInitializedAccount` failed in
  `0.571s`; update discovery returned `200` with an empty page and pinned
  release retrieval returned `404 resource_not_found`, rather than the
  required `409 account_not_initialized`.
- Root cause: the two subscription read repositories queried only subscription
  resources and did not apply the existing initialized-account read guard.
- GREEN implementation:
  - pinned-release retrieval verifies the subscriber UID in `users` before
    loading the installed release;
  - update discovery folds that verification into its existing query through a
    materialized account CTE and lateral update scan;
  - the Task 15 fixed performance bound remains one SQL query for 500
    subscription updates;
  - no transaction, lock order, provisioning, migration, or generated file
    changed.
- Focused HTTP target: PASS in `0.549s`.
- Focused privacy, pinned-read, update-discovery, update-acceptance, cursor, and
  retirement regression group: PASS in `9.202s`.
- 500-subscription discovery performance gate: PASS in `2.633s` with
  `query_count=1`.
- All seven `api/user_pmcs/...` packages: PASS.
- `go test -race ./api/user_pmcs/... -count=1`: PASS, 293 tests across seven
  packages.
- Focused repository/HTTP race command: PASS, three tests across two packages.
- `go vet ./api/user_pmcs/... ./tests/user_pmcs`: PASS with no output.
- Compile-only user-PMCS command: PASS after the sandboxed database connection
  was denied and the identical approved retry reached `miltech_ng_test`.
- Supplemental full `tests/user_pmcs` rerun: FAIL in `173.514s` only because
  the strict public-browse physical-plan assertion accepted only
  `user_pmcs_community_releases_pkey` while PostgreSQL selected the valid
  unique `user_pmcs_community_releases_checklist_revision_key`. The
  account-initialization behavior and one-query update gate passed. The
  unrelated strict assertion was not weakened.
- Behavior/test resolution commit:
  `d1d4a0c3b91d3be3ba846cba58a9b31ff9525844`
  `fix(user-pmcs): require initialized subscription readers`.
- Post-resolution-commit `git status --short`: no output.
- Documentation correction: this ledger-containing commit,
  `docs(user-pmcs): correct synchronization contract evidence`; its exact hash
  must be appended by the subsequent whole-branch closeout because a commit
  cannot contain its own hash.
- Next gate: the original Task 16 reviewer must perform a scoped re-review of
  all four findings before whole-branch closeout.

### Whole-branch Task 5 reservation resolution

- Status: implementation and focused verification complete; fresh independent
  review pending.
- Authorization: the user approved the migration 010 transaction-scoped
  content UUID reservation protocol in the controlling reservation-fix brief.
- Implementer: `/root/owned_task5_impl`.
- Actual implementation base: clean
  `4a6a2621c8a660f37e8332954bad4a3501073923`.
- RED evidence:
  - The unchanged SSI implementation failed the independent 20-user scenario
    in all five consecutive runs with PostgreSQL `40001`.
  - The schema integrity test failed because the reservation table and primary
    key did not exist.
  - The unchanged all-SSI mixed collision test passed 20/20, separating the
    collision-safety requirement from the failed availability requirement.
- Migration 010 creates only
  `user_pmcs_content_uuid_reservations(id UUID PRIMARY KEY)`.
- Test database migration:
  - verified `current_database() = miltech_ng_test` before every migration
    operation;
  - completed forward/rollback/forward;
  - final table present with zero rows.
- Development database migration:
  - verified `current_database() = miltech_ng`;
  - applied the forward migration exactly once;
  - did not run a development rollback;
  - final table present with zero rows.
- No production migration or production database connection occurred.
- Jet was regenerated from migrated `miltech_ng`. The generated reservation
  model and table files were not hand-edited.
- Protocol:
  - Create, PutDraft, and Publish gather revision/section/item/notice/step UUIDs;
  - one `INSERT ... SELECT ... ORDER BY` statement acquires the globally ordered
    reservation set inside the write transaction;
  - reservations remain held through validation and aggregate writes;
  - successful mutations explicitly delete reservations before commit;
  - rollback, callback failure, context cancellation, and backend termination
    clean up transactionally.
- Focused verification:
  - protocol/lifecycle/collision tests: 14/14 passed;
  - repeated reverse-order, same-table, cross-table, and mixed-size collisions:
    80/80 passed;
  - current Create/PutDraft/Publish/immutable-publication group: 36/36 passed;
  - API tests: 296/296 passed;
  - API race tests: 296/296 passed;
  - focused real-database race tests: 14/14 passed;
  - vet: passed with no output;
  - full-repository compile-only command: passed after the sandbox-denied
    database connection was retried with the required approval.
- Performance evidence:
  - independent 20-user repeated scenario: 20/20 passed with no `40001`,
    zero measured lock waits, 620 expected queries, zero retained reservations,
    and p50/p95 transaction latency `434.083ms/458.703ms`;
  - maximum-size reservation churn: acquisition p50/p95
    `79.718ms/109.812ms`, cleanup `44.798ms/58.631ms`, transaction
    `145.034ms/192.938ms`, WAL delta `9,922,656` bytes, and zero retained rows;
  - disjoint maximum-size writes: p50/p95 `2.090s/2.216s`, zero failures,
    zero measured lock waits, 252 queries, and zero retained rows.
- PostgreSQL 14.18 does not expose `pg_stat_force_next_flush`, and
  `pgstattuple` is unavailable. Dead-tuple observations therefore use the
  server's approximate relation statistics.
- Full-package truth: `go test ./tests/user_pmcs` was not green. It reported
  263 passing tests and three failure events in physical-plan assertions. The
  reproducible approved-index assertion observed a valid index scan through
  `user_pmcs_subscriptions_pkey` instead of the strict delta-index expectation.
  A subscription discovery assertion observed a sequential scan only in the
  full-suite state and passed in isolation. These failures were neither skipped
  nor relabeled as reservation successes.
- Commits:
  - `2455f3d9bd762270ff5349eac5fb2dc861baeec7`
    `feat(user-pmcs): add content UUID reservations`;
  - `337c99500bb1cf902a38f6365ab2ce06e1f89b86`
    `fix(user-pmcs): reserve content UUID writes`;
  - `2213de8356cc3efdb8ada44e8e06fcda1986abe8`
    `chore(user-pmcs): regenerate reservation models`.
- Pre-ledger HEAD:
  `2213de8356cc3efdb8ada44e8e06fcda1986abe8`.
- Next gate: fresh independent review of the complete reservation change and
  recorded evidence.
