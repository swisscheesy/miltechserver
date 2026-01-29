# Shops Refactoring Progress

Owner: swisscheese
Last Updated: 2026-01-29

## Principles
- No user-facing behavior changes during migration.
- Legacy endpoints stay active until the new package fully replaces them.
- No legacy file deletion until all references are removed and behavior is verified.

## Phase 1: Foundation
- [x] Step 1.1 Create shared package (authorization/errors/context)
- [x] Step 1.2 Create wrapper interfaces (core first)

## Phase 2: Extract Bounded Contexts
- [x] Settings
- [x] Invite Codes
- [x] Members
- [x] Lists + Items
- [x] Messages
- [x] Vehicles
- [x] Notifications + Items + Changes
- [x] Core

## Phase 3: Route Migration
- [x] Add `api/shops/route.go` registration
- [x] Wire sub-domain routes without changing URLs

## Phase 4: Cleanup
- [x] Remove legacy shops controller file
- [x] Remove legacy shops service/repository files
- [x] Update imports to new packages
- [ ] Run integration tests

## Notes
- Feature flags: not planned unless explicitly requested.
- Parallel compare: only after extraction of a sub-domain.
