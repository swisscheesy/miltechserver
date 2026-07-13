# Docs Equipment Details Feature Design

## Overview

Add public endpoints for querying equipment details from the `docs_equipment_details` Postgres table (488 rows, 12 columns) and browsing/downloading equipment images from Azure Blob Storage at `library/docs_equipment/images/{family}/`.

All endpoints are **public** — no authentication required.

---

## Database Schema

The `docs_equipment_details` table:

| Column | Type | Nullable |
|--------|------|----------|
| id | bigint (PK) | no |
| model | text | yes |
| lin | text | yes |
| mode | text | yes |
| description | text | yes |
| length | text | yes |
| width | text | yes |
| height | text | yes |
| weight | text | yes |
| kw | integer | yes |
| hz | text | yes |
| family | text | yes |

- **488 rows**, **16 unique family values**
- Generated Jet model/table already exist at `.gen/miltech_ng/public/model/docs_equipment_details.go` and `.gen/miltech_ng/public/table/docs_equipment_details.go`

---

## API Endpoints

### Data Endpoints (Postgres)

All data endpoints use the `response.StandardResponse` wrapper.

#### 1. `GET /api/v1/equipment-details?page=1`
**Paginated list of all equipment details**, ordered by `id`.

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| page | query int | 1 | Page number (1-indexed) |

Response body → `EquipmentDetailsPageResponse` (same pagination shape as EIC):
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [ { "id": 1, "model": "AH-1W", "lin": null, ... } ],
    "count": 40,
    "page": 1,
    "total_pages": 13,
    "is_last_page": false
  }
}
```

Page size: **40 items** (consistent with EIC).

#### 2. `GET /api/v1/equipment-details/families`
**List all unique family values.** Returns a flat array of strings.

Response:
```json
{
  "status": 200,
  "message": "",
  "data": {
    "families": ["aircraft", "armored wheeled vehicles", ...],
    "count": 16
  }
}
```

#### 3. `GET /api/v1/equipment-details/family/:family?page=1`
**Paginated list filtered by family value.**

| Param | Type | Description |
|-------|------|-------------|
| family | path string | Family name (URL-encoded, e.g. `armored%20wheeled%20vehicles`) |
| page | query int | Page number |

Response → same `EquipmentDetailsPageResponse` shape as endpoint 1.

#### 4. `GET /api/v1/equipment-details/search?q=AH-64&page=1`
**Search by model or LIN** (case-insensitive `ILIKE`).

| Param | Type | Description |
|-------|------|-------------|
| q | query string | Search term (matched against `model` and `lin`) |
| page | query int | Page number |

Response → same `EquipmentDetailsPageResponse` shape.

---

### Image Endpoints (Azure Blob Storage)

Follows the **PMCS vehicles/documents** pattern exactly.

#### 5. `GET /api/v1/equipment-details/images/families`
**List image family folders** from Azure Blob at `docs_equipment/images/`.

Uses `NewListBlobsHierarchyPager("/")` with prefix `docs_equipment/images/`.

Response:
```json
{
  "status": 200,
  "message": "",
  "data": {
    "families": [
      { "name": "aircraft", "full_path": "docs_equipment/images/aircraft/", "display_name": "AIRCRAFT" }
    ],
    "count": 16
  }
}
```

#### 6. `GET /api/v1/equipment-details/images/family/:family`
**List images in a family folder** from Azure Blob.

Uses `NewListBlobsFlatPager` with prefix `docs_equipment/images/{family}/`. Filters to image extensions only (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`).

Response:
```json
{
  "status": 200,
  "message": "",
  "data": {
    "family": "aircraft",
    "images": [
      {
        "name": "AH-64D.jpg",
        "blob_path": "docs_equipment/images/aircraft/AH-64D.jpg",
        "size_bytes": 124567,
        "last_modified": "2026-01-15T10:30:00Z"
      }
    ],
    "count": 5
  }
}
```

#### 7. `GET /api/v1/equipment-details/images/download?blob_path=docs_equipment/images/aircraft/AH-64D.jpg`
**Generate a SAS download URL** for an equipment image. Rate-limited.

Uses `shared.GenerateBlobSASURL` — same pattern as PMCS and ps-mag download endpoints. Only allows paths starting with `docs_equipment/images/` and ending with supported image extensions.

Response:
```json
{
  "status": 200,
  "message": "",
  "data": {
    "blob_path": "docs_equipment/images/aircraft/AH-64D.jpg",
    "download_url": "https://...?sv=...",
    "expires_at": "2026-03-02T22:00:00Z"
  }
}
```

---

## Architecture

Follows the existing **route → service → repository** pattern.

### Package: `api/docs_equipment`

New package under `api/` with the standard file structure:

| File | Purpose |
|------|---------|
| `route.go` | Dependencies struct, Handler struct, RegisterRoutes, HTTP handlers |
| `repository.go` | Repository interface (DB queries) |
| `repository_impl.go` | Repository implementation using raw SQL with go-jet model |
| `service.go` | Service interface (DB + Blob operations) |
| `service_impl.go` | Service implementation (delegates to repo for DB, blob client for images) |
| `response.go` | Response structs (EquipmentDetailsPageResponse, FamiliesResponse, ImageFamiliesResponse, etc.) |
| `errors.go` | Sentinel errors |

### Dependencies

```go
type Dependencies struct {
    DB         *sql.DB
    BlobClient *azblob.Client
}
```

Registered in `api/route/route.go` under the **public routes** section alongside `pol_products` and `eic`:

```go
docs_equipment.RegisterRoutes(docs_equipment.Dependencies{
    DB:         db,
    BlobClient: blobClient,
}, v1Route)
```

### Data Flow

```
HTTP Request → route.go (handler) → service_impl.go → repository_impl.go → Postgres
                                   → service_impl.go → Azure Blob Client → Azure Storage
```

- Data queries: route → service → repository (go-jet model + raw SQL for pagination)
- Image listing: route → service → Azure Blob `NewListBlobsHierarchyPager` / `NewListBlobsFlatPager`
- Image download: route → service → `shared.GenerateBlobSASURL`

---

## Key Design Decisions

1. **Page size of 40** — consistent with EIC pagination
2. **Raw SQL for DB queries** — the EIC feature uses raw SQL for complex queries; we follow the same approach since go-jet is used for simple SELECT-all patterns (like pol_products) while raw SQL is used for pagination/filtering
3. **SAS URL for images** — follows the established PMCS/ps-mag pattern, no server-side proxying
4. **Image extension whitelist** — `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp` to prevent unauthorized file access
5. **Rate limiting on download endpoint** — same `middleware.RateLimiter()` as library download
6. **Family as path param for filter** — allows clean URLs like `/equipment-details/family/aircraft`
7. **Search uses ILIKE** — case-insensitive partial match on both `model` and `lin` columns

---

## Error Handling

Following existing conventions:

| Scenario | HTTP Status | Response |
|----------|-------------|----------|
| Invalid page number | 400 | `{"error": "Invalid page number"}` |
| Empty search query | 400 | `{"error": "Search query is required"}` |
| Empty family param | 400 | `{"error": "Family parameter is required"}` |
| No results found | 404 | `response.NoItemFoundResponseMessage()` |
| DB/Blob error | 500 | `response.InternalErrorResponseMessage()` |
| Invalid blob path | 400 | Descriptive error |
| Blob not found | 404 | Not found error |
