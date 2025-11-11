# Library Feature Implementation Plan

## Overview

This document provides a detailed implementation plan for the Library feature, which allows users to browse and download PMCS (Preventive Maintenance Checks and Services) and BII (Basic Issue Items) packets stored in Azure Blob Storage. The feature follows the established architectural patterns used in the MaterialImages feature.

**Status**: Planning Phase - No code changes yet

**Last Updated**: 2025-11-10

---

## Feature Requirements

### Primary Goal
Enable users to:
1. Browse available PMCS packets organized by vehicle type
2. Browse available BII packets organized by category
3. View lists of available documents within each category
4. Download selected PDF documents

### Initial Implementation Scope (Phase 1)
For the first implementation, we will create a **single endpoint** that:
- Queries Azure Blob Storage at: `https://miltechng.blob.core.windows.net/library/pmcs/`
- Returns a JSON list of all vehicle folders (prefixes) at that location
- Each folder name represents a vehicle that has a PMCS packet available

---

## Azure Blob Storage Structure

### Expected Blob Organization
```
library/                          (container)
├── pmcs/                        (prefix/folder)
│   ├── m1151/                   (vehicle folder)
│   │   ├── pmcs-m1151.pdf
│   │   └── pmcs-m1151-supplement.pdf
│   ├── m998/                    (vehicle folder)
│   │   └── pmcs-m998.pdf
│   └── m2-bradley/              (vehicle folder)
│       └── pmcs-m2.pdf
└── bii/                         (prefix/folder)
    ├── commo/                   (category folder)
    │   └── bii-commo-list.pdf
    └── medical/                 (category folder)
        └── bii-medical-list.pdf
```

### Azure SDK Usage
- **Container**: `library`
- **Method**: `Client.NewListBlobsHierarchyPager(delimiter, options)`
- **Delimiter**: `"/"` (to simulate folder structure)
- **Prefix**: `"pmcs/"` (for phase 1 PMCS vehicles endpoint)

---

## Architecture Analysis (Based on MaterialImages)

### Current MaterialImages Pattern
The MaterialImages feature demonstrates the following architecture:

1. **Controller Layer** (`api/controller/material_images_controller.go`)
   - Handles HTTP requests
   - Validates input parameters
   - Extracts user context from Gin context
   - Calls service layer methods
   - Returns JSON responses

2. **Service Layer**
   - **Interface** (`api/service/material_images_service.go`)
   - **Implementation** (`api/service/material_images_service_impl.go`)
   - Contains business logic
   - Interacts with Azure Blob Storage
   - Calls repository for database operations
   - Transforms data between domain models and response DTOs

3. **Repository Layer**
   - **Interface** (`api/repository/material_images_repository.go`)
   - **Implementation** (`api/repository/material_images_repository_impl.go`)
   - Handles all database operations
   - Uses go-jet for type-safe SQL queries
   - Returns database models

4. **Models**
   - **Request** (`api/request/material_images_request.go`) - Input validation structs
   - **Response** (`api/response/material_images_response.go`) - Output DTOs

5. **Routing** (`api/route/material_images_route.go`)
   - Initializes dependencies (repository, service, controller)
   - Registers routes with Gin router
   - Separates public vs authenticated routes

### Key Dependencies
```go
- github.com/gin-gonic/gin                           // HTTP framework
- github.com/Azure/azure-sdk-for-go/sdk/storage/azblob  // Azure Blob Storage
- firebase.google.com/go/v4/auth                     // Firebase authentication
- github.com/go-jet/jet/v2/postgres                  // Type-safe SQL (if DB needed)
- database/sql                                       // Database operations
```

---

## Library Feature Architecture

### File Structure
Following the established pattern, create these files:

```
api/
├── controller/
│   └── library_controller.go          (NEW)
├── service/
│   ├── library_service.go              (NEW - interface)
│   └── library_service_impl.go         (NEW - implementation)
├── repository/
│   ├── library_repository.go           (NEW - interface, if DB needed later)
│   └── library_repository_impl.go      (NEW - implementation, if DB needed later)
├── request/
│   └── library_request.go              (NEW)
├── response/
│   └── library_response.go             (NEW)
└── route/
    └── library_route.go                (NEW)
```

**Note**: For Phase 1, the repository layer may not be needed since we're only querying Azure Blob Storage. However, we'll create the scaffolding to maintain consistency with the project architecture.

---

## Detailed Implementation Specification

### 1. Response Models (`api/response/library_response.go`)

```go
package response

// VehicleFolderResponse represents a vehicle folder in the PMCS library
type VehicleFolderResponse struct {
    Name        string `json:"name"`          // Vehicle name (folder name without prefix)
    FullPath    string `json:"full_path"`     // Full blob prefix (e.g., "pmcs/m1151/")
    DisplayName string `json:"display_name"`  // Human-readable name (e.g., "M1151")
}

// PMCSVehiclesResponse is the response for listing available PMCS vehicles
type PMCSVehiclesResponse struct {
    Vehicles []VehicleFolderResponse `json:"vehicles"`
    Count    int                     `json:"count"`
}

// Future: Document listing response
type DocumentResponse struct {
    Name         string `json:"name"`           // File name
    BlobPath     string `json:"blob_path"`      // Full blob path
    SizeBytes    int64  `json:"size_bytes"`     // File size
    LastModified string `json:"last_modified"`  // ISO 8601 timestamp
    DownloadURL  string `json:"download_url"`   // Temporary download URL
}

// Future: Documents list response
type DocumentsListResponse struct {
    VehicleName string             `json:"vehicle_name"`
    Documents   []DocumentResponse `json:"documents"`
    Count       int                `json:"count"`
}
```

### 2. Request Models (`api/request/library_request.go`)

```go
package request

// Currently no request validation needed for GET endpoint
// Future: Add pagination, filtering parameters
```

### 3. Service Interface (`api/service/library_service.go`)

```go
package service

import (
    "miltechserver/api/response"
)

type LibraryService interface {
    // GetPMCSVehicles returns a list of all vehicle folders in the PMCS library
    GetPMCSVehicles() (*response.PMCSVehiclesResponse, error)

    // Future endpoints:
    // GetPMCSDocuments(vehicleName string) (*response.DocumentsListResponse, error)
    // GetBIICategories() (*response.CategoriesResponse, error)
    // GetBIIDocuments(category string) (*response.DocumentsListResponse, error)
    // GenerateDownloadURL(blobPath string) (string, error)
}
```

### 4. Service Implementation (`api/service/library_service_impl.go`)

```go
package service

import (
    "context"
    "fmt"
    "strings"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

    "miltechserver/api/response"
    "miltechserver/bootstrap"
)

const (
    LibraryContainerName = "library"
    PMCSPrefix          = "pmcs/"
)

type LibraryServiceImpl struct {
    blobClient *azblob.Client
    env        *bootstrap.Env
}

func NewLibraryServiceImpl(
    blobClient *azblob.Client,
    env *bootstrap.Env,
) LibraryService {
    return &LibraryServiceImpl{
        blobClient: blobClient,
        env:        env,
    }
}

func (s *LibraryServiceImpl) GetPMCSVehicles() (*response.PMCSVehiclesResponse, error) {
    ctx := context.Background()

    // Create pager with delimiter to get hierarchical listing
    pager := s.blobClient.NewListBlobsHierarchyPager(
        LibraryContainerName,
        "/", // delimiter for folder simulation
        &azblob.ListBlobsHierarchyOptions{
            Prefix: &PMCSPrefix,
        },
    )

    vehicles := []response.VehicleFolderResponse{}

    // Iterate through pages
    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to list PMCS vehicles: %w", err)
        }

        // BlobPrefixes represent "folders"
        for _, prefix := range page.Segment.BlobPrefixes {
            if prefix.Name == nil {
                continue
            }

            fullPath := *prefix.Name
            // Extract vehicle name from path (e.g., "pmcs/m1151/" -> "m1151")
            vehicleName := strings.TrimPrefix(fullPath, PMCSPrefix)
            vehicleName = strings.TrimSuffix(vehicleName, "/")

            // Create display name (capitalize, replace hyphens with spaces)
            displayName := formatDisplayName(vehicleName)

            vehicles = append(vehicles, response.VehicleFolderResponse{
                Name:        vehicleName,
                FullPath:    fullPath,
                DisplayName: displayName,
            })
        }
    }

    return &response.PMCSVehiclesResponse{
        Vehicles: vehicles,
        Count:    len(vehicles),
    }, nil
}

// Helper function to format vehicle name for display
func formatDisplayName(name string) string {
    // Convert "m1151" -> "M1151", "m2-bradley" -> "M2 Bradley"
    display := strings.ToUpper(name)
    display = strings.ReplaceAll(display, "-", " ")
    return display
}
```

### 5. Repository Interface (`api/repository/library_repository.go`)

```go
package repository

// LibraryRepository handles database operations for library feature
// Note: Currently not needed for Phase 1 as we only query Azure Blob Storage
// This is scaffolding for future features like tracking downloads, favorites, etc.
type LibraryRepository interface {
    // Future: Track user downloads, favorites, etc.
    // RecordDownload(userID string, documentPath string) error
    // GetUserDownloadHistory(userID string) ([]DownloadRecord, error)
}
```

### 6. Repository Implementation (`api/repository/library_repository_impl.go`)

```go
package repository

import (
    "database/sql"
)

type LibraryRepositoryImpl struct {
    db *sql.DB
}

func NewLibraryRepositoryImpl(db *sql.DB) LibraryRepository {
    return &LibraryRepositoryImpl{
        db: db,
    }
}

// Future implementations will go here
```

### 7. Controller (`api/controller/library_controller.go`)

```go
package controller

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "miltechserver/api/service"
)

type LibraryController struct {
    LibraryService service.LibraryService
}

func NewLibraryController(libraryService service.LibraryService) *LibraryController {
    return &LibraryController{
        LibraryService: libraryService,
    }
}

// GetPMCSVehicles returns a list of all available PMCS vehicle folders
// GET /api/v1/library/pmcs/vehicles
func (controller *LibraryController) GetPMCSVehicles(c *gin.Context) {
    vehicles, err := controller.LibraryService.GetPMCSVehicles()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to retrieve PMCS vehicles",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, vehicles)
}
```

### 8. Routing (`api/route/library_route.go`)

```go
package route

import (
    "database/sql"

    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"

    "miltechserver/api/controller"
    "miltechserver/api/repository"
    "miltechserver/api/service"
    "miltechserver/bootstrap"
)

func NewLibraryRouter(
    db *sql.DB,
    blobClient *azblob.Client,
    env *bootstrap.Env,
    group *gin.RouterGroup,
) {
    // Initialize repository (currently unused but follows pattern)
    repo := repository.NewLibraryRepositoryImpl(db)
    _ = repo // Silence unused variable warning

    // Initialize service (no repository dependency for Phase 1)
    svc := service.NewLibraryServiceImpl(blobClient, env)

    // Initialize controller
    ctrl := controller.NewLibraryController(svc)

    // Public routes (no authentication required)
    group.GET("/library/pmcs/vehicles", ctrl.GetPMCSVehicles)

    // Future routes:
    // group.GET("/library/pmcs/:vehicle/documents", ctrl.GetPMCSDocuments)
    // group.GET("/library/bii/categories", ctrl.GetBIICategories)
    // group.GET("/library/bii/:category/documents", ctrl.GetBIIDocuments)
    // authGroup.GET("/library/download/:path", ctrl.GenerateDownloadURL) // Authenticated
}
```

### 9. Route Registration (`api/route/route.go`)

**Modification Required**: Add library router to the Setup function

```go
// In route.go Setup function, add:
NewLibraryRouter(db, blobClient, env, v1Route)
```

**Location**: After line 29, with other public routes

---

## Implementation Steps

### Phase 1: Basic Scaffold & PMCS Vehicles Endpoint

1. **Create Response Models**
   - File: `api/response/library_response.go`
   - Define `VehicleFolderResponse` and `PMCSVehiclesResponse`

2. **Create Request Models**
   - File: `api/request/library_request.go`
   - Empty for now, placeholder for future endpoints

3. **Create Service Interface**
   - File: `api/service/library_service.go`
   - Define `LibraryService` interface with `GetPMCSVehicles()` method

4. **Create Service Implementation**
   - File: `api/service/library_service_impl.go`
   - Implement `GetPMCSVehicles()` using Azure Blob SDK
   - Use `NewListBlobsHierarchyPager` with delimiter "/"
   - Parse BlobPrefixes to extract vehicle names

5. **Create Repository Scaffold**
   - File: `api/repository/library_repository.go` (interface)
   - File: `api/repository/library_repository_impl.go` (implementation)
   - Empty implementations for future use

6. **Create Controller**
   - File: `api/controller/library_controller.go`
   - Implement `GetPMCSVehicles` handler
   - Handle errors and return appropriate HTTP status codes

7. **Create Routing**
   - File: `api/route/library_route.go`
   - Initialize dependencies
   - Register GET `/api/v1/library/pmcs/vehicles` endpoint

8. **Register Router**
   - Modify: `api/route/route.go`
   - Add `NewLibraryRouter(db, blobClient, env, v1Route)` call

9. **Testing**
   - Manual testing with Postman/curl
   - Verify JSON response structure
   - Test error handling

10. **Documentation**
    - Update this document with any implementation findings
    - Create API documentation in docs/features/

---

## API Endpoint Specifications

### GET /api/v1/library/pmcs/vehicles

**Description**: Returns a list of all vehicle folders available in the PMCS library.

**Authentication**: Public (no auth required)

**Request**: None

**Response**: 200 OK
```json
{
  "vehicles": [
    {
      "name": "m1151",
      "full_path": "pmcs/m1151/",
      "display_name": "M1151"
    },
    {
      "name": "m998",
      "full_path": "pmcs/m998/",
      "display_name": "M998"
    },
    {
      "name": "m2-bradley",
      "full_path": "pmcs/m2-bradley/",
      "display_name": "M2 BRADLEY"
    }
  ],
  "count": 3
}
```

**Error Response**: 500 Internal Server Error
```json
{
  "error": "Failed to retrieve PMCS vehicles",
  "details": "detailed error message"
}
```

---

## Future Enhancements (Phase 2+)

### Additional Endpoints

1. **GET /api/v1/library/pmcs/:vehicle/documents**
   - List all documents for a specific vehicle
   - Return file names, sizes, last modified dates

2. **GET /api/v1/library/bii/categories**
   - List all BII categories (similar to PMCS vehicles)

3. **GET /api/v1/library/bii/:category/documents**
   - List all documents in a BII category

4. **GET /api/v1/library/download/:path** (Authenticated)
   - Generate temporary SAS URL for document download
   - Track download history for analytics

5. **POST /api/v1/library/favorites** (Authenticated)
   - Allow users to save favorite documents
   - Requires database table for user_library_favorites

### Database Schema (Future)

```sql
CREATE TABLE library_downloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    document_path TEXT NOT NULL,
    downloaded_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE INDEX idx_library_downloads_user ON library_downloads(user_id);

CREATE TABLE library_favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    document_path TEXT NOT NULL,
    favorited_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, document_path),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);
```

### Features to Consider

- **Search functionality**: Search across document names
- **Filtering**: Filter by vehicle type, category
- **Pagination**: For large document lists
- **Caching**: Cache folder listings to reduce Azure API calls
- **Analytics**: Track most downloaded documents
- **Versioning**: Handle multiple versions of documents

---

## Testing Strategy

### Unit Tests
- Test service layer logic
- Mock Azure Blob Client
- Verify path parsing and formatting

### Integration Tests
- Test actual Azure Blob Storage connectivity
- Verify hierarchical listing works correctly
- Test error handling for missing containers/prefixes

### Manual Testing
```bash
# Test PMCS vehicles endpoint
curl http://localhost:8080/api/v1/library/pmcs/vehicles

# Expected response format:
# {"vehicles":[{"name":"vehicle1","full_path":"pmcs/vehicle1/","display_name":"VEHICLE1"}],"count":1}
```

---

## Error Handling

### Expected Errors
1. **Azure Blob Storage connectivity issues**
   - Return 500 with "Failed to retrieve PMCS vehicles"
   - Log detailed error for debugging

2. **Container/prefix not found**
   - Return empty vehicles array with count 0
   - Don't error on empty results

3. **Permission issues**
   - Return 500 with appropriate error message
   - Verify blob storage credentials in environment

### Error Response Format
```json
{
  "error": "Human-readable error message",
  "details": "Technical details for debugging"
}
```

---

## Security Considerations

### Phase 1
- Public endpoint (no auth required)
- Read-only access to blob storage
- No user data exposure

### Future Phases
- Implement authentication for download endpoints
- Generate time-limited SAS URLs for downloads
- Track download history per user
- Implement rate limiting if needed

---

## Performance Considerations

### Azure Blob Storage
- Hierarchical listing is more expensive than flat listing
- Consider caching folder listings (Redis or in-memory)
- Pagination may be needed for large libraries

### Response Size
- Phase 1 response should be small (< 100 KB)
- Future document listings may need pagination
- Consider compression for large responses

---

## Dependencies

### Required Packages (Already in project)
- `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`
- `github.com/gin-gonic/gin`
- `miltechserver/bootstrap`

### Environment Variables (Already configured)
- `BLOB_ACCOUNT_NAME`: Azure storage account name
- `BLOB_ACCOUNT_KEY`: Azure storage account key

---

## Rollout Plan

### Phase 1: MVP (Current)
- Single endpoint: GET /library/pmcs/vehicles
- Basic error handling
- JSON response with vehicle list

### Phase 2: Document Listing
- Add document listing endpoints
- Include file metadata (size, modified date)
- Implement basic filtering

### Phase 3: Download & Authentication
- Implement authenticated download endpoint
- Generate SAS URLs for secure downloads
- Add download tracking

### Phase 4: User Features
- Favorites functionality
- Download history
- Search and advanced filtering

---

## Success Criteria

### Phase 1 Completion
- [x] All scaffold files created following project patterns
- [ ] Endpoint returns correct JSON structure
- [ ] Error handling works properly
- [ ] Integration with existing route setup
- [ ] Manual testing successful
- [ ] Documentation updated

---

## Notes & Considerations

### Azure Blob Storage Hierarchy
- Blob storage is flat; folders are simulated using prefixes
- Delimiter "/" is used to create folder-like structure
- `NewListBlobsHierarchyPager` returns `BlobPrefixes` for "folders"
- Actual files are returned as `BlobItems`

### Naming Conventions
- Follow existing patterns: snake_case for files, PascalCase for types
- Use descriptive names: `VehicleFolderResponse` not `VehicleResponse`
- Keep consistency with MaterialImages implementation

### Code Quality
- Follow Go best practices
- Use error wrapping: `fmt.Errorf("message: %w", err)`
- Document exported functions and types
- Keep functions small and focused

---

## References

### Project Files Referenced
- `api/controller/material_images_controller.go` - Controller pattern
- `api/service/material_images_service_impl.go` - Service implementation pattern
- `api/repository/material_images_repository.go` - Repository interface pattern
- `api/route/material_images_route.go` - Route registration pattern
- `api/route/route.go` - Main route setup
- `bootstrap/azure_blob.go` - Azure Blob Client initialization

### External Documentation
- [Azure SDK for Go - List Blobs Hierarchy](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob)
- [Gin Web Framework Documentation](https://gin-gonic.com/docs/)
- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)

---

## Change Log

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-11-10 | 1.0 | Initial implementation plan created | Claude |

