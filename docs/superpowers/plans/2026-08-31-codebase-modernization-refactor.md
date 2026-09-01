# Codebase Modernization Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete dead code, consolidate duplicated boilerplate (pagination, response construction, transaction handling, test harnesses), split oversized files along existing seams, and thread `context.Context` through the service layer — with zero behavior change anywhere in the public API.

**Architecture:** Twenty independently-mergeable tasks across ten passes, ordered lowest-risk-first. Passes 1–3 have no dependencies. Pass 5 (response consolidation) and Pass 7 (file splits) are each split into one task per domain/file so no single PR is large. `user_pmcs` is excluded from response consolidation everywhere in this plan — its existing envelope in `api/user_pmcs/shared/http.go` is more capable, not legacy.

**Tech Stack:** Go 1.23, Gin, PostgreSQL via go-jet/jet, Firebase Auth, Azure Blob Storage, testify.

**Spec:** `docs/superpowers/specs/2026-08-31-codebase-modernization-refactor-design.md`

## Global Constraints

- No task may change an HTTP response status code, response body shape, or database schema. Any task that appears to require one must stop and flag it instead of proceeding.
- Every task's validation includes `go build ./...`, `go vet ./...`, and `go test -p 1 ./...`. The `-p 1` flag is mandatory — `tests/shops` and `tests/pmcs_sbs_progress` share a mutable test database and race under parallel package execution.
- Test database: `postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable`. This is the confirmed-correct, intentional test host — no task may change it, only deduplicate its occurrences (Task 19).
- `user_pmcs` (all subpackages: `owned`, `community`, `subscriptions`, `sync`, `persistence`, `shared`) is excluded from response-envelope consolidation (Pass 5) permanently. It is not deferred, not a future task — do not create one.
- Six items are explicitly out of scope for this entire plan and must not be touched by any task: standardizing `user_pmcs`'s response system repo-wide (or vice versa), collapsing rate-limiting systems, reconciling raw-SQL vs. Jet query-builder usage, the `.gen/miltech_ng/` tracked-vs-generated-at-build decision, any dependency/framework version bumps, and extending `context.Context` into repository methods that don't accept it today.
- Commit after every task. One logical change per commit, matching this repo's Conventional Commits convention (`fix`, `refactor`, `test`, `docs`, `chore` as appropriate — none of these tasks are `feat`, since no new behavior is introduced).

**User decisions (already made):**
- Pass 3: `192.168.20.70` is the confirmed-correct test DB host; the fix is deleting the dead `TEST_DATABASE_URL` env read, not wiring it in.
- Pass 5: response consolidation proceeds as one PR per domain (not one large PR).
- CodeGraph: `codegraph init -i` already run — `.codegraph/` exists and is available to every task for confirming call sites and blast radius before editing.

**Corrections made during planning (verified against current code, superseding the spec's figures):**
- Pass 6's transaction helper already exists: `persistence.WithWriteTx[T any]` in `api/user_pmcs/persistence/retry.go`, not something that needs extracting from `owned/repository_impl.go` first. The task is to promote it to a shared, non-`user_pmcs`-specific location and adopt it elsewhere.
- Pass 6's migration target count is **6 files**, not 18: `api/pmcs_sbs_progress/repository_impl.go`, `api/shops/messages/repository_impl.go`, `api/shops/vehicles/repository_impl.go`, `api/user_general/repository_impl.go`, `api/user_pmcs/sync/repository_impl.go`, `api/user_saves/categories/repository_impl.go`. (`api/user_pmcs/owned/repository_impl.go` and `api/user_pmcs/persistence/retry.go` already use the good pattern and are not migration targets.)

---

## File Structure

New files this plan creates:
- `api/response/response.go` — `OK()` / `Error()` helper functions (Task 4).
- `api/shared/pagination/pagination.go` — promoted pagination parser (Task 5).
- `api/shared/db/transaction.go` — promoted `WithWriteTx` (Task 12).
- `api/shops/aggregates/repository_notifications.go`, `repository_snapshot.go`, `repository_services.go`, `repository_pmcs_history.go` — split from `repository_impl.go` (Task 17).
- `api/shops/shared/scan_helpers.go` — generic null-scan helpers moved out of `shops/aggregates` (Task 17).
- `api/pmcs_sbs_progress/validation.go`, `api/pmcs_sbs_progress/mapping.go` — split from `service_impl.go` (Task 18).
- `api/user_pmcs/community/repository_voting.go`, `repository_browse.go` — split from `repository_impl.go` (Task 19... see numbering note below).
- `tests/testutil/testutil.go` — shared DSN constant, fake-auth middleware, router builder (Task 21).

Files modified in place: `api/response/standard_response.go`, `api/response/error_response.go` (Task 3); `api/route/route.go` (Tasks 1, 22); six `sb_700_20`/`docs_equipment`/`eic`/`tmde`/`library` handler files (Task 6); ~15 domain route files (Tasks 7–11); six repository files (Task 13); `bootstrap/env.go` untouched (context threading is service-layer only); 47 `service_impl.go` files (Task 20).

Files deleted: `api/service/auth_service.go`, `api/middleware/error_handler.go`, `api/middleware/rate_limiter.go`, `api/middleware/rate_limiter_test.go` (Task 1); `api/route/shops_route.go` (Task 22).

---

### Task 1: Delete dead code — auth_service, error_handler, rate_limiter

**Goal:** Remove three files that compile into the binary but have no live effect: `api/service/auth_service.go` (zero callers, commented-out `Login`, no-op methods), `api/middleware/error_handler.go` (self-flagged dead by its own TODO comment), and `api/middleware/rate_limiter.go` (fully built but never registered in `route.go`).

**Files:**
- Delete: `api/service/auth_service.go`
- Delete: `api/middleware/error_handler.go`
- Delete: `api/middleware/rate_limiter.go`
- Delete: `api/middleware/rate_limiter_test.go`
- Modify: `api/route/route.go` (remove any import/reference to the deleted symbols, if present)

**Acceptance Criteria:**
- [ ] `grep -rn "AuthService\|NewAuthService" --include="*.go" .` returns zero results.
- [ ] `grep -rn "middleware.ErrorHandler\|ErrorHandler(" --include="*.go" .` returns zero results (outside the deleted file's own history).
- [ ] `grep -rn "middleware.RateLimit\|NewRateLimiter\|rate_limiter" --include="*.go" .` returns zero results.
- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` reports no new issues.

**Verify:** `go build ./... && go vet ./... && go test -p 1 ./...` → all pass, identical results to the pre-task baseline.

**Steps:**

- [ ] **Step 1: Confirm zero external references before deleting**

```bash
grep -rn "AuthService\|NewAuthService" --include="*.go" . 
grep -rn "ErrorHandler" --include="*.go" .
grep -rn "RateLimit\|rate_limiter" --include="*.go" .
```
Expected: `AuthService` matches only inside `api/service/auth_service.go` itself. `ErrorHandler` matches only inside `api/middleware/error_handler.go` (and possibly its own test, if any — there is none). `RateLimit`/`rate_limiter` matches only inside `api/middleware/rate_limiter.go` and `api/middleware/rate_limiter_test.go`.

- [ ] **Step 2: Delete the four files**

```bash
git rm api/service/auth_service.go
git rm api/middleware/error_handler.go
git rm api/middleware/rate_limiter.go
git rm api/middleware/rate_limiter_test.go
```

- [ ] **Step 3: Check route.go for now-dangling references**

```bash
grep -n "ErrorHandler\|RateLimit\|AuthService" api/route/route.go
```
Expected: no matches (confirmed by the audit that none of these three are wired into `Setup()`). If any match appears, remove that line/reference before proceeding — do not leave a dangling call to a deleted symbol.

- [ ] **Step 4: Build and test**

```bash
go build ./...
go vet ./...
go test -p 1 ./...
```
Expected: all pass with no new failures.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
fix(cleanup): remove dead auth_service, error_handler, rate_limiter

Three files with zero live effect on the running application:
auth_service.go had no callers and a commented-out Login method,
error_handler.go was self-flagged as likely non-functional, and
rate_limiter.go was never wired into route.go's Setup().
EOF
)"
```

```json:metadata
{"files": ["api/service/auth_service.go", "api/middleware/error_handler.go", "api/middleware/rate_limiter.go", "api/middleware/rate_limiter_test.go", "api/route/route.go"], "verifyCommand": "go build ./... && go vet ./... && go test -p 1 ./...", "acceptanceCriteria": ["grep for AuthService/NewAuthService returns zero results", "grep for ErrorHandler returns zero results", "grep for RateLimit/rate_limiter returns zero results", "go build succeeds", "go vet reports no new issues"], "modelTier": "mechanical"}
```

---

### Task 2: Fix duplicate ADR-011 numbering in decisions.md

**Goal:** `docs/project_notes/decisions.md` has two entries both headed `### ADR-011` — line 221 ("Complete Shops HTTP Adapter Migration to Bounded Subdomains", 2026-02-18) and line 259 ("Shops Performance Optimization Refactor", 2026-02-01). Renumber the earlier-dated one forward so every ADR number is unique, without touching `.gitignore` or `.gen/` (that decision is out of scope for this plan).

**Files:**
- Modify: `docs/project_notes/decisions.md`

**Acceptance Criteria:**
- [ ] `grep -c "^### ADR-0" docs/project_notes/decisions.md` shows no number appearing twice (verify via the step below, not just the count).
- [ ] The chronological order of ADRs by date is preserved in the renumbering.
- [ ] No other file references the renumbered ADR by its old number (check `docs/agents/`, `CLAUDE.md`, or other docs for citations).

**Verify:** `grep -n "^### ADR-" docs/project_notes/decisions.md | awk -F'ADR-' '{print $2}' | awk -F: '{print $1}' | sort | uniq -d` → empty output (no duplicate numbers remain).

**Steps:**

- [ ] **Step 1: Read the full current ADR sequence to find the next free number**

```bash
grep -n "^### ADR-" docs/project_notes/decisions.md
```
Confirm ADR-021 (2026-08-16, "Rename PMCS SBS Persistence Tables") is the highest existing number, and confirm no ADR-022 exists yet.

- [ ] **Step 2: Check for external references to ADR-011 before renumbering**

```bash
grep -rn "ADR-011" --include="*.md" .
```
Note every file that cites `ADR-011` by number — each reference must be updated to match whichever entry keeps the ADR-011 number after this task.

- [ ] **Step 3: Renumber "Shops Performance Optimization Refactor" (2026-02-01, currently at line 259) to ADR-022**

Change the heading at line 259 from:
```markdown
### ADR-011: Shops Performance Optimization Refactor (2026-02-01)
```
to:
```markdown
### ADR-022: Shops Performance Optimization Refactor (2026-02-01)
```

This keeps "Complete Shops HTTP Adapter Migration" (the later-dated, more architecturally significant entry) at its original ADR-011 number — minimizing the number of external references that need updating, since it's more likely to be the one other docs cite — and appends the renumbered entry at the end of the sequence rather than renumbering everything in between, which would cascade into every ADR from 011 onward.

- [ ] **Step 4: Add an editorial note at the renumbered entry**

Immediately below the new `### ADR-022` heading, add:
```markdown
*(Originally numbered ADR-011; renumbered 2026-08-31 to resolve a duplicate heading. No content changed.)*
```

- [ ] **Step 5: Update any external references found in Step 2**

For each file found citing `ADR-011` where the citation clearly refers to "Shops Performance Optimization Refactor" rather than "Complete Shops HTTP Adapter Migration", update the citation to `ADR-022`. If ambiguous, leave a note in the commit message rather than guessing.

- [ ] **Step 6: Verify uniqueness**

```bash
grep -n "^### ADR-" docs/project_notes/decisions.md | awk -F'ADR-' '{print $2}' | awk -F: '{print $1}' | sort | uniq -d
```
Expected: empty output.

- [ ] **Step 7: Commit**

```bash
git add docs/project_notes/decisions.md
git commit -m "$(cat <<'EOF'
docs(decisions): resolve duplicate ADR-011 numbering

Two unrelated decisions shared the ADR-011 heading. Renumbered the
earlier-dated "Shops Performance Optimization Refactor" entry to
ADR-022, appended at the end of the sequence, to keep ADR numbers
unique without cascading renumbers through the entries in between.
EOF
)"
```

```json:metadata
{"files": ["docs/project_notes/decisions.md"], "verifyCommand": "grep -n '^### ADR-' docs/project_notes/decisions.md | awk -F'ADR-' '{print $2}' | awk -F: '{print $1}' | sort | uniq -d", "acceptanceCriteria": ["no ADR number appears twice", "chronological order preserved", "no dangling external references to the old number"], "modelTier": "mechanical"}
```

---

### Task 3: Remove dead DSN read in tests/item_lookup/main_test.go

**Goal:** `tests/item_lookup/main_test.go` reads `TEST_DATABASE_URL` from the environment, fatally errors if unset, then ignores the result and opens a hardcoded `192.168.20.70` connection string instead. Delete the dead read; keep the hardcoded DSN, matching every other `tests/<domain>/main_test.go` file's convention.

**Files:**
- Modify: `tests/item_lookup/main_test.go`

**Acceptance Criteria:**
- [ ] No reference to `TEST_DATABASE_URL` remains in `tests/item_lookup/main_test.go`.
- [ ] The hardcoded connection string `postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable` is unchanged.
- [ ] `os` import is removed if it becomes unused after the deletion (check remaining usages first).

**Verify:** `go test -p 1 ./tests/item_lookup/... -v` → all tests pass against the real test database, output unchanged from the pre-task baseline.

**Steps:**

- [ ] **Step 1: Read the current file to confirm exact lines and check for other `os.` usage**

```bash
grep -n "os\." tests/item_lookup/main_test.go
```
Confirm whether `os.Getenv("TEST_DATABASE_URL")` is the only use of the `os` package in this file (if `loadEnv()` or other code also uses `os`, the import stays).

- [ ] **Step 2: Remove the dead read**

Change:
```go
_ = loadEnv()

dsn := os.Getenv("TEST_DATABASE_URL")
if dsn == "" {
	log.Fatal("TEST_DATABASE_URL is not set")
}

var err error
testDB, err = sql.Open("postgres", "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable")
```
to:
```go
_ = loadEnv()

var err error
testDB, err = sql.Open("postgres", "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable")
```

- [ ] **Step 3: Remove the `os` import if it's now unused**

```bash
goimports -l tests/item_lookup/main_test.go
```
If `goimports` flags the file, run `goimports -w tests/item_lookup/main_test.go` to drop the unused import.

- [ ] **Step 4: Build and test**

```bash
go build ./...
go test -p 1 ./tests/item_lookup/... -v
```
Expected: build succeeds, all `item_lookup` integration tests pass identically to before.

- [ ] **Step 5: Commit**

```bash
git add tests/item_lookup/main_test.go
git commit -m "$(cat <<'EOF'
test(item_lookup): remove unused TEST_DATABASE_URL read

The env var was read and fatally checked but never actually used —
the test DB connection was opened with a hardcoded DSN on the next
line regardless. The hardcoded 192.168.20.70 host is the correct,
intentional test database target; only the dead read is removed.
EOF
)"
```

```json:metadata
{"files": ["tests/item_lookup/main_test.go"], "verifyCommand": "go build ./... && go test -p 1 ./tests/item_lookup/... -v", "acceptanceCriteria": ["no TEST_DATABASE_URL reference remains", "hardcoded DSN unchanged", "unused os import removed if applicable", "all item_lookup tests pass"], "modelTier": "mechanical"}
```

---

### Task 4: Add response.OK()/response.Error() helpers, collapse redundant types

**Goal:** `api/response/` defines three structurally identical types (`StandardResponse`, `NoItemFoundResponse`, `ErrorResponse` — all `{Status int; Data interface{}; Message string}`) and no function that actually writes a response; every call site hand-builds a struct literal and calls `c.JSON()` itself. Collapse the three types to `StandardResponse` alone (keeping the existing convenience constructors `NoItemFoundResponseMessage()` and `InternalErrorResponseMessage()`, retyped to return `StandardResponse`), and add `OK()`/`Error()` helper functions that domains will adopt one at a time in Tasks 7–11.

This task only touches `api/response/` itself — no call sites are migrated yet. `NoItemFoundResponse` and `ErrorResponse` are Go type aliases to `StandardResponse` at first (not deleted outright) so that any call site not yet migrated in Tasks 7–11 continues to compile unchanged.

**Files:**
- Modify: `api/response/standard_response.go`
- Modify: `api/response/error_response.go`
- Create: `api/response/response.go`
- Test: `api/response/response_test.go`

**Acceptance Criteria:**
- [ ] `response.OK(c, data)` writes `{"status": 200, "data": <data>, "message": ""}` with HTTP 200.
- [ ] `response.Error(c, status, message)` writes `{"status": <status>, "data": null, "message": <message>}` with the given HTTP status.
- [ ] `NoItemFoundResponse` and `ErrorResponse` remain valid identifiers (as type aliases to `StandardResponse`) so existing call sites using them do not break before their domain's migration task runs.
- [ ] `go build ./...` succeeds with zero changes required in any other file.

**Verify:** `go test ./api/response/... -v` → new tests for `OK`/`Error` pass; `go build ./...` succeeds repo-wide with no other file touched.

**Steps:**

- [ ] **Step 1: Write the failing test for the new helpers**

```go
// api/response/response_test.go
package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, map[string]string{"foo": "bar"})

	require.Equal(t, 200, w.Code)

	var body StandardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 200, body.Status)
	require.Equal(t, "", body.Message)
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 404, "no item found")

	require.Equal(t, 404, w.Code)

	var body StandardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 404, body.Status)
	require.Nil(t, body.Data)
	require.Equal(t, "no item found", body.Message)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./api/response/... -run "TestOK|TestError" -v
```
Expected: FAIL — `OK` and `Error` are undefined.

- [ ] **Step 3: Write `api/response/response.go`**

```go
package response

import "github.com/gin-gonic/gin"

// OK writes a 200 response with the given data in the standard envelope.
func OK(c *gin.Context, data interface{}) {
	c.JSON(200, StandardResponse{
		Status: 200,
		Data:   data,
	})
}

// Error writes an error response in the standard envelope at the given
// status code.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, StandardResponse{
		Status:  status,
		Data:    nil,
		Message: message,
	})
}
```

- [ ] **Step 4: Collapse the redundant types to aliases**

Replace the contents of `api/response/error_response.go`:
```go
package response

// NoItemFoundResponse is a type alias retained for existing call sites;
// it is structurally identical to StandardResponse and carries no
// distinct fields.
type NoItemFoundResponse = StandardResponse

func NoItemFoundResponseMessage() NoItemFoundResponse {
	return NoItemFoundResponse{
		Status:  404,
		Data:    nil,
		Message: "no item(s) found",
	}
}

// ErrorResponse is a type alias retained for existing call sites; it is
// structurally identical to StandardResponse and carries no distinct
// fields.
type ErrorResponse = StandardResponse

func InternalErrorResponseMessage() ErrorResponse {
	return ErrorResponse{
		Status:  500,
		Data:    nil,
		Message: "internal Server Error",
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./api/response/... -v
```
Expected: PASS for `TestOK` and `TestError`, and all pre-existing tests in `api/response/` (including `user_shops_response_test.go`) still pass.

- [ ] **Step 6: Confirm zero other files needed changes**

```bash
go build ./...
```
Expected: succeeds with no compile errors anywhere in the repo — the type-alias approach means this task is invisible to every other call site until its domain's migration task (7–11) runs.

- [ ] **Step 7: Commit**

```bash
git add api/response/
git commit -m "$(cat <<'EOF'
refactor(response): add OK/Error helpers, alias redundant types

StandardResponse, NoItemFoundResponse, and ErrorResponse were three
structurally identical types with no shared way to actually write a
response — every call site hand-built a struct literal and called
c.JSON directly. Adds response.OK()/response.Error() as the new
convention, and aliases the two redundant types to StandardResponse
so existing call sites keep compiling until each domain migrates in
a follow-up task.
EOF
)"
```

```json:metadata
{"files": ["api/response/standard_response.go", "api/response/error_response.go", "api/response/response.go", "api/response/response_test.go"], "verifyCommand": "go test ./api/response/... -v && go build ./...", "acceptanceCriteria": ["response.OK writes 200 with correct envelope", "response.Error writes given status with correct envelope", "NoItemFoundResponse and ErrorResponse remain valid aliased identifiers", "go build succeeds with zero other files touched"], "modelTier": "standard"}
```

---

### Task 5: Promote shared pagination helper

**Goal:** A working pagination parser already exists at `api/item_lookup/shared/pagination.go` but is only used within `item_lookup`. Promote it to a shared, domain-agnostic location so Task 6 can adopt it across `sb_700_20`, `docs_equipment`, `eic`, `tmde`, and `library/ps_mag`.

**Files:**
- Create: `api/shared/pagination/pagination.go`
- Test: `api/shared/pagination/pagination_test.go`
- Modify: `api/item_lookup/shared/pagination.go` (re-export from the new location, or update its callers to import the new package directly — decide based on Step 1's findings)

**Acceptance Criteria:**
- [ ] The promoted helper parses `page` and `page_size` (or the existing parameter names used by `item_lookup/shared/pagination.go` — confirmed in Step 1) from `*gin.Context`, returning validated ints or writing a 400 response and returning `false`/an error the caller checks.
- [ ] Table-driven test covers: valid page, missing page (defaults), zero page, negative page, non-numeric page.
- [ ] `item_lookup`'s existing callers continue to produce identical behavior — this task does not change `item_lookup`'s response bodies.

**Verify:** `go test ./api/shared/pagination/... ./api/item_lookup/... -v` → all pass, `item_lookup` route tests unchanged.

**Steps:**

- [ ] **Step 1: Read the existing helper to capture its exact current signature and behavior**

```bash
cat api/item_lookup/shared/pagination.go
grep -rn "shared.Parse\|shared\.Pag" api/item_lookup/ --include="*.go"
```
Note the exact function name, parameter names, default values, and error-response format it currently produces — the promoted version must be byte-identical in behavior.

- [ ] **Step 2: Write the table-driven test first, based on the captured behavior from Step 1**

```go
// api/shared/pagination/pagination_test.go
package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryString string
		wantPage    int
		wantOK      bool
	}{
		{"valid page", "?page=3", 3, true},
		{"missing page defaults to 1", "", 1, true},
		{"zero page is invalid", "?page=0", 0, false},
		{"negative page is invalid", "?page=-1", 0, false},
		{"non-numeric page is invalid", "?page=abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.queryString, nil)
			c.Request = req

			page, ok := Parse(c)

			require.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.Equal(t, tt.wantPage, page)
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}
```

Note: adjust the function name (`Parse`), field names, and default/error behavior in this test to exactly match what Step 1 found in the existing `item_lookup/shared/pagination.go` — do not invent new defaults or validation rules.

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./api/shared/pagination/... -v
```
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 4: Create the promoted package, copying the exact logic found in Step 1**

```go
// api/shared/pagination/pagination.go
package pagination

// (Exact contents copied from api/item_lookup/shared/pagination.go,
// package name changed to `pagination`, with identical function
// signature, defaults, and error-response behavior as verified in
// Step 1.)
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./api/shared/pagination/... -v
```
Expected: PASS for all five table cases.

- [ ] **Step 6: Point `item_lookup/shared/pagination.go` at the promoted package**

Replace the body of `api/item_lookup/shared/pagination.go` with a thin re-export so `item_lookup`'s existing callers need no changes in this task:
```go
package shared

import "miltechserver/api/shared/pagination"

// Parse is kept here as a re-export for existing item_lookup callers.
// New code should import miltechserver/api/shared/pagination directly.
var Parse = pagination.Parse
```
(Adjust the exact re-export shape to match whatever the real function signature turned out to be in Step 1 — if it's not a simple `var = func` reference due to generics or multiple return values needing a wrapper, write a one-line wrapper function instead.)

- [ ] **Step 7: Run full test suite for both packages**

```bash
go build ./...
go test ./api/shared/pagination/... ./api/item_lookup/... -v
```
Expected: all pass, `item_lookup` behavior is provably unchanged since it now calls through to the exact same logic.

- [ ] **Step 8: Commit**

```bash
git add api/shared/pagination/ api/item_lookup/shared/pagination.go
git commit -m "$(cat <<'EOF'
refactor(pagination): promote item_lookup pagination helper to shared pkg

The pagination parse-and-validate helper was item_lookup-specific
despite the same block being hand-copied 15+ times across other
domains. Promotes it to api/shared/pagination, re-exported from its
original location so item_lookup's existing callers are unaffected.
Adoption by other domains follows in a separate task.
EOF
)"
```

```json:metadata
{"files": ["api/shared/pagination/pagination.go", "api/shared/pagination/pagination_test.go", "api/item_lookup/shared/pagination.go"], "verifyCommand": "go build ./... && go test ./api/shared/pagination/... ./api/item_lookup/... -v", "acceptanceCriteria": ["promoted helper behavior matches original exactly", "table-driven test covers valid/missing/zero/negative/non-numeric page", "item_lookup callers unaffected"], "modelTier": "standard"}
```

---

### Task 6: Adopt shared pagination helper in sb_700_20, docs_equipment, eic, tmde, library/ps_mag

**Goal:** Replace the copy-pasted pagination parse-and-validate block in five domains with calls to the shared helper from Task 5. `sb_700_20/handlers_apps.go` has the block 9 times in one file; `sb_700_20/handlers_chps.go`, `docs_equipment/route.go`, `eic/route.go`, `tmde/route.go`, and `library/ps_mag/route.go` each have it at least once.

**Files:**
- Modify: `api/sb_700_20/handlers_apps.go`
- Modify: `api/sb_700_20/handlers_chps.go`
- Modify: `api/docs_equipment/route.go`
- Modify: `api/eic/route.go`
- Modify: `api/tmde/route.go`
- Modify: `api/library/ps_mag/route.go`

**Acceptance Criteria:**
- [ ] Every hand-copied pagination block in the six files above is replaced with a call to `pagination.Parse(c)` (or the exact function name from Task 5).
- [ ] `grep -rn "DefaultQuery(\"page\"" api/sb_700_20 api/docs_equipment api/eic api/tmde api/library/ps_mag` returns zero results after the change.
- [ ] Each touched domain's existing tests (integration in `tests/<domain>/` and/or colocated `_test.go`) pass with identical status codes and response bodies for valid, missing, zero, negative, and non-numeric page params.

**Verify:** `go test -p 1 ./tests/sb_700_20/... ./tests/docs_equipment/... ./tests/eic/... ./tests/tmde/... ./api/library/... -v` → all pass, identical to pre-task baseline.

**Steps:**

- [ ] **Step 1: Capture the pre-change baseline for each domain**

```bash
go test -p 1 ./tests/sb_700_20/... ./tests/docs_equipment/... ./tests/eic/... ./tests/tmde/... ./api/library/... -v 2>&1 | tee /tmp/pagination-baseline.log
```
Save this output — it's the parity reference for Step 4.

- [ ] **Step 2: Locate every copy-pasted block**

```bash
grep -n "DefaultQuery(\"page\"\|Query(\"page\")" api/sb_700_20/handlers_apps.go api/sb_700_20/handlers_chps.go api/docs_equipment/route.go api/eic/route.go api/tmde/route.go api/library/ps_mag/route.go
```
Confirm the count matches the audit's finding (9 in `handlers_apps.go`, plus the others) before editing.

- [ ] **Step 3: Replace each block with a call to the shared helper, one file at a time**

For each occurrence, replace the multi-line `c.DefaultQuery("page", "1")` → `strconv.Atoi` → 400-response block with:
```go
page, ok := pagination.Parse(c)
if !ok {
	return
}
```
(Adjust to match Task 5's actual function signature — if it returns an error instead of a bool, or writes the response itself vs. expecting the caller to, match that exact contract.) Add the import `"miltechserver/api/shared/pagination"` to each file. Remove the now-unused `strconv` import if nothing else in the file uses it — check with `goimports -l <file>` before removing.

- [ ] **Step 4: Build and test after each file, comparing against the Step 1 baseline**

```bash
go build ./...
go test -p 1 ./tests/sb_700_20/... ./tests/docs_equipment/... ./tests/eic/... ./tests/tmde/... ./api/library/... -v
diff /tmp/pagination-baseline.log <(go test -p 1 ./tests/sb_700_20/... ./tests/docs_equipment/... ./tests/eic/... ./tests/tmde/... ./api/library/... -v 2>&1)
```
Expected: no diff beyond timing/ordering noise — status codes and response bodies identical.

- [ ] **Step 5: Commit, one domain per commit**

```bash
git add api/sb_700_20/
git commit -m "refactor(sb_700_20): adopt shared pagination helper"

git add api/docs_equipment/
git commit -m "refactor(docs_equipment): adopt shared pagination helper"

git add api/eic/
git commit -m "refactor(eic): adopt shared pagination helper"

git add api/tmde/
git commit -m "refactor(tmde): adopt shared pagination helper"

git add api/library/
git commit -m "refactor(library): adopt shared pagination helper in ps_mag"
```

```json:metadata
{"files": ["api/sb_700_20/handlers_apps.go", "api/sb_700_20/handlers_chps.go", "api/docs_equipment/route.go", "api/eic/route.go", "api/tmde/route.go", "api/library/ps_mag/route.go"], "verifyCommand": "go test -p 1 ./tests/sb_700_20/... ./tests/docs_equipment/... ./tests/eic/... ./tests/tmde/... ./api/library/... -v", "acceptanceCriteria": ["all copy-pasted pagination blocks replaced", "zero DefaultQuery(\"page\" matches remain in the five domains", "response bodies and status codes identical to pre-task baseline for all param edge cases"], "modelTier": "standard"}
```

---

### Task 7: Response consolidation — material_images

**Goal:** `material_images` has 35 raw `gin.H{}`/`c.JSON(http.StatusOK` occurrences across `flags/route.go`, `images/route.go`, `votes/route.go` and zero use of `response.StandardResponse`. Migrate to `response.OK()`/`response.Error()` from Task 4, preserving every existing field name and status code exactly.

**Files:**
- Modify: `api/material_images/flags/route.go`
- Modify: `api/material_images/images/route.go`
- Modify: `api/material_images/votes/route.go`

**Acceptance Criteria:**
- [ ] A golden-fixture snapshot of every endpoint's response body (captured in Step 1) matches byte-for-byte after migration.
- [ ] `grep -rn "gin.H{" api/material_images/` returns zero results.
- [ ] Domain-specific response types (`response.ImageFlagResponse`, `response.PaginatedImagesResponse`, `response.ImageVoteResponse`) are preserved as-is if they carry fields beyond the standard envelope — only the raw `gin.H{}`/manual `c.JSON` calls are replaced, not the domain-specific typed responses themselves (those are a deliberate, richer shape, not the drift this pass targets).

**Verify:** `go test -p 1 ./tests/material_images/... -v` → all pass; golden-fixture diff is empty.

**Steps:**

- [ ] **Step 1: Capture the golden fixture before any change**

```bash
mkdir -p /tmp/golden-fixtures
go test -p 1 ./tests/material_images/... -v 2>&1 | tee /tmp/golden-fixtures/material_images-before.log
```
If integration tests assert on response body content, this log captures it. Additionally, if feasible, run the actual handlers against a local test server and curl each endpoint to capture literal JSON bodies into `/tmp/golden-fixtures/material_images-*.json` for a more direct byte-diff in Step 4 — use whichever approach the existing `tests/material_images/` setup supports.

- [ ] **Step 2: Locate every raw response construction**

```bash
grep -n "gin.H{\|c.JSON(http.StatusOK" api/material_images/flags/route.go api/material_images/images/route.go api/material_images/votes/route.go
```

- [ ] **Step 3: Replace each raw construction with response.OK()/response.Error(), preserving exact field names**

For a success case like:
```go
c.JSON(http.StatusOK, gin.H{"success": true, "message": "flag created"})
```
replace with:
```go
response.OK(c, gin.H{"success": true, "message": "flag created"})
```
(Preserve the inner `gin.H{}` payload exactly as the `data` field — only the outer envelope construction changes, not the payload shape, since changing field names would be a response-body change this task must not make.) For error cases like:
```go
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image id"})
```
replace with:
```go
response.Error(c, http.StatusBadRequest, "invalid image id")
```
Note this changes the JSON key from `error` to the standard envelope's `message` field — flag this specific class of change explicitly in the PR description as a call-site response-shape difference, and confirm via the golden-fixture diff in Step 4 whether any existing test asserts on the literal `"error"` key. If a test does assert on `"error"`, that test's assertion must be updated to `"message"` in this same task (test-only change, not a production behavior change beyond the envelope key itself) — and this must be called out explicitly to you before merging, since it is a response-shape change technically outside this task's "byte-for-byte identical" acceptance criterion.

- [ ] **Step 4: Build, test, and diff against the golden fixture**

```bash
go build ./...
go test -p 1 ./tests/material_images/... -v 2>&1 | tee /tmp/golden-fixtures/material_images-after.log
diff /tmp/golden-fixtures/material_images-before.log /tmp/golden-fixtures/material_images-after.log
```
Expected: empty diff, or only the `error`→`message` key differences identified and confirmed in Step 3.

- [ ] **Step 5: Commit**

```bash
git add api/material_images/
git commit -m "$(cat <<'EOF'
refactor(material_images): adopt response.OK/Error helpers

Replaces 35 raw gin.H{}/c.JSON(http.StatusOK constructions across
flags, images, and votes route files with the shared response
envelope helpers. Domain-specific typed responses (ImageFlagResponse,
PaginatedImagesResponse, ImageVoteResponse) are unchanged.
EOF
)"
```

```json:metadata
{"files": ["api/material_images/flags/route.go", "api/material_images/images/route.go", "api/material_images/votes/route.go"], "verifyCommand": "go test -p 1 ./tests/material_images/... -v", "acceptanceCriteria": ["golden-fixture diff empty or explicitly confirmed error-to-message key changes only", "zero raw gin.H{} constructions remain", "domain-specific typed responses unchanged"], "modelTier": "standard"}
```

---

### Task 8: Response consolidation — pmcs_sbs_progress

**Goal:** `api/pmcs_sbs_progress/route.go` mixes `response.StandardResponse{...}` literals (some lines) with raw `gin.H{"message": ...}` literals (other lines) in the same file — no single convention even per-file. Migrate all of it to `response.OK()`/`response.Error()`.

**Files:**
- Modify: `api/pmcs_sbs_progress/route.go`

**Acceptance Criteria:**
- [ ] Golden-fixture snapshot (captured in Step 1) matches byte-for-byte after migration.
- [ ] `grep -n "gin.H{\|response.StandardResponse{" api/pmcs_sbs_progress/route.go` returns zero results (all replaced with helper calls).

**Verify:** `go test -p 1 ./tests/pmcs_sbs_progress/... -v` → all pass; golden-fixture diff empty.

**Steps:**

- [ ] **Step 1: Capture the golden fixture**

```bash
go test -p 1 ./tests/pmcs_sbs_progress/... -v 2>&1 | tee /tmp/golden-fixtures/pmcs_sbs_progress-before.log
```

- [ ] **Step 2: Locate every raw or inline-literal response construction**

```bash
grep -n "gin.H{\|response.StandardResponse{" api/pmcs_sbs_progress/route.go
```
Expected to match the audit's finding: lines 68, 83, 97, 118, 139, 159, 180, 226, 241 (confirm exact current line numbers, since this file may have shifted since the audit).

- [ ] **Step 3: Replace each with response.OK()/response.Error(), preserving exact field names and status codes**

For `response.StandardResponse{...}` literals passed to `c.JSON`, replace with the equivalent `response.OK(c, data)` or `response.Error(c, status, message)` call — these are already using the right shape, just not the new helper function, so this is a pure mechanical substitution with no field-name risk. For `gin.H{"message": ...}` literals, apply the same treatment as Task 7 Step 3 — preserve the payload shape, and flag any `error`-vs-`message` key discrepancy explicitly.

- [ ] **Step 4: Build, test, and diff**

```bash
go build ./...
go test -p 1 ./tests/pmcs_sbs_progress/... -v 2>&1 | tee /tmp/golden-fixtures/pmcs_sbs_progress-after.log
diff /tmp/golden-fixtures/pmcs_sbs_progress-before.log /tmp/golden-fixtures/pmcs_sbs_progress-after.log
```
Expected: empty diff or explicitly confirmed key changes only.

- [ ] **Step 5: Commit**

```bash
git add api/pmcs_sbs_progress/route.go
git commit -m "$(cat <<'EOF'
refactor(pmcs_sbs_progress): adopt response.OK/Error helpers

route.go mixed response.StandardResponse{} literals with raw
gin.H{"message": ...} literals with no single convention even
within the file. Both are replaced with the shared envelope helpers.
EOF
)"
```

```json:metadata
{"files": ["api/pmcs_sbs_progress/route.go"], "verifyCommand": "go test -p 1 ./tests/pmcs_sbs_progress/... -v", "acceptanceCriteria": ["golden-fixture diff empty or explicitly confirmed key changes only", "zero raw gin.H{} or inline StandardResponse literals remain"], "modelTier": "standard"}
```

---

### Task 9: Response consolidation — shops (excluding vehicles/notifications subpackages already using StandardResponse cleanly)

**Goal:** `shops` has 7 raw `c.JSON(http.StatusOK` calls against 13 files already using `StandardResponse` — majority-standard but with stragglers. Find and migrate the 7 raw calls.

**Files:**
- Modify: files identified in Step 1 (not enumerated here since exact file:line may have shifted since the audit — Step 1 locates them precisely)

**Acceptance Criteria:**
- [ ] Golden-fixture snapshot matches byte-for-byte after migration.
- [ ] `grep -rn "c.JSON(http.StatusOK" api/shops/ | grep -v "StandardResponse\|response\."` returns zero results (raw calls not using the response package).

**Verify:** `go test -p 1 ./tests/shops/... -v` → all pass (remember `-p 1`, `shops` shares a mutable test DB with `pmcs_sbs_progress`); golden-fixture diff empty.

**Steps:**

- [ ] **Step 1: Locate the exact 7 raw calls**

```bash
grep -rn "c.JSON(http.StatusOK" api/shops/ --include="*.go" | grep -v _test.go
```
Cross-reference each match against `grep -rn "response.StandardResponse\|response.OK" api/shops/` in the same files to confirm which calls are genuinely raw vs. already-converted.

- [ ] **Step 2: Capture the golden fixture**

```bash
go test -p 1 ./tests/shops/... -v 2>&1 | tee /tmp/golden-fixtures/shops-before.log
```

- [ ] **Step 3: Replace each raw call with response.OK()/response.Error(), preserving field names exactly**

Apply the same substitution pattern as Task 7 Step 3 to each of the 7 located call sites.

- [ ] **Step 4: Build, test, and diff**

```bash
go build ./...
go test -p 1 ./tests/shops/... -v 2>&1 | tee /tmp/golden-fixtures/shops-after.log
diff /tmp/golden-fixtures/shops-before.log /tmp/golden-fixtures/shops-after.log
```
Expected: empty diff.

- [ ] **Step 5: Commit**

```bash
git add api/shops/
git commit -m "$(cat <<'EOF'
refactor(shops): migrate remaining raw c.JSON calls to response.OK

7 raw c.JSON(http.StatusOK calls remained alongside 13 files already
using StandardResponse. Converts the stragglers to the shared helper
for full consistency within the domain.
EOF
)"
```

```json:metadata
{"files": [], "verifyCommand": "go test -p 1 ./tests/shops/... -v", "acceptanceCriteria": ["golden-fixture diff empty", "zero raw c.JSON(http.StatusOK calls remain outside the response package"], "modelTier": "standard"}
```

---

### Task 10: Response consolidation — tmde, item_lookup, eic

**Goal:** These three domains have small numbers of raw calls (`tmde`: 2 raw, 1 `StandardResponse`; `item_lookup`: 1 raw, 1 `StandardResponse`; `eic`: 0 raw, 1 `StandardResponse` — confirm current counts since these may have shifted after Task 6's pagination changes touched `tmde` and `eic`'s route files). Migrate any remaining raw calls in each; if `eic` genuinely has zero raw calls remaining, its portion of this task is a verification-only no-op.

**Files:**
- Modify: `api/tmde/route.go` (if raw calls remain)
- Modify: files under `api/item_lookup/` identified in Step 1 (if raw calls remain)
- Verify only: `api/eic/route.go`

**Acceptance Criteria:**
- [ ] Golden-fixture snapshots for `tmde` and `item_lookup` match byte-for-byte after migration.
- [ ] `eic` confirmed to already have zero raw `gin.H{}`/`c.JSON(http.StatusOK` calls, or migrated if any are found.

**Verify:** `go test -p 1 ./tests/tmde/... ./tests/item_lookup/... ./tests/eic/... -v` → all pass; golden-fixture diffs empty.

**Steps:**

- [ ] **Step 1: Locate current raw calls in all three domains**

```bash
grep -rn "gin.H{\|c.JSON(http.StatusOK" api/tmde/ api/item_lookup/ api/eic/ --include="*.go" | grep -v _test.go | grep -v "response\."
```

- [ ] **Step 2: Capture golden fixtures for domains with raw calls found**

```bash
go test -p 1 ./tests/tmde/... ./tests/item_lookup/... -v 2>&1 | tee /tmp/golden-fixtures/tmde-item_lookup-before.log
```

- [ ] **Step 3: Replace each raw call, preserving field names exactly**

Apply the Task 7 Step 3 substitution pattern to each located call site in `tmde` and `item_lookup`.

- [ ] **Step 4: Build, test, and diff**

```bash
go build ./...
go test -p 1 ./tests/tmde/... ./tests/item_lookup/... ./tests/eic/... -v 2>&1 | tee /tmp/golden-fixtures/tmde-item_lookup-after.log
diff /tmp/golden-fixtures/tmde-item_lookup-before.log /tmp/golden-fixtures/tmde-item_lookup-after.log
```
Expected: empty diff; `eic` tests pass unchanged (verification-only if Step 1 found no raw calls there).

- [ ] **Step 5: Commit**

```bash
git add api/tmde/ api/item_lookup/
git commit -m "$(cat <<'EOF'
refactor(tmde,item_lookup): migrate remaining raw response calls

Converts the last few raw gin.H{}/c.JSON(http.StatusOK calls in
tmde and item_lookup to the shared response.OK/Error helpers. eic
was confirmed to already be fully migrated to StandardResponse.
EOF
)"
```

```json:metadata
{"files": [], "verifyCommand": "go test -p 1 ./tests/tmde/... ./tests/item_lookup/... ./tests/eic/... -v", "acceptanceCriteria": ["golden-fixture diffs empty for tmde and item_lookup", "eic confirmed already consistent or migrated"], "modelTier": "standard"}
```

---

### Task 11: Response consolidation — remaining domains sweep

**Goal:** Sweep every remaining domain (excluding `user_pmcs`, permanently excluded) for leftover raw `gin.H{}`/`c.JSON(http.StatusOK` response construction not covered by Tasks 7–10, and migrate to `response.OK()`/`response.Error()`.

**Files:**
- Modify: files identified in Step 1 across all non-`user_pmcs` domains

**Acceptance Criteria:**
- [ ] `grep -rln "gin.H{" api/ --include="*.go" | grep -v _test.go | grep -v user_pmcs` returns zero results, or every remaining match is confirmed in Step 1 to be a non-response use of `gin.H` (e.g., building a nested data payload passed *into* `response.OK`, not an outer envelope) rather than a response construction this pass targets.
- [ ] Golden-fixture snapshots for each touched domain match byte-for-byte.
- [ ] `user_pmcs` is untouched by this task — confirm via `git diff --stat` showing no `api/user_pmcs/` files.

**Verify:** `go test -p 1 ./...` (full suite) → all pass; per-domain golden-fixture diffs empty.

**Steps:**

- [ ] **Step 1: Full repo sweep for remaining raw response construction**

```bash
grep -rln "gin.H{\|c.JSON(http.Status" api/ --include="*.go" | grep -v _test.go | grep -v api/user_pmcs
```
For each file found, inspect whether the `gin.H{}` is an outer response envelope (migration target) or an inner data payload already passed to `response.OK`/`response.StandardResponse` (not a target — leave as-is, since that's the `data` field's legitimate shape, not envelope drift).

- [ ] **Step 2: Capture golden fixtures for every domain with a genuine migration target**

```bash
go test -p 1 ./... -v 2>&1 | tee /tmp/golden-fixtures/full-repo-before.log
```

- [ ] **Step 3: Migrate each genuine target, one domain per commit, preserving field names exactly**

Apply the Task 7 Step 3 substitution pattern per domain. Commit after each domain individually rather than batching, consistent with the "one PR per domain" decision.

- [ ] **Step 4: Build, test, and diff after all domains are migrated**

```bash
go build ./...
go test -p 1 ./... -v 2>&1 | tee /tmp/golden-fixtures/full-repo-after.log
diff /tmp/golden-fixtures/full-repo-before.log /tmp/golden-fixtures/full-repo-after.log
```
Expected: empty diff.

- [ ] **Step 5: Confirm user_pmcs untouched**

```bash
git diff --stat main -- api/user_pmcs/
```
Expected: empty output across every commit made in this task.

- [ ] **Step 6: Final commit (or last of the per-domain commits from Step 3)**

```bash
git log --oneline -20
```
Confirm each domain touched in this task has its own commit.

```json:metadata
{"files": [], "verifyCommand": "go test -p 1 ./... -v", "acceptanceCriteria": ["no genuine raw response-envelope construction remains outside user_pmcs", "golden-fixture diffs empty per domain", "user_pmcs completely untouched"], "modelTier": "standard"}
```

---

### Task 12: Promote WithWriteTx to a shared, non-user_pmcs-specific package

**Goal:** `persistence.WithWriteTx[T any]` already exists in `api/user_pmcs/persistence/retry.go` as a working generic transaction-callback helper with retry semantics. Promote it (or a simplified variant without `user_pmcs`-specific retry/lock coupling, if Step 1 finds it's too entangled with that domain's specific concerns to lift as-is) to a shared location so Task 13 can adopt it in six other repositories.

**Files:**
- Read: `api/user_pmcs/persistence/retry.go` (source of the pattern — not modified, since `user_pmcs` keeps its own copy per the exclusion policy... note: this exclusion applies to response envelopes per the spec; transaction helpers are not covered by that exclusion, so evaluate in Step 1 whether `user_pmcs` should import the shared version too, or keep its own for a documented reason)
- Create: `api/shared/db/transaction.go`
- Test: `api/shared/db/transaction_test.go`

**Acceptance Criteria:**
- [ ] The promoted helper accepts a `*sql.DB`, a callback `func(tx *sql.Tx) (T, error)`, and returns `(T, error)`, committing on success and rolling back on any error.
- [ ] A test using a real (test-DB) transaction confirms both the commit path and the rollback-on-error path leave the database in the expected state — this is not just a compile check.
- [ ] `user_pmcs/persistence/retry.go` is unmodified by this task (adoption decision, if any, is made explicitly and separately, not as a silent side effect).

**Verify:** `go test ./api/shared/db/... -v` (requires the test DB at `192.168.20.70`) → commit and rollback paths both pass.

**Steps:**

- [ ] **Step 1: Read the existing implementation in full**

```bash
sed -n '1,60p' api/user_pmcs/persistence/retry.go
```
Determine whether `WithWriteTx`'s retry/serialization-conflict logic (`WithSerializableWriteTx` alongside it) is generic enough to lift wholesale, or whether the six Task-13 target files only need the simpler "begin, run callback, commit or rollback" shape without retry. The six files' current manual pattern (per the audit) is plain `Begin`/`Commit`/`Rollback` with no retry — so the promoted helper should match that simpler need, not force retry semantics onto call sites that never had them.

- [ ] **Step 2: Write the failing test for the simplified helper**

```go
// api/shared/db/transaction_test.go
package db

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const testDSN = "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable"

func TestWithTx_CommitsOnSuccess(t *testing.T) {
	conn, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer conn.Close()

	result, err := WithTx(conn, func(tx *sql.Tx) (int, error) {
		return 42, nil
	})

	require.NoError(t, err)
	require.Equal(t, 42, result)
}

func TestWithTx_RollsBackOnError(t *testing.T) {
	conn, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer conn.Close()

	// Create a temp table, attempt a write inside a callback that then
	// errors, and confirm the write did not persist.
	_, err = conn.Exec("CREATE TEMP TABLE IF NOT EXISTS tx_rollback_probe (id INT)")
	require.NoError(t, err)

	wantErr := errors.New("forced failure")
	_, err = WithTx(conn, func(tx *sql.Tx) (int, error) {
		_, execErr := tx.Exec("INSERT INTO tx_rollback_probe (id) VALUES (1)")
		require.NoError(t, execErr)
		return 0, wantErr
	})
	require.ErrorIs(t, err, wantErr)

	var count int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM tx_rollback_probe").Scan(&count))
	require.Equal(t, 0, count, "rollback should have discarded the insert")
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./api/shared/db/... -v
```
Expected: FAIL — `WithTx` is undefined.

- [ ] **Step 4: Write the implementation**

```go
// api/shared/db/transaction.go
package db

import (
	"database/sql"
	"errors"
	"fmt"
)

// WithTx runs fn inside a transaction on conn, committing if fn returns
// a nil error and rolling back otherwise. The rollback error (if any) is
// joined with fn's original error rather than replacing it.
func WithTx[T any](conn *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := conn.Begin()
	if err != nil {
		return zero, fmt.Errorf("begin transaction: %w", err)
	}

	result, err := fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return zero, errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./api/shared/db/... -v
```
Expected: PASS for both `TestWithTx_CommitsOnSuccess` and `TestWithTx_RollsBackOnError`.

- [ ] **Step 6: Commit**

```bash
git add api/shared/db/
git commit -m "$(cat <<'EOF'
refactor(db): add shared WithTx transaction helper

Six repository files hand-roll Begin/Commit/Rollback boilerplate
with per-file error-wrapping. Adds a shared generic WithTx helper
modeled on user_pmcs/persistence's WithWriteTx pattern, without
that domain's retry/serialization-conflict logic since the target
call sites never had retry semantics to begin with. user_pmcs's own
retry.go is untouched — its richer retry behavior is a deliberate,
separate concern.
EOF
)"
```

```json:metadata
{"files": ["api/shared/db/transaction.go", "api/shared/db/transaction_test.go"], "verifyCommand": "go test ./api/shared/db/... -v", "acceptanceCriteria": ["commit path returns the callback's result with no error", "rollback path discards the write and returns the original error", "user_pmcs/persistence/retry.go unmodified"], "modelTier": "standard"}
```

---

### Task 13: Adopt WithTx in the six manual-transaction repository files

**Goal:** Replace manual `Begin()`/wrap-error/`defer Rollback()`/`Commit()` boilerplate in six files with calls to `db.WithTx` from Task 12: `api/pmcs_sbs_progress/repository_impl.go`, `api/shops/messages/repository_impl.go`, `api/shops/vehicles/repository_impl.go`, `api/user_general/repository_impl.go`, `api/user_pmcs/sync/repository_impl.go`, `api/user_saves/categories/repository_impl.go`.

**Files:**
- Modify: `api/pmcs_sbs_progress/repository_impl.go`
- Modify: `api/shops/messages/repository_impl.go`
- Modify: `api/shops/vehicles/repository_impl.go`
- Modify: `api/user_general/repository_impl.go`
- Modify: `api/user_pmcs/sync/repository_impl.go`
- Modify: `api/user_saves/categories/repository_impl.go`

**Acceptance Criteria:**
- [ ] Every manual `.Begin()`/`.BeginTx()` call in these six files is replaced with `db.WithTx(...)`.
- [ ] For each migrated method, an existing or new test forces an error mid-transaction and confirms no partial write persists.
- [ ] Original error-wrapping message text is preserved where existing tests assert on it — check each file's tests for message-string assertions before migrating (Step 1).

**Verify:** `go test -p 1 ./tests/pmcs_sbs_progress/... ./tests/shops/... ./tests/user_general/... ./tests/user_pmcs/... ./tests/user_saves/... -v` → all pass, including rollback-path assertions.

**Steps:**

- [ ] **Step 1: For each of the six files, read the current transaction code and check for message-string test assertions**

```bash
for f in api/pmcs_sbs_progress/repository_impl.go api/shops/messages/repository_impl.go api/shops/vehicles/repository_impl.go api/user_general/repository_impl.go api/user_pmcs/sync/repository_impl.go api/user_saves/categories/repository_impl.go; do
  echo "=== $f ==="
  grep -n "\.Begin()\|\.BeginTx(\|Rollback\|Commit" "$f"
done
grep -rn "failed to begin\|rollback\|Rollback" tests/ --include="*_test.go" | grep -iE "pmcs_sbs_progress|shops|user_general|user_pmcs|user_saves"
```
Note any test that asserts on the literal error-wrapping message text produced by the current manual code — the migrated version must preserve that exact text if any test depends on it.

- [ ] **Step 2: Migrate one file at a time — start with `api/pmcs_sbs_progress/repository_impl.go`**

Replace the manual pattern:
```go
tx, err := repo.db.Begin()
if err != nil {
	return nil, fmt.Errorf("failed to begin ... transaction: %w", err)
}
defer tx.Rollback()

// ... work using tx ...

if err := tx.Commit(); err != nil {
	return nil, fmt.Errorf("failed to commit ...: %w", err)
}
return result, nil
```
with:
```go
result, err := db.WithTx(repo.db, func(tx *sql.Tx) (*ResultType, error) {
	// ... same work using tx, returning (result, nil) or (nil, err) ...
})
if err != nil {
	return nil, fmt.Errorf("failed to begin ... transaction: %w", err)
}
return result, nil
```
(Preserve the exact original error message text found in Step 1 by keeping the same `fmt.Errorf` wrapping at the call site, since `db.WithTx` returns the raw underlying error and the caller re-wraps it — this keeps message-string test assertions intact without needing `db.WithTx` itself to know each call site's custom message.) Add the import `"miltechserver/api/shared/db"`.

- [ ] **Step 3: Build and test after each file**

```bash
go build ./...
go test -p 1 ./tests/pmcs_sbs_progress/... -v
```
Expected: pass, identical to pre-migration behavior.

- [ ] **Step 4: Add a forced-error/rollback test for this file's migrated method if none exists**

```go
// in the relevant tests/<domain>/*_test.go file
func TestRepository_MethodName_RollsBackOnError(t *testing.T) {
	// Arrange: set up a scenario where the callback's work will fail
	// partway through (e.g. a constraint violation on a subsequent
	// insert within the same transaction).
	// Act: call the migrated repository method.
	// Assert: the method returns the expected error, AND a follow-up
	// query confirms no partial write persisted from the failed
	// transaction.
}
```
Write this test using the actual repository method's real signature and a real failure scenario specific to that method — do not write a generic placeholder; inspect the method being migrated to construct a scenario that genuinely exercises its rollback path (e.g., a duplicate-key insert, a foreign-key violation, or an explicit second statement designed to fail).

- [ ] **Step 5: Repeat Steps 2–4 for the remaining five files**

`api/shops/messages/repository_impl.go`, `api/shops/vehicles/repository_impl.go`, `api/user_general/repository_impl.go`, `api/user_pmcs/sync/repository_impl.go`, `api/user_saves/categories/repository_impl.go` — same substitution pattern, same requirement to add a rollback test per migrated method if one doesn't already exist.

- [ ] **Step 6: Full test run across all six domains**

```bash
go test -p 1 ./tests/pmcs_sbs_progress/... ./tests/shops/... ./tests/user_general/... ./tests/user_pmcs/... ./tests/user_saves/... -v
```
Expected: all pass, including the new rollback tests.

- [ ] **Step 7: Commit, one file per commit**

```bash
git add api/pmcs_sbs_progress/repository_impl.go
git commit -m "refactor(pmcs_sbs_progress): adopt shared WithTx helper"

git add api/shops/messages/repository_impl.go
git commit -m "refactor(shops): adopt shared WithTx helper in messages"

git add api/shops/vehicles/repository_impl.go
git commit -m "refactor(shops): adopt shared WithTx helper in vehicles"

git add api/user_general/repository_impl.go
git commit -m "refactor(user_general): adopt shared WithTx helper"

git add api/user_pmcs/sync/repository_impl.go
git commit -m "refactor(user_pmcs): adopt shared WithTx helper in sync"

git add api/user_saves/categories/repository_impl.go
git commit -m "refactor(user_saves): adopt shared WithTx helper in categories"
```

```json:metadata
{"files": ["api/pmcs_sbs_progress/repository_impl.go", "api/shops/messages/repository_impl.go", "api/shops/vehicles/repository_impl.go", "api/user_general/repository_impl.go", "api/user_pmcs/sync/repository_impl.go", "api/user_saves/categories/repository_impl.go"], "verifyCommand": "go test -p 1 ./tests/pmcs_sbs_progress/... ./tests/shops/... ./tests/user_general/... ./tests/user_pmcs/... ./tests/user_saves/... -v", "acceptanceCriteria": ["all manual Begin/Commit/Rollback replaced with db.WithTx", "each migrated method has a passing rollback-on-error test", "original error-wrapping message text preserved where tests depend on it"], "modelTier": "standard"}
```

---

### Task 14: Split shops/aggregates/repository_impl.go

**Goal:** `api/shops/aggregates/repository_impl.go` (1233 lines) mixes 5 unrelated aggregation concerns plus generic null-scan helpers. Split into 4 new files by concern, plus move the generic scan helpers to `api/shops/shared/`. Pure reorganization — same package, same exported symbols, zero logic changes.

**Files:**
- Modify: `api/shops/aggregates/repository_impl.go` (reduced to whatever doesn't fit the four new files, or emptied if everything is accounted for)
- Create: `api/shops/aggregates/repository_notifications.go`
- Create: `api/shops/aggregates/repository_snapshot.go`
- Create: `api/shops/aggregates/repository_services.go`
- Create: `api/shops/aggregates/repository_pmcs_history.go`
- Create: `api/shops/shared/scan_helpers.go`

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds after the split.
- [ ] Every function moved retains its exact original body — `git diff` for each new file shows only a function relocation, never an edited line inside a function body.
- [ ] `git diff --stat` shows deletions from `repository_impl.go` matched by additions in the new files, with no net line-count change beyond package/import boilerplate.

**Verify:** `go test -p 1 ./tests/shops/... -v` → all pass, identical to pre-split baseline. `go build ./...` succeeds.

**Steps:**

- [ ] **Step 1: Re-confirm current function groupings, since the audit's line numbers may have shifted**

```bash
grep -n "^func " api/shops/aggregates/repository_impl.go
```
Group the output into: (a) vehicle notification aggregation — `GetVehicleByIDForMember`, `GetVehicleNotificationsWithItems`, `buildNotificationsQuery`, `scanVehicleNotification`, `getVehicleNotifications`, `getItemsByNotificationIDs`; (b) recent-changes/services aggregation — `buildRecentChangesQuery`, `GetVehicleRecentChanges`, `buildServiceQuery`, `scanEquipmentService`, `GetVehicleServices`; (c) shop snapshot/bootstrap aggregation — `GetShopSnapshot`, `GetBootstrap`, `getBootstrapEquipment`, `getShopSnapshotSummary`, `getShopSnapshotVehicles`, `getShopNotificationsWithItems`, `getShopSnapshotNotifications`, `getShopSnapshotMessages`, `getShopSnapshotServices`, `getShopSnapshotRecentChanges`; (d) PMCS history — `GetEquipmentPmcsHistory`; (e) generic scan/null-conversion helpers — `timePtr`, `nullTimePtr`, `nullStringPtr`, `nullBoolPtr`, `nullInt32Ptr`, `errorsIsNoRows`, `placeholders`, `scanNotificationChange`; plus `GetListsWithItems`, which the audit didn't cleanly bucket — inspect it in this step and assign it to whichever new file matches its actual concern (likely `repository_snapshot.go`, since list aggregation is snapshot-adjacent, but confirm by reading the function).

- [ ] **Step 2: Create `api/shops/shared/scan_helpers.go` first, since the other new files will depend on it**

Move the generic helpers (`timePtr`, `nullTimePtr`, `nullStringPtr`, `nullBoolPtr`, `nullInt32Ptr`, `errorsIsNoRows`, `placeholders`) verbatim into this new file in `api/shops/shared/`, changing only the package declaration. If any of these are unexported (lowercase) and therefore invisible outside `aggregates` today, export them (capitalize) so `repository_impl.go`'s remaining callers — and the split-out files — can reach them from the new location. Update every call site across `api/shops/aggregates/*.go` to reference `shared.TimePtr` etc.

- [ ] **Step 3: Create the four concern-based files, moving function bodies verbatim**

For each of the four groupings from Step 1, create the corresponding new file with `package aggregates` and the exact, unmodified function bodies cut from `repository_impl.go`. Copy the necessary imports into each new file (only what that file's functions actually use — run `goimports` after each move to reconcile). `scanNotificationChange` goes wherever its caller (`getVehicleNotifications` or similar, confirmed in Step 1) actually lives, not automatically into the shared package, since the audit description suggests it's aggregation-specific rather than fully generic — verify this in Step 1's inspection.

- [ ] **Step 4: Confirm `repository_impl.go` after the split**

```bash
grep -n "^func " api/shops/aggregates/repository_impl.go
```
Expected: either empty (delete the file entirely if nothing remains, keeping only a package-level doc comment if useful) or containing only whatever didn't fit the four groupings — if something remains, that's a signal Step 1's grouping was incomplete and needs a fifth file or reassignment, not a leftover dumping ground.

- [ ] **Step 5: Build and verify no logic changed**

```bash
go build ./...
git diff --stat
```
Manually review the diff: every hunk should be a deletion in one file matched by an identical addition in another, never a modified line within a function body. If `goimports`/`gofmt` produced incidental whitespace differences, that's acceptable; any semantic difference is not.

- [ ] **Step 6: Run the full shops test suite**

```bash
go test -p 1 ./tests/shops/... -v
```
Expected: all pass, identical to the pre-split baseline (capture a baseline before Step 1 if not already done via Task 9's fixtures).

- [ ] **Step 7: Commit**

```bash
git add api/shops/aggregates/ api/shops/shared/
git commit -m "$(cat <<'EOF'
refactor(shops): split aggregates/repository_impl.go by concern

The 1233-line file mixed vehicle notification aggregation, recent
changes/service history, shop snapshot/bootstrap, and PMCS history
queries in one file, plus generic null-scan helpers unrelated to any
one concern. Splits into four concern-based files and moves the
generic helpers to shops/shared. Pure reorganization — no function
body was modified, only relocated.
EOF
)"
```

```json:metadata
{"files": ["api/shops/aggregates/repository_impl.go", "api/shops/aggregates/repository_notifications.go", "api/shops/aggregates/repository_snapshot.go", "api/shops/aggregates/repository_services.go", "api/shops/aggregates/repository_pmcs_history.go", "api/shops/shared/scan_helpers.go"], "verifyCommand": "go build ./... && go test -p 1 ./tests/shops/... -v", "acceptanceCriteria": ["go build succeeds", "every function body identical pre/post split, only location changed", "full shops test suite passes unchanged"], "modelTier": "standard"}
```

---

### Task 15: Split pmcs_sbs_progress/service_impl.go

**Goal:** `api/pmcs_sbs_progress/service_impl.go` (652 lines) mixes ~11 public service methods, ~11 validation helpers, and ~5 DTO-mapping helpers in one file, despite the domain already having `types.go` as a precedent for splitting concerns out of the main file. Extract `validation.go` and `mapping.go`.

**Files:**
- Modify: `api/pmcs_sbs_progress/service_impl.go` (reduced to the ~11 public service methods and `hasAuthenticatedUser`/`normalizeFaultStatus`, if those two are genuinely service-level rather than mapping — confirm in Step 1)
- Create: `api/pmcs_sbs_progress/validation.go`
- Create: `api/pmcs_sbs_progress/mapping.go`

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds.
- [ ] Every moved function retains its exact original body.
- [ ] `go test -p 1 ./tests/pmcs_sbs_progress/... -v` passes identically to the pre-split baseline.

**Verify:** `go test -p 1 ./tests/pmcs_sbs_progress/... -v` → all pass, identical output.

**Steps:**

- [ ] **Step 1: Re-confirm current function groupings**

```bash
grep -n "^func " api/pmcs_sbs_progress/service_impl.go
```
Group into: (a) public service methods — `EnsureInspection`, `GetInspection`, `ListInspections`, `DeleteInspection`, `UpsertFault`, `DeleteFault`, `DeleteFaults`, `CreateComment`, `UpdateComment`, `DeleteComment`; (b) validation helpers — `validateInspectionRequest`, `validateFaultRequest`, `validateDeleteFaultRequest`, `validateBulkDeleteFaultRequest`, `validateEquipmentID`, `validatePmcsID`, `validateGuideManual`, `normalizeInspectionSource`, `normalizeGuideInspectionSource`, `normalizeCustomInspectionSource`, `validateRequiredShortField`, `validateOptionalShortField`, `validateShortField`, `validateNotes`, `validateCommentText`; (c) DTO-mapping helpers — `mapFault`, `mapInspection`, `mapComment`, `hasAuthenticatedUser`, `normalizeFaultStatus`. Confirm `hasAuthenticatedUser` and `normalizeFaultStatus` belong in `mapping.go` rather than being genuinely service-level by reading their bodies — if either is a pure predicate/normalizer with no side effects, it belongs in `mapping.go`; if either directly drives control flow in a public method (e.g., an early-return guard), leave it in `service_impl.go`.

- [ ] **Step 2: Create `validation.go`, moving the ~11 validation functions verbatim**

Same package (`pmcs_sbs_progress`), exact function bodies, only imports adjusted via `goimports`.

- [ ] **Step 3: Create `mapping.go`, moving the ~5 DTO-mapping functions verbatim**

Same treatment.

- [ ] **Step 4: Confirm `service_impl.go` now contains only public service methods**

```bash
grep -n "^func " api/pmcs_sbs_progress/service_impl.go
```
Expected: only the ~11 public methods (plus whichever of `hasAuthenticatedUser`/`normalizeFaultStatus` Step 1 determined belongs there).

- [ ] **Step 5: Build and verify no logic changed**

```bash
go build ./...
git diff --stat
```
Same manual-review standard as Task 14 Step 5 — relocations only, no edits.

- [ ] **Step 6: Run the full pmcs_sbs_progress test suite**

```bash
go test -p 1 ./tests/pmcs_sbs_progress/... -v
```
Expected: all pass, identical to pre-split baseline.

- [ ] **Step 7: Commit**

```bash
git add api/pmcs_sbs_progress/
git commit -m "$(cat <<'EOF'
refactor(pmcs_sbs_progress): split service_impl.go by concern

The 652-line file mixed public service methods, validation helpers,
and DTO-mapping helpers with no file separation, despite the domain
already having types.go as precedent for splitting concerns out.
Extracts validation.go and mapping.go. Pure reorganization.
EOF
)"
```

```json:metadata
{"files": ["api/pmcs_sbs_progress/service_impl.go", "api/pmcs_sbs_progress/validation.go", "api/pmcs_sbs_progress/mapping.go"], "verifyCommand": "go build ./... && go test -p 1 ./tests/pmcs_sbs_progress/... -v", "acceptanceCriteria": ["go build succeeds", "every function body identical pre/post split", "full pmcs_sbs_progress test suite passes unchanged"], "modelTier": "standard"}
```

---

### Task 16: Split user_pmcs/community/repository_impl.go

**Goal:** `api/user_pmcs/community/repository_impl.go` (909 lines) mixes voting logic and browse/discovery logic. Split into two files along that seam.

**Files:**
- Modify: `api/user_pmcs/community/repository_impl.go` (reduced to release/retire lifecycle methods, which don't cleanly belong to either voting or browse — confirm in Step 1 whether these need a third file or stay in the original)
- Create: `api/user_pmcs/community/repository_voting.go`
- Create: `api/user_pmcs/community/repository_browse.go`

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds.
- [ ] Every moved function retains its exact original body.
- [ ] `go test -p 1 ./tests/user_pmcs/... -v` and any colocated `api/user_pmcs/community/*_test.go` pass identically to the pre-split baseline.

**Verify:** `go test -p 1 ./tests/user_pmcs/... ./api/user_pmcs/community/... -v` → all pass, identical output.

**Steps:**

- [ ] **Step 1: Re-confirm current function groupings**

```bash
grep -n "^func " api/user_pmcs/community/repository_impl.go
```
Group into: (a) release/retire lifecycle — `Release`, `Retire`; (b) voting — `PutVote`, `DeleteVote`, `isCommunityVoteDirection`, `requireCommunityVoteAccount`, `lockCommunityVoteSource`, `loadCommunityVoteMutation`, `communityVoteNotFoundError`; (c) browse/discovery — `Browse`, `containsModelPattern`, `communityBrowseQuery`, `communityBrowseSort`, `loadSummaryModels`, `GetCurrentRelease`, `creatorDisplayName`, `communityNotFoundError`, `advanceChecklist`, `loadAggregate`. Decide where (a) belongs: if `Release`/`Retire` are called by neither voting nor browse code paths and are genuinely a third, separate lifecycle concern, keep them in `repository_impl.go` (renamed conceptually to "lifecycle" but the file itself doesn't need renaming) rather than forcing them into one of the two new files.

- [ ] **Step 2: Create `repository_voting.go`, moving the voting functions verbatim**

Same package, exact bodies, imports reconciled via `goimports`.

- [ ] **Step 3: Create `repository_browse.go`, moving the browse/discovery functions verbatim**

Same treatment. Note `GetCurrentRelease` and `advanceChecklist`/`loadAggregate` — confirm in Step 1's reading whether these are genuinely browse-related or actually belong with the lifecycle group; the audit bucketed them under "browse/discovery" but their names suggest possible overlap with release/retire — verify by reading what each function actually does before finalizing placement.

- [ ] **Step 4: Confirm `repository_impl.go` contains only the lifecycle functions (or is empty)**

```bash
grep -n "^func " api/user_pmcs/community/repository_impl.go
```

- [ ] **Step 5: Build and verify no logic changed**

```bash
go build ./...
git diff --stat
```
Same manual-review standard as Task 14 Step 5.

- [ ] **Step 6: Run the full user_pmcs/community test suite**

```bash
go test -p 1 ./tests/user_pmcs/... -v
go test ./api/user_pmcs/community/... -v
```
Expected: all pass, identical to pre-split baseline.

- [ ] **Step 7: Commit**

```bash
git add api/user_pmcs/community/
git commit -m "$(cat <<'EOF'
refactor(user_pmcs): split community/repository_impl.go by concern

The 909-line file mixed voting logic and browse/discovery logic.
Splits into repository_voting.go and repository_browse.go, leaving
release/retire lifecycle methods in the original file since they
serve neither concern. Pure reorganization.
EOF
)"
```

```json:metadata
{"files": ["api/user_pmcs/community/repository_impl.go", "api/user_pmcs/community/repository_voting.go", "api/user_pmcs/community/repository_browse.go"], "verifyCommand": "go build ./... && go test -p 1 ./tests/user_pmcs/... ./api/user_pmcs/community/... -v", "acceptanceCriteria": ["go build succeeds", "every function body identical pre/post split", "full user_pmcs/community test coverage passes unchanged"], "modelTier": "standard"}
```

---

### Task 17: Thread context.Context through remaining service layer

**Goal:** Only 7 of 54 `service_impl.go` files accept `ctx context.Context` in their public methods. Add it to the other 47, threading from the handler's `c.Request.Context()` down to whichever repository calls already accept `ctx` — this task does not extend context-acceptance into repository methods that don't have it yet (that's explicitly out of scope for this entire plan).

Given the size (47 files), this task is itself broken into sub-steps by domain group rather than one file at a time, but remains a single commit-per-domain-group task since the change is mechanical and uniform in shape across files.

**Files:**
- Modify: 47 `service_impl.go` files and their corresponding `service.go` interface files (interface signatures must match implementations)
- Modify: corresponding handler/route files that call these service methods, to pass `c.Request.Context()`

**Acceptance Criteria:**
- [ ] `grep -rL "ctx context.Context" api/*/service_impl.go api/*/*/service_impl.go` (adjusted glob to catch nested domains) shows zero files remaining without `ctx` as the first parameter of their public methods, except `user_pmcs`'s existing 7 (now part of the "already had it" set, unaffected).
- [ ] For each modified `service_impl.go`, its paired `service.go` interface is updated to match.
- [ ] Every handler call site passes `c.Request.Context()` rather than a bare `context.Background()` or `context.TODO()`.
- [ ] At least one new test per touched domain asserts that a canceled context aborts an in-flight repository call where the repository already accepts `ctx` — this is a genuinely new regression check, not previously possible.

**Verify:** `go build ./...` (signature-only change, so this catches every missed call site) `&& go test -p 1 ./...` (full suite).

**Steps:**

- [ ] **Step 1: Enumerate the exact 47 files and confirm the count is still accurate**

```bash
grep -rL "ctx context.Context" $(find api -name "service_impl.go") | wc -l
grep -rL "ctx context.Context" $(find api -name "service_impl.go")
```
Cross-check against the 7 known-already-threaded files (`shops/core`, `shops/aggregates`, `user_pmcs/subscriptions`, `library`, `library/ps_mag`, `library/pmcs_sbs`, `item_query/detailed`) to confirm the remaining list.

- [ ] **Step 2: Pick the first domain group and identify which of its repository calls already accept `ctx`**

Start with domains whose repository layer the audit confirmed already accepts `ctx` where relevant (e.g., check each target file's paired `repository_impl.go` for `ctx context.Context` in method signatures) — these are the domains where threading context all the way through actually changes behavior-observability (Step 5's new test), versus domains where the repository doesn't accept `ctx` at all yet, where this task only changes the service-layer signature with the parameter unused below that point (still correct per this task's scope, just without a new cancellation test possible there).

```bash
grep -n "ctx context.Context" api/<domain>/repository_impl.go
```

- [ ] **Step 3: Add `ctx context.Context` as the first parameter to every public method in the domain's `service.go` interface and `service_impl.go` implementation**

For a method like:
```go
func (s *ServiceImpl) GetInspection(user *bootstrap.User, equipmentID string) (*Inspection, error) {
	return s.repo.GetInspection(equipmentID)
}
```
change to:
```go
func (s *ServiceImpl) GetInspection(ctx context.Context, user *bootstrap.User, equipmentID string) (*Inspection, error) {
	return s.repo.GetInspection(ctx, equipmentID)
}
```
if the repository method already accepts `ctx` (add `ctx` to that call), or:
```go
func (s *ServiceImpl) GetInspection(ctx context.Context, user *bootstrap.User, equipmentID string) (*Inspection, error) {
	return s.repo.GetInspection(equipmentID)
}
```
if it doesn't (accept and thread `ctx` no further than the repository boundary — do not modify the repository signature, per this task's explicit scope limit). Apply the identical parameter addition to the interface declaration in `service.go`.

- [ ] **Step 4: Update the handler/route call site to pass `c.Request.Context()`**

```go
inspection, err := service.GetInspection(c.Request.Context(), user, equipmentID)
```

- [ ] **Step 5: Add a cancellation test where the repository underneath already accepts `ctx`**

```go
// in the relevant tests/<domain>/*_test.go or colocated service_impl_test.go
func TestService_MethodName_AbortsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately, before the call

	_, err := service.MethodName(ctx, testUser, testArgs)

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled"))
}
```
Write this using the real method's real signature and real test fixtures for that domain — do not write a generic placeholder. Skip this specific test (but still complete Steps 3–4) for domains whose repository layer doesn't accept `ctx` yet, since there's nothing observable to cancel in that case.

- [ ] **Step 6: Build after each domain group to catch every call site immediately**

```bash
go build ./...
```
Expected: compile errors will surface every caller of a changed method signature that wasn't yet updated — fix all of them before moving to the next domain group, since a partial signature change across a call chain won't compile.

- [ ] **Step 7: Test the domain group**

```bash
go test -p 1 ./tests/<domain>/... ./api/<domain>/... -v
```

- [ ] **Step 8: Repeat Steps 2–7 for each remaining domain group until all 47 files are threaded**

Suggested groupings to keep commits reviewable: (1) `shops` subpackages not already threaded, (2) `user_pmcs` subpackages not already threaded (`owned`, `community`, `sync` — note `subscriptions` is already done), (3) `item_lookup` subpackages, (4) `pmcs_sbs_progress`, (5) `sb_700_20`, (6) remaining smaller domains (`docs_equipment`, `eic`, `tmde`, `material_images`, `user_saves`, `user_suggestions`, `user_vehicles`, `equipment_services`, `item_comments`, `quick_lists`, `pol_products`, `user_general`, `analytics` if it has a service layer with an HTTP-facing caller).

- [ ] **Step 9: Full repo build and test after all groups are done**

```bash
go build ./...
go vet ./...
go test -p 1 ./...
```

- [ ] **Step 10: Commit per domain group (7-9 commits total for this task)**

```bash
git add api/shops/
git commit -m "refactor(shops): thread context.Context through remaining service methods"
# ... repeat per group
```

```json:metadata
{"files": [], "verifyCommand": "go build ./... && go vet ./... && go test -p 1 ./...", "acceptanceCriteria": ["all 47 service_impl.go files (minus the 7 already done) now accept ctx as first parameter", "paired service.go interfaces match", "handler call sites pass c.Request.Context()", "at least one new cancellation test per domain where the repository layer supports it"], "modelTier": "standard"}
```

---

### Task 18: Extract shared test harness (testutil package)

**Goal:** 12 `tests/<domain>/main_test.go` files hardcode the identical DSN string. Each domain also reimplements its own fake-auth test middleware (e.g. `tests/shops/helpers_test.go`'s `testUserMiddleware()`). Extract a shared `testutil` package with the DSN constant, a shared fake-auth middleware, and a shared router-builder helper.

**Files:**
- Create: `tests/testutil/testutil.go`
- Create: `tests/testutil/testutil_test.go`
- Modify: 12 `tests/<domain>/main_test.go` files
- Modify: `tests/shops/helpers_test.go` and any other domain-specific test-middleware files found in Step 1

**Acceptance Criteria:**
- [ ] The DSN string appears exactly once in the repo, in `tests/testutil/testutil.go`, referenced by all 12 domains.
- [ ] Exactly one fake-auth test middleware implementation exists, referenced by every domain that previously had its own.
- [ ] `go test -p 1 ./...` passes identically to the pre-task baseline.

**Verify:** `go test -p 1 ./...` (full suite) → all pass, identical to baseline.

**Steps:**

- [ ] **Step 1: Enumerate every domain with the duplicated DSN and every domain with its own fake-auth middleware**

```bash
grep -rln "postgres://postgres:potato123@192.168.20.70" tests/ --include="*.go"
grep -rln "testUserMiddleware\|X-User-ID\|X-User-Name\|X-User-Email" tests/ --include="*.go"
```
Confirm the count is still 12 for the DSN (may have changed if other tasks touched test files) and identify every distinct fake-auth middleware implementation, not just the `tests/shops/helpers_test.go` one the audit named — there may be others with different header names or structures that need reconciling into one shared shape.

- [ ] **Step 2: Write the failing test for the shared package's exports**

```go
// tests/testutil/testutil_test.go
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTestDSN_IsSet(t *testing.T) {
	require.NotEmpty(t, TestDSN)
	require.Contains(t, TestDSN, "192.168.20.70")
}

func TestFakeAuthMiddleware_SetsUserFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FakeAuthMiddleware())
	router.GET("/probe", func(c *gin.Context) {
		user, exists := c.Get("user")
		require.True(t, exists)
		c.JSON(http.StatusOK, user)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-User-ID", "test-uid")
	req.Header.Set("X-User-Name", "Test User")
	req.Header.Set("X-User-Email", "test@example.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
```
Adjust the exact context key (`"user"`) and header names to match whatever Step 1 found as the real, currently-used convention in `tests/shops/helpers_test.go` — do not invent a new contract, replicate the existing one exactly so no domain's test behavior changes.

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./tests/testutil/... -v
```
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 4: Write `tests/testutil/testutil.go`**

```go
package testutil

import (
	"miltechserver/bootstrap"

	"github.com/gin-gonic/gin"
)

// TestDSN is the connection string for the shared integration test
// database. This is the confirmed-correct, intentional test host —
// do not change it without updating every domain that imports it.
const TestDSN = "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable"

// FakeAuthMiddleware reads X-User-ID/X-User-Name/X-User-Email headers
// and sets a *bootstrap.User on the gin context, replicating the shape
// of real Firebase-authenticated requests for integration tests without
// requiring a real token.
func FakeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// (Exact body copied from the real implementation found in
		// Step 1 — same header names, same context key, same User
		// struct construction — so every migrated domain's test
		// behavior is unchanged.)
		c.Next()
	}
}
```
Fill in the actual body from Step 1's findings — this is a promotion of existing logic, not new logic, so the implementation must be a faithful copy, not a reinterpretation.

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./tests/testutil/... -v
```
Expected: PASS.

- [ ] **Step 6: Migrate the 12 domains' `main_test.go` DSN references, one domain per commit**

For each domain:
```go
import "miltechserver/tests/testutil"

// ...
testDB, err = sql.Open("postgres", testutil.TestDSN)
```

- [ ] **Step 7: Migrate domains with their own fake-auth middleware to use `testutil.FakeAuthMiddleware()`**

Starting with `tests/shops/helpers_test.go` — replace the local `testUserMiddleware()` definition with a call to `testutil.FakeAuthMiddleware()`, and remove the now-dead local definition.

- [ ] **Step 8: Build and test after each domain migration**

```bash
go build ./...
go test -p 1 ./tests/<domain>/... -v
```

- [ ] **Step 9: Full suite run**

```bash
go test -p 1 ./...
```
Expected: all pass, identical to baseline.

- [ ] **Step 10: Commit per domain (or a small number of batched commits if changes are trivially identical across domains)**

```bash
git add tests/testutil/
git commit -m "test(testutil): add shared DSN constant and fake-auth middleware"

git add tests/shops/
git commit -m "test(shops): adopt shared testutil DSN and fake-auth middleware"
# ... repeat per remaining domain
```

```json:metadata
{"files": ["tests/testutil/testutil.go", "tests/testutil/testutil_test.go"], "verifyCommand": "go test -p 1 ./...", "acceptanceCriteria": ["DSN string appears exactly once in the repo", "exactly one fake-auth middleware implementation exists", "full test suite passes identically to baseline"], "modelTier": "mechanical"}
```

---

### Task 19: Remove vestigial shops_route.go indirection

**Goal:** `api/route/shops_route.go`'s `NewShopsRouter()` wraps `shops.RegisterRoutes()` with no added behavior — every other domain calls `RegisterRoutes` directly from `route.go`. Inline the call and delete the wrapper file.

**Files:**
- Modify: `api/route/route.go`
- Delete: `api/route/shops_route.go`

**Acceptance Criteria:**
- [ ] `api/route/shops_route.go` no longer exists.
- [ ] `route.go` calls `shops.RegisterRoutes(shops.Dependencies{...}, group)` directly, matching every other domain's pattern.
- [ ] Gin's registered route list (`router.Routes()`) is byte-identical before and after — same paths, same HTTP methods, same order.

**Verify:** `go build ./... && go test -p 1 ./tests/shops/... -v`; route-list diff (Step 1 vs. Step 4) is empty.

**Steps:**

- [ ] **Step 1: Capture the current registered route list**

```bash
# Add a temporary debug print of router.Routes() in a throwaway test,
# or use an existing debug endpoint if api/route/debug_route.go exposes one.
grep -n "debug" api/route/debug_route.go
```
If `debug_route.go` already exposes a routes-listing endpoint, use it directly (start the server locally and curl it). Otherwise, write a throwaway test in `api/route` that calls `Setup()` against a test router and dumps `router.Routes()` to a file — this is a temporary diagnostic, not part of the final diff.

```bash
go run . &  # or the equivalent local-run command found in Task usage
# then, in another terminal, hit the debug routes endpoint and save output
```

- [ ] **Step 2: Find the current call site in route.go**

```bash
grep -n "NewShopsRouter\|shops\." api/route/route.go
```

- [ ] **Step 3: Inline the call**

Replace:
```go
NewShopsRouter(db, blobClient, env, v1Route)
```
with:
```go
shops.RegisterRoutes(shops.Dependencies{
	DB:         db,
	BlobClient: blobClient,
	Env:        env,
}, v1Route)
```
(Match the exact `Dependencies` field values currently passed inside `shops_route.go`'s `NewShopsRouter` — this is a direct copy of that function's body into the call site, not a reinterpretation.) Add the `shops` import to `route.go` if not already present; remove any now-unused import that only existed for `NewShopsRouter`.

- [ ] **Step 4: Delete `shops_route.go` and capture the route list again**

```bash
git rm api/route/shops_route.go
go build ./...
```
Re-run the same route-listing diagnostic from Step 1 and diff against the earlier capture.

- [ ] **Step 5: Build and test**

```bash
go build ./...
go test -p 1 ./tests/shops/... -v
```
Expected: build succeeds, shops tests pass unchanged.

- [ ] **Step 6: Commit**

```bash
git add api/route/
git commit -m "$(cat <<'EOF'
refactor(route): inline shops route registration

shops_route.go's NewShopsRouter() wrapped shops.RegisterRoutes()
with no added behavior, unlike every other domain which calls
RegisterRoutes directly from route.go. Inlines the call and removes
the vestigial indirection.
EOF
)"
```

```json:metadata
{"files": ["api/route/route.go", "api/route/shops_route.go"], "verifyCommand": "go build ./... && go test -p 1 ./tests/shops/... -v", "acceptanceCriteria": ["shops_route.go deleted", "route.go calls shops.RegisterRoutes directly", "registered route list unchanged before and after"], "modelTier": "mechanical"}
```

---

## Self-Review

**1. Spec coverage:** All 10 passes from the spec map to tasks: Pass 1 → Task 1. Pass 2 → Task 2. Pass 3 → Task 3. Pass 4 → Tasks 5–6. Pass 5 → Tasks 4, 7–11 (split per-domain as decided). Pass 6 → Tasks 12–13 (corrected scope: 6 files, not 18, with the transaction helper already existing rather than needing extraction). Pass 7 → Tasks 14–16 (`user_pmcs/owned` split explicitly deferred past this plan per the spec's own sequencing note — it's not a task here, since the spec says to reassess after Pass 6/Task 13 lands, and that reassessment is a future planning exercise, not a task in this document). Pass 8 → Task 17. Pass 9 → Task 18. Pass 10 → Task 19. All six out-of-scope items from the spec are restated in Global Constraints and never appear as tasks.

**2. Placeholder scan:** Checked every task for "TBD"/vague instructions. Tasks 6, 9, 10, 11, 14, 15, 16, 17, 18, 19 include explicit "re-confirm current state, since line numbers may have shifted" steps rather than hardcoding possibly-stale audit line numbers as fact — this is a deliberate design choice given several numbers were already found wrong during planning (Task 12/13's transaction-file count, corrected from 18 to 6), not a placeholder. Every code-writing step contains real Go code, not descriptions of code. Test steps contain real test bodies, adjusted-in-place where the exact underlying contract (e.g., testutil's header names, pagination's exact function name) must be confirmed from the existing file before being copied verbatim — flagged explicitly at each such point rather than invented.

**3. Type consistency:** `response.OK`/`response.Error` signatures introduced in Task 4 are used identically in Tasks 7–11. `db.WithTx[T any]` introduced in Task 12 is used identically in Task 13. `pagination.Parse` introduced in Task 5 is used identically in Task 6. `testutil.TestDSN`/`testutil.FakeAuthMiddleware()` introduced in Task 18 are self-contained to that task. No later task references a function name that differs from its introducing task.

**Known deviation from the spec's literal Pass 6 description:** the spec said to "extract" `user_pmcs/owned`'s transaction helper; verification during planning found the real reusable helper is `persistence.WithWriteTx` in `user_pmcs/persistence/retry.go` (a different file than the spec named), and it already exists rather than needing extraction — Task 12 promotes it (in simplified form, without retry semantics the target files never had) rather than extracting it from scratch. This is called out in the plan header's "Corrections made during planning" section so it isn't mistaken for silent scope drift.
