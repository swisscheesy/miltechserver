# Material Images Refactor Progress

Last updated: 2026-01-29

## Progress Tracker

| Step | Status | Notes |
|------|--------|-------|
| 1. Create directory structure and shared package | Pending | Create `api/material_images/` with `shared/` utilities |
| 2. Extract ratelimit bounded context | Pending | Rate limiting for uploads |
| 3. Extract images bounded context | Pending | Core image CRUD operations |
| 4. Extract votes bounded context | Pending | Image voting management |
| 5. Extract flags bounded context | Pending | Image flagging/moderation |
| 6. Wire dependencies in central route | Pending | Update `api/route/route.go` |
| 7. Testing and verification | Pending | `go test ./api/material_images/...` |
| 8. Cleanup legacy files | Pending | Remove legacy material_images files |

## Metrics Snapshot

| Metric | Baseline (2026-01-29) | Current | Target |
|--------|------------------------|---------|--------|
| Largest file (material images domain) | 599 lines | 599 lines | < 200 lines |
| Methods per interface | 15 | 15 | < 7 |
| Test coverage | 0% | 0% | > 80% |
| Typed domain errors | 0 | 0 | 6 |

## Notes

- Behavior must remain identical to legacy endpoints.
- No legacy deletions until new routes are fully wired and verified.
- Mixed public/authenticated routes require careful router group handling.
- Blob storage integration should use shared utilities with nil checks.
