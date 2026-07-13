# Shop Feature Code Review — Last 2 Weeks (2026-06-26 → 2026-07-10)

## Scope

Commit range `ad688e9^..HEAD` on `pmcs_sbs_images`, 15 commits, all shop-related files:

```
ad688e9 perf(shops): scope user shop stats aggregation
dfbf523 feat(shops): add aggregate response contracts
c972b0c feat(shops): scaffold aggregate endpoints
286d154 feat(shops): add lists with items aggregate
35b6774 feat(shops): add vehicle maintenance snapshot
5159117 feat(shops): add shop snapshot endpoint
fcfc6c6 feat(shops): add bootstrap aggregate endpoint
b1766cc test(shops): guard aggregate route compatibility
ae32d6a test(shops): add aggregate performance validation
1ef4882 docs(shops): document aggregate API endpoints
785ff6d fix(shops): bound aggregate payload sections
2cc76d7 docs(shops): add mobile API refactor guide
0b95ada feat(shops): make aggregate limits optional
28a8175 feat(shops): attach notifications to shop lists
f384bf3 docs(shops): plan notification list attachments
```

32 files changed, +6,280 / −72 lines. The bulk of the change is a brand-new `api/shops/aggregates` package (route → handler → service → repository, ~1,900 lines) that bulk-fetches shop stats, lists+items, vehicle maintenance, and notifications into single mobile-facing responses — plus a change to `api/shops/vehicles/notifications` that lets a notification be attached to a shop list, and a refactor of an existing query in `api/shops/core/repository_impl.go`.

## Methodology

Ran 8 independent review passes in parallel over the full diff (line-by-line scan, removed-behavior audit, cross-file call-site tracing, reuse, simplification, efficiency, altitude/architecture, CLAUDE.md conventions), then personally re-verified every finding below against the current source before including it — reading the actual file, not just trusting the pass that raised it. Two findings from the automated passes turned out to be non-issues on inspection and are noted at the bottom rather than silently dropped.

Confidence key: **Confirmed** = I read the exact lines and traced the failure path myself. **Plausible** = a finder pass raised it with a concrete scenario but I did not independently re-derive it end-to-end.

---

## 1. Correctness bugs

### 1.1 No tracked migration for `shop_vehicle_notifications.attached_shop_list` — Confirmed, highest severity

- `migrations/` only goes up to `005_add_parent_id_to_shop_messages.sql`. There is no `006_*` migration adding the `attached_shop_list` column.
- The column is fully wired into the generated Jet code (`.gen/miltech_ng/public/table/shop_vehicle_notifications.go:30,81-83,100`) and used throughout `api/shops/vehicles/notifications/repository_impl.go` and `api/shops/aggregates/repository_impl.go`, which means `jet generate` was run against a database where the column already exists — almost certainly a local/dev DB that was hand-altered.
- No `migrate`-style tool or script exists elsewhere in the repo; `migrations/00N_*.sql` is the only tracked schema history.

**Failure scenario:** any environment provisioned by replaying `migrations/*.sql` from scratch (fresh staging, prod, or CI database) will be missing this column. Every call that touches `ShopVehicleNotifications.AllColumns` — not just the new attach-to-list feature, but the *existing* create/read/update notification endpoints — will fail with `column "attached_shop_list" does not exist`.

**Fix:** add `006_add_attached_shop_list_to_shop_vehicle_notifications.sql` (+ rollback) before this branch merges anywhere beyond the dev DB it was built against.

### 1.2 Attaching a notification to a list from another shop (or an invalid list) returns HTTP 500, not 400 — Confirmed

- `api/shops/vehicles/notifications/service_impl.go:37-56`, `validateAttachedShopList`, returns a plain `errors.New("invalid attached_shop_list")` for an empty ID, a not-found list, or a list belonging to a different shop.
- The handler forwards every service error via `c.Error(err)` (`api/shops/vehicles/notifications/handler.go:50` etc.).
- The global error middleware, `api/middleware/error_handler.go:11-31`, maps *any* error whose text doesn't contain `"no item found"` straight to `500`, echoing `err.Error()` as the client-facing message. (The file itself carries a `// TODO: Don't think this really does anything anymore.` comment — this is a pre-existing sharp edge the new code walked into rather than one it introduced.)
- `tests/shops/shops_notifications_test.go` has a test asserting the cross-shop-list case is rejected, but only checks that the request fails, not the status code — so the test currently passes despite returning the wrong status.

**Failure scenario:** mobile client PUTs a notification update with `attached_shop_list` set to a list ID from a different shop (or a stale/deleted list ID). Server returns `500 Internal Server Error` with message `"invalid attached_shop_list"` — indistinguishable from a real backend fault to client error handling, crash reporting, or on-call alerting, when it's actually a 400-class input error.

**Fix:** either give `validateAttachedShopList` a sentinel error and add a case for it in `error_handler.go`, or route it through the same `writeAggregateError`-style classification the new `aggregates` package already uses.

### 1.3 `service_date` sort order is inverted between two sibling endpoints — Confirmed

- `api/shops/aggregates/repository_impl.go:395`, `GetVehicleServices` (backs `GET /shops/vehicles/:vehicle_id/maintenance-snapshot`): `ORDER BY es.service_date DESC NULLS LAST, es.created_at DESC, es.id ASC`
- `api/shops/aggregates/repository_impl.go:969`, `getShopSnapshotServices` (backs `GET /shops/:shop_id/snapshot`): `ORDER BY es.service_date ASC NULLS LAST, es.created_at DESC, es.id ASC`

Same underlying `equipment_services` data, opposite primary sort direction. Every other paired vehicle-scoped/shop-scoped query in this file (notifications, changes, messages, lists) is consistently newest-first.

**Failure scenario:** a shop has more equipment services than `services_limit`. `GET /shops/vehicles/:id/maintenance-snapshot?services_limit=5` returns the 5 newest; `GET /shops/:shop_id/snapshot?include=services&services_limit=5` for the same vehicle's shop returns the 5 *oldest*. A mobile UI showing "recent services" on two different screens shows contradictory data, and the shop snapshot never surfaces recently completed/scheduled work once a shop has more than `services_limit` services on record.

**Fix:** change line 969 to `DESC` to match line 395 (and every other section), unless the shop-snapshot "oldest first" ordering was actually intentional — if so it needs a comment explaining why, since it's the only section that diverges.

### 1.4 Inconsistent `omitempty` makes bootstrap counts silently disappear at zero — Confirmed

`api/response/user_shops_response.go:125-135`, `ShopAggregateCounts` (used by `GET /shops/bootstrap`):

```go
type ShopAggregateCounts struct {
	Members           int64 `json:"members"`
	Vehicles          int64 `json:"vehicles"`
	Lists             int64 `json:"lists"`
	Messages          int64 `json:"messages"`
	Notifications     int64 `json:"notifications"`
	NotificationItems int64 `json:"notification_items,omitempty"`
	OpenServices      int64 `json:"open_services"`
	Services          int64 `json:"services,omitempty"`
	RecentChanges     int64 `json:"recent_changes,omitempty"`
}
```

Six of nine fields have no `omitempty`; three (`NotificationItems`, `Services`, `RecentChanges`) do, with no apparent reason for the split — the sibling `ShopSnapshotCounts` (line 137) and `VehicleMaintenanceSnapshotCounts` (line 146) structs don't use `omitempty` at all.

**Failure scenario:** a newly created or currently-inactive shop (zero notification items, zero services, zero recorded changes) calls `GET /shops/bootstrap`. Its `counts` object is missing the `notification_items`, `services`, and `recent_changes` keys entirely, while `members`, `vehicles`, `lists`, `messages`, `notifications`, `open_services` all still show `0`. A mobile client decoding into a fixed struct, or doing `data.counts.services` without an undefined check, breaks specifically for new/inactive shops — the exact accounts most likely to be onboarding.

**Fix:** drop `omitempty` from all three fields for consistency with the rest of the struct and its siblings.

### 1.5 Real internal errors are discarded with no log line — Confirmed

`api/shops/aggregates/handler.go:267-282`, `writeAggregateError`:

```go
default:
	c.JSON(http.StatusInternalServerError, response.StandardResponse{
		Status:  http.StatusInternalServerError,
		Message: ErrAggregateUnavailable.Error(),
		Data:    nil,
	})
```

The incoming `err` — which by this point carries the real DB/query failure, wrapped via `%w` all the way up from `repository_impl.go` — is matched only to decide which static message to send, then dropped. It is never logged.

**Failure scenario:** a real query failure (bad SQL, connection drop, scan error, context deadline) in any of the four aggregate endpoints returns a generic "aggregate unavailable" 500 to the client with zero trace on the server side — no log line, no error ID. On-call has nothing to search for when a mobile user reports "shop screen won't load."

**Fix:** `log` (or the project's structured logger, if one exists elsewhere — none was found in this package) the wrapped error before responding, at minimum in the `default` branch.

---

## 2. Design risk — not a bug, but worth flagging

### 2.1 Omitting a `*_limit` query parameter returns the section fully unbounded — intentional, but see caveat

Traced end-to-end:
- `parseOptionalIntQuery` (`api/shops/aggregates/handler.go:255-265`) returns `0` when the query param is absent, and rejects `0`/negative/non-numeric *explicit* values with `ErrInvalidLimit`.
- `normalizeOptionalLimit` (`api/shops/aggregates/service_impl.go:35-40`) only clamps values *above* the max; a `0` passes through unchanged.
- Every bounded query in `repository_impl.go` treats `0` as "no limit": `LIMIT NULLIF($N, 0)` (Postgres: `NULLIF(0,0)` → `NULL` → unbounded `LIMIT`), or `WHERE ($N = 0 OR rank <= $N)`.

This is not a bug — `docs/api/shops_api_efficiency_mobile.md` documents it explicitly: e.g. `lists_limit` → "Omitted Behavior: Return all lists", with `0` itself listed as an *invalid* value. So "no param" and "unlimited" are the documented contract, not an accidental regression from `0b95ada`.

**Why it's still worth flagging:** the whole point of these endpoints (per their own doc titles — "mobile efficiency") is bounded, predictable payloads. `785ff6d fix(shops): bound aggregate payload sections` suggests an unbounded-payload problem was hit once already. A mobile client that forgets to pass a limit — which is the zero-effort, most-likely-in-practice way to call these endpoints — gets exactly that unbounded behavior back, for a shop with an unbounded number of lists/items/services/changes. There's no server-side default cap protecting against a client bug or a very large shop; every call site (4 endpoints × up to 8 sections each) relies on every caller remembering to always pass a limit.

**Suggestion:** consider giving each section a non-zero *default* (distinct from "no limit"), and require an explicit opt-in (e.g. `?lists_limit=all`) for the unbounded case, rather than making "forgot to pass a param" and "explicitly want everything" the same code path.

---

## 3. Efficiency

This subsystem's stated purpose is reducing mobile round trips, so query efficiency is the central design concern — these findings work directly against that goal.

### 3.1 All aggregate sections are fetched sequentially, not concurrently — Confirmed

`api/shops/aggregates/service_impl.go`, `GetShopSnapshot` (166-200) and `GetVehicleMaintenanceSnapshot` (103-164), call each section's repository method one after another and block on each before starting the next (summary, vehicles, lists+items, notifications+items, messages, services, changes — up to 8 sequential round trips for a full shop snapshot).

The codebase already has a concurrent-fetch pattern available: `api/shops/core/repository_impl.go:19,383` uses `golang.org/x/sync/errgroup` for exactly this kind of "fetch several independent things, combine" case. The new aggregates package doesn't use it.

**Failure scenario:** each query costs roughly 5-15ms round-trip even on a healthy connection. A full shop-snapshot request with all sections enabled does 7+ sequential round trips, adding ~50-100ms of pure serialized latency that would collapse to roughly the slowest single query if the independent sections were fetched via `errgroup.Group` with a shared context — directly undercutting the "fewer round trips for mobile" premise the feature was built for.

### 3.2 Shop-membership is re-verified once per section instead of once per request — Confirmed pattern (not independently timed)

`api/shops/aggregates/repository_impl.go` — the per-section snapshot queries (`getShopSnapshotVehicles`, `getShopSnapshotNotifications`, `getShopSnapshotMessages`, `getShopSnapshotServices`, `getShopSnapshotRecentChanges`) each independently `INNER JOIN shop_members` to re-check `(shop_id, user_id)` membership, even though `service_impl.go`'s `requireShopMember` already confirmed it before any of these ran, and `getShopSnapshotSummary` re-checks it again as the first section query.

**Cost:** up to 3 redundant membership checks per request (auth layer, summary query, then again per section) instead of one. Since the summary query already returns the caller's role, the later per-section queries could filter on the already-authorized `shop_id` alone and drop the repeated `shop_members` join.

### 3.3 The added "performance" test doesn't assert on performance — Confirmed

`tests/shops/shops_aggregate_performance_test.go`:
- The whole test is gated behind `SHOP_AGGREGATE_PERF=1` and skipped by default — it does not run in normal CI.
- Even when explicitly run, it only logs p50/p95 latency and payload sizes via `t.Logf` and asserts response shape + `200` status — there is no assertion on a duration threshold, query count, or payload-size ceiling anywhere in the file.

**Failure scenario:** a regression that turns 3.1's sequential round trips into something even slower, or reintroduces an N+1 query, produces no CI failure — the test that exists specifically to catch this class of regression is both off by default and, even opted-in, has nothing to fail on.

---

## 4. Architecture / maintainability

### 4.1 The same query logic is independently re-implemented 2–4 times

The `aggregates/repository_impl.go` file (~1,150 lines) was written as fully hand-rolled raw SQL rather than using the project's generated Jet query builder that every other `api/shops/*` repository uses. Combined with building both a "vehicle-scoped" and "shop-scoped" version of most sections, this produced substantial duplication:

| Logic | Duplicated in |
|---|---|
| Per-notification ranked-items CTE + scan | `getItemsByNotificationIDs` (276-339) and `getSnapshotItemsByNotificationIDs` (804-867) — byte-for-byte identical except error-message text |
| Notification→items fan-out/grouping | `GetVehicleNotificationsWithItems` (196-233) and `getShopNotificationsWithItems` (765-802) |
| Notification-change query (title/type/is-deleted fallback logic) | Already exists in `api/shops/vehicles/notifications/changes/repository_impl.go:91,168`; re-implemented a 3rd and 4th time in `GetVehicleRecentChanges` (340-384) and `getShopSnapshotRecentChanges` (1015-1060) |
| Equipment-services fetch (shop-scoping join + field mapping) | Already exists in `api/equipment_services/queries/repository_impl.go:93` (Jet-based, paginated, shop-member-scoped); re-implemented in `GetVehicleServices` (385-440) and `getShopSnapshotServices` (958-1014), dropping pagination |
| Per-shop count subqueries (members/vehicles/lists/messages/notifications/services) | `GetBootstrap`'s bulk query (516-601) and `getShopSnapshotSummary` (665-710) |

**Cost:** any fix to one of these — a ranking rule, a null-handling fallback, a shop-scoping rule — has to be found and applied in every copy. §1.3's ordering bug is a direct, already-realized instance of exactly this risk (one of two near-identical `service_date` queries got the sort direction wrong).

**Fix direction:** at minimum, factor the identical `getItemsByNotificationIDs`/`getSnapshotItemsByNotificationIDs` pair into one function; longer-term, consider whether the vehicle-scoped and shop-scoped variants of each section can share a single parameterized query rather than two independently-maintained ones.

### 4.2 Two full, independently-maintained implementations of `UpdateVehicleNotification`

`api/shops/vehicles/notifications/repository_impl.go:157-197` branches on `update.AttachedShopListSet`: the `false` branch uses the existing Jet `UPDATE().SET()` builder; the `true` branch is a hand-written raw SQL string that re-lists `title/description/type/completed/last_updated` plus `attached_shop_list`.

**Cost:** the next field added to `shop_vehicle_notifications` has to be added to both the Jet `SET` list and the raw SQL string, or the two paths silently diverge. Jet supports conditionally-included columns in a single builder, so this duplication looks avoidable.

### 4.3 New DTOs duplicate existing ones instead of extending them

`api/response/user_shops_response.go:164-192`, `ShopSnapshotSummary`/`ShopBootstrapSummary` (id, name, details, role, is_admin, member/vehicle counts) cover the same ground as the existing `response.ShopWithStats` and `response.ShopDetailResponse`, populated by `api/shops/core/repository_impl.go:247 GetShopsWithStatsForUser`. Instead of extending that existing query/DTO with the additional counts the aggregates endpoints need (lists, messages, notifications, services), a parallel DTO and a parallel counting query were built.

**Cost:** two divergent representations of "shop with counts for this user" now exist. If `GetShopsWithStatsForUser`'s counting logic changes (e.g. to exclude soft-deleted members), the aggregates counts silently drift from the core shops-list endpoint's counts for the same shop.

### 4.4 Shop-membership scoping is reimplemented inline per query instead of reused

Every query in `aggregates/repository_impl.go` that needs to check "is this user a member of this shop" re-writes the join by hand (`INNER JOIN shop_members sm ON sm.shop_id = ... AND sm.user_id = $N`, at least 8 occurrences) rather than going through the existing `shared.ShopAuthorization` interface (`IsUserMemberOfShop`/`RequireShopMember`) already used by `service_impl.go` itself for the top-level check, and by the rest of `api/shops/*`.

**Cost:** the membership rule (soft-deletes, future role tiers, caching) now lives in 8+ duplicated SQL fragments instead of one interface; a future change to what "member" means has to be hunted down and edited in every one, and it's easy to miss one.

### 4.5 Minor: dead field `BootstrapOptions.IncludeEmptyEquipment` — Confirmed

`api/shops/aggregates/service.go:36` declares `IncludeEmptyEquipment bool`. It is never set by `parseBootstrapOptions` (no query param maps to it), never read in `service_impl.go`, and never read in `repository_impl.go`'s `getBootstrapEquipment`. Grep against the whole repo (excluding tests) confirms it's referenced nowhere else.

**Cost:** low functionally (it's inert), but a future reader will reasonably assume toggling it changes behavior. It doesn't. Either wire it up or delete it.

### 4.6 Path parameters aren't validated the way query parameters are — Plausible, worth a look

`api/shops/aggregates/handler.go:38,63,88` pass `c.Param("shop_id")`/`c.Param("vehicle_id")` straight to the service layer with no format check, while every `*_limit` and `include` query parameter in the same file goes through explicit validation (`parseOptionalIntQuery`, `parseShopSnapshotIncludes`). In practice a malformed ID will most likely just fail to match any row (or error out at the DB driver level on a UUID column), so this is more a defense-in-depth/consistency gap than a demonstrated exploit — but it's inconsistent with the care taken on the query-string side in the same file, and CLAUDE.md's project standard ("ALWAYS validate and sanitize ALL inputs") applies to path params too.

---

## 5. Passed review (raised by a finder pass, refuted on inspection)

- **"Limit-omission reintroduces the unbounded-payload bug `785ff6d` fixed."** True mechanically (see §2.1), but the doc confirms it's the *intended* contract, not a regression — reclassified from bug to design risk above rather than dropped.
- **Removed-behavior pass found nothing.** Checked the refactor of `GetShopsWithStatsForUser` (correlated subqueries → CTEs) and the `UpdateVehicleNotification` split line by line — auth/membership/ownership checks and all prior `SET` columns are preserved in both. No silently dropped validation or scoping found anywhere in the two-week window.

---

## Suggested priority order

1. §1.1 missing migration (blocks any real deploy)
2. §1.2 wrong status code on cross-shop list attach + §1.5 swallowed errors (both are "on-call has no idea what happened" bugs)
3. §1.3 ordering inconsistency, §1.4 omitempty inconsistency (both are silent client-facing data bugs)
4. §3.1/§3.2 concurrency (undercuts the feature's whole reason for existing) + §3.3 (fix the test so a future regression here is actually caught)
5. §4.1–§4.6 as opportunistic cleanup, ideally before a third aggregate endpoint gets added and the duplication compounds further
