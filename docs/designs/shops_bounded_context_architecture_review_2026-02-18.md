# Shops Bounded Context Architecture Review (2026-02-18)

## Scope

This review compares `shops` against `item_lookup` and `item_query` to answer:

1. Why `shops` still uses many files in `api/controller`.
2. Whether the current architecture is incorrect.
3. What should be changed, and how to refactor safely.

## Executive Summary

- `shops` is not fully aligned with the bounded-context style used by newer domains.
- It is not "broken" architecture, but it is inconsistent and creates avoidable coupling.
- Most likely explanation: `shops` is a partial migration in progress.
- Recommended path: incremental strangler refactor from centralized `ShopsController` to per-subdomain handlers inside `api/shops/*`, preserving routes and behavior.

## Evidence From Current Code

### 1) `item_lookup` and `item_query` are colocated by domain

- `api/item_lookup/route.go` wires repositories/services and registers routes per subdomain package (`lin`, `uoc`, `cage`, `substitute`).
- `api/item_query/route.go` wires `short` and `detailed` services directly.
- `api/item_query/short/route.go` uses local `Handler` + local `Service` interface.
- `api/item_lookup/lin/route.go` handles route logic without global controller package dependency.

### 2) `shops` subdomains still depend on a global controller package

- `api/shops/*/route.go` files import `miltechserver/api/controller`.
- Example files:
  - `api/shops/core/route.go`
  - `api/shops/members/route.go`
  - `api/shops/messages/route.go`
  - `api/shops/lists/items/route.go`
- Those routes delegate to methods on one large `*controller.ShopsController`.

### 3) `shops` has internal bounded pieces, but HTTP adapter is centralized

- `api/shops/route.go` already composes subdomain repositories/services (`core`, `settings`, `members`, `messages`, etc.).
- Then it builds `facade.Service` and injects it into `controller.NewShopsController(...)`.
- `api/controller/shops_controller_*.go` contains many handlers and repeated request concerns (auth lookup, bind/validate, param checks, JSON response shaping).

### 4) `api/controller` appears effectively shops-only

- Current imports of `miltechserver/api/controller` are under `api/shops/*` route packages.
- Other domains (`item_lookup`, `item_query`, `library`, `item_comments`, `equipment_services`, etc.) register routes directly from their own package handlers.

## Why This Happened (Most Likely)

Based on the repository state and ADR history (`docs/project_notes/decisions.md`), this looks like phased modernization:

- Newer/updated domains were migrated to colocated modules.
- `shops` moved domain internals into `api/shops/*` but kept legacy controller handlers to reduce risk.
- This is a common midpoint during large refactors with many existing endpoints.

## Is The Architecture Incorrect?

### Verdict

- **Partially correct, but inconsistent with current target style.**
- Not an immediate correctness bug.
- It does violate bounded-context intent at the HTTP adapter boundary.

### What is still good

- `shops` domain logic is already separated by subdomain (`core`, `members`, `messages`, `vehicles`, etc.).
- Dependencies are assembled in one place (`api/shops/route.go`).
- Shared authorization abstraction exists (`api/shops/shared`).

### What is problematic

- Global controller dependency from bounded subpackages (`api/shops/* -> api/controller`).
- Facade service interface is very wide (high coupling, high blast radius for changes).
- Handler concerns are duplicated across many controller files.
- Harder to test each subdomain HTTP adapter independently.
- Architectural inconsistency increases cognitive load for maintainers.

## Risks If Left As-Is

1. Slower feature delivery in shops due to wide interface coupling.
2. Higher regression risk when changing one endpoint set (controller/facade shared surface).
3. Difficulty applying reusable handler patterns across the codebase.
4. Ongoing divergence from the established colocated-module pattern.

## Recommended Target Architecture

Keep existing domain/service/repository split under `api/shops/*`, but move HTTP handlers into each bounded subpackage.

### Target shape

- `api/shops/core/handler.go`
- `api/shops/members/handler.go`
- `api/shops/messages/handler.go`
- `api/shops/lists/handler.go`
- `api/shops/lists/items/handler.go`
- `api/shops/vehicles/.../handler.go`

Each subdomain `RegisterRoutes` should accept its own narrow service interface and bind local handler methods, similar to `item_query/short`.

### Shared concerns stay shared

- Keep shared auth/context/error helpers in `api/shops/shared`.
- Keep centralized dependency composition in `api/shops/route.go`.
- Reuse route paths exactly to avoid client breakage.

## Refactor Plan (Incremental, Low Risk)

### Step 1: Establish adapter pattern for shops

- Add one handler per shops subdomain with local service interface.
- Add shared helper(s) for extracting authenticated user and common bind/validation utilities.
- Keep old controller methods intact.

### Step 2: Migrate one route group at a time

- Start with low-risk group (`settings` or `core` reads).
- Rewire `api/shops/<subdomain>/route.go` to call local handler instead of `controller.ShopsController`.
- Preserve exact URLs, request shapes, response shapes, status codes.

### Step 3: Collapse facade surface gradually

- Replace broad `facade.Service` dependency in migrated route groups with direct subdomain service dependencies.
- Keep facade temporarily for non-migrated groups.
- After all groups migrate, remove facade layer if no longer needed.

### Step 4: Remove shops controller package usage

- Delete `api/controller/shops_controller_*.go` once all shops routes no longer import `api/controller`.
- If `api/controller` becomes empty, remove package.

### Step 5: Standardize error mapping

- Move shops handlers toward typed errors + `errors.Is()` (as seen in `item_query`/`item_lookup`).
- Ensure global middleware behavior remains unchanged.

## Validation Checklist

1. Route parity test: all existing shops endpoints still registered.
2. Contract parity test: status codes + response bodies unchanged.
3. Auth parity test: unauthorized/forbidden flows unchanged.
4. Regression suite for each migrated subdomain before moving to next.
5. Manual smoke test for file upload path in `messages` endpoints.

## Suggested Migration Order

1. `settings`
2. `core`
3. `members` + `invites`
4. `lists` + `lists/items`
5. `vehicles` + `notifications` + `notification items` + `changes`
6. `messages` (last, due to multipart upload and blob side effects)

## Open Questions For swisscheese

1. Do you want strict behavior parity (including exact error messages), or only API contract parity (status + schema)?
2. Should the `facade` layer be fully removed, or retained as an internal orchestration seam?
3. Do you want this migration done behind temporary feature flags, or direct route rewiring with tests only?

## References

- `api/item_lookup/route.go`
- `api/item_lookup/lin/route.go`
- `api/item_lookup/shared/response.go`
- `api/item_query/route.go`
- `api/item_query/short/route.go`
- `api/item_query/detailed/route.go`
- `api/shops/route.go`
- `api/shops/core/route.go`
- `api/shops/messages/route.go`
- `api/controller/shops_controller_base.go`
- `api/controller/shops_controller_core.go`
- `api/controller/shops_controller_members.go`
- `api/controller/shops_controller_messages.go`
- `docs/project_notes/decisions.md`
