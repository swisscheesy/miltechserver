# PS Magazine Summary Search — Design

**Date:** 2026-03-05
**Status:** Approved

## Context

The `ps_mag_summaries` table was added to the database with two columns:
- `file_name` (PK, text) — matches the filename of a PDF in the `ps-mag/` Azure Blob container
- `summary` (nullable text) — multi-line text describing the contents of that issue

The existing ps_mag endpoints (`listIssues`, `generateDownloadURL`) let users browse and download issues. This feature adds full-text search across summaries so users can find issues by content phrase.

## API Contract

```
GET /library/ps-mag/search?q=lubrication&page=1
```

- **Authentication**: None (public route, same as existing ps-mag endpoints)
- **`q`** (required): search phrase; returns 400 if missing, empty, or fewer than 3 characters
- **`page`** (optional, default `1`): 1-indexed page number; returns 400 if non-integer or < 1
- **Page size**: 30 results per page (constant `SearchPageSize = 30`)

### Success Response — 200

```json
{
  "status": 200,
  "message": "",
  "data": {
    "results": [
      {
        "file_name": "PS_Magazine_Issue_495_February_1994.pdf",
        "matching_lines": [
          "Lubrication must be performed every 90 days.",
          "See lubrication chart on page 12."
        ]
      }
    ],
    "count": 1,
    "total_count": 23,
    "page": 1,
    "total_pages": 1,
    "query": "lubrication"
  }
}
```

- `results` — files whose summary contains the phrase; each entry includes only the lines that matched, not the full summary
- `count` — number of results on this page
- `total_count` — total matching files across all pages
- `total_pages` — ceiling division of total_count / page_size
- `query` — echoes the search phrase back to the caller

### Error Responses

| Condition | Status | Detail |
|---|---|---|
| `q` missing, empty, or < 3 chars | 400 | `ErrQueryTooShort` |
| `page` non-integer or < 1 | 400 | `ErrInvalidPage` (existing) |
| DB query fails | 500 | generic "Failed to search summaries" |

Zero results returns 200 with `"results": []` and `"total_count": 0` — no 404.

## Architecture

The `ps_mag` package gains a repository layer following the `user_suggestions` pattern. `*sql.DB` is threaded through from `library/route.go` which already holds it in `deps.DB`.

```
Route handler (route.go)
  └─ Service interface (service.go)
       └─ ServiceImpl (service_impl.go)
            ├─ blobClient  (existing)
            └─ repo Repository  (new)
                  └─ RepositoryImpl (repository_impl.go)
                        └─ *sql.DB
```

### Files Changed

| File | Change |
|---|---|
| `api/library/ps_mag/repository.go` | NEW — `Repository` interface |
| `api/library/ps_mag/repository_impl.go` | NEW — `RepositoryImpl` with raw SQL |
| `api/library/ps_mag/errors.go` | Add `ErrQueryTooShort` |
| `api/library/ps_mag/response.go` | Add `PSMagSearchResult`, `PSMagSearchResponse` |
| `api/library/ps_mag/service.go` | Add `SearchSummaries` to `Service` interface |
| `api/library/ps_mag/service_impl.go` | Add `repo` field, update `NewService`, implement `SearchSummaries` |
| `api/library/ps_mag/route.go` | Update `RegisterHandlers` signature, add `searchSummaries` handler |
| `api/library/route.go` | Pass `deps.DB` to `ps_mag.RegisterHandlers` |
| `api/library/ps_mag/repository_impl_test.go` | NEW — integration tests |
| `api/library/ps_mag/service_impl_test.go` | Add `SearchSummaries` unit tests |
| `api/library/ps_mag/route_test.go` | Add `searchSummaries` handler tests |

## Data Layer

### SQL Queries (in `repository_impl.go`)

```sql
-- Count total matching files
SELECT COUNT(*)
FROM ps_mag_summaries
WHERE summary ILIKE $1

-- Fetch paginated results
SELECT file_name, summary
FROM ps_mag_summaries
WHERE summary ILIKE $1
ORDER BY file_name ASC
LIMIT $2 OFFSET $3
```

- The ILIKE parameter is `%phrase%` — wrapped in wildcards server-side
- Passed as a prepared statement argument — no SQL injection risk
- `NULL` summaries are excluded naturally (NULL ILIKE anything = NULL, not true)

### Line Filtering (in `service_impl.go`)

After the DB returns matching rows, each summary is split by `\n` and only lines containing the phrase (case-insensitive) are kept:

```go
func filterMatchingLines(summary, query string) []string {
    lowerQuery := strings.ToLower(query)
    var matches []string
    for _, line := range strings.Split(summary, "\n") {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" && strings.Contains(strings.ToLower(trimmed), lowerQuery) {
            matches = append(matches, trimmed)
        }
    }
    return matches
}
```

Empty and whitespace-only lines are skipped entirely.

## Response Types (new in `response.go`)

```go
// PSMagSearchResult is a single file with only its matching summary lines.
type PSMagSearchResult struct {
    FileName     string   `json:"file_name"`
    MatchingLines []string `json:"matching_lines"`
}

// PSMagSearchResponse is the paginated summary search response.
type PSMagSearchResponse struct {
    Results    []PSMagSearchResult `json:"results"`
    Count      int                 `json:"count"`
    TotalCount int                 `json:"total_count"`
    Page       int                 `json:"page"`
    TotalPages int                 `json:"total_pages"`
    Query      string              `json:"query"`
}
```

## Alternatives Considered

- **PostgreSQL unnest line splitting** — filter lines entirely in SQL using `unnest(string_to_array(summary, E'\n'))`. Rejected: complex multi-level SQL, pagination on the outer file-level query becomes awkward, harder to maintain.
- **Jet ORM for the query** — use Jet's `ILike` operator instead of raw SQL. Rejected: still requires raw SQL for `COUNT(*)` pagination; plain raw SQL is simpler and consistent with ADR-013 (equipment search precedent).
