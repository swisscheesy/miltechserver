# Docs Equipment Details Implementation Plan

> **For Antigravity:** REQUIRED SUB-SKILL: Load executing-plans to implement this plan task-by-task.

**Goal:** Add public endpoints for querying equipment details from Postgres and browsing/downloading equipment images from Azure Blob Storage.

**Architecture:** New `api/docs_equipment` package following route→service→repository pattern. DB queries use raw SQL with go-jet models for scanning. Image endpoints use Azure Blob SDK hierarchy/flat pagers and `shared.GenerateBlobSASURL` for downloads.

**Tech Stack:** Go, Gin, go-jet models, Azure Blob SDK, testify

**Design Doc:** [2026-03-02-docs-equipment-details-design.md](file:///Users/swisscheese/projects/miltechserver/docs/plans/2026-03-02-docs-equipment-details-design.md)

---

## Task 1: Scaffold Package — Errors + Response Types

**Files:**
- Create: `api/docs_equipment/errors.go`
- Create: `api/docs_equipment/response.go`

**Step 1: Create `errors.go`**

```go
package docs_equipment

import "errors"

var (
	ErrNotFound        = errors.New("no equipment details found")
	ErrEmptyParam      = errors.New("required parameter is empty")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrEmptyBlobPath   = errors.New("blob path cannot be empty")
	ErrInvalidBlobPath = errors.New("invalid blob path: must start with docs_equipment/images/")
	ErrInvalidFileType = errors.New("invalid file type: only image files are allowed")
	ErrImageNotFound   = errors.New("image not found")
	ErrBlobListFailed  = errors.New("failed to list blobs from Azure")
	ErrSASGenFailed    = errors.New("failed to generate download URL")
)
```

**Step 2: Create `response.go`**

```go
package docs_equipment

import "miltechserver/.gen/miltech_ng/public/model"

// EquipmentDetailsPageResponse — paginated equipment list (matches EIC pattern).
type EquipmentDetailsPageResponse struct {
	Items      []model.DocsEquipmentDetails `json:"items"`
	Count      int                          `json:"count"`
	Page       int                          `json:"page"`
	TotalPages int                          `json:"total_pages"`
	IsLastPage bool                         `json:"is_last_page"`
}

// FamiliesResponse — unique family values from the DB.
type FamiliesResponse struct {
	Families []string `json:"families"`
	Count    int      `json:"count"`
}

// ImageFamilyFolder — a family folder in blob storage.
type ImageFamilyFolder struct {
	Name        string `json:"name"`
	FullPath    string `json:"full_path"`
	DisplayName string `json:"display_name"`
}

// ImageFamiliesResponse — list of image family folders.
type ImageFamiliesResponse struct {
	Families []ImageFamilyFolder `json:"families"`
	Count    int                 `json:"count"`
}

// ImageItem — a single image blob.
type ImageItem struct {
	Name         string `json:"name"`
	BlobPath     string `json:"blob_path"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
}

// FamilyImagesResponse — images in a family folder.
type FamilyImagesResponse struct {
	Family string      `json:"family"`
	Images []ImageItem `json:"images"`
	Count  int         `json:"count"`
}

// ImageDownloadResponse — SAS download URL.
type ImageDownloadResponse struct {
	BlobPath    string `json:"blob_path"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
}
```

**Step 3: Verify compilation**

Run: `cd /Users/swisscheese/projects/miltechserver && go build ./api/docs_equipment/...`
Expected: clean build, no errors.

**Step 4: Commit**

```bash
git add api/docs_equipment/errors.go api/docs_equipment/response.go
git commit -m "feat(docs_equipment): scaffold errors and response types"
```

---

## Task 2: Repository Interface + Implementation

**Files:**
- Create: `api/docs_equipment/repository.go`
- Create: `api/docs_equipment/repository_impl.go`

**Step 1: Create `repository.go`**

```go
package docs_equipment

type Repository interface {
	GetAllPaginated(page int) (EquipmentDetailsPageResponse, error)
	GetFamilies() (FamiliesResponse, error)
	GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error)
	SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error)
}
```

**Step 2: Create `repository_impl.go`**

Uses raw SQL with `database/sql` (consistent with EIC pattern). Page size constant of 40. Each method:
- Queries data with `LIMIT/OFFSET`
- Queries total count
- Calculates `totalPages` and `isLastPage`
- Scans into `model.DocsEquipmentDetails` fields

Key queries:
- `GetAllPaginated`: `SELECT * FROM docs_equipment_details ORDER BY id LIMIT $1 OFFSET $2`
- `GetFamilies`: `SELECT DISTINCT family FROM docs_equipment_details WHERE family IS NOT NULL ORDER BY family`
- `GetByFamilyPaginated`: same as GetAll + `WHERE LOWER(family) = LOWER($1)`
- `SearchPaginated`: `WHERE model ILIKE $1 OR lin ILIKE $1` with `%query%` pattern

The scan helper maps all 12 columns to `model.DocsEquipmentDetails` fields.

```go
package docs_equipment

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

const pageSize = 40

type repositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repositoryImpl{db: db}
}

func scanEquipmentItem(rows *sql.Rows) (model.DocsEquipmentDetails, error) {
	var item model.DocsEquipmentDetails
	err := rows.Scan(
		&item.ID, &item.Model, &item.Lin, &item.Mode,
		&item.Description, &item.Length, &item.Width,
		&item.Height, &item.Weight, &item.Kw, &item.Hz, &item.Family,
	)
	return item, err
}

const selectAll = `SELECT id, model, lin, mode, description, length, width, height, weight, kw, hz, family FROM docs_equipment_details`

func (r *repositoryImpl) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)

	query := selectAll + ` ORDER BY id LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to query equipment details: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details`).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count equipment details: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}

func (r *repositoryImpl) GetFamilies() (FamiliesResponse, error) {
	rows, err := r.db.Query(`SELECT DISTINCT family FROM docs_equipment_details WHERE family IS NOT NULL ORDER BY family`)
	if err != nil {
		return FamiliesResponse{}, fmt.Errorf("failed to query families: %w", err)
	}
	defer rows.Close()

	var families []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return FamiliesResponse{}, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, f)
	}
	if err := rows.Err(); err != nil {
		return FamiliesResponse{}, fmt.Errorf("failed to iterate families: %w", err)
	}

	return FamiliesResponse{Families: families, Count: len(families)}, nil
}

func (r *repositoryImpl) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	if strings.TrimSpace(family) == "" {
		return EquipmentDetailsPageResponse{}, ErrEmptyParam
	}
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)

	query := selectAll + ` WHERE LOWER(family) = LOWER($1) ORDER BY id LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(query, strings.TrimSpace(family), pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to query by family: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details WHERE LOWER(family) = LOWER($1)`, strings.TrimSpace(family)).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count by family: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found for family: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}

func (r *repositoryImpl) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	if strings.TrimSpace(query) == "" {
		return EquipmentDetailsPageResponse{}, ErrEmptyParam
	}
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)
	searchPattern := "%" + strings.TrimSpace(query) + "%"

	stmt := selectAll + ` WHERE model ILIKE $1 OR lin ILIKE $1 ORDER BY id LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(stmt, searchPattern, pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details WHERE model ILIKE $1 OR lin ILIKE $1`, searchPattern).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count search results: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found matching query: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}

func collectItems(rows *sql.Rows) ([]model.DocsEquipmentDetails, error) {
	var items []model.DocsEquipmentDetails
	for rows.Next() {
		item, err := scanEquipmentItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan equipment item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}
	return items, nil
}

func buildPageResponse(items []model.DocsEquipmentDetails, page, totalCount int) EquipmentDetailsPageResponse {
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	return EquipmentDetailsPageResponse{
		Items:      items,
		Count:      len(items),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}
}
```

**Step 3: Verify compilation**

Run: `cd /Users/swisscheese/projects/miltechserver && go build ./api/docs_equipment/...`
Expected: clean build.

**Step 4: Commit**

```bash
git add api/docs_equipment/repository.go api/docs_equipment/repository_impl.go
git commit -m "feat(docs_equipment): add repository interface and implementation"
```

---

## Task 3: Service Interface + Implementation

**Files:**
- Create: `api/docs_equipment/service.go`
- Create: `api/docs_equipment/service_impl.go`

**Step 1: Create `service.go`**

```go
package docs_equipment

import "context"

type Service interface {
	// DB operations
	GetAllPaginated(page int) (EquipmentDetailsPageResponse, error)
	GetFamilies() (FamiliesResponse, error)
	GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error)
	SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error)

	// Blob operations
	ListImageFamilies() (*ImageFamiliesResponse, error)
	ListFamilyImages(family string) (*FamilyImagesResponse, error)
	GenerateImageDownloadURL(ctx context.Context, blobPath string) (*ImageDownloadResponse, error)
}
```

**Step 2: Create `service_impl.go`**

DB methods delegate to repo. Image methods follow the PMCS library pattern:
- `ListImageFamilies`: uses `NewListBlobsHierarchyPager("/")` with prefix `docs_equipment/images/`
- `ListFamilyImages`: uses `NewListBlobsFlatPager` with prefix `docs_equipment/images/{family}/`, filters to image extensions
- `GenerateImageDownloadURL`: validates path prefix + extension, checks blob exists, calls `shared.GenerateBlobSASURL`

```go
package docs_equipment

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"miltechserver/api/library/shared"
)

const (
	containerName = "library"
	imagePrefix   = "docs_equipment/images/"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

type serviceImpl struct {
	repo       Repository
	blobClient *azblob.Client
}

func NewService(repo Repository, blobClient *azblob.Client) Service {
	return &serviceImpl{repo: repo, blobClient: blobClient}
}

func (s *serviceImpl) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.GetAllPaginated(page)
}

func (s *serviceImpl) GetFamilies() (FamiliesResponse, error) {
	return s.repo.GetFamilies()
}

func (s *serviceImpl) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.GetByFamilyPaginated(strings.TrimSpace(family), page)
}

func (s *serviceImpl) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	return s.repo.SearchPaginated(strings.TrimSpace(query), page)
}

func (s *serviceImpl) ListImageFamilies() (*ImageFamiliesResponse, error) {
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(containerName)
	prefix := imagePrefix
	pager := containerClient.NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{
		Prefix: &prefix,
	})

	var families []ImageFamilyFolder
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, p := range page.Segment.BlobPrefixes {
			if p.Name == nil {
				continue
			}
			fullPath := *p.Name
			name := strings.TrimPrefix(fullPath, imagePrefix)
			name = strings.TrimSuffix(name, "/")
			if name == "" {
				continue
			}
			displayName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " "))
			families = append(families, ImageFamilyFolder{
				Name:        name,
				FullPath:    fullPath,
				DisplayName: displayName,
			})
		}
	}

	return &ImageFamiliesResponse{Families: families, Count: len(families)}, nil
}

func isImageFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return allowedImageExts[ext]
}

func (s *serviceImpl) ListFamilyImages(family string) (*FamilyImagesResponse, error) {
	if strings.TrimSpace(family) == "" {
		return nil, ErrEmptyParam
	}
	ctx := context.Background()
	containerClient := s.blobClient.ServiceClient().NewContainerClient(containerName)
	prefix := imagePrefix + strings.TrimSpace(family) + "/"
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	var images []ImageItem
	for pager.More() {
		pg, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBlobListFailed, err)
		}
		for _, blob := range pg.Segment.BlobItems {
			if blob.Name == nil {
				continue
			}
			blobPath := *blob.Name
			parts := strings.Split(blobPath, "/")
			fileName := parts[len(parts)-1]
			if !isImageFile(fileName) {
				slog.Debug("Skipping non-image blob", "blobPath", blobPath)
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
			images = append(images, ImageItem{
				Name:         fileName,
				BlobPath:     blobPath,
				SizeBytes:    sizeBytes,
				LastModified: lastModified,
			})
		}
	}

	return &FamilyImagesResponse{
		Family: strings.TrimSpace(family),
		Images: images,
		Count:  len(images),
	}, nil
}

func (s *serviceImpl) GenerateImageDownloadURL(ctx context.Context, blobPath string) (*ImageDownloadResponse, error) {
	if strings.TrimSpace(blobPath) == "" {
		return nil, ErrEmptyBlobPath
	}
	blobPath = path.Clean(blobPath)
	if !strings.HasPrefix(blobPath, imagePrefix) {
		return nil, ErrInvalidBlobPath
	}
	if !isImageFile(blobPath) {
		return nil, ErrInvalidFileType
	}

	blobClient := s.blobClient.ServiceClient().NewContainerClient(containerName).NewBlobClient(blobPath)
	if _, err := blobClient.GetProperties(ctx, nil); err != nil {
		slog.Error("Equipment image blob not found", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrImageNotFound, err)
	}

	sasResult, err := shared.GenerateBlobSASURL(ctx, s.blobClient, containerName, blobPath)
	if err != nil {
		slog.Error("Failed to generate SAS for equipment image", "blobPath", blobPath, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrSASGenFailed, err)
	}

	return &ImageDownloadResponse{
		BlobPath:    blobPath,
		DownloadURL: sasResult.URL,
		ExpiresAt:   sasResult.ExpiresAt.Format(time.RFC3339),
	}, nil
}
```

**Step 3: Verify compilation**

Run: `cd /Users/swisscheese/projects/miltechserver && go build ./api/docs_equipment/...`

**Step 4: Commit**

```bash
git add api/docs_equipment/service.go api/docs_equipment/service_impl.go
git commit -m "feat(docs_equipment): add service interface and implementation"
```

---

## Task 4: Route Handler + Registration

**Files:**
- Create: `api/docs_equipment/route.go`
- Modify: `api/route/route.go` (add import + registration call)

**Step 1: Create `route.go`**

7 handler methods matching the 7 endpoints from the design doc. Follows the EIC handler pattern for pagination/error handling, and the library handler pattern for image download with error switching.

```go
package docs_equipment

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"

	"miltechserver/api/middleware"
	"miltechserver/api/response"
)

type Dependencies struct {
	DB         *sql.DB
	BlobClient *azblob.Client
}

type Handler struct {
	service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
	repo := NewRepository(deps.DB)
	svc := NewService(repo, deps.BlobClient)
	registerHandlers(router, svc)
}

func registerHandlers(router *gin.RouterGroup, svc Service) {
	handler := Handler{service: svc}

	// Data endpoints
	router.GET("/equipment-details", handler.getAllPaginated)
	router.GET("/equipment-details/families", handler.getFamilies)
	router.GET("/equipment-details/family/:family", handler.getByFamily)
	router.GET("/equipment-details/search", handler.search)

	// Image endpoints
	router.GET("/equipment-details/images/families", handler.listImageFamilies)
	router.GET("/equipment-details/images/family/:family", handler.listFamilyImages)
	router.GET("/equipment-details/images/download", middleware.RateLimiter(), handler.generateImageDownloadURL)
}

func (h *Handler) getAllPaginated(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetAllPaginated(page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) getFamilies(c *gin.Context) {
	data, err := h.service.GetFamilies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) getByFamily(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family parameter is required"})
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.GetByFamilyPaginated(family, page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) search(c *gin.Context) {
	q := c.Query("q")
	if strings.TrimSpace(q) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query (q) is required"})
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}
	data, err := h.service.SearchPaginated(q, page)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, response.NoItemFoundResponseMessage())
		} else {
			c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) listImageFamilies(c *gin.Context) {
	data, err := h.service.ListImageFamilies()
	if err != nil {
		slog.Error("Failed to list image families", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list image families", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) listFamilyImages(c *gin.Context) {
	family := c.Param("family")
	if strings.TrimSpace(family) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family parameter is required"})
		return
	}
	data, err := h.service.ListFamilyImages(family)
	if err != nil {
		slog.Error("Failed to list family images", "error", err, "family", family)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list images", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: data})
}

func (h *Handler) generateImageDownloadURL(c *gin.Context) {
	blobPath := c.Query("blob_path")
	result, err := h.service.GenerateImageDownloadURL(c.Request.Context(), blobPath)
	if err != nil {
		switch {
		case errors.Is(err, ErrImageNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found", "details": "The requested image does not exist"})
		case errors.Is(err, ErrEmptyBlobPath), errors.Is(err, ErrInvalidBlobPath), errors.Is(err, ErrInvalidFileType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL", "details": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, response.StandardResponse{Status: 200, Message: "", Data: result})
}
```

**Step 2: Register in `api/route/route.go`**

Add import `"miltechserver/api/docs_equipment"` and register under public routes after `eic.RegisterRoutes(...)`:

```go
docs_equipment.RegisterRoutes(docs_equipment.Dependencies{
	DB:         db,
	BlobClient: blobClient,
}, v1Route)
```

**Step 3: Verify compilation**

Run: `cd /Users/swisscheese/projects/miltechserver && go build ./...`

**Step 4: Commit**

```bash
git add api/docs_equipment/route.go api/route/route.go
git commit -m "feat(docs_equipment): add HTTP handlers and register routes"
```

---

## Task 5: In-Package Unit Tests

**Files:**
- Create: `api/docs_equipment/route_test.go`
- Create: `api/docs_equipment/service_impl_test.go`

**Step 1: Create `route_test.go`** — tests handler HTTP status codes using service stubs (follows pol_products test pattern).

Tests to include:
- `TestGetAllPaginatedSuccess` — stub returns data, expect 200
- `TestGetAllPaginatedError` — stub returns error, expect 500
- `TestGetAllPaginatedInvalidPage` — page=0, expect 400
- `TestGetFamiliesSuccess` — expect 200
- `TestGetByFamilySuccess` — expect 200
- `TestGetByFamilyEmpty` — blank family URL, expect 400 or 404
- `TestSearchMissingQuery` — no `q` param, expect 400
- `TestSearchSuccess` — expect 200

**Step 2: Create `service_impl_test.go`** — tests service delegates to repo correctly, trims input (follows EIC service test pattern).

Tests to include:
- `TestServiceTrimsSearchQuery` — verify whitespace trimmed
- `TestServiceTrimsFamily` — verify whitespace trimmed
- `TestServiceDelegatesToRepo` — verify repo called with correct args

**Step 3: Run unit tests**

Run: `cd /Users/swisscheese/projects/miltechserver && go test ./api/docs_equipment/... -v`
Expected: all tests PASS.

**Step 4: Commit**

```bash
git add api/docs_equipment/route_test.go api/docs_equipment/service_impl_test.go
git commit -m "test(docs_equipment): add handler and service unit tests"
```

---

## Task 6: Integration Tests

**Files:**
- Create: `tests/docs_equipment/main_test.go`
- Create: `tests/docs_equipment/helpers_test.go`
- Create: `tests/docs_equipment/handlers_test.go`

Follows the `tests/eic/` pattern exactly: `TestMain` connects to test DB, helpers provide router setup and JSON request helpers.

**Integration tests to include:**
- `TestEquipmentDetailsBlankParams` — invalid page, expect 400
- `TestEquipmentDetailsPaginated` — page 1, verify response shape
- `TestEquipmentFamilies` — verify families returned
- `TestEquipmentByFamily` — fetch first family, verify filtered results
- `TestEquipmentSearch` — search with a known model value
- `TestEquipmentInternalError` — bad DB connection, expect 500

**Run:** `cd /Users/swisscheese/projects/miltechserver && go test ./tests/docs_equipment/... -v`

**Commit:**
```bash
git add tests/docs_equipment/
git commit -m "test(docs_equipment): add integration tests"
```

---

## Task 7: Static Analysis + Final Verification

**Step 1: Run full test suite**

Run: `cd /Users/swisscheese/projects/miltechserver && go test ./api/docs_equipment/... ./tests/docs_equipment/... -v -count=1`

**Step 2: Run vet**

Run: `cd /Users/swisscheese/projects/miltechserver && go vet ./api/docs_equipment/...`

**Step 3: Build**

Run: `cd /Users/swisscheese/projects/miltechserver && go build ./...`

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(docs_equipment): complete docs equipment details feature"
```
