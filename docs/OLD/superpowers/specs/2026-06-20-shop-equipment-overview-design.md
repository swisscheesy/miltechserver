# Shop Equipment Overview Endpoint Design

Date: 2026-06-20
Status: Approved design

## Objective

Add an authenticated endpoint that returns every shop the current user belongs to and a compact overview of every equipment record in those shops. The response is intentionally unbounded and is intended for an overview screen rather than synchronization or export.

The design target is a user belonging to as many as 100 shops with 25,000 equipment records in total. With a warm PostgreSQL cache, the endpoint must achieve p95 latency below one second.

## Current Implementation

Shops use bounded packages under `api/shops`. The `core` package owns user-level shop retrieval, including:

- `GET /api/v1/auth/shops`, which returns shops for the authenticated user;
- `GET /api/v1/auth/shops/user-data`, which returns authenticated user data and shop statistics;
- the existing handler, service, and repository boundaries for membership-rooted shop aggregates.

Equipment is stored in `shop_vehicle` and exposed per shop through `api/shops/vehicles`. The existing `GetShopVehicles` flow performs a separate membership check and then queries one shop. Calling that flow once per shop would create an N+1 query pattern and is not suitable for the new aggregate endpoint.

The generated Jet models expose complete `shops`, `shop_members`, and `shop_vehicle` records. The new endpoint needs only a projection of those records and therefore must use dedicated response DTOs rather than generated database models.

## API Contract

### Route

```text
GET /api/v1/auth/shops/equipment/overview
```

The route is registered under the existing Firebase-authenticated route group. It accepts no user ID or shop ID. The authenticated Firebase UID is read only from Gin request context.

### Successful response

The endpoint uses the existing `response.StandardResponse` envelope:

```json
{
  "status": 200,
  "message": "Shop equipment overview retrieved successfully",
  "data": {
    "shops": [
      {
        "id": "shop-id",
        "name": "Shop 1",
        "details": "Shop description",
        "role": "member",
        "equipment_count": 2,
        "equipment": [
          {
            "id": "equipment-id",
            "admin": "12345",
            "model": "M1097",
            "serial": "ABC123",
            "niin": "012345678"
          }
        ]
      }
    ]
  }
}
```

Response rules:

- A user with no shop memberships receives `200` with `{"shops": []}`.
- A shop with no equipment remains in the response with `equipment_count: 0` and `equipment: []`.
- `equipment_count` equals the length of the returned `equipment` array.
- Shops are ordered by `shops.created_at DESC`.
- Equipment within each shop is ordered by `shop_vehicle.save_time DESC`, then `shop_vehicle.id DESC` as a deterministic tie-breaker.
- The response contains no pagination metadata, continuation token, truncation indicator, or total-result cap.

### Response DTOs

Define dedicated DTOs in the existing response package:

- an endpoint response containing `shops`;
- a shop overview containing `id`, `name`, `details`, `role`, `equipment_count`, and `equipment`;
- a compact equipment overview containing `id`, `admin`, `model`, `serial`, and `niin`.

The DTOs prevent accidental response growth when generated Jet models gain columns.

## Component Placement

Implement the endpoint in `api/shops/core`:

- `route.go` registers the route;
- `handler.go` extracts request context and writes the response;
- `service.go` and `service_impl.go` expose the authenticated use case;
- `repository.go` and `repository_impl.go` expose and implement the aggregate query.

This placement is consistent with the existing user-level Shops aggregate endpoints. It avoids creating a cross-domain facade or placing a user-level aggregate in the per-shop `vehicles` package.

## Query Design

Execute one parameterized Jet query rooted in membership:

1. Start from `shop_members`.
2. Filter `shop_members.user_id` by the authenticated Firebase UID.
3. Inner join `shops` on `shops.id = shop_members.shop_id`.
4. Left join `shop_vehicle` on `shop_vehicle.shop_id = shops.id`.
5. Select only the approved DTO fields plus ordering fields needed for deterministic output.
6. Apply the approved shop and equipment ordering.
7. Map the joined result into nested response DTOs through Jet query-result mapping.

The left join is required so shops without equipment are not removed. The membership predicate performs authorization within the data query, eliminating separate per-shop authorization checks.

The handler passes `c.Request.Context()` through the service and repository. The repository executes the Jet statement with that context so client disconnection cancels database work. The endpoint adds no server-imposed deadline.

The query is a single PostgreSQL statement, so all returned rows come from one statement snapshot. Changes committed after the query begins may appear only in the next request.

## Database Schema and Index Assessment

The deployed database currently has the necessary access-path indexes:

- `idx_shop_members_user_id` on `shop_members(user_id)`;
- the unique `shop_members_shop_id_user_id_key` on `shop_members(shop_id, user_id)`;
- `idx_shop_vehicle_shop_save_time` on `shop_vehicle(shop_id, save_time DESC)`;
- primary-key indexes on `shops.id` and `shop_vehicle.id`.

The live development data inspected during design contained four shops, six memberships, and eight equipment records. That dataset is too small to validate production-scale planner behavior; sequential scans are rational at that size.

No schema migration is part of the initial implementation. An additional index may be proposed only when a representative 100-shop/25,000-equipment benchmark and `EXPLAIN (ANALYZE, BUFFERS)` demonstrate a specific bottleneck. This avoids redundant indexes and their write/storage costs.

## Transport and Serialization

Use normal buffered JSON serialization. Do not stream the response. Buffered serialization ensures query or encoding failures can be returned before response headers and a partial JSON document are sent.

Production currently has no application-level response compression. Apply gzip only to this endpoint when the request advertises `Accept-Encoding: gzip`. Clients that do not request gzip receive ordinary JSON. Endpoint-scoped compression avoids changing unrelated API behavior.

The endpoint has:

- no pagination or truncation;
- no response-size cap;
- no application cache or conditional cache;
- no endpoint deadline;
- no endpoint-specific rate limiter.

## Performance Design

Performance comes from bounded field selection and a set-based query, not concurrency or caching:

- one database round trip;
- no per-shop membership checks;
- no per-shop equipment queries;
- no database count query;
- only approved response columns;
- `equipment_count` derived from the mapped slice;
- Jet nested result mapping instead of manual per-shop query loops;
- request-context cancellation;
- gzip for supporting clients.

Record query duration, total handler duration, shop count, and equipment count in structured logs. Do not log equipment field values or response bodies.

The accepted performance target is warm-cache p95 below one second for 100 shops and 25,000 total equipment records. The implementation must measure rather than assume compliance.

## Error Handling

The endpoint is all-or-nothing:

- Missing authenticated user context returns `401 Unauthorized`.
- Query and result-mapping failures flow through the existing error middleware and return a generic `500 Internal Server Error` without database details.
- A canceled request cancels database work through context; no partial response is written.
- JSON or compression setup failures return a generic server error before a response body is committed.
- Empty membership is a successful empty result, not `404`.

## Security

- The Firebase UID is accepted only from authenticated Gin context.
- The membership predicate limits all shop and equipment rows to the authenticated user.
- All query values are bound parameters; no user-controlled SQL fragments are constructed.
- Dedicated DTOs prevent unrelated or future database fields from leaking into the response.
- No response is cached, preventing cross-user cache contamination.
- Logs contain aggregate counts and identifiers needed for diagnosis, not equipment content.

The user explicitly chose not to add endpoint-specific rate limiting. This leaves repeated expensive requests as a residual abuse risk to be handled by existing authentication and infrastructure controls.

## Testing Strategy

### Contract and integration tests

Add Shops integration coverage for:

- authenticated success through the production route registration;
- multiple shops with equipment assigned to the correct parent;
- a member seeing only shops they belong to;
- equipment from non-member shops never appearing;
- shops with no equipment;
- a user with no memberships;
- correct membership role;
- correct `equipment_count`;
- exact compact equipment projection;
- descending shop and equipment ordering, including equal `save_time` tie-breaking;
- database failure returning a generic server error;
- response arrays encoded as `[]`, not `null`.

### Context and compression tests

Verify that:

- request cancellation reaches the repository/database operation;
- `Accept-Encoding: gzip` produces a gzip response;
- decompressed gzip JSON matches the normal JSON contract;
- a client without gzip support receives uncompressed JSON;
- compression remains scoped to this endpoint.

### Performance validation

Create a repeatable benchmark fixture with one user, 100 shops, and 25,000 equipment records. Capture:

- warm-cache p50 and p95 latency;
- database query time;
- total handler time;
- allocations and peak memory per request;
- compressed and uncompressed response sizes;
- `EXPLAIN (ANALYZE, BUFFERS)` output.

The plan must define how the fixture is loaded and cleaned up without affecting ordinary integration data. Performance results must be recorded before claiming the target is met.

## Alternatives Considered

### Two batched queries

Fetch shops first and all matching equipment second, then group in Go. This reduces repeated shop columns in database results but adds a round trip and requires a read-only repeatable-read transaction for snapshot consistency. The extra orchestration is not justified before the single-query design is benchmarked.

### PostgreSQL JSON aggregation

Build nested JSON in PostgreSQL. This reduces result-row transfer to Go but moves response construction and memory pressure into the database, weakens type-safe DTO mapping, and makes the response contract more tightly coupled to SQL. It is a possible evidence-driven optimization, not the initial design.

### Per-shop equipment loading

Call the existing vehicle service once for every shop. This creates N+1 database and authorization queries and is rejected.

### Pagination

Paginate shops or equipment. This would provide stronger upper bounds, but the approved contract explicitly requires a single unbounded response.

### Caching

Cache user overview responses or return conditional `304` responses. This is rejected because the approved contract requires each request to query current data.

## Scope Boundaries

This feature does not:

- change existing `/shops`, `/shops/user-data`, or per-shop vehicle endpoints;
- change shop membership or equipment permissions;
- add pagination, caching, rate limiting, or server deadlines;
- return complete shop or equipment database models;
- change equipment creation, update, deletion, or ordering behavior;
- add a database migration without measured evidence;
- edit generated Jet files by hand.

Jet-generated models must be regenerated only if a later evidence-backed schema migration changes the database schema.
