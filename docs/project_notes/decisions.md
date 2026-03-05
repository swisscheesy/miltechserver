# Architectural Decisions

This file logs architectural decisions (ADRs) for the miltechserver project. Use bullet lists for clarity.

## Format

Each decision should include:
- Date and ADR number
- Context (why the decision was needed)
- Decision (what was chosen)
- Alternatives considered
- Consequences (trade-offs, implications)

## Existing Architecture

Based on the current project setup:

### ADR-001: Use Gin Web Framework (Established)

**Context:**
- Need a high-performance HTTP web framework for Go
- API server handling military tech data lookups

**Decision:**
- Use Gin as the web framework

**Consequences:**
- Fast HTTP routing and middleware support
- Well-documented and widely adopted
- Good performance characteristics for API workloads

### ADR-002: Use Jet for Database Querying (Established)

**Context:**
- Need type-safe SQL query building for PostgreSQL
- Want to avoid raw SQL strings and reduce SQL injection risks

**Decision:**
- Use Jet for database querying and model generation

**Consequences:**
- Type-safe queries at compile time
- Auto-generated models from database schema
- Learning curve for team unfamiliar with Jet

### ADR-003: Use Firebase Auth for Authentication (Established)

**Context:**
- Need secure user authentication
- Want to offload auth complexity to managed service

**Decision:**
- Use Firebase Auth for user authentication

**Consequences:**
- Managed authentication with multiple providers
- JWT token verification in Go middleware
- Dependency on Firebase/Google services

### ADR-004: Use Azure Blob Storage for External Data (Established)

**Context:**
- Need to store images and large binary data
- Require scalable, cost-effective object storage

**Decision:**
- Use Azure Blob Storage for external data storage

**Consequences:**
- Scalable object storage
- Integration with Azure ecosystem
- Need to handle connection/credential management

## New Decisions

<!-- Add new ADRs below this line -->

### ADR-005: Refactor EIC Domain into Colocated Module (2026-01-31)

**Context:**
- EIC domain had large monolithic repository with repeated SQL and scanning logic
- Code was scattered across controller/service/repository/route directories
- Error handling relied on string matching and duplication made changes risky

**Decision:**
- Consolidate EIC into `api/eic` with shared query builder and row scanner
- Use typed errors for not-found/invalid cases while preserving external behavior
- Keep response types in `api/response` to maintain API contract

**Alternatives considered:**
- Full bounded-context decomposition (lookup/browse sub-packages)
- Keep legacy structure and only deduplicate SQL

**Consequences:**
- Significantly reduced SQL and scanning duplication
- Clearer ownership and lower maintenance overhead for EIC lookups
- Requires verification before removing legacy files and adding tests afterward

### ADR-006: Refactor Library Domain into Colocated Module (2026-01-31)

**Context:**
- Library domain was split across controller/service/repository/route/response/request folders
- Error handling relied on string matching in the controller
- Repository layer was unused scaffolding for future features

**Decision:**
- Consolidate library into `api/library` with colocated route, service, errors, and response types
- Use typed errors with `errors.Is()` in handlers
- Remove unused repository and request scaffolding

**Alternatives considered:**
- Keep legacy structure and only replace string error matching
- Decompose into sub-contexts (pmcs/bii/favorites) before those features exist

**Consequences:**
- Clearer domain ownership and reduced file scatter
- Safer error handling with typed errors
- Adds tests for route/service validation but still requires manual API verification

### ADR-007: Refactor Item Comments Domain into Colocated Module (2026-01-31)

**Context:**
- item_comments logic was split across controller/service/repository/route/request/response directories
- Mixed public + authenticated routes needed consistent registration and error handling
- Existing typed errors and validation were already solid; refactor was for organization consistency

**Decision:**
- Consolidate item_comments into `api/item_comments` with colocated route, service, repository, errors, and types
- Keep raw SQL join for author display names
- Rewire main router to `item_comments.RegisterRoutes` (public + auth groups)
- Add unit + integration tests; defer legacy deletion until validation is complete

**Alternatives considered:**
- Keep legacy structure and only update routing
- Convert the raw SQL join to Jet (riskier, not required for refactor)

**Consequences:**
- Clearer ownership with fewer directories and consistent route registration
- Minimal LOC reduction since behavior and validation were preserved
- Legacy files remain until manual/API validation confirms the new module

### ADR-008: Refactor Item Query Domain into Colocated Module (2026-01-31)

**Context:**
- item_query logic was split across controller/service/repository/route with a 517 LOC detailed repository file
- Detailed queries silently ignored errors from most helper queries
- Short query handlers used string matching on error text for 404s

**Decision:**
- Consolidate item_query into `api/item_query` with shared, short, and detailed subpackages
- Preserve endpoints and response shapes; keep status code behavior unchanged
- Log detailed query failures server-side instead of swallowing errors
- Keep analytics tracking via a small interface and wire the real implementation
- Replace string matching with typed errors + `errors.Is()` in short handlers

**Alternatives considered:**
- Keep legacy structure and only fix error handling
- Return partial-data errors in the response body

**Consequences:**
- File sizes are smaller and concerns are separated cleanly
- Clients see no API shape or status changes while server logs now surface failures
- Manual verification is still recommended for detailed query correctness

### ADR-009: Refactor Small Domains into Colocated Modules (2026-01-31)

**Context:**
- analytics, item_quick_lists, and user_general were split across controller/service/repository/route/response/request folders
- These domains lagged behind the colocated module pattern used elsewhere
- user_general contained a logging format bug and relied on string error matching

**Decision:**
- Consolidate the domains into `api/analytics`, `api/quick_lists`, and `api/user_general`
- Add typed errors for user_general and replace string matching with `errors.Is()`
- Consolidate quick_lists response types and user_general request types within each domain
- Update route registration to use `RegisterRoutes()` and wire analytics via a `New()` constructor
- Add unit and route tests for quick_lists and user_general

**Alternatives considered:**
- Keep legacy structure and only fix the user_general logging bug
- Update route registration without moving files
- Leave analytics in shared service/repository directories

**Consequences:**
- Consistent module layout and dependency wiring for small domains
- Easier maintenance and clearer ownership of request/response types
- No external behavior changes, but new tests provide coverage for refactored handlers

### ADR-010: Item Query Performance Optimization (2026-01-31)

**Context:**
- Detailed item query endpoint (`GET /api/v1/queries/items/detailed`) executed ~45 sequential database queries
- Each request made serial round-trips to 10 query functions, each containing 1-8 internal queries
- Default Go connection pool (MaxIdleConns=2) was insufficient for parallel workloads
- No request context propagation meant queries couldn't be cancelled on client disconnect
- Identical NIIN lookups hit the database repeatedly with no caching

**Decision:**
- Implement 7 optimizations as designed in `docs/designs/item_query_performance_optimization_design.md`:
  1. **Connection Pool Tuning**: Configure `MaxOpenConns=50`, `MaxIdleConns=25`, with connection recycling via `ConnMaxLifetime=5min`
  2. **Top-Level Parallelization**: Use `errgroup` to execute all 10 query functions concurrently in `repository_impl.go`
  3. **Context Propagation**: Thread `context.Context` from handler → service → repository → queries; use `QueryContext` instead of `Query`
  4. **Inner Query Parallelization**: Apply `errgroup` within each query function for independent sub-queries
  5. **In-Memory Caching**: Add TTL-based cache (24h) at service layer with background cleanup
  6. **Async Analytics**: Use buffered channel for fire-and-forget analytics in short query service
  7. **Database Indexes**: Create migration script for NIIN indexes on all queried tables

**Alternatives considered:**
- Single PostgreSQL stored function returning all data (rejected: harder to maintain, less flexible)
- Redis caching (deferred: single instance deployment doesn't require distributed cache)
- Query result streaming (rejected: response structure requires full data assembly)

**Consequences:**
- Expected 10-40x performance improvement (from ~45 sequential to ~1-2 parallel round-trips)
- Cache hits return in <1ms without database load
- Queries respect request cancellation via context
- Partial data returned on individual query failures (logged but non-fatal)
- Connection pool settings configurable via `DB_MAX_OPEN_CONNS` and `DB_MAX_IDLE_CONNS` env vars
- Migration `003_create_item_query_indexes.sql` must be run to create NIIN indexes

### ADR-011: Complete Shops HTTP Adapter Migration to Bounded Subdomains (2026-02-18)

**Context:**
- `shops` domain logic was already separated into bounded subpackages under `api/shops/*`
- HTTP route adapters still depended on global `api/controller/shops_controller_*.go` handlers
- `api/shops/facade` exposed a very wide interface coupling unrelated subdomains together
- Other domains (`item_lookup`, `item_query`, `library`, etc.) had already adopted colocated route/handler/service modules

**Decision:**
- Move shops HTTP handlers into subdomain-local files:
  - `api/shops/core/handler.go`
  - `api/shops/settings/handler.go`
  - `api/shops/members/handler.go`
  - `api/shops/members/invites/handler.go`
  - `api/shops/lists/handler.go`
  - `api/shops/lists/items/handler.go`
  - `api/shops/messages/handler.go`
  - `api/shops/vehicles/handler.go`
  - `api/shops/vehicles/notifications/handler.go`
  - `api/shops/vehicles/notifications/items/handler.go`
  - `api/shops/vehicles/notifications/changes/handler.go`
- Rewire each `api/shops/*/route.go` to register endpoints directly from local handlers and local service interfaces
- Remove `api/shops/facade` and `api/controller/shops_controller_*.go`
- Preserve existing route paths and response schemas
- Keep direct migration approach (no feature flag path)

**Alternatives considered:**
- Keep the controller/facade architecture and only reorganize folders
- Introduce feature flags and dual-route wiring for phased runtime switching
- Migrate only one shops subdomain and defer full completion

**Consequences:**
- Shops now matches the bounded-context HTTP adapter pattern used by other refactored domains
- Reduced coupling by eliminating monolithic controller and facade interfaces
- Route registration now depends on narrow subdomain service contracts
- Existing API contract remains unchanged while internal architecture is simplified
- Full suite verification was required due wide route wiring changes (`go test ./...`)

### ADR-011: Shops Performance Optimization Refactor (2026-02-01)

**Context:**
- Shops endpoints executed repeated authorization checks per request and used COUNT-based membership queries
- Vehicle notifications loaded items with an N+1 pattern
- Shop stats admin subquery scanned all admins instead of filtering by user
- Blob cleanup used unbounded sequential deletes with no timeout
- Paginated messages relied on offset-based SQL without cursor opt-in
- Several handlers allocated slices without pre-sizing, and vehicle service had no-op assignments

**Decision:**
- Add request-scoped authorization caching via Gin context and a cached wrapper
- Replace COUNT membership/admin checks with LIMIT 1 existence checks
- Fix notification N+1 by fetching items in a single IN query and grouping in memory
- Filter admin_check subquery by user_id in the shops stats query
- Add blob listing timeout and bounded concurrent deletions (best-effort, log-only failures)
- Implement optional cursor pagination for shop messages with `before_id`/`after_id` and `next_cursor` response field; keep existing page/limit behavior
- Pre-allocate known-size slices and remove no-op vehicle field assignments
- Do not add the invite code index (intentionally omitted)

**Alternatives considered:**
- Leave authorization checks uncached (rejected: redundant per-request queries)
- Use Redis for cross-request caching (deferred: not required for single instance)
- Keep offset-only pagination (rejected: degrades with deep history)
- Move blob deletion to background jobs (deferred: out of current scope)

**Consequences:**
- Fewer DB round-trips on hot request paths and faster notification retrieval
- Cursor pagination is opt-in and backwards compatible; pagination metadata is omitted for cursor responses
- Blob cleanup completes faster but still tolerates partial failures without surfacing to users
- Manual index creation remains required; invite code lookup still relies on existing schema

### ADR-012: LIN SearchByPage Performance Optimization (2026-02-15)

**Context:**
- Mobile users reported slow load times on the LIN lookup page
- `SearchByPage` executed two separate queries against `lookup_lin_niin` (a view joining `nsn` 7.3M rows / 698 MB with `army_lin_to_niin` 18K rows)
- The COUNT query forced a full nested-loop join on every page request (114ms), even though the count (~18,204) rarely changes
- OFFSET-based pagination degraded linearly: page 1 at 0.28ms, last page at 271ms (968x slower)
- No ORDER BY clause meant pagination results were non-deterministic (correctness bug)
- Two DB roundtrips doubled mobile network latency impact
- Full analysis documented in `docs/designs/lin_searchbypage_performance_analysis.md`

**Decision:**
- **Switch from view to materialized view**: Replace `lookup_lin_niin` (view) with `lookup_lin_niin_mat` (materialized view) across all repository queries. The materialized view pre-computes the join, eliminating the 698 MB `nsn` table from the hot query path. Queries now scan a ~1 MB precomputed dataset instead of joining on every request.
- **Cache the total count with 15-day TTL**: Add an in-memory count cache to `RepositoryImpl` using `sync.RWMutex`. The COUNT query only executes on first request and after TTL expiry. 15-day TTL chosen because this is reference data that changes only during bulk data imports.
- **Add deterministic ORDER BY**: Add `ORDER BY lin ASC, niin ASC` to the paginated query to fix non-deterministic pagination. Users will no longer see duplicates or miss rows when navigating pages.

**Alternatives considered:**
- Window function `COUNT(*) OVER()` to combine into single query (rejected: still computes full count on every request; caching is more effective)
- Keyset/cursor pagination (deferred: requires API contract change and mobile app update; OFFSET is acceptable for 911 pages with materialized view)
- Redis for count caching (rejected: single-value cache doesn't justify distributed cache dependency)
- Background goroutine cleanup for cache (rejected: single-value cache with 15-day TTL doesn't need periodic cleanup)

**Consequences:**
- COUNT query eliminated on cache hits (114ms -> 0ms); first request still pays ~4ms against materialized view
- Data queries against materialized view are faster (no join overhead) and deterministic (ORDER BY)
- Materialized view must be refreshed when underlying data changes (`REFRESH MATERIALIZED VIEW CONCURRENTLY lookup_lin_niin_mat`)
- API response JSON is unchanged (`LookupLinNiinMat` has identical fields/tags to `LookupLinNiin`); no mobile app changes needed
- Tests updated to reference `lookup_lin_niin_mat`; existing test behavior preserved
- OFFSET degradation on later pages remains but is mitigated by querying a small materialized table instead of a 698 MB join

### ADR-013: Equipment Details Public API with Image Browsing (2026-03-02)

**Context:**
- New `docs_equipment_details` table added to Postgres (488 rows, 12 columns, 16 equipment families)
- Equipment images stored in Azure Blob Storage at `library/docs_equipment/images/{family}/`
- Mobile app needs endpoints to browse equipment data, filter by family/search by model or LIN, and view/download associated images
- All endpoints must be public (no authentication) to support offline-capable mobile workflows
- Multiple existing patterns to follow: EIC (paginated DB queries), PMCS library (blob folder listing + SAS download)

**Decision:**
- Create new colocated module `api/docs_equipment` with standard `route → service → repository` architecture
- **4 data endpoints** querying Postgres:
  1. `GET /equipment-details?page=N` — paginated list (40/page, ordered by ID)
  2. `GET /equipment-details/families` — unique family values for filter UI
  3. `GET /equipment-details/family/:family?page=N` — filter by family (case-insensitive)
  4. `GET /equipment-details/search?q=term&page=N` — search model or LIN (ILIKE partial match)
- **3 image endpoints** querying Azure Blob Storage:
  5. `GET /equipment-details/images/families` — list family image folders via hierarchy pager
  6. `GET /equipment-details/images/family/:family` — list images in a family folder (flat pager, image-extension whitelist)
  7. `GET /equipment-details/images/download?blob_path=...` — generate 1-hour SAS read URL (rate-limited)
- Use raw SQL with `database/sql` for DB queries (not Jet query builder) — consistent with EIC pagination pattern where dynamic WHERE clauses and LIMIT/OFFSET are cleaner in raw SQL
- Use `shared.GenerateBlobSASURL` for image download — same User Delegation SAS pattern as PMCS and PS Magazine
- Register as public routes in `api/route/route.go` alongside `eic` and `pol_products`

**Alternatives considered:**
- Use Jet query builder for DB queries (rejected: raw SQL is simpler for pagination with dynamic filters; EIC set the precedent)
- Server-side image proxying/streaming instead of SAS URLs (rejected: SAS URLs offload bandwidth to Azure CDN, consistent with all existing download endpoints, and simpler for mobile caching)
- Single combined endpoint returning equipment data + image URLs together (rejected: not all families have images, images are in blob storage not DB, separate concerns are cleaner)
- Separate `api/docs_equipment_images` package for image endpoints (rejected: all endpoints share the same domain context and Dependencies struct; splitting would add unnecessary complexity)

**Consequences:**
- Mobile app can build a complete equipment browser with category filtering, search, and image gallery
- No authentication overhead — endpoints are immediately accessible
- Image download URL generation is rate-limited to prevent abuse
- Image extension whitelist (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) prevents unauthorized file access through the download endpoint
- Page size of 40 is consistent with EIC but may need tuning based on mobile app UX feedback
- If equipment data grows significantly, consider adding database indexes on `family`, `model`, and `lin` columns
