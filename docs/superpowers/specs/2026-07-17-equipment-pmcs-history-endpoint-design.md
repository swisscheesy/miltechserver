# Equipment + PMCS History Aggregate Endpoint — Design

**Goal:** Give the mobile team a single endpoint that returns every piece of equipment a user has access to, across all their shops, with each equipment's PMCS SBS inspection history nested underneath — so they no longer have to call a vehicle-list endpoint and then one PMCS-history call per vehicle.

**Context:** The PMCS SBS inspection history feature (`docs/superpowers/specs/2026-07-16-pmcs-sbs-inspection-history-design.md`, implemented on branch `pmcs_sbs_images`) introduced `GET /pmcs-sbs/equipment/:equipment_id/pmcs`, which lists one vehicle's inspection history. This new endpoint is the cross-vehicle overview built on top of that same data.

## Decisions

- **Scope:** All equipment across all shops the user is a member of, in one call — no shop grouping, no `:shop_id` path parameter. Matches the existing `GetShopEquipmentOverview` access model (`shop_members.user_id` defines what's visible).
- **PMCS detail depth:** Summary only per inspection (`id`, `guide_manual`, `performed_date`, `fault_count`, `created_at`) — the same fields as `InspectionSummaryResponse` in `api/pmcs_sbs_progress`. No nested fault line-items. The mobile client fetches full fault detail for one inspection on demand via the existing `GET /pmcs-sbs/equipment/:equipment_id/pmcs/:pmcs_id`.
- **Payload limits:** None. No pagination or per-vehicle cap on inspection count — every accessible vehicle and its complete inspection history is always returned.
- **Endpoint location:** `api/shops/aggregates/`, alongside the existing bootstrap/snapshot/lists-with-items/maintenance-snapshot endpoints — this package already exists specifically to compose data across domains for a user's shops.
- **Zero-history equipment:** Included, with `historical_pmcs: []`. The equipment list is "everything you have access to," independent of whether a PMCS has ever been recorded against it.

## Endpoint

```
GET /api/v1/auth/shops/equipment-pmcs-history
```

No query parameters, no path parameters. Registered in `api/shops/aggregates/route.go` alongside the other 4 routes, gzip-wrapped the same way:

```go
router.GET("/shops/equipment-pmcs-history", gzip.Gzip(gzip.DefaultCompression), handler.getEquipmentPmcsHistory)
```

## Response Shape

```json
{
  "status": 200,
  "data": {
    "equipment": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B1",
        "model": "M1152A1",
        "serial": "SER-B1",
        "niin": "",
        "historical_pmcs": [
          {
            "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-16T14:30:00Z",
            "fault_count": 3,
            "created_at": "2026-07-16T14:31:02.123456Z"
          },
          {
            "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
            "guide_manual": "pmcs_sbs/hmmwv/hmmwv_up_armor_pmcs.json",
            "performed_date": "2026-07-09T09:00:00Z",
            "fault_count": 0,
            "created_at": "2026-07-09T09:05:44.001122Z"
          }
        ]
      },
      {
        "id": "9d8e7f6a-2222-4b3c-8d4e-000000000002",
        "shop_id": "b2f1c1b4-1111-4a2b-9c3d-000000000001",
        "admin": "B2",
        "model": "M998",
        "serial": "SER-B2",
        "niin": "",
        "historical_pmcs": []
      }
    ],
    "count": 2
  },
  "message": ""
}
```

`historical_pmcs` is ordered most-recent-`performed_date`-first, matching the existing per-vehicle list endpoint's ordering. `count` is the number of equipment entries in `equipment` (not a total across any pagination — there is none here).

## Response Types (`api/response/user_shops_response.go`)

New types added alongside the existing `ShopEquipmentSummary`/`ShopBootstrapSummary`/etc.:

```go
type EquipmentPmcsHistoryResponse struct {
	Equipment []EquipmentWithPmcsHistory `json:"equipment"`
	Count     int                        `json:"count"`
}

type EquipmentWithPmcsHistory struct {
	ShopEquipmentSummary
	ShopID         string              `json:"shop_id"`
	HistoricalPmcs []PmcsHistorySummary `json:"historical_pmcs"`
}

type PmcsHistorySummary struct {
	ID            uuid.UUID `json:"id"`
	GuideManual   string    `json:"guide_manual"`
	PerformedDate time.Time `json:"performed_date"`
	FaultCount    int       `json:"fault_count"`
	CreatedAt     time.Time `json:"created_at"`
}
```

`EquipmentWithPmcsHistory` embeds the existing `ShopEquipmentSummary` (`ID`, `Admin`, `Model`, `Serial`, `Niin`) rather than redeclaring those fields, following the same embedding convention already used by `ShopListWithItems` (embeds `ShopListWithUsername`) and `ShopDetailResponse` (embeds `model.Shops`). `PmcsHistorySummary` is field-for-field identical to `pmcs_sbs_progress.InspectionSummaryResponse` but declared locally in the `response` package to avoid a cross-package type dependency, consistent with how this package already defines its own response types rather than importing other feature packages' types.

## Architecture

New code follows the existing `api/shops/aggregates/` package skeleton exactly — no new package.

- **`route.go`** — one new registration line (above).
- **`handler.go`** — new `getEquipmentPmcsHistory(c *gin.Context)`: auth check via the existing `getUser`-equivalent helper in this package, call `Service.GetEquipmentPmcsHistory(user)`, write `response.StandardResponse{Status: 200, Data: result, Message: ""}` on success, `writeAggregateError(c, err)` on failure (existing helper, already used by all 4 other handlers).
- **`service.go` / `service_impl.go`** — new interface method `GetEquipmentPmcsHistory(user *bootstrap.User) (*response.EquipmentPmcsHistoryResponse, error)`. No request options to validate or limits to normalize (no caps, no query params), so this is a thin pass-through to the repository — the simplest of the five service methods in this package once added.
- **`repository.go` / `repository_impl.go`** — new method `GetEquipmentPmcsHistory(user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error)`. **Query style note:** this package's existing repository methods (e.g. `bootstrap`) use raw SQL via `database/sql`, not jet, but the two specific queries this method mirrors (`GetShopEquipmentOverview` and `ListInspections`'s fault-count batching) are both jet queries. This method uses jet for both queries, prioritizing consistency with the queries it's directly modeled on over consistency with this package's other, unrelated repository methods — jet's compile-time column/type checking is a better fit for two queries built by copying an existing join shape:
  1. **Query 1 (jet):** all `shop_vehicle` rows across every shop the user is a member of — same join shape as `GetShopEquipmentOverview` (`api/shops/core/repository_impl.go:146-183`): `shops` ⋈ `shop_members` ⋈ `shop_vehicle`, filtered by `shop_members.user_id = $1`, selecting `shop_vehicle.id, shop_vehicle.shop_id, shop_vehicle.admin, shop_vehicle.model, shop_vehicle.serial, shop_vehicle.niin`.
  2. **Short-circuit:** if Query 1 returns zero rows, return `[]response.EquipmentWithPmcsHistory{}` immediately — skip Query 2 entirely (no vehicles means no possible inspections, and there's no reason to issue an `IN ()` query for a known-empty set).
  3. **Query 2 (jet):** batched fetch of every `pmcs_sbs_inspections` row `WHERE equipment_id IN (<vehicle ids from Query 1>)`, with `fault_count` computed via the same `GROUP BY pmcs_id` + `IN` batching technique already used in `pmcs_sbs_progress.RepositoryImpl.ListInspections` (`api/pmcs_sbs_progress/repository_impl.go`) — one query for inspection rows across all vehicles, one query for fault counts across all those inspection ids, no per-row queries. Ordered by `equipment_id, performed_date DESC`.
  4. **Merge in Go:** build `map[string][]response.PmcsHistorySummary` keyed by `equipment_id` from Query 2's results, then iterate Query 1's vehicles in their original order, attaching each one's slice from the map — defaulting to `[]response.PmcsHistorySummary{}` (never `nil`) when the map has no entry, so the JSON field is always `[]`, never `null`.

## Access Control

Identical to every other endpoint in this package and in `pmcs_sbs_progress`: visibility is entirely determined by `shop_members.user_id = <caller>`. A vehicle the caller's shop memberships don't include is never fetched by Query 1, so it's structurally impossible for it to appear in Query 2 or the response — no separate per-vehicle authorization check is needed because the join itself is the authorization boundary (same as `GetShopEquipmentOverview`).

## Error Handling

No per-resource error surface — there's no path parameter to be invalid and no request body to malform, so the only failure modes are:

| Condition | Handling |
|---|---|
| Missing/invalid authentication | `401`, existing auth-check helper in this package |
| User has zero shop memberships | Not an error — `{"equipment": [], "count": 0}` |
| Unexpected DB error (either query) | `500` via the existing `writeAggregateError` / `ErrAggregateUnavailable` sentinel, same as this package's other 4 endpoints |

## Testing

Following this package's existing test conventions (`handler_test.go`, `service_impl_test.go`, and a real-Postgres repository test):

- **Repository, real Postgres:** user with multiple shops and multiple vehicles per shop; some vehicles with multiple inspections (including at least one clean/zero-fault inspection); one vehicle with zero inspections. Asserts: correct grouping per vehicle, `performed_date DESC` ordering within each vehicle's `historical_pmcs`, and that the zero-inspection vehicle appears with `historical_pmcs: []`.
- **Repository, real Postgres:** user with zero shop memberships → `[]response.EquipmentWithPmcsHistory{}`, no error, and (inspectable via a query-count assertion or mock) Query 2 is never issued.
- **Repository, real Postgres:** cross-user isolation — a vehicle belonging to a shop the requesting user is not a member of never appears in the result, even if that vehicle has PMCS history.
- **Service:** thin pass-through — asserts the repository's error/result flow through unchanged (no transformation logic to test beyond that, since there's no validation step).
- **Handler:** `401` with no authenticated user in context; successful call returns `200` with the service's result wrapped in `response.StandardResponse`.

## Out of Scope

- Any pagination, filtering (by `guide_manual`, date range, etc.), or per-vehicle inspection cap — explicitly rejected in favor of an unbounded response.
- Full fault line-item detail inline — explicitly rejected in favor of summary-only, with drill-down via the existing single-inspection endpoint.
- Shop-level grouping of the equipment list — explicitly rejected in favor of a flat list with `shop_id` on each entry.
- Any change to the existing `GET /pmcs-sbs/equipment/:equipment_id/pmcs` or other `pmcs_sbs_progress` endpoints — this is a new, additive aggregate endpoint only.
