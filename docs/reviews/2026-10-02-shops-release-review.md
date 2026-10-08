# Shops pre-release code review — 2026-10-02

**Scope:** `cleanup` branch vs `main` (50 commits, `d3b18ce..4abd3a1`), focused on `api/shops/**`, migrations 016–018 and the shops integration tests. About 9,000 lines added and 2,100 removed across the shops feature.

**Method:** static review of the diff and the surrounding code, plus `go build ./...`, `go vet` and `go test ./api/...`.

**Not verified:**

- The isolated integration suite was **not run**: `scripts/test-shops-isolated.sh` requires `rg` on `PATH` (`brew install ripgrep`).
- Three items were **not reviewed** because tool access to them was denied during the session:
  - `api/shops/messages/service_impl.go`
  - the `response.ShopMessageResponse` type
  - `.gen/miltech_ng/public/model/shop_messages.go`. One line of this file surfaced in a broad grep and drives finding C1; nothing else in it was read.

## Summary

| ID | Severity | Finding | New in this build? |
|----|----------|---------|--------------------|
| C1 | **Critical** | Local `.gen` `shop_messages` model has `InsertionNumber` tagged `json:"insertion_number"`. A build from this tree leaks the field into legacy message responses, and breaks every message read and send on any DB without migration 018. | Yes (018) |
| H1 | **High** | Renaming a shop silently resets `admin_only_lists` to `false`. | No (Jan 2026), still ships |
| H2 | High (UX) | Non-creator admins cannot rename a shop; they get a generic error. | No |
| M1 | Medium | Unit suite is red: `api/route` test panics on nil `Env`. | Yes |
| M2 | Medium (UX) | Batch list-item removal now fails the whole batch if any item is already gone. | Yes |
| M3 | Medium | Lock-order inversion between message inserts: deadlock risk during a rolling deploy, with no retry. | Yes (018 + new lock) |
| M4 | Medium (UX) | New error messages that released clients can only show as "Unable to complete Shops request" / HTTP 500. | Yes |
| M5 | Medium (UX) | Message catch-up answers 400 when the client watermark is ahead of the server; the client keeps failing until the user re-enters the shop. | Yes |
| M6 | Medium | Vehicle and shop-update mutations still use check-then-act authorization outside the transaction. | Partly |
| M7 | Medium (ops) | `generateSchema` runs on every startup and `log.Fatalf`s on failure. It is also the root cause of C1. | No |
| L1–L6 | Low | Timestamp drift in receipts, unbounded atomic-save batch, dead code, audit gaps, legacy cursor skip | Mixed |

The core designs hold up well; see "Verified correct" at the end. The release blockers are **C1** and **H1**. M1 should be fixed so CI is green before tagging.

---

## Critical

### C1. Regenerated `shop_messages` model leaks `insertion_number` and couples every message path to migration 018

**Evidence**

- `.gen/miltech_ng/public/model/shop_messages.go:23`: `InsertionNumber *int64 \`json:"insertion_number"\``
- [api/shops/messages/repository_impl.go:46-49](../../api/shops/messages/repository_impl.go#L46-L49): `shopMessageResponseRow` embeds `model.ShopMessages`.
- Every legacy read selects `ShopMessages.AllColumns`:
  - `getShopMessageResponseByID` (line 104)
  - `GetShopMessages` (line 123)
  - `GetShopMessagesPaginated` (line 141)
  - `GetShopMessagesByCursor` (line 166)
- [Dockerfile](../../Dockerfile) does `COPY . .` with no `.dockerignore`. `.gen/shop_messages.go` is not tracked in git, so the image ships whatever `.gen` is on the build machine.

**What happens if this tree is built and deployed**

1. **Against a DB with 018 applied:** every legacy message response gains an `insertion_number` key. That breaks the legacy-shape guarantee in ADR-022 and `docs/testing/shops-release-contracts.md`. `TestMessageSyncLegacyCompatibility` exists to catch this, but it is an integration test and was not run.
2. **Against a DB *without* 018 (production today, per the runbook):** `AllColumns` makes Jet emit `shop_messages.insertion_number`, so Postgres raises `column "insertion_number" does not exist`. All three message list endpoints return 500. `CreateShopMessage` reads the row back inside its transaction (line 90), so the insert **rolls back**: users cannot send messages either. The Messages tab is fully broken.

**Root cause:** `generateSchema` in `main.go` regenerates `.gen/` on every `go run .` from whatever DB the env points at. Running locally against a migrated DB was enough to produce this model.

**Fix options, best first**

1. Make the legacy reads independent of the model's column set: select an explicit column list, as the insert already does, or project into a struct that has no `InsertionNumber`. This removes the coupling permanently.
2. Have `tools/jetregen` and `main.go` emit `json:"-"` for `shop_messages.insertion_number`. ADR-022 already prescribes this. Note that this alone does **not** fix failure 2: `AllColumns` would still select the column.
3. Either way: before building the release image, run `scripts/test-shops-isolated.sh -run TestMessageSyncLegacyCompatibility ./tests/shops`, and run `git status .gen` plus a grep for `InsertionNumber` on the build machine.

---

## High

### H1. Updating a shop resets `admin_only_lists` to false

**Evidence:** [api/shops/core/repository_impl.go:67-87](../../api/shops/core/repository_impl.go#L67-L87)

```go
updateStmt := Shops.UPDATE(Shops.Name, Shops.UpdatedAt).SET(
    Shops.Name.SET(String(shop.Name)),
    Shops.UpdatedAt.SET(TimestampzT(*shop.UpdatedAt)),
    Shops.AdminOnlyLists.SET(Bool(shop.AdminOnlyLists)),   // <- always written
)
```

[core/handler.go:233-237](../../api/shops/core/handler.go#L233-L237) builds `model.Shops{ID, Name, Details}`. `request.UpdateShopRequest` has no `admin_only_lists` field, so `shop.AdminOnlyLists` is always `false`.

When `SET` receives `ColumnAssigment`s, Jet ignores the `UPDATE(...)` column list and writes **every** assignment. This was verified in `go-jet/jet/v2/postgres/update_statement.go:47-60`.

**Impact:** an admin turns on "admin-only lists". Any later rename or details edit silently turns it off, and non-admin members regain list create, edit and delete rights. This disables a permission control without anyone being told. It dates from `17f0c51` (2026-01-28), so it is not new. It still ships in this build, and no test covers it: the authorization regression tests set the flag with raw SQL.

**Fix:** remove the `AdminOnlyLists.SET(...)` lines; the settings endpoint owns that column. Add an integration test: set `admin_only_lists=true`, call `PUT /shops/:id`, and assert the setting is still `true`.

### H2. Admins who did not create a shop cannot rename it

**Evidence:**

- [core/service_impl.go](../../api/shops/core/service_impl.go) `UpdateShop` checks `IsUserShopAdmin`.
- The repository's `WHERE` clause adds `Shops.CreatedBy.EQ(user)` ([repository_impl.go:89-92](../../api/shops/core/repository_impl.go#L89-L92)).
- A promoted admin passes the service check, then the `UPDATE ... RETURNING` matches 0 rows and Jet returns `qrm: no rows in result set`.
- A legacy client sees HTTP 500 "Unable to complete Shops request"; a contract-2 client sees 404 `not_found`.

**Impact:** a confusing failure for promoted admins. `DeleteShop` has the same creator check on top of the admin check. That may be intended, but for update it contradicts the service's own error text ("only shop admins can update shops").

**Fix:** decide the rule. If admins may rename, drop the `CreatedBy` predicate and authorize with `shared.LockShopMutation` inside a transaction, as `DeleteShop` already does.

---

## Medium

### M1. Unit test suite fails: nil-pointer panic in `api/route`

`go test ./api/...`: 1025 passed, **2 failed**.

```
TestSetupShopVehicleUsageUnauthorizedResponsesUseStandardEnvelope/missing_authorization_header
panic: runtime error: invalid memory address or nil pointer dereference
miltechserver/api/shops.RegisterRoutes  api/shops/route.go:35
miltechserver/api/route.Setup           api/route/route.go:155
```

The test calls `Setup(nil, router, nil, nil, nil)`. [api/shops/route.go:35-36](../../api/shops/route.go#L35-L36) now dereferences `deps.Env.ShopsAtomicNotificationSaveEnabled` and `deps.Env.ShopsMessageSyncEnabled`; those reads were added in `2572f2b` and `20848ba`. Production always passes an `Env`, so this is test-only, but CI is red.

**Fix:** treat a nil `Env` as "all flags off" in `RegisterRoutes`, or pass `&bootstrap.Env{}` in the test.

### M2. `RemoveListItemBatch` is no longer tolerant of already-removed items

**Evidence:**

- [lists/items/repository_impl.go:329](../../api/shops/lists/items/repository_impl.go#L329) now calls `AuthorizeListMutation(..., itemIDs, ...)`.
- [shared/authorization.go:283](../../api/shops/shared/authorization.go#L283) fails with "list item not found" for **any** missing ID.

Before this branch, only `itemIDs[0]` had to exist (service preflight), and the `DELETE ... IN (...)` ignored missing rows.

**Impact:**

- Two members clearing the same list at once: the second user's whole batch fails.
- A client retrying after a timeout where the first attempt committed: the retry errors.
- Legacy clients see HTTP 500.

The new code also **closes a real hole**: the old batch delete would remove item IDs from other lists or shops. Keep that protection.

**Fix:** skip IDs that no longer exist, as `deleteLegacyItems(..., requireItem=false)` already does for notification items. Keep the same-shop and permission checks for the IDs that do exist.

### M3. Message-insert lock-order inversion (deadlock window during rollout)

**Evidence:**

- New-binary `CreateShopMessage` takes `shops` `FOR UPDATE` first ([messages/repository_impl.go:72](../../api/shops/messages/repository_impl.go#L72), via [authorization.go:265](../../api/shops/shared/authorization.go#L265)), then inserts. The 018 trigger then takes the `shop_message_counters` row lock.
- A writer that skips `LockShopMutation` locks in the reverse order. Examples are the **previous server binary** during a rolling deploy, or ad-hoc SQL. Its `BEFORE INSERT` trigger takes the counter row lock first, then the FK check takes `FOR KEY SHARE` on the `shops` row. `FOR KEY SHARE` conflicts with `FOR UPDATE`.

**Impact:** Postgres detects the deadlock (`40P01`) and aborts one transaction. `CreateShopMessage` uses `sharedb.WithTx`, not the retrying `WithNotificationMutation`, so the user gets a send failure. The window only exists while 018 is applied and the fleet is mixed, or when out-of-band writers run, so impact is low. ADR-022 lists only the migration-vs-app deadlock as an accepted risk, so this one is not documented.

**Fix options:**

- Retry `40P01` in `CreateShopMessage`.
- Or document this risk in ADR-022's consequences, and keep the old-binary/new-binary overlap after 018 as short as possible.

**Related scaling note, not a bug today:** `LockShopMutation` takes `FOR UPDATE` on the `shops` row for **every** shop mutation, including message posts and list item edits. All writes within one shop are serialized, and any child-row FK check on that shop waits. That is fine at current shop sizes. If a large shop shows write latency, the first lever is `FOR SHARE` for ordinary mutations, with `FOR UPDATE` kept for settings and role changes.

### M4. Released clients get generic 500s for new failure modes

**How it works:**

- [shared/contract.go:56-70](../../api/shops/shared/contract.go#L56-L70): without the `X-MilTech-Shops-Contract: 2` header, every `c.Error` path answers HTTP 500, or 404 only for "no item found".
- Any message not in the allowlists becomes "Unable to complete Shops request".
- Preserving those statuses is intentional, per `shops-release-contracts.md`.

**New messages released clients will only see as a generic 500:**

| Situation | Message | Location |
|-----------|---------|----------|
| Deleting a list an equipment service uses | `list is in use` (allowlisted, but HTTP 500 to legacy clients) | [lists/repository_impl.go:222](../../api/shops/lists/repository_impl.go#L222) |
| Leaving a shop while membership changes | `shop membership changed; retry leaving the shop` (not allowlisted, so replaced with the generic text) | [members/repository_impl.go:268](../../api/shops/members/repository_impl.go#L268) |
| Third lock-race retry exhausted on a notification | `notification attachment changed while acquiring locks` (generic) | [shared/notification_mutation.go:15](../../api/shops/shared/notification_mutation.go#L15) |

**UX impact:** a user of 3.7.0+41 who deletes a list attached to a service sees a generic failure and no reason. Consider adding a short note in the client's release notes. Also confirm the next mobile build maps `list_in_use` to a human message.

### M5. Message catch-up returns 400 when the client watermark is ahead of the server

**Evidence:** [messages/sync_repository.go:206-211](../../api/shops/messages/sync_repository.go#L206-L211). If `after` is greater than the shop's current counter, the request returns `invalid`. That happens after a DB restore, if an instance points at a different DB, or if a counter row is recreated. Per the runbook, the mobile client then shows "Message refresh failed" on every poll (about 10 s) until the user leaves and re-enters the shop.

**Fix:** return a distinct code such as `watermark_ahead` / 409, so the client can re-run `initial` automatically. At minimum, add this case to the runbook's "capability true but sync fails" section.

### M6. Some mutations still authorize outside the transaction

`3917e08` ("enforce current membership on mutations") moved most writes to lock-then-recheck inside the transaction. These still use the old check-then-act pattern:

| Mutation | Current authorization |
|----------|----------------------|
| `vehicles.UpdateShopVehicle`, `UpdateShopVehicleUsage` | service-level membership, admin and creator checks |
| `vehicles.AdjustShopVehicleUsage` | `RequireShopMember` before the transaction |
| `vehicles.CreateShopVehicle` | service-level check only |
| `core.UpdateShop` (see H1/H2) | service-level check only |
| `invites.CreateInviteCode`, `DeactivateInviteCode`, `DeleteInviteCode` | service-level check only |

A member removed mid-request can still complete one write. The window is milliseconds, so this is low risk, but it is inconsistent with the rest of the feature.

Separately, [vehicles/service_impl.go:160 and 174](../../api/shops/vehicles/service_impl.go#L160-L174) run `UpdateShopVehicleUsage` and then `UpdateShopVehicle` as **two separate autocommit writes**. If the second fails, the usage change persists without the detail change.

### M7. Schema generation on every startup (operational)

`main.go` `generateSchema` connects to the DB and rewrites `./.gen` at startup, then calls `log.Fatalf` on any error. In the container this needs:

- a writable working directory, and
- catalog access for the app role.

A read-only filesystem or a catalog permission change becomes a crash loop. It is also the mechanism behind C1. This is pre-existing, but with schema changes landing it is now the most likely way to break a release. Consider gating it behind `DEBUG=true` or removing it from `main`; `tools/jetregen` covers the developer workflow.

---

## Low

- **L1. Receipt timestamp drift.** [atomic_save_repository.go:154](../../api/shops/vehicles/notifications/atomic_save_repository.go#L154) re-stamps `committed_at` with a second `time.Now()` after the writes. The returned `committed_at` is therefore a few ms later than the notification's `save_time`/`last_updated` and the audit rows. If the client uses `committed_at` as the notification's `last_updated`, the values will not match. Use `receipt.CommittedAt` in that `UPDATE` (the `RETURNING` still normalizes precision).
- **L2. Unbounded atomic-save item count.** Only the 1 MiB body cap applies, which allows thousands of items. Each item costs one or two queries while the shop row is locked `FOR UPDATE`, blocking every other write in that shop. Add an explicit cap, for example 500.
- **L3. New-item ID race in atomic save.** The ownership lookup at line 218 is unlocked. A concurrent insert of the same item UUID elsewhere hits the PK, and the user gets 500 instead of 400. Rare in practice.
- **L4. Dead branch.** [sync_handler.go:72](../../api/shops/messages/sync_handler.go#L72): `else if _, ok := query["limit"]` can never be true because `GetQuery` already returned `ok`.
- **L5. Audit gap for new item fields.** [items/legacy_mutation.go:112](../../api/shops/vehicles/notifications/items/legacy_mutation.go#L112) does not select `nickname`/`unit_of_measure`, so `items_removed` audit entries omit them.
- **L6. Legacy cursor pagination skips ties.** `GetShopMessagesByCursor` filters on `created_at < cursor` only, so messages sharing a timestamp are skipped. This is pre-existing; the v2 sync path fixes it with `(created_at, id)`.

---

## Deployment-order dependencies (verify before rollout)

1. **Migration 017 must be applied before this binary.** Legacy notification item creation ([items/legacy_mutation.go:42](../../api/shops/vehicles/notifications/items/legacy_mutation.go#L42)) and the maintenance snapshot ([aggregates/repository_notifications.go:172-181](../../api/shops/aggregates/repository_notifications.go#L172-L181)) now reference `nickname`/`unit_of_measure`. Without 017, adding notification items fails for **all** clients, including released ones.
2. **Migration 016** is needed only when `SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED=true`.
3. **Migration 018:** until C1 is fixed, this binary also requires 018, regardless of the flag. After the C1 fix it requires 018 only for the flag, as ADR-022 intends.
4. Known failing tests recorded on this branch: `TestAtomicNotificationItemFieldsSurviveReleasedClientSaves` and `scripts/test-shops-isolated_test.py`. Plus M1 above.

## Verified correct (no action)

- **Message sync consistency.** Reads run in a single `REPEATABLE READ, READ ONLY` transaction. The watermark comes from the counter row, whose lock the trigger holds until commit, so a watermark never covers an uncommitted lower number. History uses a `(created_at, id)` keyset. Overlap between catch-up and history yields duplicates, not gaps, so clients must dedupe by `id`.
- **Atomic notification save idempotency.**
  - The operation claim, writes and audits commit together.
  - A concurrent duplicate blocks on `ON CONFLICT` until the first transaction resolves.
  - Replays are fingerprint-checked and re-authorized.
  - Create targets are deterministic (UUIDv5 of user and operation).
  - Released-client omissions of `nickname`/`unit` are handled with `COALESCE` and `omitempty` in the fingerprint.
- **Lock ordering in shared helpers** is consistent: shop → member → lists → vehicle → notification → items. `WithNotificationMutation` retries only `40P01`/`40001` and lock-change errors.
- **Attachment FK** is `ON DELETE SET NULL`, so a deleted list cannot strand a notification in `LockNotificationMutation`.
- **No double auditing:** the service skips audits when the repository implements `OwnsLegacyNotificationAudits`.
- **Error sanitization:** unknown error strings never reach clients (allowlist plus fixed prefixes), and authorization DB errors are logged, not echoed.
- **Migration 018** is atomic, re-runnable after a timeout or deadlock, and adds the FK after seeding to avoid the documented deadlock. Its rollback reverses every object.

## Suggested release checklist

1. Fix C1 (explicit legacy columns) and H1 (drop the `AdminOnlyLists` assignment), and add tests for both.
2. Fix M1 so `go test ./api/...` is green.
3. `brew install ripgrep`, then run `scripts/test-shops-isolated.sh -count=1 ./tests/shops` and then `./tests/equipment_services`, separately: the wrapper rejects `-p 1`, and the packages race if run together.
4. On the build machine: confirm `.gen` matches the intended schema before `docker build`.
5. Apply 017 (and 018 if C1 is not fixed) to the target DB before deploying the binary.
6. Decide M2 and M5 before the next mobile release, since both change client-visible behaviour.
