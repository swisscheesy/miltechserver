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
