# POL Products Feature Design Document

**Date**: 2026-02-09
**Author**: System Design
**Status**: Design Phase
**Feature**: Public POL Products Lookup

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Current Implementation Analysis](#current-implementation-analysis)
3. [Requirements](#requirements)
4. [Proposed Solution](#proposed-solution)
5. [Technical Design](#technical-design)
6. [API Specification](#api-specification)
7. [Implementation Plan](#implementation-plan)
8. [Testing Strategy](#testing-strategy)
9. [Risks and Tradeoffs](#risks-and-tradeoffs)
10. [Future Considerations](#future-considerations)

---

## Executive Summary

This design introduces a single public endpoint to return the full contents of the `pol_products` table (Petroleum, Oils, and Lubricants product reference data). The table contains approximately 239 rows of static reference data with military specification, NIIN, container size, and symbol information.

The feature follows the existing flat bounded context pattern established by `quick_lists`, using the standard layered architecture (route → service → repository) with Jet for database access and the standard response wrapper.

---

## Current Implementation Analysis

### Existing Patterns

The project uses bounded context directories under `api/`. For simple, single-purpose features, all files live flat in one directory:

```
api/quick_lists/
├── repository.go          # Repository interface
├── repository_impl.go     # Implementation with Jet queries
├── response.go            # Domain-specific response struct
├── route.go               # Dependencies, Handler, RegisterRoutes, handlers
├── route_test.go          # Handler-level tests with service stubs
├── service.go             # Service interface
├── service_impl.go        # Service implementation
└── service_impl_test.go   # Service-level tests with repo stubs
```

Key conventions observed:
- **Dependency injection**: A `Dependencies` struct holds `*sql.DB` (and other deps as needed). `RegisterRoutes(deps, router)` wires everything up.
- **Route registration**: Public routes are registered on `v1Route` in `api/route/route.go` without auth middleware.
- **Response wrapping**: All handlers return `response.StandardResponse{Status, Message, Data}` where `Data` contains a domain-specific struct with a typed data slice and a `count` field.
- **Error responses**: `response.InternalErrorResponseMessage()` for 500s.
- **Jet usage**: Queries use generated table/model types from `.gen/miltech_ng/public/`.

### Relevant Generated Models

**Model** (`.gen/miltech_ng/public/model/pol_products.go`):
```go
type PolProducts struct {
    FigureHeader   string `json:"figure_header"`
    Specification  string `sql:"primary_key" json:"specification"`
    MilitarySymbol string `json:"military_symbol"`
    ContainerSize  string `json:"container_size"`
    Niin           string `sql:"primary_key" json:"niin"`
}
```

**Table** (`.gen/miltech_ng/public/table/pol_products.go`):
- `table.PolProducts` with `AllColumns`, `MutableColumns`, `DefaultColumns`
- Columns: `FigureHeader`, `Specification`, `MilitarySymbol`, `ContainerSize`, `Niin`

### Database Schema

| Column           | Type | Nullable | Notes                        |
|------------------|------|----------|------------------------------|
| figure_header    | text | No       | Product category grouping    |
| specification    | text | No       | Military spec (composite PK) |
| military_symbol  | text | No       | Military symbol designation  |
| container_size   | text | No       | Container size description   |
| niin             | text | No       | NSN NIIN (composite PK)      |

- **Primary Key**: `(specification, niin)`
- **Row Count**: ~239 rows (small, static reference data)

---

## Requirements

### Functional
1. A single GET endpoint that returns all rows from the `pol_products` table.
2. The endpoint is public (no authentication required).
3. The response must use the standard response structure with a typed data array and count.

### Non-Functional
- No pagination required (~239 rows, small static dataset).
- No rate-limiting constraints for this endpoint.
- No input parameters to validate.

---

## Proposed Solution

### Endpoint
- `GET /api/v1/pol-products`

### Architecture
New flat bounded context directory following the `quick_lists` pattern:

```
api/pol_products/
├── repository.go          # Repository interface
├── repository_impl.go     # Jet query implementation
├── response.go            # PolProductsResponse struct
├── route.go               # Dependencies, Handler, RegisterRoutes, handlers
├── route_test.go          # Handler tests with service stub
├── service.go             # Service interface
├── service_impl.go        # Service implementation
└── service_impl_test.go   # Service tests with repo stub
```

Request flow:
```
GET /api/v1/pol-products
  → route.go (Handler.getPolProducts)
    → service_impl.go (ServiceImpl.GetPolProducts)
      → repository_impl.go (RepositoryImpl.GetPolProducts)
        → Jet SELECT from table.PolProducts
```

---

## Technical Design

### 1. Response Structure (`response.go`)

```go
package pol_products

import "miltechserver/.gen/miltech_ng/public/model"

type PolProductsResponse struct {
    Products []model.PolProducts `json:"products"`
    Count    int                 `json:"count"`
}
```

Follows the established pattern where domain-specific response structs include a typed slice and a count (identical to `QuickListsClothingResponse`, `QuickListsWheelsResponse`, etc.).

### 2. Repository Layer

**Interface** (`repository.go`):
```go
package pol_products

type Repository interface {
    GetPolProducts() (PolProductsResponse, error)
}
```

**Implementation** (`repository_impl.go`):
```go
package pol_products

import (
    "database/sql"

    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/.gen/miltech_ng/public/table"

    . "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
    return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetPolProducts() (PolProductsResponse, error) {
    var products []model.PolProducts
    stmt := SELECT(
        table.PolProducts.AllColumns,
    ).FROM(table.PolProducts)

    if err := stmt.Query(repo.db, &products); err != nil {
        return PolProductsResponse{}, err
    }

    return PolProductsResponse{
        Products: products,
        Count:    len(products),
    }, nil
}
```

### 3. Service Layer

**Interface** (`service.go`):
```go
package pol_products

type Service interface {
    GetPolProducts() (PolProductsResponse, error)
}
```

**Implementation** (`service_impl.go`):
```go
package pol_products

type ServiceImpl struct {
    repo Repository
}

func NewService(repo Repository) Service {
    return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) GetPolProducts() (PolProductsResponse, error) {
    return service.repo.GetPolProducts()
}
```

### 4. Route Layer (`route.go`)

```go
package pol_products

import (
    "database/sql"
    "net/http"

    "github.com/gin-gonic/gin"

    "miltechserver/api/response"
)

type Dependencies struct {
    DB *sql.DB
}

type Handler struct {
    service Service
}

func RegisterRoutes(deps Dependencies, router *gin.RouterGroup) {
    repo := NewRepository(deps.DB)
    svc := NewService(repo)
    registerHandlers(router, svc)
}

func registerHandlers(router *gin.RouterGroup, svc Service) {
    handler := Handler{service: svc}
    router.GET("/pol-products", handler.getPolProducts)
}

func (handler *Handler) getPolProducts(c *gin.Context) {
    data, err := handler.service.GetPolProducts()
    if err != nil {
        c.JSON(http.StatusInternalServerError, response.InternalErrorResponseMessage())
        return
    }

    c.JSON(http.StatusOK, response.StandardResponse{
        Status:  http.StatusOK,
        Message: "",
        Data:    data,
    })
}
```

### 5. Route Registration (`api/route/route.go`)

Add to the "All Public Routes" section:

```go
import "miltechserver/api/pol_products"

// In Setup(), under "All Public Routes":
pol_products.RegisterRoutes(pol_products.Dependencies{DB: db}, v1Route)
```

### 6. Error Handling

| Scenario         | HTTP Status | Response                                  |
|------------------|-------------|-------------------------------------------|
| Success          | 200         | `StandardResponse` with `PolProductsResponse` |
| Database error   | 500         | `response.InternalErrorResponseMessage()` |

Since this endpoint returns all rows with no input parameters, there is no 400 (bad request) or 404 (not found) case. An empty result set is still a valid 200 response with `count: 0`.

---

## API Specification

### POL Products Lookup

**Endpoint:** `GET /api/v1/pol-products`

**Authentication:** None (public)

**Parameters:** None

**Response (200):**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "products": [
      {
        "figure_header": "COMBAT/TACTICAL ENGINE OILS",
        "specification": "MIL-PRF-2104",
        "military_symbol": "OE/HDO-15/40",
        "container_size": "1-QT",
        "niin": "01-421-1427"
      },
      {
        "figure_header": "COMBAT/TACTICAL ENGINE OILS",
        "specification": "MIL-PRF-2104",
        "military_symbol": "OE/HDO-15/40 (SAE 15W-40)",
        "container_size": "1-QT",
        "niin": "01-438-6076"
      }
    ],
    "count": 239
  }
}
```

**Response (500):**
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

## Implementation Plan

1. Create the `api/pol_products/` directory.
2. Create `response.go` with the `PolProductsResponse` struct.
3. Create `repository.go` (interface) and `repository_impl.go` (Jet query).
4. Create `service.go` (interface) and `service_impl.go` (pass-through).
5. Create `route.go` with `Dependencies`, `Handler`, `RegisterRoutes`, and handler method.
6. Register the route in `api/route/route.go` under the public routes section.
7. Create `route_test.go` and `service_impl_test.go` with stub-based tests.
8. Run tests and verify endpoint against the database.

---

## Testing Strategy

### Unit Tests

**Service Layer** (`service_impl_test.go`):
- Stub the `Repository` interface.
- Test that `GetPolProducts()` returns data from the repo on success.
- Test that `GetPolProducts()` propagates errors from the repo.

**Handler Layer** (`route_test.go`):
- Stub the `Service` interface.
- Test `GET /pol-products` returns 200 with valid data.
- Test `GET /pol-products` returns 500 when service returns an error.

### Manual Verification
- Hit `GET /api/v1/pol-products` and verify:
  - Response matches the expected JSON structure.
  - `count` matches the actual number of items in the `products` array.
  - All 239 rows are present.
  - All fields are non-null strings in every row.

---

## Risks and Tradeoffs

- **Full table return**: Returning all 239 rows in a single response is acceptable for this dataset size. The response payload is approximately 25-30 KB, well within normal limits.
- **No filtering**: This design intentionally omits filtering/search capabilities. The table is small enough that client-side filtering is practical and avoids unnecessary server complexity.
- **No caching**: No server-side caching is implemented. The data is static reference data that changes only with database updates. If needed, HTTP cache headers could be added later.

---

## Future Considerations

- Add HTTP cache headers (`Cache-Control`, `ETag`) if the endpoint sees high traffic, since the underlying data only changes with database refreshes.
- Add optional query parameters for filtering by `figure_header` or `specification` if client-side filtering proves insufficient.
- Consider grouping the response by `figure_header` if the frontend needs categorized views.
