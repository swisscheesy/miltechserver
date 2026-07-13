# SB 700-20 New Search Endpoints Design

**Date:** 2026-05-14
**Branch:** sb700-20

## Overview

Extend the existing SB 700-20 API with 7 new search endpoints. The existing endpoints only support searching by `lin` (or `lin_zmm_lin` for the H tables). This adds the ability to search by alternative fields: `new_lin` on app_e and app_g, `lin_zmm_sublin`/`lin_zmmsublin` on app_h1 and app_h2, and `ric` on chp_4, chp_6, and chp_8.

All endpoints are public (no auth), mounted under `/api/v1`, and return lists since none of the search fields are sole primary keys.

## New Routes (7 total)

```
GET /sb700-20/app-e/search-new-lin/:new_lin
GET /sb700-20/app-g/search-new-lin/:new_lin
GET /sb700-20/app-h1/search-sublin/:sublin
GET /sb700-20/app-h2/search-sublin/:sublin
GET /sb700-20/chp-4/search-ric/:ric
GET /sb700-20/chp-6/search-ric/:ric
GET /sb700-20/chp-8/search-ric/:ric
```

## Field Mapping

| Route | URL Param | DB Column | Column Type | Returns |
|---|---|---|---|---|
| `app-e/search-new-lin` | `:new_lin` | `new_lin` | `*string` | `[]Sb70020AppE` |
| `app-g/search-new-lin` | `:new_lin` | `new_lin` | `*string` | `[]Sb70020AppG` |
| `app-h1/search-sublin` | `:sublin` | `lin_zmm_sublin` | `string` (PK component) | `[]Sb70020AppH1` |
| `app-h2/search-sublin` | `:sublin` | `lin_zmmsublin` | `string` (PK component) | `[]Sb70020AppH2` |
| `chp-4/search-ric` | `:ric` | `ric` | `*string` | `[]Sb70020Chp4` |
| `chp-6/search-ric` | `:ric` | `ric` | `*string` | `[]Sb70020Chp6` |
| `chp-8/search-ric` | `:ric` | `ric` | `*string` | `[]Sb70020Chp8` |

Note: AppH1 uses `lin_zmm_sublin` (underscore before "sublin") while AppH2 uses `lin_zmmsublin` (no underscore — different column name from a different table). Both expose `:sublin` in the URL; the column difference is handled internally.

## New Methods (7 per layer)

Added to `Repository`, `Service`, and their implementations:

```
GetAppEByNewLIN(newLin string) ([]Sb70020AppE, error)
GetAppGByNewLIN(newLin string) ([]Sb70020AppG, error)
GetAppH1BySubLIN(sublin string) ([]Sb70020AppH1, error)
GetAppH2BySubLIN(sublin string) ([]Sb70020AppH2, error)
GetChp4ByRIC(ric string) ([]Sb70020Chp4, error)
GetChp6ByRIC(ric string) ([]Sb70020Chp6, error)
GetChp8ByRIC(ric string) ([]Sb70020Chp8, error)
```

## Files Changed

| File | Change |
|---|---|
| `api/sb_700_20/repository.go` | Add 7 method signatures to `Repository` interface |
| `api/sb_700_20/service.go` | Add 7 method signatures to `Service` interface |
| `api/sb_700_20/repository_apps.go` | Implement `GetAppEByNewLIN`, `GetAppGByNewLIN`, `GetAppH1BySubLIN`, `GetAppH2BySubLIN` |
| `api/sb_700_20/repository_chps.go` | Implement `GetChp4ByRIC`, `GetChp6ByRIC`, `GetChp8ByRIC` |
| `api/sb_700_20/service_impl.go` | Add 7 thin delegation methods with `TrimSpace`/`ToUpper` normalization |
| `api/sb_700_20/handlers_apps.go` | Add `searchAppEByNewLIN`, `searchAppGByNewLIN`, `searchAppH1BySubLIN`, `searchAppH2BySubLIN` |
| `api/sb_700_20/handlers_chps.go` | Add `searchChp4ByRIC`, `searchChp6ByRIC`, `searchChp8ByRIC` |
| `api/sb_700_20/route.go` | Register 7 new routes in `RegisterHandlers` |
| `docs/sb700-20-new-search-endpoints.md` | New user-facing API reference doc |

## Repository Implementation Pattern

Each new method follows the exact same pattern as existing search methods:

1. `strings.TrimSpace` the param; return `ErrEmptyParam` if blank
2. Build `SELECT … WHERE col.EQ(String(param))` using Jet
3. Return `ErrNotFound` if result slice is empty
4. Return the full slice on success

No pagination — these are point-search endpoints, not list endpoints.

## Service Implementation Pattern

Thin delegation with normalization:

```
strings.TrimSpace(strings.ToUpper(param))
```

Same normalization applied to all existing search methods.

## Handler Implementation Pattern

Each handler:

1. Reads the path param (`c.Param("new_lin")`, `c.Param("sublin")`, or `c.Param("ric")`)
2. Returns 400 if the trimmed param is blank
3. Calls the corresponding service method
4. Maps errors: `ErrNotFound` → 404, `ErrEmptyParam` → 400, else → 500
5. Returns 200 with `response.StandardResponse{Status: 200, Data: items}`

## Error Responses

Same error handling as all existing endpoints:

- `400` — blank or whitespace-only path param
- `404` — no records found for the given value
- `500` — unexpected database error

## Constraints

- All 7 endpoints are public — registered on `v1Route`, not `authRoutes`
- No DB migrations required — querying existing columns
- No new files — all additions go into the existing 8 files listed above
- Page size / pagination not applicable — these are direct field searches
