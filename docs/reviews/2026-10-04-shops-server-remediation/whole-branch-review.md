# Independent whole-candidate review

Date: 2026-10-04. Reviewer: final whole-branch review seat. **Local code assessment: changes requested.** No Critical finding confirmed; six Important findings and five grouped Minor findings follow. This is neither an integration approval nor a release approval.

## Candidate, requirements, and review method

The input is `task-0-final-branch-review.diff`, the supplied cumulative 224-file candidate delta from base `518ef4e828abeb64c203e567287b730799222da7`, indexed by `whole-branch-review-index.md`. HEAD still points at the base; a base-to-HEAD comparison would not review this candidate. The index comprises 223 text entries plus the empty `.gen/.jetgen.lock` metadata entry, spanning 32,832 artifact lines. The original unrelated baseline changes, CodeGraph data, and review workspace are excluded from the supplied artifact. The removed credential URI was scrubbed before artifact publication; I did not derive a raw Git diff or inspect its value.

I applied the SeniorCodeReviewer template, Go skill, Database Optimizer skill, and Supabase Postgres Best Practices skill. The binding requirements are `docs/superpowers/specs/2026-10-03-shops-server-release-remediation-design.md`, the matching implementation plan, and `docs/audits/2026-10-02-shops-server-production-review.md`, interpreted as a product vision rather than an exhaustive list of permitted failure modes. I read the constraints first, the acceptance document, Task 28 report/review and verification records, final open items, rulings, and the relevant deferred ledger entries. Earlier scoped reviews are evidence, not a substitute for this review.

I reviewed the entire indexed candidate in domain passes: production Go and cross-layer behavior; lifecycle/schema/DDL; generation/build/runner/operational safety; unit and physical-database tests; and documentation/evidence. Most artifact passes displayed current-side additions and hunk context, suppressing obsolete deleted implementation lines; earlier production passes also displayed deletions. This is a complete current-candidate review of the indexed files, not a claim that every removed line was reread verbatim. The very large one-line source manifest was reviewed structurally rather than printed as hundreds of hashes. Truncated output ranges were recovered in subsequent targeted reads. The earlier scratch progress file records intermediate pending work and is superseded by this report and its appended closure.

For concrete caller and lock questions outside the diff, I reached for CodeGraph first. Its stale inventory required targeted current-source fallback. Those traces covered the actual notification mutation/Shop/member/list/vehicle lock path, legacy combined notification reads, authorization/error mapping, and the retained membership helper. I did not broadly review unrelated baseline code or Flutter/client source.

## Strengths

- Current authority is resolved from persisted resources and rechecked under the relevant Shop/member/resource locks. The retained notification mutation helper supplies a real shared serialization point and bounded whole-transaction retry for recognized lock races.
- The explicit message DTO separates the legacy public shape from generated database models. Canonical tagged generation, source manifests, staged publication, mandatory tagged startup generation and compatibility validation materially improve reproducibility.
- Message assets use immutable storage targets, scoped references, durable cleanup jobs, state/lease fences, and late-PUT reconciliation. Cloud deletion is outside database transactions, and destructive adoption of unproven historical assets is avoided.
- Notification metadata repair preserves original operation identity/fingerprint, uses retained evidence without resurrecting item rows, and detects stale repair evidence. The atomic save, audit, and replay work is substantially stronger than the original implementation.
- Tests include actual PostgreSQL lock barriers, role/grant behavior, cancellation, retained metadata, queue/state races, and complete PMCS capacity. Task 28 accurately records post-commit username lookup ambiguity: a failed response can follow exactly one committed business mutation, without hidden retry or compensation.
- The acceptance artifacts distinguish host evidence, disposable physical-database evidence, and unexecuted live/Docker/device gates. Expected fault-injection diagnostics are not misrepresented as production failures or pristine logs.

## Important findings

### I1 — The legacy notifications-with-items route still assembles different database snapshots

**References:** `api/shops/vehicles/notifications/route.go:13`; `service_impl.go:159`; `repository_impl.go:31`, `:46`, `:61`, `:87`, `:103` in the same package. Related corrected implementation: `api/shops/aggregates/` repeatable-read transactions. Requirement: design section 9.1 and the F23 consistency objective.

The active `GET /shops/vehicles/:vehicle_id/notifications-with-items` path checks the vehicle and membership through pool reads, then fetches notification headers and their items in separate pool queries. An atomic writer can commit changed header fields and replacement items between the two reads. A successful response can therefore pair the old header with the new items, a state that never existed in the database. Authorization is also not evaluated as part of that response's database snapshot.

This is a plan coverage defect: limiting Task 21's file list to `api/shops/aggregates` omitted an existing combined read boundary. The live route and the user's consistency expectation remain in scope even though the package name differs.

**Fix:** Use one context-bound, read-only repeatable-read transaction for the persisted vehicle binding, membership decision, notification headers, and items. Keep the existing DTO/envelope and omitted-limit behavior. Ensure every helper uses that transaction, rather than opening a second connection. Coordinate the query-shape repair with I6.

**Focused validation:** A physical writer barrier between the header and item stages must return either the complete old or complete new notification, never a hybrid. Verify membership as of the same read snapshot, revocation on the next request, cancellation, and unchanged response shape.

### I2 — Cleanup database phases can block indefinitely despite bounded cloud operations and expiring leases

**References:** `api/shops/messages/cleanup_worker.go:58`, `:75`, `:87`, `:89`, `:97`, `:112`; `api/shops/messages/asset_repository_impl.go:340` (Prepare), `:441` (Finish), `:478` (Reconcile), and its `loadAsset` locking helper; `main.go` cleanup-worker startup.

Only the cloud operation receives the 30-second deadline capped by the job lease. Reconcile, Claim, Prepare, and Finish all receive the process-lifetime context. Prepare can wait for an asset row lock; Finish can wait for its cleanup-job lock; connection acquisition and other database work can wait too. An idle transaction holding the needed row, or a depleted pool, can stall the single serial worker indefinitely. The two-minute lease expiring does not cancel that in-flight SQL. The iteration warning is emitted only when the operation returns, so a stalled worker need not produce the expected failure signal. Additional processes can encounter the same blocked asset.

This is separate from the already deferred ten-job/shared-lease throughput case. A durable queue remains recoverable after process restart, but durable recovery alone does not make its running worker bounded.

**Fix:** Give each database phase a bounded child context. Bound job preparation by both an operation budget and the remaining lease. Acknowledgment must use a fresh, still-live bounded child of the worker context, or safely leave the lease for recovery when acknowledgment cannot complete. Preserve cancellation on shutdown, state/lease validation, and the rule that cloud calls run with no database locks held.

**Focused validation:** Assert deadlines for each repository phase in worker tests. In a physical test, hold the asset/job lock or exhaust the pool, show timely cancellation, then release the blocker and demonstrate recovery and progress on another job. Include the existing multi-job lease-expiry scenario without weakening lease fences.

### I3 — Migration 020 rollback has an empty-check/drop race that can erase newly committed cleanup proof

**References:** `migrations/020_rollback_shop_message_asset_lifecycle.sql:1–13`; `docs/migrations/shop_message_asset_lifecycle.md:68`; migration source pins in `scripts/apply-shops-remediation-migrations.sh` and downstream guard/provenance artifacts.

The reverse migration checks all three lifecycle tables for rows before acquiring a writer-blocking table lock. Its SELECT locks permit concurrent INSERTs. An upload reservation or cleanup job can commit after the relevant empty check and before DROP acquires its exclusive lock. The rollback then drops newly populated durable state even though the stated safety contract is to refuse populated lifecycle tables.

An operational instruction to stop writers is necessary for a rollout, but it does not replace the reverse script's advertised refusal property. This is a concrete concurrent empty-to-populated race, not an assertion that a named target was changed.

**Fix:** Acquire appropriate writer-blocking locks on every lifecycle table before the first emptiness check, in a documented order consistent with lifecycle operations and with bounded operational lock waiting. Only then check emptiness and drop. Migration 020 is eligible for this candidate repair; the immutable historical set is 015–018. Update the migration source pins, guard fixtures/matrix, generated/provenance evidence, and any dependent acceptance record.

**Focused validation:** Use two physical connections and an explicit barrier to race insertion against rollback. The outcome must either serialize rollback before the writer can create state or refuse rollback and retain committed proof; it must never silently drop a successfully committed reservation/job.

### I4 — Build-context exclusions do not cover ordinary logs and local temporary artifacts

**References:** `.dockerignore:14–41`; `Dockerfile:15`; `scripts/test-server-container-inputs.sh:77–81`. Requirement: design section 12.2/F10 excludes local logs and temporary artifacts from build inputs/layers.

The ignore file covers credential names and many repository/tool caches, but does not exclude ordinary `*.log` files, log directories, or general local temporary artifacts. `COPY . .` sends an unexcluded `server.log`, `logs/worker.log`, or scratch directory into the builder. Being Git-ignored does not exclude a file from Docker's context. The current sentinel test inventory does not exercise these ordinary log/temp paths.

This identifies a missing protection, not an observed secret in a file or a verified leak from a built image. The actual Docker context/layer checks remain unexecuted because the daemon probe failed.

**Fix:** Add appropriate root and nested log/temp exclusions while preserving required generated models and runtime/build assets. Add representative root and nested log/temp sentinels to the build-context/layer guard.

**Focused validation:** Verify the guard detects an intentionally omitted pattern and that required inputs remain included. Run the actual Docker context, layer, runtime, and mount acceptance when a local daemon is available; static pattern inspection cannot certify those gates.

### I5 — The active release runbook still instructs the obsolete generation and legacy-cursor behavior

**References:** `docs/testing/shops-release-contracts.md:297–308` and `:405–413`; current explicit message DTO, mandatory tagged generation, and legacy cursor guard implementation.

The still-active “Legacy-shape guarantee” says the response embeds the Jet model, that `shop_messages` is intentionally not regenerated, and that the generated `InsertionNumber` must be hidden with `json:"-"`. The candidate instead uses an explicit DTO and requires canonical tagged regeneration without hand-editing generated schema output. A later appended gate does not clearly retire the earlier operational instruction.

The operator section also says legacy catch-up silently omits NULL insertion numbers. The current universal cursor guard fails closed instead. Keeping both descriptions as current instructions can lead an operator to skip required generation or diagnose the wrong sync behavior during a release.

**Fix:** Replace the obsolete active directions, or mark them explicitly historical and link to the current behavior. State the explicit DTO, mandatory tagged regeneration, unready legacy cursor 503 behavior, numeric reset 409 behavior, and current migration/fleet-fence requirements together. Preserve historical audit evidence rather than rewriting it as if the old behavior never existed.

**Focused validation:** Review all release instructions that mention Jet message embedding, generated-field tag edits, intentionally skipped regeneration, NULL legacy anchors, or cursor recovery against current code. This is a document consistency repair, not a reason to rerun successful application suites alone.

### I6 — Unbounded notification reads still exceed PostgreSQL's bind-parameter capacity

**References:** `api/shops/aggregates/repository_notifications.go:156–192`; `api/shops/vehicles/notifications/repository_impl.go:87–103`. Callers include Shop snapshot/vehicle-maintenance notification aggregation and the legacy notifications-with-items route.

The aggregate helper binds one parameter per selected notification plus a limit parameter. Selecting 65,535 notifications therefore produces 65,536 bindings and exceeds the driver's/PostgreSQL protocol limit of 65,535. The legacy helper binds one parameter per notification and fails at 65,536 notifications. This is a deterministic conditional capacity failure of the omitted-limit/unbounded contract, even when each notification has no items. It is not evidence that the deployed fleet currently reaches that population.

The complete PMCS fixture demonstrates the repaired PMCS query shape; it does not validate these other unbounded queries. Restricting the fix to the F35 named PMCS files left the same class of failure at another user-facing boundary.

**Fix:** Use an array binding or a relational membership-scoped query with a bounded parameter count. Preserve the exact selected parent set, per-parent item limit, deterministic ordering, authorization snapshot, and omitted-limit semantics. Do not repair this by silently truncating notifications or inventing a production cap. Audit the analogous ID-expanded bootstrap equipment helper while making the query-shape change; it also adds a limit parameter to a selected Shop-ID list.

**Focused validation:** Assert bounded parameter shape, then exercise physical notification counts immediately around the two thresholds with the owning routes. Prove all selected parents/items are retained and limits/order remain correct. Coordinate the legacy route test with I1's transaction change. No claim about production latency or memory budgets follows from merely clearing the protocol ceiling.

## Minor findings

### M1 — Vehicle-deletion audit failures bypass the common monitored, sanitized event

**References:** `api/shops/vehicles/repository_impl.go:309–321`; `api/shops/vehicles/notifications/items/service_impl.go:418` common warning helper; `docs/testing/shops-release-contracts.md:422–433` collector expectations.

Vehicle deletion deliberately preserves the business mutation when its savepoint-protected `vehicle_deleted` audit insert fails. That policy is allowed, but the remaining warning is a different raw-error message with only the vehicle ID. The documented collector watches `legacy_notification_audit_failed`, so this path lacks the same actor/Shop/correlation/category signal and may be absent from the intended audit-gap alert. There is still a warning; I do not claim silent deletion or an observed secret leak.

Use a common sanitized event contract with actor, Shop, vehicle, correlation, and stable failure category while preserving the savepoint and business-write semantics. Inject one audit failure and assert one sanitized event, committed deletion, and collector coverage.

### M2 — The unused unlocked membership upsert remains callable through the core repository interface

**References:** `api/shops/core/repository.go:18`; `api/shops/core/repository_impl.go:349`; Task 11 deferred item.

`AddMemberToShop` retains the standalone role upsert outside the current lock/authorization path. Targeted caller inspection found no production caller, so this is not an active authorization bypass. Remove the obsolete interface method/implementation and its obsolete cancellation-only case, or make its safe internal contract explicit, to avoid accidental reuse as membership logic evolves. Preserve tests of the actual supported member mutation paths.

### M3 — Equipment deletion logs success before its transaction commits

**Reference:** `api/equipment_services/core/repository_impl.go`, `Delete` transaction callback; Task 8 deferred item.

The success log occurs inside the transaction callback. A later commit failure can leave a “deleted” message for a deletion that did not commit. Move the success event after a successful transaction return. This is an observability defect, not evidence that the transaction helper commits failed work.

### M4 — The performance sampler is not stopped on an early test failure

**References:** `tests/shops/shops_aggregate_performance_test.go:605`, `:632`; Task 28 deferred item.

The sampler stops through the returned finish closure. A preceding `require`/FailNow can bypass that closure and leave the sampler querying until package exit, obscuring the original failure. Register an idempotent stop/wait in `t.Cleanup`, and reuse it from finish. The successfully completed measurements recorded for this candidate are not invalidated by this failure-path defect.

### M5 — A few durable evidence labels are stale or transposed

**References:** `docs/project_notes/bugs.md:21–25`; `docs/testing/shops-database.md:609–614`; `docs/migrations/shop_message_allocator_lock_order.md`, current-writer transaction row.

The bug notes transpose F29/F30 (privileges versus unsafe migration 017), the database document still presents the now-repaired F32 regression as currently failing, and the allocator table labels the current writer `sharedb.WithTx` instead of `WithTxContext`. Correct the identifiers and mark old failing evidence as historical. Keep original diagnostic context and link current verification rather than deleting the history. These can be repaired with I5's document pass.

## Deferred, parked, and open-check disposition

The table explicitly carries the deferred items identified in the ledger and scoped reviews. “Follow-up” means an assessed nonblocking test/maintenance gap, not an owner waiver or an assertion that missing evidence exists.

| Item | Disposition and reason |
|---|---|
| Task 4 negative catalog type/nullability, output/namespace/backups/symlink cases | Follow-up negative coverage. Source guards exist; later physical incompatible mandatory-column coverage addresses part of the risk but does not replace every generator/path case. No concrete publication escape was found. |
| Task 5 cleanup warnings for image tags never created | Follow-up diagnostic cleanup: track created tags before removal. A warning from an early failed build is not successful layer/runtime validation. Actual Docker gates remain open. |
| Task 6 failed-marker-query diagnostic redaction branch | Follow-up negative test. BAD_MARKER tests mismatch, not a failing marker query. The source suppresses that diagnostic; add an explicit failing-query fixture to protect the behavior. |
| Tasks 6/11/12/14/18/19/21/22/23 focused equipment selections matching no tests; Task 24 focused Shops selection matching no tests | Zero coverage remains zero coverage for those selections. Final owning-package/full physical runs supply separate evidence. No duplicate rerun is needed simply to relabel a no-match command. |
| Task 8 denial assertions accepting generic legacy 500 | Follow-up precision. Persisted-state assertions and the authority source path remain meaningful. Strengthen version-2 status/code/body/cause checks while retaining the documented legacy envelope compatibility. |
| Task 8 pre-commit equipment deletion log | M3: real minor observability defect; fix location of success log. |
| Task 11 unused core membership helper | M2: no production caller found; remove obsolete unsafe entry point, not an active exploit claim. |
| Task 12 discard log falsely saying cloud object deleted | Closed by the later queued-event change and its unit regression: `shop_message_image_cleanup_queued` records queued work rather than completed deletion. |
| Task 12 generated Python cache | Closed in the candidate cleanup/exclusion evidence. Not product code or an outstanding acceptance result. |
| Tasks 13/18/24/26/27/28 expected WARN/ERROR output capture and matrix categorization | Follow-up targeted event assertions. Existing injected failures, denial/cancellation, and category checks explain diagnostics; do not relabel them as new production failures or call the logs pristine. Live alert routing remains unexecuted. M1 identifies the specific remaining unmatched event. |
| Task 14 ten sequential cloud jobs can outlast their shared two-minute lease | Existing lease fences/recovery preserve safety; potential throughput loss and deterministic clock coverage remain. Fold the clock scenario into I2's worker regression; do not extend work beyond expired lease to make the test pass. |
| Task 18 direct service/repository invalid second batch element no-touch test | Follow-up defense regression. HTTP matrix plus pre-transaction validation provide current evidence; add direct entry-point coverage without claiming validation is absent. |
| Task 22 direct NULL anchor and exact legacy `data:null`/omitted-code versus version-2 reset envelope tests | Follow-up contract precision. Branches exist; existing deleted-anchor/internal failure tests are not the exact missing NULL/serialized-envelope cases. I5 must describe the implemented guard accurately. |
| Task 26 expected startup category warning capture | Follow-up exact event assertion; folded into diagnostic row above, not a startup failure finding. |
| Task 27 zero-exit/no-receipt and mismatched-PID wrapper branches | Follow-up negative coverage. Current source refuses uncertain completion and does not auto-retry; the matching-PID fake does not prove every refusal branch. |
| Task 28 performance sampler early failure | M4. Recorded completed metrics remain usable within their stated limitations. |
| Legacy current-writer late commit behind timestamp anchor | **OPEN, unresolved behavior.** Physical evidence intentionally demonstrates it; no lossless legacy claim and no owner waiver. Preserve-versus-monotonic timestamp policy still requires a decision. |
| In-range numeric cursor ABA after restore | **OPEN.** Ahead-of-counter reset handling does not detect an in-range reused cursor. Requires the explicit epoch/recovery/client gate, not a claim that all restores are covered. |
| C06 active-code population, fleet instance count, and UID/edge abuse budget | **UNKNOWN/BLOCKED.** Owner inputs unanswered. No invented production population limit or approved security budget. Removal remains not-a-ban under the recorded policy; unsupported controls stay rejected. |
| Original F09 credential and separate Task 18 MCP credential incident | **OPEN owner confirmation.** These are separate obligations. Task 18 known source identity is Docker `crystaldba/postgres-mcp`, role `postgres`, database `miltech_ng`; no credential value was inspected or repeated here. No rotation/contact was performed. |
| Named migrations/repair targets | **UNEXECUTED.** `miltech_ng_test` first, then separately authorized `miltech_ng`, with same-session identity/schema and source pins. UNPINNED is not a usable named-target approval. |
| Actual application DB role, grants and real legacy writes | **UNEXECUTED on named targets.** Disposable role tests do not establish production `DB_USERNAME`, ownership or privileges. |
| Fleet build/schema/flags, external counter-first/message-first writers and asset-writer compatibility | **OPEN inventory and fencing gate.** Local lock proof is not evidence that every deployed or external writer participates. |
| Docker context/layer/runtime/credential mount/read-only generated tree | **UNEXECUTED.** The bounded actual daemon probe exited 3 after 10.067 seconds with daemon-unavailable evidence. I4 is a static protection defect independent of that unavailable acceptance. |
| Real TMDE view definition | **UNKNOWN.** The explicitly documented empty synthetic fixture is a compile ABI, not proof of a live catalog definition or production behavior. |
| Service-date flags and released request/parser/device/signed-artifact acceptance | **UNEXECUTED; flags remain false.** Service dates/reads, field mappings, owner request fixtures, client parsing, physical devices and signed artifacts are separate gates. |
| Required live alert routes | **UNEXECUTED.** Tests and warning counts are not proof that a deployed collector pages the right owner. |
| Complete PMCS capacity and production budget | Local complete fixture measured roughly 25.9 MB response and 656.6 MB cumulative allocation per request. No peak-RSS, cold/concurrent fleet, or production budget clearance follows. Existing overview p95 evidence is separate (74.803 ms plain / 85.609 ms gzip against its stated one-second target). I6 remains a distinct deterministic capacity defect. |
| Task 28 username propagation/cancellation after commit | Reviewed with cross-layer semantics: nine propagated service paths and persisted-once/cache-cancellation evidence address the scoped defect. A response failure after commit remains documented ambiguity, with no automatic business retry/compensation guarantee invented. |
| Local integration, push, deployment and feature activation | Not authorized or performed by this review. No readiness claim substitutes for these steps. |

## Verification evidence and limits

I independently parsed `.gen/source-manifest.json` and hashed all **749 current input paths**. All matched the manifest. The current source digest is **`bc5ad7ab73d9049ff7bac44910f10bc6eb337fed99b90c2ab9fa75c00834ba7e`**. This validates the recorded source identity, not semantic correctness.

I independently hashed all **26 log files** referenced by `task-28-verification-records.json`; there were no mismatches. I inspected/parsed the recorded outcomes rather than rerunning reported suites:

| Recorded verification | Evidence consumed |
|---|---|
| Final host tests | 616 top-level tests / 1,305 including subtests, pass; three inherited PSMag skips. |
| Unit race | 144 / 434, pass; zero skips. |
| Full guarded physical runner, migrations/capacities | 297 / 974, pass; zero skips. Recorded injected diagnostics: 39 ERROR / 1 WARN. |
| Physical race | 32 / 123, pass; zero skips. Recorded injected diagnostics: 18 ERROR / 3 WARN. |
| Build and vet | Recorded exit 0. |
| Python guards / runner / publication | 11 / 12 / 4 tests, recorded pass. |
| Actual bounded Docker probe | Exit 3, 10.067 seconds, daemon unavailable; downstream Docker acceptance unexecuted. |

Task 28's independently reviewed evidence records all 32 tracked PMCS generated files unchanged from HEAD and 18 migration files 015–023 unchanged from its start snapshot. I consumed that scoped provenance; I did not independently rerun those Git comparisons. I did independently validate the current manifest files and verification log hashes as stated above.

No new application/test suite was run by this reviewer. The new findings are supported by concrete source paths and interleavings/query shapes; the focused tests above are required regression directions for the fix wave, not tests I claim already failed. No production/named database, cloud target, credential, environment/configuration file, full process argument list, deployment, or client/device was inspected or changed. No subagent was dispatched. No source, index, HEAD, branch, or commit was changed; only the authorized reviewer workspace report/progress artifacts were written.

All indexed candidate files have current-side review coverage. Boundaries not covered are removed obsolete lines not displayed by the filtered passes, deliberately excluded baseline files, external/deployed state, unavailable Docker execution, and clients/devices. There is no residual unread indexed production area being silently graded green. Source proof and existing successful tests do not imply that the newly identified behaviors were tested by those suites.

## Declined to judge

Each line names behavior considered but not prescribed or certified by this review. These are explicit executor/owner rulings to retain, not silent exclusions. In particular, I1 and I6 were judged despite narrow task file lists because active combined reads and unbounded semantics are user-facing requirements.

- **Change legacy timestamp allocation to guarantee lossless late-commit catch-up:** the known omission remains OPEN; the accepted preserve-timestamp behavior and the unanswered preserve-versus-monotonic policy choice require an owner decision. I do not approve the omission or call it fixed.
- **Implement a new numeric synchronization epoch/restore protocol:** detecting in-range restore ABA requires the separate owner/client recovery contract. Current ahead-cursor 409 behavior is insufficient to certify it; the gate remains OPEN.
- **Choose new invite-code length, UID/fleet limits, bans, expiry or abuse budgets:** required C06 population/budget inputs are absent and those policy changes were not authorized. I do not waive the rollout security gate.
- **Automatically expire ready unattached drafts or delete/adopt historical assets without proof:** the recorded product policy explicitly disallows destructive cleanup of unproven assets. I assessed current scoped-reference cleanup, not a new retention policy.
- **Replace all legacy best-effort auditing with a mandatory outbox/gap-free design:** the accepted policy permits business success after an audit failure. I judged the required observable failure behavior (M1), rather than silently treating a stronger redesign as required or completed.
- **Invent production history caps, cold/concurrent SLOs, fleet memory limits or indexes from the local PMCS measurement:** production capacity inputs are missing; preserve the unbounded contract and retain the budget gate. I6's protocol ceiling is a concrete judged defect, not this unmeasured budget question.
- **Guarantee compatibility with arbitrary external or counter-first writers:** local code cannot establish their inventory or deployed participation. The fleet/external-writer fence remains required; no bridge is certified against unknown writers.
- **Migrate all unrelated account/general/PMCS transaction helpers to request contexts:** those baseline paths are outside the explicit Shops/direct equipment-services remediation scope; no concrete Shops caller discovered here justified expanding that migration. This does not excuse I2's worker context defect.
- **Certify an overall tagged-generation runtime bound:** the inspected startup/schema deadline and safe publication do not establish a production generation-duration budget. No overall generator duration acceptance was supplied, so I make no such claim.
- **Perform or certify named-target schema/repair, actual roles, credential rotation, live cloud behavior, unavailable Docker runtime/layers, real TMDE definition, client/device/signed artifacts, alert routing or deployment/activation:** these require the separately enumerated identities, authorization, owner decisions, infrastructure or artifacts. Their open status is preserved above rather than inferred from local tests.

## Verdict and next step

**Changes requested on the local candidate.** Resolve I1–I6 in one coordinated fix wave, explicitly dispose of M1–M5 and the listed deferred items, and apply scoped independent re-review to the changed contracts. The corrected candidate needs focused regressions, required owning checks, and refreshed source/provenance artifacts; any source or migration edit invalidates the current digest as evidence for that later candidate. There is no need to rerun unchanged suites merely to restate their current result.

Even after local findings are resolved, legacy late-commit policy, restore ABA recovery, C06 population/budget, credentials, named targets, fleet writers, actual roles, Docker, real TMDE, client/device/signed artifacts, alerts, and deployment/activation remain separate gates. No merge, release, migration, push or deployment approval is granted by this report.
