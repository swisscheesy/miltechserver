# PS Magazine Download Analytics Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Increment an `analytics_event_counters` row each time a PS Magazine SAS download URL is successfully generated.

**Architecture:** Add `IncrementPSMagDownload` to the `analytics.Service` interface and implement it in `analytics.ServiceImpl` (which owns label normalization). Wire `analytics.Service` into `ps_mag.ServiceImpl` via constructor injection, mirroring the existing PMCS pattern in `library.ServiceImpl`. Update the route wiring in `library/route.go` to pass the already-available `deps.Analytics` through.

**Tech Stack:** Go 1.21+, Gin, Jet ORM, `github.com/stretchr/testify/require` for tests.

---

### Task 1: Add `IncrementPSMagDownload` to the analytics package (TDD)

**Files:**
- Create: `api/analytics/service_impl_test.go`
- Modify: `api/analytics/service.go`
- Modify: `api/analytics/service_impl.go`

---

**Step 1: Create the test file with a repo stub and failing tests**

Create `api/analytics/service_impl_test.go`:

```go
package analytics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// repoStub captures the arguments of the last IncrementCounter call.
type repoStub struct {
	capturedEventType   string
	capturedEntityKey   string
	capturedEntityLabel string
	err                 error
}

func (r *repoStub) IncrementCounter(eventType, entityKey, entityLabel string) error {
	r.capturedEventType = eventType
	r.capturedEntityKey = entityKey
	r.capturedEntityLabel = entityLabel
	return r.err
}

func TestFormatPSMagLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard filename",
			input: "PS_Magazine_Issue_004_September_1951.pdf",
			want:  "Issue 004 September 1951",
		},
		{
			name:  "no PS_Magazine_ prefix",
			input: "Some_Other_File.pdf",
			want:  "Some Other File",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "filename without extension",
			input: "PS_Magazine_Issue_001_January_1951",
			want:  "Issue 001 January 1951",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatPSMagLabel(tc.input))
		})
	}
}

func TestIncrementPSMagDownload_StandardFilename(t *testing.T) {
	repo := &repoStub{}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("PS_Magazine_Issue_004_September_1951.pdf")

	require.NoError(t, err)
	require.Equal(t, "ps_mag_download", repo.capturedEventType)
	require.Equal(t, "PS_MAGAZINE_ISSUE_004_SEPTEMBER_1951.PDF", repo.capturedEntityKey)
	require.Equal(t, "ISSUE 004 SEPTEMBER 1951", repo.capturedEntityLabel)
}

func TestIncrementPSMagDownload_EmptyFilename(t *testing.T) {
	repo := &repoStub{}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("")

	require.NoError(t, err)
	require.Empty(t, repo.capturedEventType) // repo must not be called for empty input
}

func TestIncrementPSMagDownload_RepoError(t *testing.T) {
	repo := &repoStub{err: errors.New("db down")}
	svc := NewService(repo)

	err := svc.IncrementPSMagDownload("PS_Magazine_Issue_001_January_1951.pdf")

	require.Error(t, err)
}
```

**Step 2: Run the tests to confirm they fail**

```bash
cd /Users/swisscheese/projects/miltechserver
go test ./api/analytics/...
```

Expected: compile error — `formatPSMagLabel` and `IncrementPSMagDownload` are undefined.

---

**Step 3: Add `IncrementPSMagDownload` to the Service interface**

Edit `api/analytics/service.go` — add the new method after `IncrementPMCSManualDownload`:

```go
type Service interface {
	IncrementItemSearchSuccess(niin string, nomenclature string) error
	IncrementPMCSManualDownload(entityKey string, entityLabel string) error
	IncrementPSMagDownload(filename string) error
	IncrementCounter(eventType string, entityKey string, entityLabel string) error
}
```

---

**Step 4: Implement the method and helper in service_impl.go**

In `api/analytics/service_impl.go`:

Add the constant (alongside the existing two):
```go
const (
	analyticsEventItemSearchSuccess  = "item_search_success"
	analyticsEventPMCSManualDownload = "pmcs_manual_download"
	analyticsEventPSMagDownload      = "ps_mag_download"
)
```

Add `formatPSMagLabel` after `sanitizePMCSKey`:
```go
// formatPSMagLabel derives a human-readable label from a PS Magazine filename.
// Example: "PS_Magazine_Issue_004_September_1951.pdf" → "Issue 004 September 1951"
func formatPSMagLabel(filename string) string {
	label := strings.TrimPrefix(filename, "PS_Magazine_")
	if idx := strings.LastIndex(label, "."); idx != -1 {
		label = label[:idx]
	}
	label = strings.ReplaceAll(label, "_", " ")
	return strings.TrimSpace(label)
}
```

Add `IncrementPSMagDownload` after `IncrementPMCSManualDownload`:
```go
func (service *ServiceImpl) IncrementPSMagDownload(filename string) error {
	normalizedKey := normalizeAnalyticsKey(filename)
	if normalizedKey == "" {
		return nil
	}
	label := normalizeAnalyticsKey(formatPSMagLabel(filename))
	if label == "" {
		label = normalizedKey
	}
	return service.IncrementCounter(analyticsEventPSMagDownload, normalizedKey, label)
}
```

**Step 5: Run the tests to confirm they pass**

```bash
go test ./api/analytics/...
```

Expected: all tests PASS.

**Step 6: Commit**

```bash
git add api/analytics/service.go api/analytics/service_impl.go api/analytics/service_impl_test.go
git commit -m "feat(analytics): add IncrementPSMagDownload with label normalization"
```

---

### Task 2: Wire analytics into ps_mag.ServiceImpl (TDD)

**Files:**
- Modify: `api/library/ps_mag/service_impl.go`
- Modify: `api/library/ps_mag/service_impl_test.go`

---

**Step 1: Add failing tests to service_impl_test.go**

At the bottom of `api/library/ps_mag/service_impl_test.go`, append:

First add the import for `"errors"` if not already there (it already is), and add `"miltechserver/api/analytics"` to the import block.

```go
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"miltechserver/api/analytics"
)
```

Then add the stub and tests at the bottom of the file:

```go
// analyticsStub satisfies analytics.Service for ps_mag unit testing.
type analyticsStub struct {
	capturedFilename string
	err              error
}

func (a *analyticsStub) IncrementItemSearchSuccess(_, _ string) error  { return nil }
func (a *analyticsStub) IncrementPMCSManualDownload(_, _ string) error { return nil }
func (a *analyticsStub) IncrementCounter(_, _, _ string) error         { return nil }
func (a *analyticsStub) IncrementPSMagDownload(filename string) error {
	a.capturedFilename = filename
	return a.err
}

func TestTrackPSMagDownload_CallsAnalytics(t *testing.T) {
	stub := &analyticsStub{}
	svc := &ServiceImpl{
		analytics: stub,
		cache:     newIssueCache(5 * time.Minute),
	}

	err := svc.trackPSMagDownload("ps-mag/PS_Magazine_Issue_004_September_1951.pdf")

	require.NoError(t, err)
	require.Equal(t, "PS_Magazine_Issue_004_September_1951.pdf", stub.capturedFilename)
}

func TestTrackPSMagDownload_NilAnalytics(t *testing.T) {
	svc := &ServiceImpl{cache: newIssueCache(5 * time.Minute)}

	// Must not panic when analytics is nil.
	err := svc.trackPSMagDownload("ps-mag/PS_Magazine_Issue_004_September_1951.pdf")

	require.NoError(t, err)
}

func TestTrackPSMagDownload_AnalyticsReturnsError(t *testing.T) {
	stub := &analyticsStub{err: errors.New("db down")}
	svc := &ServiceImpl{
		analytics: stub,
		cache:     newIssueCache(5 * time.Minute),
	}

	// trackPSMagDownload surfaces the error so GenerateDownloadURL can log it.
	err := svc.trackPSMagDownload("ps-mag/PS_Magazine_Issue_004_September_1951.pdf")

	require.Error(t, err)
}
```

Also update the existing `TestGenerateDownloadURLValidation` — change:
```go
svc := NewService(nil, nil)
```
to:
```go
svc := NewService(nil, nil, nil)
```

**Step 2: Run the tests to confirm they fail**

```bash
go test ./api/library/ps_mag/...
```

Expected: compile error — `ServiceImpl` has no `analytics` field, `trackPSMagDownload` undefined, `NewService` wrong arity.

---

**Step 3: Update ServiceImpl and NewService in service_impl.go**

In `api/library/ps_mag/service_impl.go`:

Add the import at the top:
```go
import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"miltechserver/api/analytics"
	"miltechserver/api/library/shared"
)
```

Update `ServiceImpl` to add the analytics field:
```go
type ServiceImpl struct {
	blobClient *azblob.Client
	repo       Repository
	cache      *issueCache
	analytics  analytics.Service
}
```

Update `NewService` to accept and store analytics:
```go
func NewService(blobClient *azblob.Client, db *sql.DB, analyticsService analytics.Service) Service {
	return &ServiceImpl{
		blobClient: blobClient,
		repo:       NewRepository(db),
		cache:      newIssueCache(10 * time.Minute),
		analytics:  analyticsService,
	}
}
```

Add `trackPSMagDownload` after `GenerateDownloadURL`:
```go
func (s *ServiceImpl) trackPSMagDownload(blobPath string) error {
	if s.analytics == nil {
		return nil
	}
	parts := strings.Split(blobPath, "/")
	filename := parts[len(parts)-1]
	if strings.TrimSpace(filename) == "" {
		return nil
	}
	return s.analytics.IncrementPSMagDownload(filename)
}
```

In `GenerateDownloadURL`, after the `sasResult` block and before the `return`, add the tracking call (mirrors the PMCS pattern in `library/service_impl.go`):
```go
	if trackErr := s.trackPSMagDownload(blobPath); trackErr != nil {
		slog.Warn("Failed to track PS Mag download analytics",
			"blobPath", blobPath,
			"error", trackErr)
	}

	return &DownloadURLResponse{
		BlobPath:    blobPath,
		DownloadURL: sasResult.URL,
		ExpiresAt:   sasResult.ExpiresAt.Format(time.RFC3339),
	}, nil
```

**Step 4: Run the tests to confirm they pass**

```bash
go test ./api/library/ps_mag/...
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add api/library/ps_mag/service_impl.go api/library/ps_mag/service_impl_test.go
git commit -m "feat(ps-mag): wire analytics into ServiceImpl to track downloads"
```

---

### Task 3: Update route wiring

**Files:**
- Modify: `api/library/ps_mag/route.go`
- Modify: `api/library/route.go`

No new tests needed — both files use internal `registerHandlers` functions in their tests, which are unaffected by this change.

---

**Step 1: Update RegisterHandlers in ps_mag/route.go**

Add `"miltechserver/api/analytics"` to the import block in `api/library/ps_mag/route.go`.

Change the public `RegisterHandlers` signature and body:

```go
// RegisterHandlers wires ps_mag routes into the public router group.
// Called from api/library/route.go.
func RegisterHandlers(publicGroup *gin.RouterGroup, blobClient *azblob.Client, db *sql.DB, analyticsService analytics.Service) {
	svc := NewService(blobClient, db, analyticsService)
	registerHandlers(publicGroup, svc)
}
```

**Step 2: Pass deps.Analytics in library/route.go**

In `api/library/route.go`, update the `ps_mag.RegisterHandlers` call in `RegisterRoutes`:

```go
func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
	svc := NewService(deps.BlobClient, deps.Env, deps.Analytics)
	registerHandlers(publicGroup, authGroup, svc)
	ps_mag.RegisterHandlers(publicGroup, deps.BlobClient, deps.DB, deps.Analytics)
}
```

**Step 3: Build to verify the whole project compiles**

```bash
go build ./...
```

Expected: exits 0, no output.

**Step 4: Run the full test suite for affected packages**

```bash
go test ./api/analytics/... ./api/library/...
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add api/library/ps_mag/route.go api/library/route.go
git commit -m "feat(ps-mag): pass analytics through route wiring to enable download tracking"
```
