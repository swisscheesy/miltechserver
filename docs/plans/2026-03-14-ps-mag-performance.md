# PS Magazine Performance Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the Azure Blob Storage round-trip on every `/library/ps-mag/issues` request and fix supporting performance and correctness issues to make initial loading fast and reliable.

**Architecture:** The primary bottleneck is `listAllIssues()` calling Azure Blob Storage on every request with no caching. An in-memory TTL cache (mirroring the existing pattern in `api/item_query/detailed/cache.go`) stores the full issue list for 10 minutes, reducing subsequent request latency from ~500ms to <1ms. Secondary improvements add request context propagation, HTTP cache headers, User Delegation Key caching, ILIKE wildcard escaping, and a slice pre-allocation.

**Tech Stack:** Go, Gin, Azure SDK for Go (`azblob`), `sync.RWMutex`, `database/sql`, testify

---

## Background: Root Cause Summary

From the performance analysis performed 2026-03-14:

| # | Issue | Severity | Location |
|---|-------|----------|----------|
| 1 | No caching of blob listing — Azure API call on every request | **Critical** | `service_impl.go:116-166` |
| 2 | `context.Background()` instead of request context in `listAllIssues` | **High** | `service_impl.go:117` |
| 3 | No HTTP `Cache-Control` headers on issue list response | **Medium** | `route.go:104` |
| 4 | User Delegation Key fetched fresh on every download URL request | **Medium** | `shared/sas.go:47-54` |
| 5 | `%` and `_` in search phrase not escaped before ILIKE pattern | **Low/Correctness** | `repository_impl.go:22` |
| 6 | Issues slice not pre-allocated in `listAllIssues` | **Low** | `service_impl.go:124` |

Database is well-designed (GIN trigram index, parameterized queries, 330 rows). The search path is not the bottleneck.

---

### Task 1: In-memory cache for blob listing

Adapt the existing `Cache` pattern from `api/item_query/detailed/cache.go` for the ps_mag issue list. The ps_mag list has no per-key variation — it is a single global slice — so a simpler single-entry TTL cache is appropriate.

**Files:**
- Create: `api/library/ps_mag/cache.go`
- Create: `api/library/ps_mag/cache_test.go`

**Step 1: Write the failing cache tests**

Create `api/library/ps_mag/cache_test.go`:

```go
package ps_mag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIssueCache_MissOnEmpty(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	_, ok := c.get()
	require.False(t, ok)
}

func TestIssueCache_HitAfterSet(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	issues := []PSMagIssueResponse{
		{Name: "test.pdf", IssueNumber: 1},
	}
	c.set(issues)

	got, ok := c.get()
	require.True(t, ok)
	require.Equal(t, issues, got)
}

func TestIssueCache_MissAfterExpiry(t *testing.T) {
	c := newIssueCache(1 * time.Millisecond)
	c.set([]PSMagIssueResponse{{Name: "test.pdf"}})

	time.Sleep(5 * time.Millisecond)

	_, ok := c.get()
	require.False(t, ok)
}

func TestIssueCache_SetOverwritesPrevious(t *testing.T) {
	c := newIssueCache(5 * time.Minute)
	c.set([]PSMagIssueResponse{{Name: "old.pdf"}})
	c.set([]PSMagIssueResponse{{Name: "new.pdf"}})

	got, ok := c.get()
	require.True(t, ok)
	require.Equal(t, "new.pdf", got[0].Name)
}

func TestIssueCache_GetReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not corrupt the cache.
	c := newIssueCache(5 * time.Minute)
	c.set([]PSMagIssueResponse{{Name: "original.pdf"}})

	got, _ := c.get()
	got[0].Name = "mutated.pdf"

	got2, _ := c.get()
	require.Equal(t, "original.pdf", got2[0].Name)
}
```

**Step 2: Run to verify they FAIL**

```bash
go test ./api/library/ps_mag/... -run TestIssueCache -v
```
Expected: compile error — `newIssueCache` undefined.

**Step 3: Create `api/library/ps_mag/cache.go`**

```go
package ps_mag

import (
	"sync"
	"time"
)

// issueCache is a single-entry TTL cache for the full ps-mag issue list.
// Thread-safe for concurrent access.
type issueCache struct {
	mu        sync.RWMutex
	issues    []PSMagIssueResponse
	expiresAt time.Time
	ttl       time.Duration
}

func newIssueCache(ttl time.Duration) *issueCache {
	return &issueCache{ttl: ttl}
}

// get returns a copy of the cached issue list and true if the cache is warm and
// not expired. Returns nil and false on a cache miss.
func (c *issueCache) get() ([]PSMagIssueResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.issues == nil || time.Now().After(c.expiresAt) {
		return nil, false
	}
	// Return a copy so callers cannot mutate the cached slice.
	cp := make([]PSMagIssueResponse, len(c.issues))
	copy(cp, c.issues)
	return cp, true
}

// set stores issues in the cache and resets the expiry clock.
func (c *issueCache) set(issues []PSMagIssueResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.issues = issues
	c.expiresAt = time.Now().Add(c.ttl)
}
```

**Step 4: Run tests to verify they PASS**

```bash
go test ./api/library/ps_mag/... -run TestIssueCache -v
```
Expected: all 5 tests PASS.

**Step 5: Commit**

```bash
git add api/library/ps_mag/cache.go api/library/ps_mag/cache_test.go
git commit -m "feat(ps-mag): add single-entry TTL cache for issue list"
```

---

### Task 2: Wire the cache into ServiceImpl

Add the `issueCache` field to `ServiceImpl`, update `NewService` to initialise it, and update `listAllIssues` to check the cache on reads and populate it on a miss.

**Files:**
- Modify: `api/library/ps_mag/service_impl.go`
- Modify: `api/library/ps_mag/service_impl_test.go`

**Step 1: Write the failing service cache test**

Append to `api/library/ps_mag/service_impl_test.go`:

```go
func TestListIssues_UsesCacheOnSecondCall(t *testing.T) {
	// The azureCallCount lets us detect how many times the blob pager was invoked.
	// We cannot easily stub Azure here, so we test via ServiceImpl directly by
	// pre-populating the cache and confirming the Azure client is NOT called.
	//
	// Build a ServiceImpl with a warm cache and a nil blobClient.
	// If listAllIssues tries to use blobClient it will panic — proving the cache
	// was bypassed. If it succeeds the cache was used.
	cached := []PSMagIssueResponse{
		{Name: "PS_Magazine_Issue_1_January_1951.pdf", IssueNumber: 1, Month: "January", Year: 1951},
	}
	c := newIssueCache(5 * time.Minute)
	c.set(cached)

	svc := &ServiceImpl{
		blobClient: nil, // panics if called
		repo:       &repoStub{},
		cache:      c,
	}

	result, err := svc.ListIssues(context.Background(), 1, "asc", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, "PS_Magazine_Issue_1_January_1951.pdf", result.Issues[0].Name)
}
```

You will need `"context"` and `"time"` in the imports of `service_impl_test.go` if not already present.

**Step 2: Run to verify it FAILS**

```bash
go test ./api/library/ps_mag/... -run TestListIssues_UsesCacheOnSecondCall -v
```
Expected: compile error — `cache` field does not exist on `ServiceImpl`, `ListIssues` has wrong signature.

**Step 3: Update `ServiceImpl` and `listAllIssues` in `service_impl.go`**

Update the struct:
```go
type ServiceImpl struct {
	blobClient *azblob.Client
	repo       Repository
	cache      *issueCache
}
```

Update `NewService` (10-minute TTL — adjust if desired):
```go
func NewService(blobClient *azblob.Client, db *sql.DB) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		repo:       NewRepository(db),
		cache:      newIssueCache(10 * time.Minute),
	}
}
```

Add `"time"` to imports if not already present.

Update `listAllIssues` to accept a context and check the cache:
```go
// listAllIssues fetches every blob under ps-mag/ and parses metadata from filenames.
// Results are cached for 10 minutes; the cache is shared across all requests.
// Blobs that do not match the filename convention are silently skipped.
func (s *ServiceImpl) listAllIssues(ctx context.Context) ([]PSMagIssueResponse, error) {
	if cached, ok := s.cache.get(); ok {
		return cached, nil
	}

	containerClient := s.blobClient.ServiceClient().NewContainerClient(PSMagContainerName)
	prefix := PSMagPrefix
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	issues := make([]PSMagIssueResponse, 0, 512)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, blob := range page.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			parts := strings.Split(blobPath, "/")
			fileName := parts[len(parts)-1]

			issueNum, month, year, ok := parseIssueFilename(fileName)
			if !ok {
				slog.Debug("Skipping non-matching ps-mag blob", "blobPath", blobPath)
				continue
			}

			var sizeBytes int64
			if blob.Properties != nil && blob.Properties.ContentLength != nil {
				sizeBytes = *blob.Properties.ContentLength
			}
			var lastModified string
			if blob.Properties != nil && blob.Properties.LastModified != nil {
				lastModified = blob.Properties.LastModified.Format(time.RFC3339)
			}

			issues = append(issues, PSMagIssueResponse{
				Name:         fileName,
				BlobPath:     blobPath,
				IssueNumber:  issueNum,
				Month:        month,
				Year:         year,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	s.cache.set(issues)
	return issues, nil
}
```

**Step 4: Update `ListIssues` signature to accept `context.Context`**

The `Service` interface change comes in Task 3. For now, update the `ServiceImpl.ListIssues` receiver method to accept ctx and pass it through:

```go
func (s *ServiceImpl) ListIssues(ctx context.Context, page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	order = strings.ToLower(order)
	if order != "asc" && order != "desc" {
		return nil, ErrInvalidOrder
	}

	issues, err := s.listAllIssues(ctx)
	if err != nil {
		return nil, err
	}

	if year != nil {
		issues = filterByYear(issues, *year)
	}
	if issueNumber != nil {
		issues = filterByIssueNumber(issues, *issueNumber)
	}

	sortIssues(issues, order)

	pageItems, totalPages := paginateIssues(issues, page, PageSize)

	return &PSMagIssuesResponse{
		Issues:     pageItems,
		Count:      len(pageItems),
		TotalCount: len(issues),
		Page:       page,
		TotalPages: totalPages,
		Order:      order,
	}, nil
}
```

Add `"context"` to imports in `service_impl.go` if not already present.

**Step 5: Run the cache test to verify it PASSES**

```bash
go test ./api/library/ps_mag/... -run TestListIssues_UsesCacheOnSecondCall -v
```
Expected: PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/service_impl.go
git commit -m "feat(ps-mag): wire issue list cache into ServiceImpl and listAllIssues"
```

---

### Task 3: Update Service interface and handler for context propagation

`ListIssues` on the `Service` interface must accept `context.Context` so the handler can pass the request context and enable proper cancellation. Update the interface, the service stub in tests, and the handler.

**Files:**
- Modify: `api/library/ps_mag/service.go`
- Modify: `api/library/ps_mag/route.go`
- Modify: `api/library/ps_mag/route_test.go`

**Step 1: Update `service.go`**

```go
package ps_mag

import "context"

// Service provides methods for accessing PS Magazine issues from Azure Blob Storage.
type Service interface {
	// ListIssues returns a paginated, optionally filtered list of PS Magazine issues.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	// page is 1-indexed. order must be "asc" or "desc".
	// year and issueNumber are optional — pass nil to skip the filter.
	ListIssues(ctx context.Context, page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error)

	// GenerateDownloadURL creates a 1-hour SAS URL for a ps-mag blob.
	// ctx should be the request context so Azure calls are cancelled on client disconnect.
	// blobPath must start with "ps-mag/" and end with ".pdf".
	GenerateDownloadURL(ctx context.Context, blobPath string) (*DownloadURLResponse, error)

	// SearchSummaries returns a paginated list of PS Magazine issues whose summaries
	// contain query. Only the lines matching query are returned per file.
	// query must be at least 3 characters. page is 1-indexed.
	SearchSummaries(query string, page int) (*PSMagSearchResponse, error)
}
```

**Step 2: Update `serviceStub.ListIssues` in `route_test.go`**

```go
func (s *serviceStub) ListIssues(_ context.Context, page int, order string, year *int, issueNumber *int) (*PSMagIssuesResponse, error) {
	return s.listResp, s.listErr
}
```

**Step 3: Update `listIssues` handler in `route.go` to pass request context**

Find the call at the bottom of `listIssues`:
```go
result, err := h.service.ListIssues(page, order, year, issueNumber)
```
Change to:
```go
result, err := h.service.ListIssues(c.Request.Context(), page, order, year, issueNumber)
```

**Step 4: Verify it compiles**

```bash
go build ./api/library/ps_mag/...
```
Expected: no errors.

**Step 5: Run all package tests**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/service.go api/library/ps_mag/route.go api/library/ps_mag/route_test.go
git commit -m "feat(ps-mag): propagate request context through ListIssues for cancellation support"
```

---

### Task 4: Add HTTP Cache-Control headers to issue list endpoint

The issue list is public, paginated, and rarely changes. Clients and any reverse proxy can safely cache it for a short window.

**Files:**
- Modify: `api/library/ps_mag/route.go`
- Modify: `api/library/ps_mag/route_test.go`

**Step 1: Write the failing test**

Append to `route_test.go`:

```go
func TestListIssuesResponseHasCacheControlHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	stub := &serviceStub{
		listResp: &PSMagIssuesResponse{
			Issues:     []PSMagIssueResponse{},
			TotalPages: 1,
			Order:      "asc",
		},
	}
	registerHandlers(router.Group("/api/v1"), stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/library/ps-mag/issues", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "public, max-age=300", resp.Header().Get("Cache-Control"))
}
```

**Step 2: Run to verify it FAILS**

```bash
go test ./api/library/ps_mag/... -run TestListIssuesResponseHasCacheControlHeader -v
```
Expected: FAIL — header not present.

**Step 3: Add the header in `route.go`**

In the `listIssues` handler, immediately before the final `c.JSON(...)` success response, add:

```go
c.Header("Cache-Control", "public, max-age=300")
c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
```

**Step 4: Run test to verify it PASSES**

```bash
go test ./api/library/ps_mag/... -run TestListIssuesResponseHasCacheControlHeader -v
```
Expected: PASS.

**Step 5: Run all package tests**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/route.go api/library/ps_mag/route_test.go
git commit -m "feat(ps-mag): add Cache-Control header to issue list response"
```

---

### Task 5: Cache User Delegation Key in shared/sas.go

`GetUserDelegationCredential` is an Azure AD call made on every download URL request. The UDK has a 1-hour validity window; caching it for 45 minutes eliminates one of the two Azure round-trips per download.

**Files:**
- Modify: `api/library/shared/sas.go`

> **Note:** There are no existing tests for `shared/sas.go` — the UDK call requires a live Azure Managed Identity. The change is structural (add a package-level cache variable) and can be manually verified. Write a unit test for the cache helper only.

**Step 1: Add a UDK cache struct**

The `shared` package will hold a package-level `udkCache` instance. Add the struct and its constructor at the top of `sas.go`, before `GenerateBlobSASURL`:

```go
import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// udkEntry caches a User Delegation Key and its expiry.
type udkEntry struct {
	mu        sync.Mutex
	key       *service.UserDelegationCredential
	expiresAt time.Time
}

// packageUDKCache is the module-level UDK cache shared across all callers.
var packageUDKCache udkEntry
```

**Step 2: Add the `getOrRefreshUDK` helper**

Add this function immediately before `GenerateBlobSASURL`:

```go
// getOrRefreshUDK returns a cached User Delegation Key if still valid, or fetches
// a new one from Azure AD and caches it for 45 minutes.
func getOrRefreshUDK(ctx context.Context, svcClient *service.Client, expiresAt time.Time) (*service.UserDelegationCredential, error) {
	packageUDKCache.mu.Lock()
	defer packageUDKCache.mu.Unlock()

	// Reuse the cached key if it covers the requested expiry with 5 minutes of margin.
	if packageUDKCache.key != nil && packageUDKCache.expiresAt.After(expiresAt.Add(5*time.Minute)) {
		return packageUDKCache.key, nil
	}

	// Request a key valid for 45 minutes from now.
	keyExpiry := time.Now().UTC().Add(45 * time.Minute)
	udk, err := svcClient.GetUserDelegationCredential(
		ctx,
		service.KeyInfo{
			Start:  strPtr(time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339)),
			Expiry: strPtr(keyExpiry.Format(time.RFC3339)),
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user delegation credential: %w", err)
	}

	packageUDKCache.key = udk
	packageUDKCache.expiresAt = keyExpiry
	return udk, nil
}
```

**Step 3: Update `GenerateBlobSASURL` to use the helper**

Replace the `svcClient.GetUserDelegationCredential(...)` call block (lines 47-57 in the original) with:

```go
udk, err := getOrRefreshUDK(ctx, svcClient, expiresAt)
if err != nil {
	return nil, err
}
```

The rest of the function is unchanged.

**Step 4: Verify it compiles**

```bash
go build ./api/library/shared/...
```
Expected: no errors.

**Step 5: Build the full project**

```bash
go build ./...
```
Expected: no errors.

**Step 6: Commit**

```bash
git add api/library/shared/sas.go
git commit -m "perf(ps-mag): cache User Delegation Key for 45 minutes to reduce Azure AD calls"
```

---

### Task 6: Escape ILIKE wildcard characters in repository

A user searching for `%` or `_` gets unintended LIKE wildcards — they match everything. Escape these characters before building the pattern.

**Files:**
- Modify: `api/library/ps_mag/repository_impl.go`
- Modify: `api/library/ps_mag/repository_impl_test.go`

**Step 1: Write the failing unit test**

This is a unit test for the escaping logic only — no DB needed. Append to `repository_impl_test.go`:

```go
func TestEscapeILIKEPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"oil", "oil"},
		{"100%", `100\%`},
		{"_bolt", `\_bolt`},
		{"100% oil_level", `100\% oil\_level`},
		{`back\slash`, `back\\slash`},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := escapeLIKEPattern(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}
```

**Step 2: Run to verify it FAILS**

```bash
go test ./api/library/ps_mag/... -run TestEscapeILIKEPattern -v
```
Expected: compile error — `escapeLIKEPattern` undefined.

**Step 3: Add `escapeLIKEPattern` and use it in `SearchSummaries`**

In `repository_impl.go`, add the helper and update `SearchSummaries`:

```go
// escapeLIKEPattern escapes special LIKE/ILIKE metacharacters %, _, and \ in s
// so they are treated as literals. Uses \ as the escape character, which is the
// PostgreSQL default for LIKE patterns.
func escapeLIKEPattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
```

Add `"strings"` to imports in `repository_impl.go`.

Update `SearchSummaries` to use it:
```go
func (r *RepositoryImpl) SearchSummaries(phrase string, page, pageSize int) ([]summaryRow, int, error) {
	pattern := "%" + escapeLIKEPattern(phrase) + "%"
	offset := (page - 1) * pageSize
	// ... rest unchanged
```

**Step 4: Run tests to verify they PASS**

```bash
go test ./api/library/ps_mag/... -run TestEscapeILIKEPattern -v
```
Expected: all 5 sub-tests PASS.

**Step 5: Run all package tests**

```bash
go test ./api/library/ps_mag/... -v
```
Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/library/ps_mag/repository_impl.go api/library/ps_mag/repository_impl_test.go
git commit -m "fix(ps-mag): escape ILIKE metacharacters in search phrase to prevent wildcard injection"
```

---

### Task 7: Final verification

**Step 1: Run the full test suite**

```bash
go test ./... 2>&1 | tail -30
```
Expected: all packages PASS, no compile errors.

**Step 2: Build a production binary**

```bash
go build -o /tmp/miltechserver_check ./... && rm /tmp/miltechserver_check
```
Expected: exits cleanly.

**Step 3: Verify the cache is effective (manual smoke test)**

If you have a running local server:

```bash
# First request — populates the cache (slow, ~500ms+)
time curl -s "http://localhost:8080/api/v1/library/ps-mag/issues" | jq '.data.total_count'

# Second request — served from cache (fast, <10ms)
time curl -s "http://localhost:8080/api/v1/library/ps-mag/issues" | jq '.data.total_count'
```
Expected: same `total_count` value; second request significantly faster.

**Step 4: Verify Cache-Control header**

```bash
curl -sI "http://localhost:8080/api/v1/library/ps-mag/issues" | grep -i cache-control
```
Expected: `Cache-Control: public, max-age=300`

**Step 5: Final commit if anything was touched**

If nothing changed, no commit needed.

---

## Not In Scope

The following was considered but excluded per YAGNI:

- **Store issue metadata in PostgreSQL** — Eliminates the Azure dependency entirely but requires a schema migration, data import pipeline, and Azure sync mechanism. The caching approach solves the immediate problem.
- **Slice pre-allocation** (`make([]PSMagIssueResponse, 0, 512)`) — Already incorporated into the updated `listAllIssues` in Task 2 as a one-liner.
- **Request timeout on Azure calls** — The request context propagated in Task 3 already enables client-side cancellation. An explicit timeout (e.g., `context.WithTimeout`) can be added in a follow-up if Azure latency becomes a production concern.
