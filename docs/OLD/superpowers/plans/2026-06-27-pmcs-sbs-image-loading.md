# PMCS SBS Image Loading Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a public PMCS SBS endpoint that streams raw PNG bytes for guide item images stored in Azure Blob Storage.

**Architecture:** Keep the feature inside the existing public `api/library/pmcs_sbs` package. The endpoint accepts the selected guide JSON `blob_path` and an extensionless `image_name`, derives the PNG blob path under the guide's `images/<guide-name>/` folder, validates path safety, and streams the PNG with `Content-Type: image/png`.

**Tech Stack:** Go, Gin, Azure Blob Storage SDK, `io`, `path`, `strings`, `slog`, `testify/require`.

**Spec:** `docs/superpowers/specs/2026-06-27-pmcs-sbs-image-loading-design.md`

---

## File Structure

- Modify `api/library/pmcs_sbs/errors.go`: add image-specific validation errors.
- Modify `api/library/pmcs_sbs/response.go`: add image download stream type.
- Modify `api/library/pmcs_sbs/service.go`: add `GetImage`.
- Modify `api/library/pmcs_sbs/service_impl.go`: add validation, path derivation, and Azure `DownloadStream`.
- Modify `api/library/pmcs_sbs/service_impl_test.go`: add pure validation and path derivation tests.
- Modify `api/library/pmcs_sbs/route.go`: register and implement `GET /library/pmcs-sbs/image`.
- Modify `api/library/pmcs_sbs/route_test.go`: add handler tests for success and error mapping.
- Modify `api/route/route_test.go`: assert public route registration.
- Modify `docs/api/pmcs-sbs-api.md`: document image loading for mobile.

No Postgres, Jet, migrations, authenticated route, generated file, or PMCS SBS fault API changes are required.

Unrelated current worktree note: `docs/api/pmcs_sbs_bulk_fault_delete_mobile.md` is untracked before this plan and must remain out of this plan unless explicitly requested.

---

## Task 1: Service Contract And Validation Helpers

**Files:**
- Modify: `api/library/pmcs_sbs/errors.go`
- Modify: `api/library/pmcs_sbs/response.go`
- Modify: `api/library/pmcs_sbs/service.go`
- Modify: `api/library/pmcs_sbs/service_impl.go`
- Test: `api/library/pmcs_sbs/service_impl_test.go`

- [ ] **Step 1: Add service tests for image path derivation and validation**

In `api/library/pmcs_sbs/service_impl_test.go`, add tests that call pure helper methods before any Azure client is touched:

```go
func TestBuildImageBlobPath(t *testing.T) {
	imageBlobPath, err := buildImageBlobPath(
		"pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json",
		"Before_12",
	)

	require.NoError(t, err)
	require.Equal(t, "pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png", imageBlobPath)
}
```

Add table tests for invalid guide paths:

- blank guide path -> `ErrEmptyBlobPath`
- `pmcs_sbs/HMMWV/file.pdf` -> `ErrInvalidFileType`
- `pmcs_sbs/HMMWV/../secret.json` -> `ErrInvalidBlobPath`
- `pmcs_sbs\\HMMWV\\file.json` -> `ErrInvalidBlobPath`
- `pmcs/other/file.json` -> `ErrInvalidBlobPath`

Add table tests for invalid image names:

- blank image name -> `ErrEmptyImageName`
- `Before_12.png` -> `ErrInvalidImageName`
- `folder/Before_12` -> `ErrInvalidImageName`
- `folder\\Before_12` -> `ErrInvalidImageName`
- `../Before_12` -> `ErrInvalidImageName`
- `Before.12` -> `ErrInvalidImageName`

- [ ] **Step 2: Run tests and confirm the expected failure**

```bash
go test ./api/library/pmcs_sbs -run 'TestBuildImageBlobPath|TestValidate.*Image|Test.*Image.*Validation' -count=1
```

Expected: compile failure because the helper and new errors do not exist.

- [ ] **Step 3: Add image errors**

In `api/library/pmcs_sbs/errors.go`, add:

```go
ErrEmptyImageName   = errors.New("image name cannot be empty")
ErrInvalidImageName = errors.New("invalid image name: must be an extensionless PNG basename")
```

- [ ] **Step 4: Add image stream type**

In `api/library/pmcs_sbs/response.go`, add:

```go
type ImageDownload struct {
	Body          io.ReadCloser
	ContentLength int64
	ContentType   string
	FileName      string
	BlobPath      string
}
```

Import `io` in that file.

- [ ] **Step 5: Extend the service interface**

In `api/library/pmcs_sbs/service.go`, add:

```go
GetImage(ctx context.Context, guideBlobPath string, imageName string) (*ImageDownload, error)
```

- [ ] **Step 6: Implement validation and path derivation helpers**

In `api/library/pmcs_sbs/service_impl.go`, add focused helpers:

- `cleanGuideBlobPath(blobPath string) (string, error)`
- `cleanImageName(imageName string) (string, error)`
- `buildImageBlobPath(guideBlobPath string, imageName string) (string, error)`

Validation rules:

- guide path trims whitespace, rejects backslashes, uses `path.Clean`, requires `pmcs_sbs/`, and requires `.json`;
- image name trims whitespace, rejects `/`, `\`, `.`, `..`, and any value changed by `path.Clean`;
- image name is extensionless and receives `.png` only during path derivation.

- [ ] **Step 7: Verify service helper tests pass**

```bash
go test ./api/library/pmcs_sbs -run 'TestBuildImageBlobPath|Test.*Image.*Validation|TestGetFileContentValidation|TestGetFilesValidation' -count=1
```

- [ ] **Step 8: Commit Task 1**

```bash
git add api/library/pmcs_sbs/errors.go api/library/pmcs_sbs/response.go api/library/pmcs_sbs/service.go api/library/pmcs_sbs/service_impl.go api/library/pmcs_sbs/service_impl_test.go
git commit -m "feat(pmcs-sbs): add image path validation"
```

---

## Task 2: Azure Image Download Service

**Files:**
- Modify: `api/library/pmcs_sbs/service_impl.go`
- Test: `api/library/pmcs_sbs/service_impl_test.go`

- [ ] **Step 1: Implement `GetImage`**

In `ServiceImpl`, add:

```go
func (s *ServiceImpl) GetImage(ctx context.Context, guideBlobPath string, imageName string) (*ImageDownload, error)
```

Behavior:

- derive the image blob path with `buildImageBlobPath`;
- create a blob client in container `LibraryContainerName`;
- call `DownloadStream(ctx, nil)`;
- map Azure download failures to `ErrFileNotFound`, matching the existing content endpoint's current behavior;
- return `ImageDownload` with:
  - `Body`: Azure response body;
  - `ContentLength`: Azure content length when present, otherwise `-1`;
  - `ContentType`: `image/png`;
  - `FileName`: `imageName + ".png"`;
  - `BlobPath`: derived image blob path.

Do not read the full image into memory.

- [ ] **Step 2: Add nil-safe validation coverage**

Add a test using `NewService(nil)` that calls `GetImage` with invalid inputs and confirms validation returns before the nil Azure client can be touched.

- [ ] **Step 3: Verify package tests**

```bash
go test ./api/library/pmcs_sbs -count=1
```

- [ ] **Step 4: Commit Task 2**

```bash
git add api/library/pmcs_sbs/service_impl.go api/library/pmcs_sbs/service_impl_test.go
git commit -m "feat(pmcs-sbs): stream images from blob storage"
```

---

## Task 3: Public Image Route And Handler Tests

**Files:**
- Modify: `api/library/pmcs_sbs/route.go`
- Modify: `api/library/pmcs_sbs/route_test.go`
- Modify: `api/route/route_test.go`

- [ ] **Step 1: Extend route test stubs**

In `api/library/pmcs_sbs/route_test.go`, extend `serviceStub` with:

```go
imageResp *ImageDownload
imageErr  error
```

Implement:

```go
func (s *serviceStub) GetImage(_ context.Context, _ string, _ string) (*ImageDownload, error) {
	return s.imageResp, s.imageErr
}
```

Extend `capturingStub` to record `capturedImageBlobPath` and `capturedImageName`.

- [ ] **Step 2: Add handler tests**

Add route tests for:

- success returns `200`, `Content-Type: image/png`, `Content-Disposition: inline; filename="Before_12.png"`, and PNG bytes;
- missing `blob_path` returns `400`;
- missing `image_name` returns `400`;
- `ErrInvalidBlobPath` returns `400`;
- `ErrInvalidImageName` returns `400`;
- `ErrFileNotFound` returns `404`;
- generic error returns `500`;
- handler passes `blob_path` and `image_name` unchanged to the service.

Use `io.NopCloser(strings.NewReader("png-bytes"))` for success test image body.

- [ ] **Step 3: Run handler tests and confirm expected failure**

```bash
go test ./api/library/pmcs_sbs -run 'TestGetImage|Test.*Image' -count=1
```

Expected: compile or route failure until the handler is implemented.

- [ ] **Step 4: Register route**

In `registerHandlers`, add the public route beside the content route:

```go
publicGroup.GET("/library/pmcs-sbs/image", middleware.RateLimiter(), h.getImage)
```

- [ ] **Step 5: Implement handler**

In `api/library/pmcs_sbs/route.go`, add `getImage`.

Handler behavior:

- require nonblank `blob_path`;
- require nonblank `image_name`;
- call `h.service.GetImage(c.Request.Context(), blobPath, imageName)`;
- map `ErrFileNotFound` to `404`;
- map `ErrEmptyBlobPath`, `ErrInvalidBlobPath`, `ErrInvalidFileType`, `ErrEmptyImageName`, and `ErrInvalidImageName` to `400`;
- map all other errors to `500`;
- on success, `defer image.Body.Close()`;
- set `Cache-Control`;
- stream with `c.DataFromReader(http.StatusOK, image.ContentLength, image.ContentType, image.Body, extraHeaders)`.

Extra headers:

```go
map[string]string{
	"Content-Disposition": fmt.Sprintf(`inline; filename="%s"`, image.FileName),
	"Cache-Control":      "public, max-age=86400",
}
```

- [ ] **Step 6: Add public route registration test**

In `api/route/route_test.go`, add:

```go
func TestSetupRegistersPmcsSbsImageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	Setup(nil, router, nil, nil, nil)

	requireRouteRegistered(t, router, http.MethodGet, "/api/v1/library/pmcs-sbs/image")
}
```

- [ ] **Step 7: Verify route and handler tests**

```bash
go test ./api/library/pmcs_sbs ./api/route -count=1
```

- [ ] **Step 8: Commit Task 3**

```bash
git add api/library/pmcs_sbs/route.go api/library/pmcs_sbs/route_test.go api/route/route_test.go
git commit -m "feat(pmcs-sbs): add public image endpoint"
```

---

## Task 4: Mobile API Documentation

**Files:**
- Modify: `docs/api/pmcs-sbs-api.md`

- [ ] **Step 1: Update overview**

Change the PMCS SBS public API overview from three endpoints to four:

1. List folders
2. List files
3. Fetch content
4. Fetch image

- [ ] **Step 2: Add Fetch Image section**

Add a new section after Fetch Content:

```http
GET /library/pmcs-sbs/image?blob_path=<guide_json_blob_path>&image_name=<extensionless_image_name>
```

Document:

- no authentication required;
- success returns binary PNG bytes, not JSON;
- `Content-Type: image/png`;
- `blob_path` must be the selected guide JSON path from the List Files response;
- `image_name` must be the exact extensionless string from the guide item's `images` array;
- example using `Before_12`.

- [ ] **Step 3: Add error table**

Document:

- missing `blob_path` -> `400`;
- missing `image_name` -> `400`;
- invalid guide path -> `400`;
- invalid image name -> `400`;
- image missing from blob storage -> `404`;
- rate limit exceeded -> `429`;
- Azure/storage failure -> `500`.

- [ ] **Step 4: Update mobile workflow**

Add guidance:

- load guide JSON first;
- read item `images`;
- request each image on demand;
- treat `404` as missing content and continue rendering the step.

- [ ] **Step 5: Commit Task 4**

```bash
git add docs/api/pmcs-sbs-api.md
git commit -m "docs(pmcs-sbs): document image loading endpoint"
```

---

## Task 5: Focused Verification And Final Review

**Files:**
- Verify all touched files from Tasks 1-4.

- [ ] **Step 1: Run focused tests**

```bash
go test ./api/library/pmcs_sbs ./api/library ./api/route -count=1
```

Expected: PASS.

- [ ] **Step 2: Optional broader route-adjacent check**

If the focused tests pass and time allows:

```bash
go test ./api/library/... ./api/route -count=1
```

Expected: PASS, unless unrelated baseline failures appear.

- [ ] **Step 3: Review final diff**

```bash
git status --short
git diff --stat HEAD
git diff HEAD -- api/library/pmcs_sbs api/route/route_test.go docs/api/pmcs-sbs-api.md
```

Confirm:

- no generated files changed;
- no Postgres or Jet files changed;
- no authenticated PMCS SBS fault files changed;
- untracked `docs/api/pmcs_sbs_bulk_fault_delete_mobile.md` remains unrelated unless the user explicitly includes it.

- [ ] **Step 4: Final commit if verification required fixes**

If Task 5 reveals a small test/doc correction, commit it:

```bash
git add api/library/pmcs_sbs api/route/route_test.go docs/api/pmcs-sbs-api.md
git commit -m "test(pmcs-sbs): verify image loading endpoint"
```

Before committing this optional correction, re-run `git status --short` and confirm only files touched by this plan are staged. Skip this commit if no code changes are needed after Tasks 1-4.

---

## Acceptance Criteria

- `GET /api/v1/library/pmcs-sbs/image` is registered publicly.
- Endpoint accepts `blob_path` and `image_name`.
- Endpoint returns raw PNG bytes with `Content-Type: image/png`.
- Endpoint derives image blobs under `images/<guide-name>/<image_name>.png`.
- Endpoint rejects unsafe guide paths and unsafe image names.
- Endpoint returns `404` for missing image blobs.
- Endpoint does not parse guide JSON or validate `images[]` membership.
- Endpoint does not touch Postgres, Jet, authenticated PMCS SBS fault routes, or generated files.
- PMCS SBS public API docs describe the new mobile image-loading contract.
- Focused verification passes:

```bash
go test ./api/library/pmcs_sbs ./api/library ./api/route -count=1
```
