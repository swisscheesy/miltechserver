# Equipment Services Refactor Tracker

**Created:** 2026-01-30
**Owner:** swisscheese
**Status:** In Progress

## Progress Log

- 2026-01-30: Initialized new module structure and shared utilities.
- 2026-01-30: Added integration tests and ran `go test ./tests/equipment_services` (pass).
- 2026-01-30: Added edge case coverage (auth, validation, pagination, mismatch, completion date behavior) and re-ran tests.

## Implementation Checklist

### Phase 1: Foundation
- [x] 1.1 Create directory structure
- [x] 1.2 Create shared/errors.go
- [x] 1.3 Create shared/context.go
- [x] 1.4 Create shared/authorization.go
- [x] 1.5 Create shared/mappers.go (with username cache)
- [x] 1.6 Create central route.go skeleton

### Phase 2: Bounded Contexts
- [x] 2.1 Extract core CRUD context
- [x] 2.2 Extract queries context
- [x] 2.3 Extract calendar context
- [x] 2.4 Extract status context
- [x] 2.5 Extract completion context

### Phase 3: Wiring
- [x] 3.1 Wire dependencies in equipment_services/route.go
- [x] 3.2 Update main route registration
- [x] 3.3 Remove legacy controller/service/repository/routes

### Phase 4: Verification & Cleanup
- [x] 4.1 Create integration tests
- [ ] 4.2 Manual API testing
- [x] 4.3 Remove legacy files (after verification)

## Metrics

| Metric | Current | Target | Notes |
|--------|---------|--------|-------|
| Largest file (LOC) | TBD | < 200 | Recalculate after cleanup |
| Methods per interface | <= 4 | < 7 | Per context |
| Code duplication (mappers) | Centralized | 1 shared | Username cache added |
| Test coverage | Added endpoint tests | > 80% | Tests added but not yet executed |
| N+1 username queries | Reduced | 1 per unique user | Per-request cache |

## Notes

- Legacy code removed after tests passed. Manual API validation still pending.
