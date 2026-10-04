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

### ADR-022: Shops Performance Optimization Refactor (2026-02-01)

*(Originally numbered ADR-011; renumbered 2026-08-31 to resolve a duplicate heading. No content changed.)*

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

### ADR-014: PMCS Step-by-Step JSON API as Library Sub-Package (2026-05-22)

**Context:**
- PMCS Step-by-Step (SBS) files are structured JSON documents stored in Azure Blob Storage under `pmcs_sbs/<vehicle>/` prefixes
- Mobile app needs to browse available vehicle folders, list JSON files within a folder, and fetch raw JSON content for offline use
- The `api/library/ps_mag` sub-package already demonstrated a viable pattern for blob-backed library content with its own handler, service interface, service implementation, and tests colocated under `api/library/`
- Content proxy endpoint carries non-trivial bandwidth cost, warranting rate limiting
- Blob content must be validated as JSON before being forwarded to clients — corrupt blobs should fail loudly, not silently return garbage

**Decision:**
- Create `api/library/pmcs_sbs/` as a new sub-package following the exact `ps_mag` pattern: `errors.go`, `response.go`, `service.go`, `service_impl.go`, `route.go`, and tests colocated in the package
- Expose three public endpoints registered via `api/library/route.go`:
  1. `GET /library/pmcs-sbs/folders` — list top-level folders via Azure hierarchy pager
  2. `GET /library/pmcs-sbs/:folder/files` — list `.json` files in a folder via Azure flat pager
  3. `GET /library/pmcs-sbs/content?blob_path=...` — proxy raw JSON content (rate-limited)
- Use a `Service` interface with all three methods accepting `context.Context` so Azure SDK calls respect HTTP request cancellation — the original plan omitted context on `GetFolders`/`GetFiles` and was corrected before implementation
- Apply `path.Clean` + `pmcs_sbs/` prefix check in `GetFileContent` to prevent directory traversal attacks; apply `strings.ContainsAny(folderName, "./\\")` guard in `GetFiles` for the same reason
- Cap `io.ReadAll` at 10 MB via `io.LimitReader` to prevent memory exhaustion from oversized or adversarial blobs
- Validate blob content with `json.Valid` before returning it; return `ErrInvalidJSON` on failure
- Return `json.RawMessage` from `GetFileContent` so JSON is proxied inline without double-encoding in `StandardResponse.Data`
- Strip internal error details from all 500 responses across `pmcs_sbs`, `ps_mag`, and the parent `library` package — Azure SDK error strings (containing account names, endpoint URLs, and request IDs) should log to slog but never reach clients

**Alternatives considered:**
- SAS URL redirect instead of content proxy (rejected: JSON files are small text, not large binaries; proxying avoids exposing Azure storage account details to clients and is consistent with how the mobile app expects to receive structured content)
- Adding DB/analytics dependency like `ps_mag` (rejected: pmcs_sbs is blob-only with no search index or download tracking requirement at this time)
- Single flat endpoint returning all folders and files together (rejected: folder list and file list serve different UI states; separating them avoids over-fetching)
- Skipping `json.Valid` check and trusting blob content (rejected: a corrupted or replaced blob would silently propagate invalid JSON to clients; the check is O(n) and inexpensive for the expected file sizes)

**Consequences:**
- Three new public endpoints are live with no authentication requirement
- Rate limiting on the content proxy endpoint prevents bandwidth abuse; folder/file listing endpoints are not rate-limited (consistent with `ps_mag`)
- Path traversal attacks are blocked at both the folder-listing and content-download layers
- Memory per content request is bounded at 10 MB regardless of blob size
- Context propagation ensures all Azure calls cancel immediately when a client disconnects
- 500 error responses across all library handlers no longer expose Azure internals (this fix applied retroactively to `ps_mag` and the parent `library` package as part of this work)
- `ps_mag` and `pmcs_sbs` now share the same security posture on error responses; future library sub-packages should follow the same pattern

### ADR-015: Unbounded Shop Equipment Overview (2026-06-20)

**Context:**
- The overview client needs every shop the authenticated user belongs to and compact equipment identity fields in one current response.
- The accepted representative load is 100 shops and 25,000 equipment records, with warm-cache p95 below one second.
- Existing per-shop vehicle retrieval would create an N+1 query pattern.

**Decision:**
- Add `GET /api/v1/auth/shops/equipment/overview` under `api/shops/core`.
- Use one membership-filtered Jet query with a left join to equipment and request-context cancellation.
- Return compact DTOs, preserve empty shops, and apply endpoint-scoped gzip.
- Keep the response unbounded and uncached, with no endpoint deadline or endpoint-specific rate limiter.
- Keep the existing schema because representative evidence did not justify another index.

**Consequences:**
- The endpoint performs one database round trip and limits rows through membership in the query.
- Memory and bandwidth grow linearly with equipment count; gzip reduces transfer size but not DTO allocation.
- If workloads exceed the accepted bound or p95 target, revisit the transport contract before speculative indexing or caching.

### ADR-016: Add Backward-Compatible Shops Aggregate Read Endpoints (2026-07-03)

**Context:**
- Existing Shops clients can load complex screens by chaining multiple narrow endpoints.
- The existing `GET /shops/equipment/overview` aggregate proved the set-based additive endpoint pattern.
- Some current query shapes needed internal fixes without response-contract changes.

**Decision:**
- Keep all legacy endpoints active and backward compatible.
- Add aggregate read endpoints for list trees, vehicle maintenance snapshots, shop snapshots, and Shops bootstrap.
- Make high-cardinality section limits optional on new aggregate endpoints: omitted limit parameters return all rows for that section, while supplied limit parameters are positive integers capped by endpoint-specific maximums. Existing legacy endpoint contracts remain unchanged.
- Require representative `EXPLAIN (ANALYZE, BUFFERS)` evidence before adding new indexes.

**Consequences:**
- New clients can reduce round trips substantially.
- Old clients continue using existing endpoints.
- Aggregate endpoints must maintain dedicated top-level DTOs and avoid accidental response growth when reusing generated nested model shapes.

### ADR-017: PMCS SBS Inspection History (2026-07-16)

**Context:**
- `pmcs_sbs_faults` keyed faults by `(equipment_id, guide_manual, section_id, item_index)` with no inspection-event dimension — saving a fault always overwrote the prior state for that checklist item, and a clean (zero-fault) inspection left no trace at all
- Users need a historical view of every PMCS performed on a vehicle: the date it was performed and the faults found during it, including clean passes
- A 2026-06-21 design (`docs/OLD/superpowers/plans/2026-06-21-pmcs-sbs-faults-only.md`) deliberately removed a prior `pmcs_sbs_equipment` + `pmcs_sbs_completions` pair of tables to simplify the feature to "faults only" — this ADR reintroduces an inspection-event concept, reversing part of that simplification, because the product requirement now demands it
- The mobile client already autosaves faults one at a time with no start/submit workflow; a full explicit lifecycle (start/finish endpoints, in_progress/completed status) would be a much bigger behavior change than the requirement called for

**Decision:**
- Add `pmcs_sbs_inspections` as the parent of `pmcs_sbs_faults`, FK'd to `shop_vehicle` with `ON DELETE CASCADE`, carrying `guide_manual`, `performed_date`, and `created_by`
- The client generates the inspection id (UUID) and sends it with every fault save in that session — this is what distinguishes a new inspection from a continuation of the last one, not a server-side heuristic like date-bucketing
- Inspection creation is a hybrid of implicit and explicit: saving a fault implicitly creates its parent inspection if one doesn't exist yet (preserving today's autosave contract), and a separate explicit `PUT .../pmcs/:pmcs_id` endpoint exists for the zero-fault clean-completion case
- `performed_date` is client-supplied, not a server write-timestamp, since field techs may inspect offline and sync later
- `guide_manual` is immutable after an inspection's first creation; a mismatched `guide_manual` (or a `pmcs_id` reused under a different `equipment_id`) on a later request is rejected with `ErrInspectionConflict` (409) rather than silently accepted
- No existing `pmcs_sbs_faults` data is migrated — the rows were current-state-only snapshots with no date-performed concept

**Alternatives considered:**
- Full explicit start/finish lifecycle with an in_progress/completed status (rejected: bigger behavior change than the mobile autosave pattern needed; also considered a single atomic submit-everything-at-the-end call, rejected for the same reason)
- One inspection per calendar day, bucketed server-side (rejected: would incorrectly merge two real inspections performed on the same vehicle on the same day)
- Embedding faults as a JSONB array on the inspection row instead of a separate table (rejected: breaks the existing per-fault autosave contract — concurrent shop members editing different faults on the same inspection would race on read-modify-write of the same JSON blob, and per-fault CHECK constraints can't be enforced on individual JSONB array elements)
- Migrating existing fault rows into synthetic "legacy" inspection records (rejected: the data was not judged valuable enough to justify the migration complexity)

**Consequences:**
- Equipment can now have unlimited PMCS inspections over time, each independently listing its own faults, including inspections with zero faults
- The fault API's identity changed from `(equipment_id, guide_manual, section_id, item_index)` to `(pmcs_id, section_id, item_index)` — a breaking change for API consumers, documented in `docs/api/pmcs_sbs_inspections_mobile.md`
- The Flutter mobile client requires a corresponding Drift schema migration (add `pmcs_id` to its local `PmcsSbsFaultsTable` mirror) and client-side generation/tracking of the inspection UUID per session before this can ship end-to-end — tracked as a follow-up in the `miltech` repo, out of scope for this server-side change
- Every write to `pmcs_sbs_faults` now goes through a transaction that also touches `pmcs_sbs_inspections` (to implicitly create the parent row), adding one extra `INSERT ... ON CONFLICT` per fault save compared to the old single-table upsert

### ADR-018: PMCS SBS Inspection Notes + Comments (2026-07-21)

**Context:**
- `pmcs_sbs_inspections` (ADR-017) records only mechanical inspection facts (`equipment_id`, `guide_manual`, `performed_date`, `performed_by`) — there was no field for the inspecting user to leave free-text context, and no way for other shop members to discuss an inspection after the fact
- Users need (1) a single free-text note per inspection, and (2) a multi-user comment thread: any shop member with access to the vehicle can post a comment, edit/delete their own, and see all comments when the inspection is fetched

**Decision:**
- Add a nullable `notes TEXT` column directly to `pmcs_sbs_inspections`, mutable through the existing `PUT .../pmcs/:pmcs_id` upsert (same access as `performed_date` — any shop member, no separate endpoint)
- Add a new child table `pmcs_sbs_inspection_comments` (`id`, `pmcs_id` FK `ON DELETE CASCADE`, `author_id` FK with no delete action, `text`, `created_at`, `updated_at`), following the same relationship shape as `pmcs_sbs_faults` to its parent
- Comments are a flat chronological list — no threading/`parent_id`, since the product requirement was "leave a comment on an inspection," not reply chains
- Comments are edit/soft-delete by author, reusing the exact `api/item_comments` pattern: a `PUT`/`DELETE` on a comment checks `author_id == caller.UserID` (403 otherwise), and delete is a soft-delete (`UPDATE text = 'Deleted by user'`), not a row delete — no new `is_deleted` column needed
- `GET .../pmcs/:pmcs_id` embeds the full `comments` array (with resolved author username via the same `LEFT JOIN users` pattern used for `performed_by_username`) and `GET .../pmcs` (list) gets a batched `comment_count` per inspection, mirroring the existing `fault_count` query exactly
- Comment access control reuses `requireVehicleAccess` (shop-membership check) — there is no role gate, matching how the rest of the inspection record already works

**Alternatives considered:**
- A dedicated `GET .../comments` endpoint (rejected: comments are always wanted alongside inspection detail per the stated requirement; a separate endpoint would just add an extra round trip for the common case)
- Threaded replies via `parent_id` like `item_comments` (rejected: no product requirement for replies, and it adds mobile-client complexity for no stated benefit)
- Restricting note edits to only the inspection's `performed_by` user (rejected: inconsistent with how the rest of the inspection record — `guide_manual`, `performed_date` — is already editable by any shop member with vehicle access)

**Consequences:**
- `InspectionResponse` grew two fields (`notes`, `comments`); `InspectionSummaryResponse` grew one (`comment_count`) — additive, non-breaking for existing API consumers
- `author_id` on `pmcs_sbs_inspection_comments` has no `ON DELETE` action (matching the live `item_comments_author_id_fkey` constraint), so a user with existing comments cannot be hard-deleted from `users` without first handling their comments — same constraint that already exists for `item_comments` authors
- The Flutter mobile client needs corresponding UI to display/edit notes and render the comment thread on the inspection detail screen — out of scope for this server-side change, tracked separately in the `miltech` repo

### ADR-019: Reclaim Orphaned User-PMCS Release on Final Unsubscribe (2026-07-30)

**Context:**
- User-PMCS account deletion retains immutable community releases that are
  pinned by active external subscriptions, while nulling the deleted owner's
  identity and tombstoning the checklist
- If account deletion completes before the last subscriber unsubscribes, that
  final unsubscribe can leave an owner-null, tombstoned checklist and retired
  source with a zero-pin release tree that no account can reach
- The approved v1 design otherwise limits release reclamation to checklist or
  account deletion and does not authorize general unsubscribe garbage
  collection

**Decision:**
- On unsubscribe, reclaim retained release content only when the subscription
  is the final active pin and the source is already owner-null, its checklist
  is tombstoned, and the community source is retired
- Preserve ordinary unsubscribe as a target-row operation and lock only active
  pins needed to prove final-pin eligibility
- Do not garbage-collect active sources, owned checklist history, or ordinary
  unpinned release history during unsubscribe

**Alternatives considered:**
- Leave the retained zero-pin release tree indefinitely
- Run broad release garbage collection on every unsubscribe
- Add a periodic background release collector

**Consequences:**
- Account-deletion-first and unsubscribe-first interleavings converge without
  leaking unreachable retained content
- Active and owned history remains protected from unsubscribe cleanup
- Final-pin unsubscribe performs additional eligibility checks and
  deterministic locking, while ordinary unsubscribe avoids lock amplification
  across historical subscription rows

### ADR-020: Model Shop PMCS Inspections as a Guide-or-Custom Source Union (2026-08-07)

**Context:**
- Shop inspection history previously required every inspection to reference a
  guide JSON path, while mobile clients can now perform user-created PMCS
  checklists against Shop equipment
- Fabricating a guide path for a custom checklist would make provenance
  dishonest and could make clients attempt to load content that does not exist
- Historical inspection records must remain useful when a private checklist is
  device-local, retired, unsubscribed, deleted by its owner, or removed from
  server-side revision history

**Decision:**
- Keep guide and custom inspections in `pmcs_sbs_inspections` and enforce an
  exclusive `source_type = guide | custom` union with database CHECK
  constraints
- Store custom checklist ID, revision ID, revision number, and display name as
  immutable snapshot provenance on the inspection; deliberately add no foreign
  keys to User-PMCS checklist or revision tables
- Continue accepting legacy guide writes that omit `source_type` only when the
  request contains a valid `guide_manual` and no custom fields; all responses
  explicitly return `source_type` and omit inapplicable source fields
- Reuse the authenticated Shop equipment boundary for both source types. The
  custom checklist name, fault data, notes, and comments are visible to every
  current member who can access that Shop equipment, while the authored
  checklist tree is never copied into Shop inspection storage
- Preserve the existing write-through lifecycle: first-fault autosave and
  explicit clean completion use the same inspection record, immutable source
  tuple, and transactional fault persistence

**Alternatives considered:**
- Encode custom identity in a synthetic `custom/...json` guide path (rejected:
  corrupts guide provenance and loading semantics)
- Create separate custom inspection and fault tables/routes (rejected:
  duplicates authorization, history, comments, deletion, and lifecycle logic)
- Reference live User-PMCS checklist/revision rows with foreign keys (rejected:
  device-local content may not exist on the server and historical Shop records
  must survive content retirement or deletion)

**Consequences:**
- Existing guide request routes remain compatible, while inspection detail,
  list, and Shops aggregate history gain additive source-discriminated fields
- A reused inspection UUID must match equipment ID and the complete immutable
  source tuple; mismatches return the existing inspection conflict response
- Custom provenance and faults outlive the originating checklist content, but
  the server cannot reconstruct the full authored checklist tree from an
  inspection record
- The schema migration is reversible only while no custom inspection rows
  exist; rollback refuses to discard custom history

### ADR-021: Rename PMCS SBS Persistence Tables to User PMCS Convention (2026-08-16)

**Context:**
- The PostgreSQL tables `pmcs_sbs_inspections`, `pmcs_sbs_faults`, and
  `pmcs_sbs_inspection_comments` used a legacy prefix that did not follow the
  established `user_pmcs_` convention used by related server-side PMCS tables
- The physical names were referenced by generated Jet identifiers, handwritten
  repositories, integration-test SQL, constraint names, and index names, while
  remaining internal to the server's HTTP contract
- Only the non-production `miltech_ng_test` and `miltech_ng` databases were in
  scope, so the schema and application could be changed as one coordinated
  cutover

**Decision:**
- Rename the tables to `user_pmcs_inspections`, `user_pmcs_faults`, and
  `user_pmcs_inspection_comments` with one metadata-only PostgreSQL transaction
- Rename the 15 table-derived constraints and two explicit secondary indexes
  without changing their definitions; allow primary-key constraint renames to
  carry their backing index names. Add FK-leading indexes on
  `user_pmcs_inspections.performed_by` and
  `user_pmcs_inspection_comments.author_id` to satisfy the existing
  `user_pmcs_%` schema invariant
- Bound lock acquisition with a local timeout, acquire all three table locks in
  one deterministic order, and make a partial rename fail atomically
- Provide an exact inverse rollback, rehearse forward/rollback/forward on
  `miltech_ng_test`, apply forward once to `miltech_ng`, and compare row
  fingerprints and catalog definitions before and after
- Regenerate Jet from the migrated `miltech_ng` schema with the repository's
  canonical JSON-tag template, then mechanically update internal Go and test
  references

**Alternatives considered:**
- Compatibility views under the legacy names (rejected: unnecessary for a
  coordinated development cutover and unsafe for the existing write patterns)
- Create-copy-swap replacement tables (rejected: adds data-copy and dependency
  reconstruction risks to a metadata-only change)
- Dual writes or synchronization triggers (rejected: no old and new application
  versions need to run concurrently)

**Consequences:**
- Old and new binaries require their matching schema names and cannot run
  concurrently during the cutover
- Inspection, fault, comment, authorization, cascade, ordering, Shop aggregate,
  route, and JSON behavior do not change; the two additive indexes improve FK
  maintenance and preserve the repository's schema-integrity contract
- Migration `014` is data-preserving and reversible, and its rollback behavior
  is exercised only on `miltech_ng_test`; production is not part of this
  decision or migration execution
- Historical migrations and ADRs retain the table names that were correct when
  those records were written

### ADR-022: Shops Message Sync Numbering by Trigger, Gated by Flag and Probe (2026-09-29)

**Context:**
- Message synchronization needs a per-shop, commit-ordered `insertion_number` so
  clients can catch up from a watermark without missing a message
- Released clients and older server binaries insert messages and must keep
  working against the expanded schema; the `/shops/capabilities` answer must not
  promise sync before the schema exists
- Historical context at ADR-022 adoption: `response.ShopMessageResponse` embedded
  the Jet model. The explicit DTO decision below supersedes that implementation;
  the legacy JSON key set remains a compatibility contract

**Decision:**
- Migration 018 adds `shop_messages.insertion_number`, a `shop_message_counters`
  table and a `BEFORE INSERT` trigger that is the **sole allocator**; it
  overwrites any supplied value, so every writer (old binaries, ad-hoc SQL) is
  numbered and application code never sets it
- `message_sync` is advertised only when `SHOPS_MESSAGE_SYNC_ENABLED=true` and a
  catalog probe finds the counter table and an enabled trigger (capability =
  flag AND probe); the probe is not backfill or fleet proof, so rollout gates
  stay operator-verified
- Superseded on 2026-10-03: `response.ShopMessageResponse` now explicitly owns
  the nine legacy fields, and legacy reads/readback select explicit columns.
  Regenerate after every authorized migration/data repair through
  `tools/jetregen`; `insertion_number` keeps its normal generated JSON tag but
  cannot leak through the message DTO. Sync and aggregates use the same DTO.
- Startup still requires tagged Jet generation before route registration, using
  the application's actual database pool/port. The shared generator validates
  legacy message column types/nullability, allowing pre/post-018 and compatible
  extra columns, and publishes staged canonical `miltech_ng/<schema>` output
  under a cross-process lock with backup/rollback. Publication uses two renames,
  not an atomic exchange for readers outside that lock.
- The generation manifest records the real source identity and schema catalog
  digest separately from the canonical namespace. Runtime generation does not
  rebuild or certify the running binary; release builds must use intended schema
  inputs and pass the later migration/function/grant readiness gates. The
  disposable fixture lacks `LookupLinNiinMat`, so its focused message probe is
  not evidence of a complete freshly generated application build.
- The migration takes `shops` then `shop_messages` locks (the cascading-delete
  order) and adds the counter foreign key after seeding; it is applied to live
  databases only through a checksum-pinned runner, `miltech_ng_test` first
- `insertion_number` stays nullable (optional `NOT NULL` hardening deferred
  until production is verified)

**Alternatives considered:**
- Application-side counter increment (rejected: older binaries and other writers
  would bypass it)
- A sequence per shop or a global sequence (rejected: sequences are not
  commit-ordered, so a watermark could skip a later-committing lower number)
- Regenerating the jet model (rejected: adds `insertion_number` to every legacy
  response)
- Extra `(shop_id, id)` index for reconcile (not added: about 0.4 ms saved per
  100-ID chunk, an eighth index on every insert; revisit as migration 019)

**Consequences:**
- Not zero downtime: message reads and writes block for the migration's hold
  (about 1.1 s at 100,000 rows on a laptop; production row counts unknown)
- Accepted residual deadlock: a transaction that touches `shop_messages` and then
  writes `shops` deadlocks with the migration; the migration is the victim, is
  atomic and can be re-run; none found in the repositories checked (not an
  exhaustive audit)
- Every message insert now also writes `shop_message_counters` under the
  invoker's privileges; the application role must be verified on
  `miltech_ng_test` before production
- The migration must be applied before the flag is turned on; the flag must be
  uniform across the fleet
- Runner pre-018 schema checksums are unpinned until the operator records them
- Details: `docs/testing/shops-database.md`,
  `docs/testing/shops-message-sync-measurements.md`,
  `docs/testing/shops-release-contracts.md`

### Shops remediation operational checkpoint (2026-10-04)

The 019–023 runner takes one explicitly authorized forward action and stops for
mandatory tagged generation. Named targets stay UNPINNED pending separate owner
identity/source/schema/data and actual DB_USERNAME proof, test first then production.
Every later action requires hash-pinned preceding target/schema/source generation
evidence with all 32 PMCS hashes unchanged. No automatic reverse or repair. See
[the release gate sheet](../testing/shops-server-remediation-release.md). This
adds operational constraints without rewriting historical ADR evidence.

Task 22 late-commit legacy cursor and numeric restore ABA limitations remain OPEN.
C06 owner population/fleet/edge budget remains unknown; no numeric limit is assumed.
