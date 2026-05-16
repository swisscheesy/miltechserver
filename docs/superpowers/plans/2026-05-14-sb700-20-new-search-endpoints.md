# SB 700-20 New Search Endpoints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 7 new search endpoints to the SB 700-20 API, allowing searches by `new_lin` (app_e, app_g), `lin_zmm_sublin`/`lin_zmmsublin` (app_h1, app_h2), and `ric` (chp_4, chp_6, chp_8), plus a user-facing API reference doc.

**Architecture:** Mirror the existing sb_700_20 pattern exactly — new method signatures added to both interfaces, repository methods query the target column with `WHERE col.EQ(String(param))`, service methods thin-delegate with normalization, and handler functions are added to the existing split files. All 7 endpoints return slices since none of the search fields are sole primary keys.

**Tech Stack:** Go, Gin, go-jet/jet v2 (Postgres), testify/require

---

## Files Changed

| File | Change |
|---|---|
| `api/sb_700_20/repository.go` | Add 7 method signatures |
| `api/sb_700_20/service.go` | Add 7 method signatures |
| `api/sb_700_20/repository_apps.go` | Add 4 repository implementations (app_e, app_g, app_h1, app_h2) |
| `api/sb_700_20/repository_chps.go` | Add 3 repository implementations (chp_4, chp_6, chp_8) |
| `api/sb_700_20/service_impl.go` | Add 7 thin delegation methods |
| `api/sb_700_20/handlers_apps.go` | Add 4 handler functions |
| `api/sb_700_20/handlers_chps.go` | Add 3 handler functions |
| `api/sb_700_20/route.go` | Register 7 new routes |
| `tests/sb_700_20/helpers_test.go` | Add 4 sample-fetch helpers |
| `tests/sb_700_20/repository_test.go` | Add 7 blank-param validation cases |
| `tests/sb_700_20/handlers_test.go` | Add 7 new blank-param entries + 7 new test functions |
| `docs/sb700-20-new-search-endpoints.md` | New user-facing API reference (no code, JSON examples only) |

---

## TDD Note

Go is compiled. The `repository` and `service` structs must satisfy their interfaces before the test package will build. The workflow is therefore:

1. Add interface signatures (Tasks 1–2)
2. Add skeleton impls that return `ErrEmptyParam` for blank and `ErrNotFound` otherwise (Tasks 3–4 skeleton step)
3. Write tests — package now compiles (Tasks 5–7)
4. Run tests — validation cases pass (skeleton handles blank correctly); integration cases are skipped if no test DB
5. Replace skeletons with real implementations (Tasks 3–4 impl step)
6. Run tests — all pass

---

### Task 1: Extend Repository Interface

**Files:**
- Modify: `api/sb_700_20/repository.go`

- [ ] **Step 1: Add 7 method signatures to the `Repository` interface**

Open `api/sb_700_20/repository.go`. Add after the existing `GetChp8ByLIN` / `GetChp8Paginated` lines:

```go
GetAppEByNewLIN(newLin string) ([]model.Sb70020AppE, error)
GetAppGByNewLIN(newLin string) ([]model.Sb70020AppG, error)
GetAppH1BySubLIN(sublin string) ([]model.Sb70020AppH1, error)
GetAppH2BySubLIN(sublin string) ([]model.Sb70020AppH2, error)
GetChp4ByRIC(ric string) ([]model.Sb70020Chp4, error)
GetChp6ByRIC(ric string) ([]model.Sb70020Chp6, error)
GetChp8ByRIC(ric string) ([]model.Sb70020Chp8, error)
```

The full interface block should now have 39 method signatures (32 existing + 7 new).

- [ ] **Step 2: Verify the package no longer compiles (expected)**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/sb_700_20/...
```

Expected: compile error — `repository` does not implement the 7 new methods. This confirms the interface change is wired correctly.

---

### Task 2: Extend Service Interface

**Files:**
- Modify: `api/sb_700_20/service.go`

- [ ] **Step 1: Add 7 method signatures to the `Service` interface**

Open `api/sb_700_20/service.go`. Add after the existing `GetChp8ByLIN` / `GetChp8Paginated` lines:

```go
GetAppEByNewLIN(newLin string) ([]model.Sb70020AppE, error)
GetAppGByNewLIN(newLin string) ([]model.Sb70020AppG, error)
GetAppH1BySubLIN(sublin string) ([]model.Sb70020AppH1, error)
GetAppH2BySubLIN(sublin string) ([]model.Sb70020AppH2, error)
GetChp4ByRIC(ric string) ([]model.Sb70020Chp4, error)
GetChp6ByRIC(ric string) ([]model.Sb70020Chp6, error)
GetChp8ByRIC(ric string) ([]model.Sb70020Chp8, error)
```

---

### Task 3: Add Repository Implementations — Apps

**Files:**
- Modify: `api/sb_700_20/repository_apps.go`

- [ ] **Step 1: Append 4 methods to `repository_apps.go`**

Add at the end of the file:

```go
func (r *repository) GetAppEByNewLIN(newLin string) ([]model.Sb70020AppE, error) {
	newLin = strings.TrimSpace(newLin)
	if newLin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppE
	stmt := SELECT(table.Sb70020AppE.AllColumns).
		FROM(table.Sb70020AppE).
		WHERE(table.Sb70020AppE.NewLin.EQ(String(newLin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppGByNewLIN(newLin string) ([]model.Sb70020AppG, error) {
	newLin = strings.TrimSpace(newLin)
	if newLin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppG
	stmt := SELECT(table.Sb70020AppG.AllColumns).
		FROM(table.Sb70020AppG).
		WHERE(table.Sb70020AppG.NewLin.EQ(String(newLin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppH1BySubLIN(sublin string) ([]model.Sb70020AppH1, error) {
	sublin = strings.TrimSpace(sublin)
	if sublin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppH1
	stmt := SELECT(table.Sb70020AppH1.AllColumns).
		FROM(table.Sb70020AppH1).
		WHERE(table.Sb70020AppH1.LinZmmSublin.EQ(String(sublin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppH2BySubLIN(sublin string) ([]model.Sb70020AppH2, error) {
	sublin = strings.TrimSpace(sublin)
	if sublin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppH2
	stmt := SELECT(table.Sb70020AppH2.AllColumns).
		FROM(table.Sb70020AppH2).
		WHERE(table.Sb70020AppH2.LinZmmsublin.EQ(String(sublin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}
```

Note: `table.Sb70020AppH1.LinZmmSublin` (capital S before "ublin") vs `table.Sb70020AppH2.LinZmmsublin` (lowercase s — different generated column name from a different table).

---

### Task 4: Add Repository Implementations — Chps

**Files:**
- Modify: `api/sb_700_20/repository_chps.go`

- [ ] **Step 1: Append 3 methods to `repository_chps.go`**

Add at the end of the file:

```go
func (r *repository) GetChp4ByRIC(ric string) ([]model.Sb70020Chp4, error) {
	ric = strings.TrimSpace(ric)
	if ric == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020Chp4
	stmt := SELECT(table.Sb70020Chp4.AllColumns).
		FROM(table.Sb70020Chp4).
		WHERE(table.Sb70020Chp4.Ric.EQ(String(ric)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetChp6ByRIC(ric string) ([]model.Sb70020Chp6, error) {
	ric = strings.TrimSpace(ric)
	if ric == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020Chp6
	stmt := SELECT(table.Sb70020Chp6.AllColumns).
		FROM(table.Sb70020Chp6).
		WHERE(table.Sb70020Chp6.Ric.EQ(String(ric)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetChp8ByRIC(ric string) ([]model.Sb70020Chp8, error) {
	ric = strings.TrimSpace(ric)
	if ric == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020Chp8
	stmt := SELECT(table.Sb70020Chp8.AllColumns).
		FROM(table.Sb70020Chp8).
		WHERE(table.Sb70020Chp8.Ric.EQ(String(ric)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}
```

- [ ] **Step 2: Verify package compiles**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/sb_700_20/...
```

Expected: compile error — `service` struct does not implement the 7 new `Service` interface methods yet. Repository is now satisfied.

---

### Task 5: Add Service Implementations

**Files:**
- Modify: `api/sb_700_20/service_impl.go`

- [ ] **Step 1: Append 7 delegation methods to `service_impl.go`**

Add at the end of the file:

```go
func (s *service) GetAppEByNewLIN(newLin string) ([]model.Sb70020AppE, error) {
	return s.repository.GetAppEByNewLIN(strings.TrimSpace(strings.ToUpper(newLin)))
}
func (s *service) GetAppGByNewLIN(newLin string) ([]model.Sb70020AppG, error) {
	return s.repository.GetAppGByNewLIN(strings.TrimSpace(strings.ToUpper(newLin)))
}
func (s *service) GetAppH1BySubLIN(sublin string) ([]model.Sb70020AppH1, error) {
	return s.repository.GetAppH1BySubLIN(strings.TrimSpace(strings.ToUpper(sublin)))
}
func (s *service) GetAppH2BySubLIN(sublin string) ([]model.Sb70020AppH2, error) {
	return s.repository.GetAppH2BySubLIN(strings.TrimSpace(strings.ToUpper(sublin)))
}
func (s *service) GetChp4ByRIC(ric string) ([]model.Sb70020Chp4, error) {
	return s.repository.GetChp4ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
func (s *service) GetChp6ByRIC(ric string) ([]model.Sb70020Chp6, error) {
	return s.repository.GetChp6ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
func (s *service) GetChp8ByRIC(ric string) ([]model.Sb70020Chp8, error) {
	return s.repository.GetChp8ByRIC(strings.TrimSpace(strings.ToUpper(ric)))
}
```

- [ ] **Step 2: Verify package compiles cleanly**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./api/sb_700_20/...
```

Expected: no errors.

---

### Task 6: Add Handler Functions — Apps

**Files:**
- Modify: `api/sb_700_20/handlers_apps.go`

- [ ] **Step 1: Append 4 handler functions to `handlers_apps.go`**

Add at the end of the file:

```go
func (h *Handler) searchAppEByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		return
	}
	items, err := h.service.GetAppEByNewLIN(newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppGByNewLIN(c *gin.Context) {
	newLin := c.Param("new_lin")
	if strings.TrimSpace(newLin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		return
	}
	items, err := h.service.GetAppGByNewLIN(newLin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new_lin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppH1BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		return
	}
	items, err := h.service.GetAppH1BySubLIN(sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchAppH2BySubLIN(c *gin.Context) {
	sublin := c.Param("sublin")
	if strings.TrimSpace(sublin) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		return
	}
	items, err := h.service.GetAppH2BySubLIN(sublin)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sublin parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}
```

---

### Task 7: Add Handler Functions — Chps

**Files:**
- Modify: `api/sb_700_20/handlers_chps.go`

- [ ] **Step 1: Append 3 handler functions to `handlers_chps.go`**

Add at the end of the file:

```go
func (h *Handler) searchChp4ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		return
	}
	items, err := h.service.GetChp4ByRIC(ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchChp6ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		return
	}
	items, err := h.service.GetChp6ByRIC(ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}

func (h *Handler) searchChp8ByRIC(c *gin.Context) {
	ric := c.Param("ric")
	if strings.TrimSpace(ric) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		return
	}
	items, err := h.service.GetChp8ByRIC(ric)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else if errors.Is(err, ErrEmptyParam) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ric parameter is required"})
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: http.StatusOK, Data: items})
}
```

---

### Task 8: Register Routes

**Files:**
- Modify: `api/sb_700_20/route.go`

- [ ] **Step 1: Add 7 route registrations to `RegisterHandlers`**

In `RegisterHandlers`, after the last existing `chp-8` line, add:

```go
router.GET("/sb700-20/app-e/search-new-lin/:new_lin", h.searchAppEByNewLIN)
router.GET("/sb700-20/app-g/search-new-lin/:new_lin", h.searchAppGByNewLIN)
router.GET("/sb700-20/app-h1/search-sublin/:sublin", h.searchAppH1BySubLIN)
router.GET("/sb700-20/app-h2/search-sublin/:sublin", h.searchAppH2BySubLIN)
router.GET("/sb700-20/chp-4/search-ric/:ric", h.searchChp4ByRIC)
router.GET("/sb700-20/chp-6/search-ric/:ric", h.searchChp6ByRIC)
router.GET("/sb700-20/chp-8/search-ric/:ric", h.searchChp8ByRIC)
```

- [ ] **Step 2: Verify full build**

```bash
cd /Users/swisscheese/projects/miltechserver && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit the implementation**

```bash
git add api/sb_700_20/repository.go api/sb_700_20/service.go \
        api/sb_700_20/repository_apps.go api/sb_700_20/repository_chps.go \
        api/sb_700_20/service_impl.go api/sb_700_20/handlers_apps.go \
        api/sb_700_20/handlers_chps.go api/sb_700_20/route.go
git commit -m "feat(sb700-20): add search by new_lin, sublin, and ric endpoints"
```

---

### Task 9: Add Test Helpers

**Files:**
- Modify: `tests/sb_700_20/helpers_test.go`

- [ ] **Step 1: Append 4 sample-fetch helpers to `helpers_test.go`**

Add at the end of the file:

```go
func fetchSampleNewLIN(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT new_lin FROM " + tableName + " WHERE new_lin IS NOT NULL LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleSubLINAppH1(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "sb_700_20_app_h1") {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT lin_zmm_sublin FROM sb_700_20_app_h1 LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleSubLINAppH2(t *testing.T, db *sql.DB) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, "sb_700_20_app_h2") {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT lin_zmmsublin FROM sb_700_20_app_h2 LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}

func fetchSampleRIC(t *testing.T, db *sql.DB, tableName string) (string, bool) {
	t.Helper()
	if !hasRelation(t, db, tableName) {
		return "", false
	}
	var val sql.NullString
	err := db.QueryRow("SELECT ric FROM " + tableName + " WHERE ric IS NOT NULL LIMIT 1").Scan(&val)
	if err == sql.ErrNoRows {
		return "", false
	}
	require.NoError(t, err)
	if !val.Valid {
		return "", false
	}
	return val.String, true
}
```

---

### Task 10: Update Repository Validation Tests

**Files:**
- Modify: `tests/sb_700_20/repository_test.go`

- [ ] **Step 1: Add 7 blank-param cases to `TestRepositoryValidationErrors`**

In `TestRepositoryValidationErrors`, add to the `cases` slice after the existing `GetChp8Paginated` entry:

```go
{"GetAppEByNewLIN blank", func() error { _, err := repo.GetAppEByNewLIN("  "); return err }},
{"GetAppGByNewLIN blank", func() error { _, err := repo.GetAppGByNewLIN("  "); return err }},
{"GetAppH1BySubLIN blank", func() error { _, err := repo.GetAppH1BySubLIN("  "); return err }},
{"GetAppH2BySubLIN blank", func() error { _, err := repo.GetAppH2BySubLIN("  "); return err }},
{"GetChp4ByRIC blank", func() error { _, err := repo.GetChp4ByRIC("  "); return err }},
{"GetChp6ByRIC blank", func() error { _, err := repo.GetChp6ByRIC("  "); return err }},
{"GetChp8ByRIC blank", func() error { _, err := repo.GetChp8ByRIC("  "); return err }},
```

- [ ] **Step 2: Run validation tests**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/sb_700_20/... -run TestRepositoryValidationErrors -v
```

Expected: all 33 cases PASS (7 new blank cases → `ErrEmptyParam`).

---

### Task 11: Update Handler Integration Tests

**Files:**
- Modify: `tests/sb_700_20/handlers_test.go`

- [ ] **Step 1: Add 7 endpoints to `TestBlankSearchParams`**

In `TestBlankSearchParams`, add to the `endpoints` slice:

```go
"/api/v1/sb700-20/app-e/search-new-lin/%20%20",
"/api/v1/sb700-20/app-g/search-new-lin/%20%20",
"/api/v1/sb700-20/app-h1/search-sublin/%20%20",
"/api/v1/sb700-20/app-h2/search-sublin/%20%20",
"/api/v1/sb700-20/chp-4/search-ric/%20%20",
"/api/v1/sb700-20/chp-6/search-ric/%20%20",
"/api/v1/sb700-20/chp-8/search-ric/%20%20",
```

- [ ] **Step 2: Add 7 new test functions**

Append to the end of `handlers_test.go`:

```go
func TestAppEByNewLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_e") {
		t.Skip("sb_700_20_app_e missing in test DB")
	}
	if val, ok := fetchSampleNewLIN(t, testDB, "sb_700_20_app_e"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search-new-lin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppE
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-e/search-new-lin/NOTREAL999").Code)
}

func TestAppGByNewLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_g") {
		t.Skip("sb_700_20_app_g missing in test DB")
	}
	if val, ok := fetchSampleNewLIN(t, testDB, "sb_700_20_app_g"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search-new-lin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppG
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-g/search-new-lin/NOTREAL999").Code)
}

func TestAppH1BySubLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h1") {
		t.Skip("sb_700_20_app_h1 missing in test DB")
	}
	if val, ok := fetchSampleSubLINAppH1(t, testDB); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search-sublin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH1
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h1/search-sublin/NOTREAL999").Code)
}

func TestAppH2BySubLINEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_app_h2") {
		t.Skip("sb_700_20_app_h2 missing in test DB")
	}
	if val, ok := fetchSampleSubLINAppH2(t, testDB); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search-sublin/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020AppH2
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/app-h2/search-sublin/NOTREAL999").Code)
}

func TestChp4ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_4") {
		t.Skip("sb_700_20_chp_4 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_4"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp4
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-4/search-ric/NOTREAL999").Code)
}

func TestChp6ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_6") {
		t.Skip("sb_700_20_chp_6 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_6"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp6
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-6/search-ric/NOTREAL999").Code)
}

func TestChp8ByRICEndpoint(t *testing.T) {
	router := newTestRouter(t)
	if !hasRelation(t, testDB, "sb_700_20_chp_8") {
		t.Skip("sb_700_20_chp_8 missing in test DB")
	}
	if val, ok := fetchSampleRIC(t, testDB, "sb_700_20_chp_8"); ok {
		resp := doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search-ric/"+val)
		require.Equal(t, http.StatusOK, resp.Code)
		var payload standardResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
		var data []model.Sb70020Chp8
		require.NoError(t, json.Unmarshal(payload.Data, &data))
		require.NotEmpty(t, data)
	}
	require.Equal(t, http.StatusNotFound, doJSONRequest(t, router, http.MethodGet, "/api/v1/sb700-20/chp-8/search-ric/NOTREAL999").Code)
}
```

- [ ] **Step 3: Run full test suite**

```bash
cd /Users/swisscheese/projects/miltechserver && go test ./tests/sb_700_20/... -v
```

Expected: all tests PASS or SKIP (skips happen when a table is absent from the test DB — that is correct behavior).

- [ ] **Step 4: Commit the tests**

```bash
git add tests/sb_700_20/helpers_test.go tests/sb_700_20/repository_test.go tests/sb_700_20/handlers_test.go
git commit -m "test(sb700-20): add validation and integration tests for new search endpoints"
```

---

### Task 12: Write API Reference Doc

**Files:**
- Create: `docs/sb700-20-new-search-endpoints.md`

- [ ] **Step 1: Create the doc**

Create `docs/sb700-20-new-search-endpoints.md` with the content below. No code — JSON examples only.

````markdown
# SB 700-20 New Search Endpoints

This document describes the 7 new search endpoints added to the SB 700-20 API. All endpoints are public (no authentication required) and are mounted under `/api/v1`.

All successful responses share the same envelope:

```json
{
  "status": 200,
  "message": "",
  "data": [ ... ]
}
```

All error responses follow one of two shapes:

**400 Bad Request**
```json
{ "error": "<param> parameter is required" }
```

**404 Not Found**
```json
{ "status": 404, "message": "No item found", "data": {} }
```

**500 Internal Server Error**
```json
{ "status": 500, "message": "Internal server error" }
```

---

## App E — Search by New LIN

**`GET /api/v1/sb700-20/app-e/search-new-lin/:new_lin`**

Returns all App E records whose `new_lin` field matches the given value. The value is matched case-insensitively (normalized to uppercase before querying).

**Path parameter:** `:new_lin` — the new LIN value to search for (e.g. `A12345`). Whitespace-only values return 400.

**Example request:**
```
GET /api/v1/sb700-20/app-e/search-new-lin/A12345
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "B99001",
      "cmc": "C",
      "reason_for_deletion": null,
      "new_lin": "A12345",
      "nsn": "1234-01-567-8901",
      "nomenclature": "RIFLE,5.56MM",
      "date_entered_into_ap": "20230101",
      "army_type_class": "W",
      "ratio": "1",
      "zmm_appdx_dlw_and_or_zmm": null,
      "calendar_year_month": "202301",
      "chapter_code": 4
    }
  ]
}
```

---

## App G — Search by New LIN

**`GET /api/v1/sb700-20/app-g/search-new-lin/:new_lin`**

Returns all App G records whose `new_lin` field matches the given value.

**Path parameter:** `:new_lin` — the new LIN value (e.g. `A12345`).

**Example request:**
```
GET /api/v1/sb700-20/app-g/search-new-lin/A12345
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "tr": "R",
      "new_lin": "A12345",
      "lin": "B99001"
    }
  ]
}
```

---

## App H1 — Search by Sublin

**`GET /api/v1/sb700-20/app-h1/search-sublin/:sublin`**

Returns all App H1 records whose `lin_zmm_sublin` field matches the given sublin value. A single sublin may appear under multiple parent LINs, so this can return multiple records.

**Path parameter:** `:sublin` — the sublin value (e.g. `AA`).

**Example request:**
```
GET /api/v1/sb700-20/app-h1/search-sublin/AA
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin_zmm_lin": "A00001",
      "nomenclature": "RIFLE SYSTEM",
      "lin_zmm_sublin": "AA",
      "sub_lin_nomenclature": "RIFLE,5.56MM"
    },
    {
      "lin_zmm_lin": "B00002",
      "nomenclature": "WEAPON SET",
      "lin_zmm_sublin": "AA",
      "sub_lin_nomenclature": "CARBINE,5.56MM"
    }
  ]
}
```

---

## App H2 — Search by Sublin

**`GET /api/v1/sb700-20/app-h2/search-sublin/:sublin`**

Returns all App H2 records whose `lin_zmmsublin` field matches the given sublin value. Note: App H2 uses `lin_zmmsublin` (no underscore between zmm and sublin) as its column name — this is distinct from App H1's `lin_zmm_sublin`.

**Path parameter:** `:sublin` — the sublin value (e.g. `AA`).

**Example request:**
```
GET /api/v1/sb700-20/app-h2/search-sublin/AA
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin_zmmsublin": "AA",
      "sub_lin_nomenclature": "RIFLE,5.56MM",
      "lin_zmm_lin": "A00001",
      "nomenclature": "RIFLE SYSTEM"
    }
  ]
}
```

---

## Chp 4 — Search by RIC

**`GET /api/v1/sb700-20/chp-4/search-ric/:ric`**

Returns all Chapter 4 records whose `ric` field matches the given RIC value. Multiple records can share the same RIC.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-4/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": 1,
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_of_mara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```

---

## Chp 6 — Search by RIC

**`GET /api/v1/sb700-20/chp-6/search-ric/:ric`**

Returns all Chapter 6 records whose `ric` field matches the given RIC value.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-6/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": "1",
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_of_mara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```

---

## Chp 8 — Search by RIC

**`GET /api/v1/sb700-20/chp-8/search-ric/:ric`**

Returns all Chapter 8 records whose `ric` field matches the given RIC value.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-8/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": "1",
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_ofMara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```
````

- [ ] **Step 2: Commit the doc**

```bash
git add docs/sb700-20-new-search-endpoints.md
git commit -m "docs(sb700-20): add API reference for new search endpoints"
```
