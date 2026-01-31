# EIC Refactoring Progress

**Owner:** swisscheese
**Started:** 2026-01-31
**Status:** In Progress

## Checklist

### Phase 1: Foundation
- [x] Create directory structure (`api/eic`)
- [x] Add typed errors (`errors.go`)
- [x] Add shared query builder (`query_builder.go`)
- [x] Add shared row scanner (`scanner.go`)

### Phase 2: Core Implementation
- [x] Create repository interface + implementation
- [x] Create service interface + implementation
- [x] Create route handlers + route registration

### Phase 3: Wiring
- [x] Wire `api/route/route.go` to new EIC module

### Phase 4: Verification & Cleanup
- [ ] Verify endpoints behave identically (manual checks)
- [x] Remove legacy EIC files after verification
- [x] Add tests (post-refactor)

### Phase 5: Documentation
- [x] Add ADR-005 in `docs/project_notes/decisions.md`

## Metrics Tracking

| Metric | Baseline (2026-01-30) | Current | Target |
|--------|------------------------|---------|--------|
| Total LOC (EIC domain) | 884 | 682 | ~400 |
| Repository LOC | 484 | 243 | ~150 |
| Duplicated SQL Blocks | 4 | 1 | 1 |
| Files (scattered) | 7 | 7 | 7 |
| Largest EIC file | 484 | 243 | <150 |

## Notes

- Behavior must remain identical to legacy implementation.
- Legacy EIC files remain until manual verification is complete.
- Tests will be added after the refactor is validated.
