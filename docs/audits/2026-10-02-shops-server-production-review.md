# Shops server production code review — 2026-10-02

**Release assessment: not ready for shipment from the reviewed inputs.** There are reachable authorization and data-loss defects, a generated-model compatibility failure, and failing verification checks. Passing compilation and migrations do not resolve those problems.

## Scope and evidence

- Repository: `/Users/swisscheese/projects/miltechserver`, branch `cleanup`, HEAD `4abd3a19fb0efe1ac51d549e4f7a9866c0f8b8a5`.
- Primary recent-change window: `d3b18ce..4abd3a1`, 49 commits, September 26–30. The review also traced August changes and older reachable behavior. `main..HEAD` contains 138 commits; it is not the same 49-commit window.
- Reviewed all Shops packages: core, members/invites, settings, lists/items, vehicles/usage, notifications/items/history, aggregates, capabilities, messages and sync. Direct dependencies reviewed include equipment services, Firebase authentication, production route/error middleware, transaction helpers, request/response types, generated models, migrations 015–018, build/generation inputs, migration runners and relevant tests. The Shops and equipment-service directories contain 147 files.
- **No Flutter/client source was reviewed.** Compatibility conclusions below use server request contracts and supported server request sequences, not assumptions about a particular shipped client.
- Review method: independent subsystem source reviews, recent history/diff inspection, CodeGraph, fresh build/vet/tests, and temporary Go overlay reproductions. The existing untracked release review and findings-revalidation documents were preserved and treated as leads requiring verification.
- No application source, tests, generated models, migrations, settings, credentials or project notes were changed. This new report is the only intentional repository write. No commit, deployment, application startup or connection to an existing database was performed.
- Database verification used only clusters created by `scripts/test-shops-isolated.sh`, the approved checksum-pinned fixture, its later migrations, and the disposable-instance marker. The wrapper removed each cluster. Neither `miltech_ng_test` nor `miltech_ng` was contacted.

**Evidence labels:** **Reproduced** means observed in this review's local/disposable checks; **Source-confirmed** means the complete reachable source path establishes the behavior; **Conditional** identifies the additional schema, concurrency, rollout or capacity condition. No finding establishes that an exploit has occurred in production.

**Severity:** P1 should block shipment or the affected rollout until repaired; P2 is a material correctness, UX or verification problem requiring resolution or an explicit release decision; P3 is lower-impact contract accuracy. Findings share causes and should not be treated as 37 independent patches.

## Findings ledger

| ID | Priority | Finding | Change provenance / evidence |
| --- | --- | --- | --- |
| F01 | P1 | Production authenticated errors can finish as empty HTTP 200 | Legacy wiring; September contracts depend on it; reproduced group semantics |
| F02 | P1 | Service completion/deletion accept admin authority from another Shop | Legacy; disposable route reproduction |
| F03 | P1 | Message text authorizes deletion of a foreign image | Legacy; source-confirmed |
| F04 | P1 | Orphan cleanup can delete another member's published image | Legacy; source-confirmed |
| F05 | P1 | Several writers remain authorized after removal/demotion commits | Legacy and August additions; PATCH removal race reproduced |
| F06 | P1 | Revoked sessions and disabled Firebase users remain authorized | Legacy; pinned SDK and source-confirmed |
| F07 | P1 | Shop metadata edits clear list restrictions; create ignores them | Legacy; reproduced, including false-setting rejection |
| F08 | P1 | Generated messages change JSON and require 018 even with sync off | Current build input/recent schema interaction; reproduced |
| F09 | P1 | Transaction unit tests contain a committed shared-DB credential | September helper tests; source-confirmed; deliberately not run |
| F10 | P1 | Container build copies local secrets and embeds the Firebase key | Existing release packaging; source-confirmed, image not built |
| F11 | P1 | Foreign reply parents create cross-Shop deletion dependencies | Legacy; source-confirmed; cascade impact on pinned schema |
| F12 | P1 | Creator/admin usage-only PUT erases equipment metadata | Legacy full writer plus August role routing; reproduced |
| F13 | P1 | A revoked invitation can still admit a delayed join | Legacy; source-confirmed interleaving |
| F14 | P2 | A delayed duplicate join can overwrite an admin role | Legacy admission upsert; source-confirmed interleaving |
| F15 | P2 | Shop creation suppresses failure to create its first membership | Legacy; source-confirmed failure path |
| F16 | P2 | Final departures can strand users or leave memberless Shops | Legacy lifecycle plus incomplete recent locking; source-confirmed |
| F17 | P2 | Promoted-admin rename passes authorization but cannot persist | Legacy; source-confirmed |
| F18 | P2 | Legacy item replacement loses new nickname/unit metadata | September enrichment interaction; reproduced server sequence |
| F19 | P2 | Base equipment readings accept negative effective usage | Legacy gap retained by August hardening; source-confirmed |
| F20 | P2 | Equipment admin-number edit reports success without saving | Legacy; reproduced |
| F21 | P2 | Legacy item and notification validation permits invalid records | Legacy paths retained by refactors; list bulk reproduced |
| F22 | P2 | Request cancellation is discarded at database boundaries | Existing gaps plus September transaction regression; source-confirmed |
| F23 | P2 | Aggregates combine multiple committed versions | Existing multi-query design; source-confirmed concurrency property |
| F24 | P2 | Legacy message paging skips timestamp ties/newer bursts | Legacy; source-confirmed examples |
| F25 | P2 | Multipart limit runs after request parsing | Legacy; source-confirmed; proxy bounds unknown |
| F26 | P2 | Service edits/repeated completion rewrite historical completion time | Legacy; source-confirmed |
| F27 | P2 | Declared service filters are ignored and paging lacks tie order | Legacy; source-confirmed |
| F28 | P2 | Invite expiry/use limits are accepted but discarded | Legacy request/writer mismatch; source-confirmed |
| F29 | P2 | Message migration privilege gate can pass without required grants | New rollout documentation; verified Postgres semantics |
| F30 | P1 | Migration 017 guide instructs unsafe generation and wrong target order | September 29 documentation regression |
| F31 | P2 | Nil configuration crashes production-route unit tests | New capability flag wiring; fresh suite failure |
| F32 | P2 | Enrichment integration test asserts the wrong audit kind/count | New test; fresh suite failure and source explanation |
| F33 | P2 | Isolation wrapper regression tests fail at import | Recent wrapper/test drift; fresh failure |
| F34 | P2 | Legacy item audits omit newly meaningful unit/nickname fields | September enrichment exposes old projection; source-confirmed |
| F35 | P2 | PMCS history has a deterministic parameter-limit failure | Existing design; capacity-dependent, source/pinned driver confirmed |
| F36 | P2 | Batch list removal newly rejects stale non-first item IDs | September authorization tightening; compatibility narrowing |
| F37 | P3 | Bulk removal success counts overstate actual deletions | Legacy; source-confirmed |

## Security and release blockers

### F01 — Error middleware is absent from the production authenticated group

**Evidence:** `api/route/route.go:112–120` attaches `ErrorHandler` to `v1Route`, then constructs `authRoutes` independently from the engine. Authenticated Shops and equipment-service routes are registered on that sibling group at `:155–160`. `api/shops/shared/contract.go:16–38` installs response writers but does not process `c.Errors`. Handlers such as `api/shops/members/handler.go:37–40` call `c.Error(err)` and return without a body.

**Trigger/impact:** Invalid joins, denied reads/writes, missing resources and database failures can produce empty HTTP 200. A caller receives a success status for a failed operation and no useful failure envelope. The negotiated error machinery cannot fix a middleware chain that never invokes it. Direct `response.Error` paths still write their responses; this does not affect every endpoint failure equally.

**Fresh verification:** A temporary test reproduced the exact sibling-group arrangement with the real contract/error middleware: recording an error produced status 200 and an empty body. This isolated routing semantics, not live Firebase authentication. The integration routers instead install `ErrorHandler` globally (`tests/shops/helpers_test.go:44`, `tests/equipment_services/helpers_test.go:36`), which masks the assembly defect.

**Repair/acceptance:** Install error handling on the actual authenticated chain while retaining Shops redaction/negotiation. Exercise actual `route.Setup` with controlled authentication and failing handlers under legacy and contract 2: invalid invite, nonmember read, denied settings update, not-found and repository failure must produce nonempty correctly classified envelopes. Preserve intentional legacy status behavior separately from fixing empty success responses.

### F02 — Equipment-service completion/deletion authorize the wrong Shop

**Evidence:** `api/equipment_services/shared/authorization.go:107–120` accepts admin status in supplied `shopID`. `completion/service_impl.go:33–41` and `core/service_impl.go:182–190` then mutate by service ID. `completion/repository_impl.go:38–43` and `core/repository_impl.go:156–164` require only membership in the service's actual Shop.

**Reproduced:** A user who is admin of A and an ordinary member of B completed and deleted a service created by B's owner through `/shops/A/equipment-services/<B-service-id>`. Both requests returned 200 and the persisted row changed/disappeared. Existing equipment-service tests passed despite this missing permission case.

**Impact:** An ordinary member can obtain effective admin mutation rights in any Shop they belong to by supplying another Shop where they are admin. This is not unrestricted outsider access: membership in B is still required.

**Repair/acceptance:** Resolve persisted service ownership, bind it to the URL Shop, and enforce creator/admin permission in that same Shop inside the write transaction. Test wrong-Shop URLs, A-admin/B-member denial, actual B-admin/creator success, and unchanged rows on denial. The newer update repository already rechecks actual-Shop permissions and should retain that protection.

### F03 — Message text is treated as authority to delete Azure assets

**Evidence:** `api/shops/messages/service_impl.go:239–256` deletes an authorized message then passes its text to blob cleanup. `repository_impl.go:368–393,470–495` extracts an image marker and the suffix after `/shop-message-images/`, then deletes that key with the configured Azure client. Host, Shop, uploader, asset association and surviving references are not validated.

**Trigger:** A member creates their own text containing `[IMAGE:https://anything/shop-message-images/<foreign-shop>/<known-upload-id>.jpg]`, then deletes their own message. The message-row authorization succeeds, while cleanup targets the foreign object. The host is ignored: this is object-deletion authorization failure, not SSRF.

**Impact/condition:** Existing images in other Shops can become unavailable if the key is known and Azure credentials permit deletion. No Azure request or production exploit was executed in this review.

**Repair/acceptance:** Destructive cleanup must use server-owned upload identity/ownership and reference state. URL parsing alone cannot establish ownership, especially for copied same-Shop references. Test foreign-Shop/account URLs, copied URLs, still-referenced assets and unproven historical assets; unrelated objects must survive deletion of the attacker's message.

### F04 — Orphan cleanup has no orphan or uploader authorization

**Evidence:** `api/shops/messages/route.go:15–16`, `handler.go:276–295`, `repository_impl.go:337–365`. `DELETE /shops/messages/image/:message_id?shop_id=<shop>` verifies membership only, then deletes `<shop>/<supplied-id>.<extension>`.

**Trigger/impact:** A member reads another author's image URL and supplies its upload ID. The endpoint can delete an image still attached to that author's published message. It does not check who uploaded it or whether it is unreferenced. Upload IDs and message IDs are independently generated (`service_impl.go:65,276`), so a message-ID author check is insufficient.

**Repair/acceptance:** Persist upload ownership and attachment/reference state. Enforce the agreed uploader/admin policy and permit orphan cleanup only for truly unreferenced assets. Test another member's asset, a published/shared image, former membership and a genuine uploader-owned orphan. Decline destructive cleanup when historical ownership cannot be proved.

### F05 — Current-authority checks do not cover every writer

**Evidence and affected operations:**

| Writer | Preflight | Persistence gap |
| --- | --- | --- |
| Vehicle create, metadata PUT, absolute usage PUT, usage PATCH | `api/shops/vehicles/service_impl.go:39–64,120–175,183–210` | `repository_impl.go:29–47,84–124,128–162,165–227` lacks transaction-bound caller checks; PATCH repository has no caller identity |
| Shop rename | `api/shops/core/service_impl.go:67–76` | `core/repository_impl.go:89–95` uses ID/creator rather than current membership/admin |
| Invite create/deactivate/delete | `api/shops/members/invites/service_impl.go:43–49,113–122,141–150` | `invites/repository_impl.go:23–36,85–112` uses standalone writes |
| Service complete/delete | `api/equipment_services/shared/authorization.go:107–120` | Completion/delete DML rechecks membership but not current creator/admin role |

**Reproduced PATCH race:** A disposable transaction held the vehicle row lock. The member's PATCH passed preflight and blocked on that row. The current server's member-removal endpoint committed removal; releasing the blocker let PATCH return 200 and persist usage. Revocation had committed before the delayed write.

**Other source-confirmed interleavings:** A removed creator can rename through the immutable creator predicate; a removed member can finish invite creation; a demoted noncreator admin can complete/delete another user's service because membership alone still satisfies the DML. These are separate from F02's wrong-Shop authority.

**History:** September `3917e08` reauthorized many mutations but left these paths outside the repair. August member-usage/PATCH additions retained preflight-only authorization.

**Repair/acceptance:** Pass typed caller identity through all writer repositories and coordinate Shop → membership → owned resource authorization with removal/demotion. Test every row in the table using deterministic barriers. A mutation authorized under the coordinating lock can commit before removal; removal/demotion committed first must prevent the later authorization/write. Do not weaken existing locks merely to reduce contention.

### F06 — Firebase disablement and token revocation are not enforced

**Evidence:** `api/middleware/authentication.go:36` calls `VerifyIDToken`; `:57–68` fetches the Firebase user but uses only DisplayName. It never rejects `Disabled` or checks `TokensValidAfterMillis`. Pinned Firebase v4.15.2 explicitly distinguishes the normal verifier from checked revocation/disablement verification. [Firebase session revocation documentation](https://firebase.google.com/docs/auth/admin/manage-sessions#detect_id_token_revocation)

**Impact:** A still-valid bearer token from a disabled user or revoked session can continue accessing Shops while database membership exists. Ordinary signature/expiry verification is present; this is the missing revocation/disablement decision, not acceptance of unsigned tokens.

**Repair/acceptance:** Use checked SDK verification or equivalent checks against the user already fetched. Avoid an unnecessary second lookup when implementing the same checks. Test active, disabled, revoked, expired and deleted users; denied authentication must never execute the handler.

**Related availability defect:** `authentication.go:57–60` logs user lookup failure and continues, then dereferences `username.DisplayName` at `:66`. The SDK returns nil on lookup failure. Deleted users or Firebase outages therefore panic; recovery yields a bare 500. Return on lookup failure and distinguish invalid/deleted users from transient identity-service failures. Test `(nil,error)` without a panic or leaked dependency error. `bootstrap/authentication.go:33–42` also returns a nil auth client on initialization failure; startup should not silently publish an unusable authenticated API.

### F07 — List restriction intent is lost during Shop create/update

**Evidence:** `api/shops/core/handler.go:233–237` builds an update without `AdminOnlyLists`; `core/repository_impl.go:67–85` explicitly writes its default false. The create handler accepts the field at `:82–84`, but the insert omits it (`repository_impl.go:44–51`). Installed Jet v2.13.0 serializes every supplied column assignment, so the extra SET is real despite the smaller `UPDATE(...)` column list.

**Reproduced:** Create with `admin_only_lists:true` persisted false. Set true, then rename: persisted false again. The legacy setter rejects an explicit false with HTTP 400 because `api/request/shops_request.go:196–198` applies `binding:"required"` to a plain bool. The unified pointer-based settings endpoint supports false.

**Impact:** A metadata edit silently turns off a permission control and enables ordinary-member list/item mutations. Requested protected creation is also silently permissive.

**Repair/acceptance:** Persist create intent, isolate metadata changes from settings, and distinguish omitted from explicit false. Test create true, rename with/without details, denial of member writes after rename, and true/false/omission through both settings APIs.

### F08 — Generated message models break the guarded compatibility boundary

**Evidence:** Current ignored `.gen/miltech_ng/public/model/shop_messages.go:23` has `InsertionNumber *int64` tagged `json:"insertion_number"`. `api/response/user_shops_response.go:65–68` embeds the model. Legacy message reads use `ShopMessages.AllColumns` at `api/shops/messages/repository_impl.go:104,123,141,166,289`; create reads back the inserted row inside its transaction. The generated table also includes the new column.

**Fresh failures:** Nine message sync/legacy tests failed on an unexpected `insertion_number` response key. A temporary disposable pre-018 reproduction, with sync disabled, made message POST fail and confirmed the inserted row rolled back. Direct legacy reads also select the nonexistent column. The aggregate and raw sync projections avoid that SQL coupling, but still embed the changed model for JSON and gain a null key.

**Impact:** This checkout violates the exact legacy key contract. A binary compiled with these generated inputs cannot use the legacy message CRUD paths on an unmigrated database, despite the flag. This is a build-input-dependent outage; it does not prove the production schema or deployed binary has this state.

**Root/build boundary:** `main.go:32,56–95` regenerates ignored code on every startup. Generation happens after compilation: it cannot change the running binary, but changes future build inputs. It also requires startup catalog access/writable storage, hardcodes generation port 5432 despite a configurable connection port, and crashes on failure. `tools/jetregen/main.go:50–55` gives every column a public JSON tag; it lacks the special exclusion required by ADR-022. Docker copies locally generated inputs.

**Repair/acceptance:** Use explicit stable legacy projections and a stable public DTO; hiding the JSON field alone does not fix AllColumns SQL. Make generation a controlled build/developer step with reproducible schema inputs and intentional internal-field tags. Verify both pre-018 and post-018 schemas, flags off/on, message create/read/update/delete, aggregate rows and exact JSON keys. Retain unrelated tracked generated-model checks.

### F09 — A shared database credential is committed in a nominal unit package

**Evidence:** `api/shared/db/transaction_test.go:12` declares a fixed credential-bearing PostgreSQL URI; tests at `:14–47` use it directly. They bypass disposable database validation, and one executes DDL/DML. The credential is deliberately omitted from this report and tool output.

**Impact:** `go test ./api/...` can contact a shared target unexpectedly; anyone with access to the source/history can recover the credential. Removal from the current file does not revoke it or erase history. Its present validity and target identity were not tested.

**Fresh verification boundary:** This package was explicitly excluded from execution. It was compiled/vetted. The remaining test packages ran with `TEST_DB_URL`, `TEST_DATABASE_URL` and `TEST_DATABASE_MARKER` unset so optional unrelated repository tests could not inherit a target.

**Repair/acceptance:** Replace these helper tests with a network-free SQL driver or marker-validated disposable integration tests. Have the credential owner rotate/revoke the exposed credential and assess historical exposure. Test that unit runs never dial a database; retain meaningful commit/rollback behavior checks.

### F10 — Release packaging includes credential-bearing files

**Evidence:** `Dockerfile:29` performs `COPY . .`; no `.dockerignore` exists. Local `.env` and `fire_auth_key.json` exist (contents were not read). `Dockerfile:45` explicitly copies the Firebase key into the final image. Gitignore does not exclude files from Docker's context or image layers.

**Impact/condition:** Building this workspace sends local credentials into the build context/builder layer and puts the Firebase service-account key in the runtime artifact. Anyone able to pull/export that image can obtain the key. The precise privileges, registry exposure and deployed image were not inspected; this is packaging evidence, not a claim that the key is publicly exposed.

**Repair/acceptance:** Exclude local secrets from the build context and provide runtime Firebase credentials through deployment-managed secret/workload identity mechanisms. Verify image/context contents without printing secrets and preserve functional authentication. Rotate exposed keys when the artifact's actual distribution requires it. This is within scope because it supplies the Shops authentication credential.

### F11 — Cross-Shop reply parents bypass lifecycle isolation

**Evidence:** `api/request/shops_request.go:55–59`, `messages/handler.go:38–42`, `service_impl.go:56–72`, `repository_impl.go:70–87`. ParentID passes unchanged after destination-Shop authorization. The FK verifies global existence, not same-Shop ownership.

**Trigger:** Create a B reply whose parent is a message in A. After removal from B, deleting the still-authorized parent in A can remove B replies/descendants under the pinned `ON DELETE CASCADE` FK (`tests/testutil/shops_schema_baseline.sql:5077–5078`).

**Qualification:** Migration 005 declares SET NULL, while the approved physical baseline and current cascade tests use CASCADE. Foreign-parent acceptance is source-confirmed; cascading unauthorized deletion is confirmed for the fixture, not a freshly inspected live schema.

**Repair/acceptance:** Verify parent existence and destination identity in the create transaction. Preserve intended same-Shop cascade semantics. Test inaccessible/foreign parents, same-Shop replies, concurrent parent deletion and no cross-Shop cascade. Audit historical relationships before introducing constraints or rewriting data.

### F12 — Usage-only vehicle updates erase metadata for privileged callers

**Evidence:** `api/shops/vehicles/service_impl.go:149–165` routes usage-only PUT to the preserving writer only when the caller is neither creator nor admin. Creator/admin instead reaches the full writer at `:174`; `handler.go:213–225` maps omitted strings to empty values, and `repository_impl.go:85–93` persists them. UOC defaults to UNK.

**Reproduced:** On a populated vehicle, an owner sent only vehicle ID/admin/base and tracked readings. HTTP 200 cleared NIIN, model, serial and comment and set UOC to UNK. Ordinary-member coverage asserts preservation, but its owner request supplies full details (`tests/shops/shops_vehicles_test.go:580–619`). No client source is needed to establish that this supported server payload behaves differently by role.

**Repair/acceptance:** Decide usage-only intent independently of privilege; keep all unrelated metadata for the same partial request from member, creator and promoted admin. Preserve intentional full metadata updates and the documented usage reconciliation contract. Test database fields after each response, not just status.

### F13 — Invite revocation is not atomic with admission

**Evidence:** `api/shops/members/service_impl.go:41–59` reads active invite state then separately creates membership. `invites/repository_impl.go:42–53,85–127` reads/deactivates/deletes independently. Membership insertion receives no invite identity or state.

**Interleaving/impact:** Join reads active; administrator revokes/deletes and commits; delayed join creates membership. The admission credential no longer exists or is inactive, yet it grants member access.

**Repair/acceptance:** Admission must resolve and lock Shop/invite, recheck active validity and create membership in one transaction coordinated with revocation. Test claim-before-revoke and revoke-before-claim with deterministic connections, including deletion and expired/limited credentials if implemented. Late denied claims must leave no membership.

## Other correctness and UX findings

### F14 — Duplicate admission can demote a concurrently promoted user

`api/shops/members/service_impl.go:50–59` performs a preflight not-member check; `members/repository_impl.go:95–105` upserts with `DO UPDATE role=<requested role>`. Two joins can pass the check. One creates membership and an admin promotes it; the delayed join writes role `member` over that promotion. This is a source-confirmed legacy race. Use admission insert semantics that never alter an existing role; promotion owns role changes. Test duplicate claims, response-loss retries and promotion between pending claims. Preserve an intentional stable already-member outcome.

### F15 — Shop creation can succeed without creator membership

`api/shops/core/service_impl.go:47–60` commits the Shop first and merely logs failure to add its admin membership. The handler can still return 201. A constraint/transient failure leaves an inaccessible memberless Shop; retry creates another ID. Create Shop and first membership in one transaction. Inject failure of the second insert and assert neither row persists and no successful response is sent. This predates the recent refactors.

### F16 — Departure decisions are outside the lifecycle transaction

`api/shops/members/service_impl.go:82–94` chooses remove versus delete from an unlocked count. Two members can both see count=2 and choose removal; serialization in `members/repository_impl.go:114–152` does not reevaluate final cleanup, so both memberships disappear while the Shop survives. Separately, after the creator leaves, a last remaining noncreator enters cleanup but `repository_impl.go:271–287` requires the original creator, preventing departure.

Perform current count/last-member transition under the coordinating Shop lock and distinguish explicit creator-only deletion from internal final-member cleanup. The existing revalidation document records creator-only explicit deletion and successor-admin handoff as prior design decisions; this review does not authorize expanding destructive permissions. Test two simultaneous leaves, leave versus removal, creator-first departure and noncreator final departure. Verify no inaccessible memberless Shop or trapped sole member remains. Do not silently choose a new ownership policy while repairing the transaction boundary.

### F17 — Promoted admins cannot rename despite passing the service policy

`api/shops/core/service_impl.go:67–76` admits any current admin, but the rename WHERE adds `Shops.CreatedBy=user` (`core/repository_impl.go:89–92`). A promoted admin passes authorization then matches no row. Fix the rename rule consistently at service and persistence layers and test both original and promoted admins. **Creator-only explicit deletion is not reported as a defect**, because the prior design document selects it; final-member cleanup remains F16. F01 currently makes service failure presentation worse.

### F18 — New enrichment is lost through the supported legacy replacement sequence

Atomic save preserves omitted nickname/unit on an existing item UUID (`api/shops/vehicles/notifications/atomic_save_repository.go:205–225`). Legacy DELETE removes the row; subsequent legacy INSERT creates a new UUID with omitted fields as NULL (`items/legacy_mutation.go:42,112–125`). It has no durable logical-item enrichment to recover.

**Reproduced:** Add quantity 2, nickname Front hub, unit KT; DELETE; re-add the same notification/NIIN with quantity 3 and omitted fields. The replacement persisted NULL nickname/unit. Thus omission-preserving atomic UPDATE does not cover separate legacy replacement. This is a September enrichment compatibility risk, not proof of a particular client implementation.

The existing revalidation document records the intended retention policy. A repair must establish logical item identity and collision behavior: physical UUID uniqueness does not ensure one NIIN per notification. Preserve enrichment according to that policy without resurrecting removed quantity, borrowing another notification's fields, or replaying stale explicit edits. Test the full legacy sequence, delayed/retried re-add, multiple units/duplicate NIINs and alternating old/new writes. Existing data needs an approved collision decision before new uniqueness/backfill work.

### F19 — Effective base usage can be negative

`api/request/shops_request.go:73–74,85–86` and vehicle create/full update have no nonnegative base-reading validation; only tracked readings are validated/constrained. `vehicles/repository_impl.go:183–201` falls back to base mileage/hours when tracked values are nil. Creating mileage=-100 can therefore make small normal adjustments fail range checks later. Validate new base input consistently with the effective usage domain; inspect existing invalid rows before any migration/data correction. Test both fields, omitted defaults and nil tracked readings. August hardening did not close this legacy gap.

### F20 — Equipment admin-number edits are silently discarded

The request requires `admin` (`api/request/shops_request.go:78–89`), and `vehicles/handler.go:215` passes it through, but `repository_impl.go:85–102` omits the Admin SET. The temporary owner PUT supplied NEW-ADMIN; persisted admin remained the original value. This legacy defect makes apparent renames fail silently and affects displayed audit labels. Save Admin on authorized metadata edits, while keeping usage-only intent restricted. Verify subsequent GET/database value for creator/admin and that ordinary usage updates cannot rename it.

### F21 — Equivalent writers enforce different input rules

| Path | Source evidence | Accepted invalid input / effect |
| --- | --- | --- |
| List-item bulk | `api/request/shops_request.go:180–183`; `lists/items/handler.go:165–186` | Slice lacks `dive`; nested omitted fields bypass required validation. Reproduced `{list_id:L, items:[{list_id:L}]}` persisted empty NIIN/name and quantity 0 |
| Notification-item bulk | `shops_request.go:117–128`; `notifications/items/handler.go:132–154`; `items/service_impl.go:164–173` | Same missing nested validation; optional enrichment checks do not validate core fields |
| Single/update list items and single notification items | Quantity uses `required` without positive-domain rule | Negative nonzero quantities pass; whitespace required text passes |
| Notification legacy PUT | `notifications/service_impl.go:231–278`; `legacy_mutation.go:68` | Required string type accepts ZZ/whitespace, while create and atomic allow M1/PM/MW |

Pinned schemas permit the malformed core values. Some overlong legacy values reach varchar limits and fail in SQL rather than validation. Apply equivalent positive-quantity, trimmed required-content, field-length and supported-type rules to all relevant writers. Preserve supported NIIN/NSN formats; do not invent a narrower format/cap. Test an invalid second bulk element rejects the whole request, all single/update variants, unsupported types and no row/audit changes on failure.

Notification bulk additionally requires an outer `notification_id` but ignores it (`shops_request.go:126–128`, `items/handler.go:139–157`): outer A plus all nested B mutates authorized B. This is a target-consistency bug, **not an IDOR**, because persisted nested-target authorization protects B. Compare outer/nested IDs and test contradictory requests with no writes. In contrast, multi-list batches across several lists in one Shop are explicitly supported; their outer list ID is not treated as an authorization defect.

### F22 — Context threading does not reach transaction/database lifetimes

Handlers pass request context, but many repository/auth interfaces do not. List/member/settings/legacy notification writers use Background contexts for transaction and locks; equipment-service calls also lose context before DB/auth/username operations. Examples: `lists/repository_impl.go:174–180`, `lists/items/repository_impl.go:334–339`, `members/repository_impl.go:115–120`, `settings/repository_impl.go:115–120`, `notifications/legacy_mutation.go:17,42,78`, `items/legacy_mutation.go:22,73`, `equipment_services/core/repository_impl.go:28–40,94–120`.

**Direct recent regression:** `1a5733f` replaced PATCH usage's `BeginTx(ctx,nil)` with `sharedb.WithTx`; `api/shared/db/transaction.go:16` uses `Begin()`/Background. QueryContext still cancels individual queries, but pool acquisition and transaction lifetime no longer share HTTP context. Cancellation between the final query and Commit no longer has the original transaction cancellation behavior.

Abandoned requests can keep occupying connections/Shop locks and later commit; retries of non-idempotent legacy creates can duplicate work. Thread context through every boundary, using context-bound transactions and statements. Test canceled pool acquisition, blocked Shop/vehicle locks, cancellation before persistence/commit and lock release. Cancellation cannot undo an already committed mutation or prove the outcome of an ambiguous response.

### F23 — Atomic writes do not make multi-query aggregate reads atomic

`api/shops/aggregates/repository_snapshot.go:144–216` reads summary and sections separately. `repository_notifications.go:57–93` reads headers then items; maintenance service composition and PMCS history similarly span statements. Each uses the pool without a shared read snapshot.

An atomic writer can commit between header/items or inspection/count queries, yielding old headers with new items or inconsistent totals/sections. This is source-confirmed; no deterministic tearing reproduction was run. Use one read-only REPEATABLE READ transaction/queryable connection for each aggregate and its authority check. Plain default READ COMMITTED still permits per-statement versions. Test a writer committing between component reads and verify the response describes one version.

### F24 — Legacy message cursors can lose messages

`api/shops/messages/repository_impl.go:160–176` compares only created_at and orders only timestamp. `service_impl.go:151–168` truncates and uses the last returned ID as continuation. Timestamp ties are excluded by strict subsequent bounds. For newer polling, newest-first truncation does not drain a burst: if 10 newer messages exist and limit=2, returning 10/9 then continuing after 9 returns 10 again while 1–8 are never drained.

Use deterministic timestamp/ID bounds plus direction-appropriate continuation while retaining the legacy envelope. Test equal timestamps over several pages, newer bursts, both directions and deleted anchors. V2 fixes healthy-path history/catch-up ordering but does not repair these existing endpoints.

### F25 — Upload resource consumption precedes the 5 MiB check

`api/shops/messages/handler.go:204–225` calls PostForm/FormFile before testing header.Size; membership is later in `repository_impl.go:308–315`. There is no request bound before multipart parsing. An authenticated nonmember can force substantial parsing, bandwidth and temporary-storage work before rejection. Proxy limits were not inspected, so the source establishes an application-boundary gap rather than a measured production denial of service.

Apply a total body bound before parsing, with intentional multipart-overhead allowance, and reject before unnecessary Azure work. Test oversized fields/multiple parts as well as a large file, nonmember uploads and missing Shop ID; measure bounded body consumption and no Azure calls.

### F26 — Editing completed service metadata changes its historical date

`api/equipment_services/core/service_impl.go:152–160` sets CompletionDate=now whenever is_completed=true and the optional date is omitted, even if already complete. `completion/repository_impl.go:24–43` also overwrites completion time for repeated no-date completion calls. Description edits/retries can turn historical maintenance dates into edit/retry dates.

Preserve existing completion time for omission on already-completed records, retaining explicit date edits and reopen semantics. Test metadata edits, duplicate/response-loss completion, concurrent completion, explicit date and reopen. This is legacy lifecycle behavior; fixing it does not require activating date-only service mapping.

### F27 — Service queries silently ignore filters and have unstable tie order

`api/request/equipment_services_request.go:31–39` accepts status, but `queries/repository_impl.go:25–56` never applies it. Equipment-specific `queries/route.go:78–110` binds is_completed/service_type/status then passes only dates/limit/offset. Accepted filters can return unrelated rows and counts. Define/enforce supported semantics or reject unsupported filters explicitly; test completed/open/type filters on both routes. Preserve the unresolved service-date mapping gate.

Both paged queries also lack a unique ordering key (`queries/repository_impl.go:79–82,126–129`): shop order is created_at only; equipment order is service_date only. Equal/null dates can produce duplicates/omissions across pages. Add a deterministic ID tie-breaker/intentional null ordering and test ties. Offset paging still shifts under concurrent inserts; a tie-breaker alone cannot guarantee a stable multi-request snapshot.

Malformed RFC3339 input is inconsistently classified. Shop list/calendar errors are dynamic wrapped time.Parse failures (`queries/repository_impl.go:42–54`, `calendar/service_impl.go:37–44`), so contract 2 classification defaults to internal_error after F01 is repaired; the equipment handler returns validation 400. Classify these known input failures safely and consistently without exposing parse internals or inadvertently changing negotiated legacy statuses.

### F28 — Security-related invite options have no effect

`api/request/shops_request.go:39–43` accepts max_uses/expires_at; `invites/handler.go:29–37` forwards only Shop ID. `invites/service_impl.go:57–66`, current generated invitation model and repository insert have no corresponding persistence. Join checks active status only. A one-use or already-expired request can create an active reusable invitation.

Implement validated atomic expiry/use accounting or explicitly reject unsupported controls. Do not silently advertise restrictions. Test invalid dates/limits, expired credentials, one-use simultaneous claims and intentional unlimited invites. Present validity of real codes was not tested.

### F29 — Documented privilege checks test any privilege, not all

`docs/testing/shops-database.md:422,428` uses comma lists in `has_table_privilege(...,'INSERT,UPDATE[,SELECT]')`. PostgreSQL treats that as **any** listed privilege. A SELECT-only app role can pass the counter-table gate while the migration-018 invoker trigger's INSERT/UPDATE fails on every subsequent legacy or sync message send. Turning sync off does not disable the trigger. [PostgreSQL 14 privilege inquiry documentation](https://www.postgresql.org/docs/14/functions-info.html)

The separately required real application-role insert should catch this in the test target; therefore this is a false-positive gate, not proof of a live outage. Use separate privilege checks combined with AND, verify grants/default-privilege assumptions for the newly created table, and rehearse with SELECT-only/INSERT-only/UPDATE-only versus fully authorized roles. Preserve the real legacy-shaped send test before production.

### F30 — Migration 017's guide violates the generator and database workflow

`docs/migrations/shop_notification_item_nickname_uom_migration.md:28–30` instructs applying to miltech_ng, using plain `jet`, then applying to miltech_ng_test. This conflicts with AGENTS' explicit prohibition of plain Jet and the established test-first identity-checked workflow. Plain Jet can wipe ignored outputs and omit snake_case JSON tags, affecting public APIs beyond the new fields.

Correct the guide to the tagged `go run ./tools/jetregen` workflow with approved identities/target order and canonical migration procedure. Do not execute it as part of this review. After controlled generation, verify tracked generated files, legacy response keys and F08's internal-field handling. This documentation regression was introduced September 29 (`e3656e2`).

## Verification and lower-impact contract defects

### F31 — Capability registration panics on nil Env

Fresh safe API tests failed at `api/shops/route.go:35`, dereferencing `deps.Env` through `route.Setup(nil,router,nil,nil,nil)`. New atomic/message flags added mandatory reads despite route setup/tests supporting absent config elsewhere. Production bootstrap normally supplies Env, so this is a registration/test contract failure, not a demonstrated production startup panic with valid config.

Use consistent explicit defaults or require/validate configuration at the supported boundary. Run the full production route tests, not only one repaired test. The panic aborts the route package before later cases can execute.

### F32 — Notification enrichment test asks for the wrong audit representation

Fresh `TestAtomicNotificationItemFieldsSurviveReleasedClientSaves` fails at `tests/shops/shops_notification_item_fields_test.go:74` (expected 1, actual 0). It queries change_type=items_updated, but production intentionally stores change_type=update with an items_updated payload (`atomic_save_repository.go:245–257`). The scenario also makes two real changes to item 0, so merely changing the queried kind may still leave an incorrect count expectation.

Field assertions before that line passed. This failure is not evidence that atomic omission preservation is broken. Assert each operation's actual update-kind/payload and default-only no-change behavior, then rerun the isolated suite. Do not change production audit semantics solely to satisfy this assertion.

### F33 — Isolation-wrapper Python tests cannot import

Fresh `python3 scripts/test-shops-isolated_test.py` exits 1: `ValueError: substring not found` at line 8. Its source slicing expects `psql_local -v marker=`, but the current wrapper supplies an explicit database argument. The real wrapper provisioned, verified migrations, executed tests and removed its clusters successfully in this review.

Repair the test's dependency on obsolete textual anchors and cover current helper arguments/target refusals. Passing provisioning does not make a broken regression test green.

### F34 — Legacy item history loses enrichment detail

`notifications/items/service_impl.go:326–346,363–377` builds old NIIN/name/quantity-only audit payloads. `items/legacy_mutation.go:112–125` snapshots removal without nickname/unit. Quantity 3 with unit DZ is recorded without information distinguishing three dozen from three each, and after removal the row no longer preserves its label/unit.

Include item identity and persisted enrichment in add/remove snapshots before deletion, retaining explicit legacy best-effort audit behavior unless separately changed. Test nondefault units/nicknames, null/default values, single/bulk operations and enriched removal. This became materially incomplete with September 29 field persistence.

### F35 — Complete PMCS history exceeds a hard bind-parameter limit

`api/shops/aggregates/repository_pmcs_history.go:67–76,79–98,105–124` loads accessible inspections then binds one UUID per inspection into fault/comment count IN lists. Pinned lib/pq v1.10.9 rejects more than 65,535 parameters; 65,536 accessible inspections therefore fail the whole history endpoint. The equipment-ID query also grows a bound list. This is a demonstrated source/driver ceiling, not a measured current customer dataset or immediate outage.

Use membership-scoped relational queries, array binding or bounded batches without truncating complete history. Test construction beyond the limit and a meaningful disposable capacity fixture. The September repository split preserved this function; no provenance regression was found.

### F36 — New strict batch authorization narrows stale-delete compatibility

`api/shops/lists/items/repository_impl.go:339` invokes `AuthorizeListMutation` for every requested item; `shared/authorization.go:278–286` rejects any missing one. Earlier flow already required the **first** item to exist in service preflight, but later missing IDs were tolerated by DELETE IN. Concurrent deletion/retry can now fail the entire otherwise valid batch when a non-first ID disappears.

This tightening also closes real foreign-item deletion holes; retain that protection. If idempotent stale deletion is the intended compatibility rule, authorize every surviving row, define all-missing behavior, and repair the service's first-ID anchoring as well. Never infer authority solely from one survivor. Test stale non-first/first/all IDs, foreign survivors, mixed Shops and current policy under concurrent deletion.

### F37 — Bulk deletion counts report request length

`api/shops/lists/items/handler.go:219–229` and `notifications/items/handler.go:227` report len(item_ids). SQL deletes each ID once; notification deletion deduplicates and tolerates absent IDs (`items/legacy_mutation.go:83–84,107–114`). Duplicate list IDs overreport; duplicate/missing notification IDs can report 3 when one row was removed.

Return actual affected counts or explicitly validate duplicates within the established envelope. Test duplicate IDs, partial misses, distinct IDs and concurrent removal. Clarify requested-count semantics if preserving legacy behavior instead of changing it.

## Conditional concerns and explicit release limits

These are not additional unconditional production bugs or claims of current deployed state.

1. **Mixed-writer message deadlock after 018.** Current message create locks shops FOR UPDATE, then inserts/counter trigger; old-style writers insert/counter first, then acquire the Shop FK key-share lock. Each can wait on the other's lock. Source: `messages/repository_impl.go:70–87`, `shared/authorization.go:264–269`, `migrations/018_add_shop_message_insertion_numbers.sql:85–92`. A uniform current fleet does not have this normal insert inversion. Existing raw-insert concurrency tests do not exercise a mixed old/current writer. Rehearse a deterministic mixed-writer cycle; avoid overlap or introduce a reviewed consistent-order/retry strategy. Do not silently modify pinned 018 or weaken membership serialization.
2. **Capability does not prove complete readiness.** Atomic capability is the raw flag (`shops/route.go:33–35`, `capabilities/route.go:39`); no atomic schema/access probe exists. A flag-on wrong/missing 016/017 target advertises readiness but saves fail. Current rollout design intentionally relies on operator/fleet gates and permits callable-before-advertisement activation; retain that intent. Message readiness checks counter-table/trigger presence, not every column/index/function/grant/backfill/fleet invariant. Capability true is not a substitute for verifying the exact release build and application role.
3. **Counter reset/reapply recovery.** Catch-up returns invalid when after/through exceeds the current counter (`messages/sync_repository.go:206–211`). Restore, wrong target or rollback/reapply can make a previously valid watermark unusable. The server does not distinguish this recovery case from malformed input. Provide a deliberate recoverable/reset signal if clients need automatic reinitialization; do not hide a fleet/target mismatch. No client polling behavior was reviewed.
4. **Unnumbered messages can escape catch-up validation.** NULL insertion numbers are filtered out before scanner validation at `sync_repository.go:212–224`; a catch-up can advance without returning them. Initial/history/reconcile fail closed only for encountered/requested rows. This requires broken/bypassed numbering or external changes, not healthy writers, and the runbook acknowledges it. Rehearse catch-up under a broken trigger/data invariant before treating fail-closed coverage as exhaustive.
5. **Per-Shop write contention is unmeasured.** Transaction authorization serializes many writers on the Shop row. Atomic requests have a 1 MiB body bound but no item-count bound and several statements per changed item. Canceled legacy work worsens the contention (F22). Aggregate omitted limits deliberately mean unlimited; changing defaults/caps would change the established contract. Measure representative concurrency/payloads before selecting batching or budgets; there is no measured production latency limit here.
6. **Invite guessing and removal/rejoin policy.** Codes contain 32 bits of randomness (`invites/service_impl.go:159–166`); Shops does not register its own rate limiter in the reviewed route chain. Edge limits and active-code counts are unknown. The expected attempts against N active uniformly random codes are approximately 2^32/N. Verify edge/application throttling before judging practical exposure. A removed user with a retained active code can rejoin; any ban-versus-removal policy needs an explicit decision.
7. **Blob lifecycle leaks remain.** Cascaded reply deletions clean only the prefetched parent's image; edits do not reconcile removed asset references; multiple markers clean only the first. Cleanup failures log and return success, with no durable retry/reconciliation shown. Quantify storage impact and design reference-aware cleanup together with F03/F04 rather than broadening unsafe URL deletion.
8. **History semantics are limited.** Legacy audits commit after business writes and can fail best effort; atomic audits commit with the operation. History labels prefer current title/admin over snapshots; empty history can be data:null; equal audit timestamps have no deterministic event tie order. Some histories are unbounded and Shop history caps at 500. These are actual source behaviors requiring contract decisions or capacity measurements, not all demonstrated bugs. Receipt committed_at and resource timestamps represent different events; timestamp equality is not established as a requirement.
9. **Service date activation remains deliberately unavailable.** service_dates/service_reads are false and the date-only parser is disconnected. `docs/testing/shops-service-date-mapping.md` retains an unresolved mapping gate. No live timestamp/session-timezone interpretation was inspected; do not infer it from historical migrations or activate the parser as a review repair.
10. **Live migration runners remain deliberately gated.** The message runner contains UNPINNED schema checksums and refuses before contact. That is an operator preparation gate, not evidence of a broken migration. Required application-role proof, named-target identity, fleet consistency and migration order remain unverified. Migration 017 rollback discards nickname/unit values as documented; no rollback against an existing target is authorized by this audit.

## Fresh verification results

All commands used `rtk`; raw output was retained where needed. Actual local executable reported `go1.23.3 darwin/arm64`; module declares Go 1.23.0/toolchain 1.23.4. Test results apply to this local environment and exact checkout/generated inputs.

| Check | Fresh result | Meaning / limits |
| --- | --- | --- |
| `go build ./...` | Exit 0 | Compilation only; application was not started and schema was not regenerated |
| `go vet ./api/... ./bootstrap/... ./tools/...` | Exit 0 | Includes compilation/static checks of the credential-bearing test; it was not executed |
| Safe API + bootstrap tests, `-count=1 -json` | Exit 1 | 83 packages selected, excluding only api/shared/db; 39 package passes, 43 no-test skips, 1 failed package. JSON events: 1,050 test/subtest passes, 3 skips, 2 failures belonging to one panic tree; not 1,050 independent scenarios |
| Production route unit package | Fails | Nil Env panic aborts the package; later tests not completed |
| Isolated Shops + equipment services, `-v -count=1` plus both migration verification switches | Exit 1 | 138 top-level tests passed, 10 failed, 2 skipped across the two packages; Shops failed, equipment services passed |
| Disposable migration verification | Six successful checks | Populated notification upgrade; empty reverse; populated reverse refusal preserving receipt; populated message upgrade; NULL-created_at refusal with no partial column; message reverse/reapply |
| Isolation-wrapper Python tests | Exit 1 | Import-time substring failure, F33 |
| Temporary audit overlay reproductions | Eight observations confirmed | Six in first run, then removal-race and replacement-metadata tests; these assert current defects, not repairs or additional independent passing release coverage |
| `git diff --check` | Exit 0 | No intentional application/source changes |

The integration command was:

```sh
env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  GOCACHE=/private/tmp/shops-server-production-review-20261002/go-cache \
  rtk proxy bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  -v -count=1 ./tests/shops ./tests/equipment_services
```

The wrapper's advisory lock serialized the two packages' shared-table access. Initial sandbox loopback binding was denied; the same approved disposable test scope was then run with tool approval. There was no switch to an existing target. The first attempt to add a virtual overlay test file failed Go vet's file lookup; mapping the temporary tests over an existing test file allowed execution without editing that file.

Nine message failures were all exact-key assertions exposing insertion_number:

- TestMessageSyncRoutesInitialReturnsWatermark
- TestMessageSyncRoutesInitialEmptyShopStartsAtZero
- TestMessageSyncRoutesCatchUpBurstOver100HoldsBound
- TestMessageSyncRoutesCatchUpSkipsDeletedGaps
- TestMessageSyncRoutesHistoryEqualTimestampsAndDeletedAnchor
- TestMessageSyncRoutesReconcile
- TestMessageSyncRoutesReconcileParentDeleteCascades
- TestMessageSyncRoutesMemberRemovedBetweenChunks
- TestMessageSyncLegacyCompatibility

The tenth failure was TestAtomicNotificationItemFieldsSurviveReleasedClientSaves, explained in F32. Performance tests were opt-in and skipped; race/load tests, production container build, live Firebase/Azure calls, production schema/grants/identity inspection and deployed-fleet validation were not performed.

Temporary evidence files are under `/private/tmp/shops-server-production-review-20261002/`: unit.log, integration.log, reproductions.log, concurrency-replacement.log, audit_review_test.go and overlay.json. They contain only review/disposable evidence; the real shared credential was never printed. The private 714 MiB Go cache was removed after verification; the small evidence files remain. The report's conclusions and reproduction descriptions above are self-contained even if temporary evidence is later removed.

## Protections verified in source and existing test coverage

- Lists/items and notification mutations reauthorize persisted ownership and current membership under coordinating locks. Mixed-Shop batches and foreign references are rejected before writes. No direct current list/notification-item IDOR was confirmed.
- List deletion blocks equipment-service dependencies, including completed history, and detaches notification references transactionally. Attachment writers coordinate referenced-list locks and verify Shop ownership.
- Atomic notification save claims a user-scoped operation, checks fingerprint conflicts, reauthorizes replay, uses deterministic create identity, writes rows/audits/receipt together and preserves receipt-only replay. Injected item/audit/receipt failure rollback tests passed. The F32 test failure should not be mistaken for broken same-ID omission preservation.
- Sync reads use a read-only repeatable-read transaction with membership and watermark in the same snapshot. Commit-ordered trigger allocation, bounded held catch-up, deleted-number gaps, keyset history and Shop-scoped reconciliation are coherently implemented. An unsigned history cursor is pagination input; forging it does not bypass Shop authorization.
- Unknown error text is redacted by the scoped response writer/classifier. In particular, the upload handler's formatted error uses response.Error, so the current registered Shops chain sanitizes it; it is not confirmed as an internal-error text leak. F01 concerns c.Error paths that write no response.
- The custom/guide PMCS history union carries provenance fields without exposing authored checklist trees. The repository split preserved that behavior. Capacity/snapshot issues remain F23/F35.
- The disposable wrapper rejects inherited targets/unsupported packages, verifies its marker before tests and cleans clusters. No existing-database or production acceptance is implied by those checks.

## Recommended repair and release sequence

1. Close asset/service authorization holes and every uncovered transaction-authority path (F02–F06, F11, F13). Reproduce with controlled security cases and real disposable concurrency; retain all current permission protections.
2. Repair error assembly, list-restriction intent, role-independent usage updates and admission/departure lifecycle (F01, F07, F12, F14–F17). Define any remaining ownership/admission policy explicitly; preserve prior creator-only destructive intent.
3. Make release inputs reproducible and remove credential exposure/unsafe operational instructions (F08–F10, F29–F30). Rotate actual exposed credentials through their owner. Do not infer successful rotation from source cleanup.
4. Cover legacy enrichment replacement, equivalent input validation, metadata persistence, service/history/pagination behavior and cancellation/read snapshots (F18–F28, F34–F37). Resolve data collision/mapping gates before schema/data changes; preserve accepted legacy payloads and negotiated status rules.
5. Repair verification defects (F31–F33), rerun safe unit/static checks and full disposable suites, then add the previously missing security/lifecycle scenarios. A known failure should remain visible until resolved; neither a focused passing run nor a bad assertion is release evidence.
6. Separately verify exact production binary/generated inputs, existing-database identities/schema/grants, application-role legacy writes, mixed-writer cutover, fleet flags, proxy upload limits and representative performance. Existing targets require their own authorization and identity checks. Shipment, migration activation and deployment remain outside this review's completed scope.

This review identifies repair work and provides evidence; it does not authorize implementation, existing-database access or production rollout.
