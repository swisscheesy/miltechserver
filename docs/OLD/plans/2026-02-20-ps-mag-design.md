# PS Magazine Feature — Design Document

**Date:** 2026-02-20
**Status:** Approved
**Branch:** shop_refactor (to be implemented on a new feature branch)

---

## Overview

Add a PS Magazine browsing and download feature to the existing `library` bounded context. PS Magazine issues are stored as flat PDF files in Azure Blob Storage under the `ps-mag/` prefix of the `library` container. Files follow a strict naming convention that encodes all metadata needed for filtering and display — no database is required.

**Filename convention:**
```
PS_Magazine_Issue_###_Month_Year.pdf
Example: PS_Magazine_Issue_495_February_1994.pdf
```

---

## Goals

- Allow users to list PS Magazine issues in paginated batches of 50, ordered by issue number ASC or DESC
- Allow users to filter issues by year (from filename)
- Allow users to filter issues by exact issue number (from filename)
- Allow users to generate a time-limited SAS download URL for any issue
- No analytics tracking for PS Magazine downloads
- No authentication required (all routes are public)

---

## Architecture

### Placement

The feature lives as a sub-package of the existing `library` bounded context, following the same pattern as `api/item_lookup/lin/`, `api/item_lookup/uoc/`, etc.

```
api/library/
├── errors.go
├── response.go
├── route.go            ← adds one call to ps_mag.RegisterHandlers()
├── service.go
├── service_impl.go
└── ps_mag/
    ├── errors.go       ← sentinel errors
    ├── response.go     ← PSMagIssueResponse, PSMagIssuesResponse, DownloadURLResponse
    ├── route.go        ← Handler struct + RegisterHandlers()
    ├── service.go      ← Service interface
    └── service_impl.go ← blob listing, filename parsing, filter/sort/paginate/SAS
```

No changes are required to `api/route/route.go`. The library's existing `RegisterRoutes` call in `route.go` already wires the blob client and credential, so the parent just passes them down to the sub-package.

### Dependency flow

```
route/route.go
  → library.RegisterRoutes(deps)
      → ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.BlobCredential)
          → ps_mag.NewService(blobClient, credential)
```

The `ps_mag` package receives only what it needs: the blob client and the shared key credential. It does not depend on the DB, analytics, or env — keeping it lean.

---

## API Routes

All routes are public (registered on `publicGroup`).

### `GET /library/ps-mag/issues`

Returns a paginated list of PS Magazine issues, with optional filters.

**Query parameters:**

| Parameter | Type   | Default | Description                                      |
|-----------|--------|---------|--------------------------------------------------|
| `page`    | int    | `1`     | Page number, 1-indexed. Must be > 0.             |
| `order`   | string | `asc`   | Sort direction by issue number: `asc` or `desc`. |
| `year`    | int    | —       | Optional. Filter to issues published in this year. |
| `issue`   | int    | —       | Optional. Filter to an exact issue number.       |

**Success response:** `200 OK`
```json
{
  "status": 200,
  "message": "",
  "data": {
    "issues": [
      {
        "name": "PS_Magazine_Issue_495_February_1994.pdf",
        "blob_path": "ps-mag/PS_Magazine_Issue_495_February_1994.pdf",
        "issue_number": 495,
        "month": "February",
        "year": 1994,
        "size_bytes": 4821032,
        "last_modified": "2024-01-15T10:30:00Z"
      }
    ],
    "count": 1,
    "total_count": 1,
    "page": 1,
    "total_pages": 1,
    "order": "asc"
  }
}
```

**Error responses:**

| Condition | Status | Body |
|-----------|--------|------|
| `page` < 1 | 400 | `{"error": "Invalid request", "details": "page must be greater than 0"}` |
| `order` not `asc`/`desc` | 400 | `{"error": "Invalid request", "details": "order must be 'asc' or 'desc'"}` |
| Azure listing fails | 500 | `{"error": "Failed to list issues", "details": "..."}` |

---

### `GET /library/ps-mag/download`

Generates a 1-hour SAS URL for downloading a specific issue.

**Query parameters:**

| Parameter   | Type   | Description                                                              |
|-------------|--------|--------------------------------------------------------------------------|
| `blob_path` | string | Full blob path, e.g. `ps-mag/PS_Magazine_Issue_495_February_1994.pdf` |

**Success response:** `200 OK`
```json
{
  "status": 200,
  "message": "",
  "data": {
    "blob_path": "ps-mag/PS_Magazine_Issue_495_February_1994.pdf",
    "download_url": "https://...",
    "expires_at": "2026-02-20T11:30:00Z"
  }
}
```

**Error responses:**

| Condition | Status | Body |
|-----------|--------|------|
| `blob_path` empty | 400 | `{"error": "Invalid request", "details": "blob path cannot be empty"}` |
| `blob_path` missing `ps-mag/` prefix | 400 | `{"error": "Invalid request", "details": "invalid blob path: must start with ps-mag/"}` |
| `blob_path` not ending in `.pdf` | 400 | `{"error": "Invalid request", "details": "invalid file type: only PDF files can be downloaded"}` |
| Blob does not exist in Azure | 404 | `{"error": "Issue not found", "details": "..."}` |
| SAS generation fails | 500 | `{"error": "Failed to generate download URL", "details": "..."}` |

---

## Response Types (`ps_mag/response.go`)

```go
// PSMagIssueResponse represents a single parsed PS Magazine issue.
type PSMagIssueResponse struct {
    Name         string `json:"name"`
    BlobPath     string `json:"blob_path"`
    IssueNumber  int    `json:"issue_number"`
    Month        string `json:"month"`
    Year         int    `json:"year"`
    SizeBytes    int64  `json:"size_bytes"`
    LastModified string `json:"last_modified"` // RFC3339
}

// PSMagIssuesResponse is the paginated listing response.
type PSMagIssuesResponse struct {
    Issues     []PSMagIssueResponse `json:"issues"`
    Count      int                  `json:"count"`       // items on this page
    TotalCount int                  `json:"total_count"` // total matching issues (across all pages)
    Page       int                  `json:"page"`
    TotalPages int                  `json:"total_pages"`
    Order      string               `json:"order"`
}

// DownloadURLResponse contains a time-limited SAS URL for an issue.
type DownloadURLResponse struct {
    BlobPath    string `json:"blob_path"`
    DownloadURL string `json:"download_url"`
    ExpiresAt   string `json:"expires_at"` // RFC3339
}
```

---

## Service Interface (`ps_mag/service.go`)

```go
type Service interface {
    // ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
    // page is 1-indexed. order must be "asc" or "desc".
    // year and issueNumber are optional filters — nil means no filter applied.
    ListIssues(page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error)

    // GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
    // blobPath must start with "ps-mag/" and end with ".pdf".
    GenerateDownloadURL(blobPath string) (*DownloadURLResponse, error)
}
```

---

## Service Implementation Logic (`ps_mag/service_impl.go`)

### Constants

```go
const (
    PSMagContainerName = "library"
    PSMagPrefix        = "ps-mag/"
    PageSize           = 50
)
```

### Filename parsing

```go
var issueRegex = regexp.MustCompile(
    `^PS_Magazine_Issue_(\d+)_([A-Za-z]+)_(\d{4})\.pdf$`,
)
```

Blobs that do not match this pattern are silently skipped during listing. This is defensive — stray files in the prefix do not cause errors.

### `ListIssues` flow

1. List all blobs from Azure with prefix `ps-mag/` using `NewListBlobsFlatPager`
2. For each blob: extract filename, run `parseIssueFilename`, skip non-matching
3. Apply year filter if `year != nil`
4. Apply issueNumber filter if `issueNumber != nil`
5. Sort the filtered slice by `IssueNumber` ASC or DESC
6. Compute `TotalCount = len(filtered)` and `TotalPages = ceil(TotalCount / PageSize)`
7. Slice to `[(page-1)*PageSize : page*PageSize]` — clamp upper bound to avoid out-of-range
8. Return `PSMagIssuesResponse`

### `GenerateDownloadURL` flow

1. Validate `blobPath` is not empty → `ErrEmptyBlobPath`
2. Validate prefix `ps-mag/` → `ErrInvalidBlobPath`
3. Validate suffix `.pdf` (case-insensitive) → `ErrInvalidFileType`
4. Call `blobClient.GetProperties(ctx, nil)` to confirm blob exists → `ErrIssueNotFound`
5. Generate SAS token: 1-hour expiry, read-only, HTTPS-only, with `SharedKeyCredential`
6. Return `DownloadURLResponse`

---

## Sentinel Errors (`ps_mag/errors.go`)

```go
var (
    ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
    ErrInvalidBlobPath = errors.New("invalid blob path: must start with ps-mag/")
    ErrInvalidFileType = errors.New("invalid file type: only PDF files can be downloaded")
    ErrIssueNotFound   = errors.New("issue not found")
    ErrBlobListFailed  = errors.New("failed to list blobs from Azure")
    ErrSASGenFailed    = errors.New("failed to generate download URL")
    ErrInvalidPage     = errors.New("page must be greater than 0")
    ErrInvalidOrder    = errors.New("order must be 'asc' or 'desc'")
)
```

---

## HTTP Error Mapping (route.go)

| Error sentinel | HTTP Status |
|---------------|-------------|
| `ErrIssueNotFound` | 404 |
| `ErrInvalidBlobPath`, `ErrInvalidFileType`, `ErrEmptyBlobPath`, `ErrInvalidPage`, `ErrInvalidOrder` | 400 |
| All others | 500 |

---

## Tests

### `ps_mag/service_impl_test.go` — pure unit tests, no Azure dependency

| Test | Description |
|------|-------------|
| `TestParseIssueFilename` | Valid filename returns correct issueNumber, month, year |
| `TestParseIssueFilenameInvalid` | Non-matching names return `ok=false` |
| `TestListIssuesFilterByYear` | Year filter excludes non-matching issues |
| `TestListIssuesFilterByIssue` | Issue number filter returns exactly one result |
| `TestListIssuesPaginationASC` | Page 1 returns correct window in ASC order |
| `TestListIssuesPaginationDESC` | Same window in DESC order |
| `TestListIssuesEmptyResult` | Filters that match nothing return empty slice, not error |
| `TestGenerateDownloadURLValidation` | Empty/invalid paths return correct sentinel errors |

### `ps_mag/route_test.go` — HTTP handler tests with `serviceStub`

| Test | Description |
|------|-------------|
| `TestListIssuesSuccess` | 200 with valid params |
| `TestListIssuesDefaultParams` | 200 with no params (defaults: page=1, order=asc) |
| `TestListIssuesInvalidPage` | 400 for `page=0` |
| `TestListIssuesInvalidOrder` | 400 for `order=sideways` |
| `TestDownloadSuccess` | 200 with valid `blob_path` |
| `TestDownloadMissingBlobPath` | 400 for missing `blob_path` param |
| `TestDownloadNotFound` | 404 when service returns `ErrIssueNotFound` |
| `TestDownloadInvalidPath` | 400 when service returns `ErrInvalidBlobPath` |
| `TestDownloadServerError` | 500 on unexpected error |

---

## Files Changed / Created

| File | Action |
|------|--------|
| `api/library/route.go` | Modified — add `ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.BlobCredential)` |
| `api/library/ps_mag/errors.go` | Created |
| `api/library/ps_mag/response.go` | Created |
| `api/library/ps_mag/service.go` | Created |
| `api/library/ps_mag/service_impl.go` | Created |
| `api/library/ps_mag/route.go` | Created |
| `api/library/ps_mag/service_impl_test.go` | Created |
| `api/library/ps_mag/route_test.go` | Created |

---

## Out of Scope

- Caching of the Azure blob listing (add later if profiling warrants it)
- Authentication (all routes are public)
- Analytics tracking for downloads
- Upload or delete operations
- Pagination by anything other than issue number (e.g. by year)
