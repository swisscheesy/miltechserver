# Equipment Details API — Mobile Integration Guide

**Base URL:** `https://<host>/api/v1`
**Authentication:** None required — all endpoints are public
**Content-Type:** `application/json`

---

## Overview

The Equipment Details API provides two groups of endpoints:

1. **Data Endpoints** — query equipment details from the database (dimensions, weight, model, LIN, family classification)
2. **Image Endpoints** — browse and download equipment images from blob storage, organized by family

Equipment data covers military vehicles, aircraft, generators, trailers, and other items. There are ~488 records across 16 equipment families.

---

## Data Endpoints

### 1. List All Equipment (Paginated)

Returns a paginated list of all equipment details, ordered by ID.

**`GET /equipment-details`**

#### Query Parameters

| Parameter | Type    | Required | Default | Description |
|-----------|---------|----------|---------|-------------|
| `page`    | integer | No       | `1`     | Page number. Must be ≥ 1. |

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "id": 1,
        "model": "AH-1W",
        "lin": null,
        "mode": "REDUCED",
        "description": "HELICOPTER ATTACK",
        "length": "546",
        "width": "128",
        "height": "165",
        "weight": "10200",
        "kw": null,
        "hz": null,
        "family": "aircraft"
      },
      {
        "id": 2,
        "model": "AH-1W",
        "lin": null,
        "mode": "FLYAWAY",
        "description": "HELICOPTER ATTACK",
        "length": "546",
        "width": "576",
        "height": "165",
        "weight": "14750",
        "kw": null,
        "hz": null,
        "family": "aircraft"
      }
    ],
    "count": 40,
    "page": 1,
    "total_pages": 13,
    "is_last_page": false
  }
}
```

#### Item Object Fields

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | integer | No | Unique row identifier |
| `model` | string | Yes | Equipment model designation (e.g. `"AH-64D"`, `"M1037"`) |
| `lin` | string | Yes | Line Item Number (e.g. `"H48918"`, `"R14216"`) |
| `mode` | string | Yes | Configuration mode (e.g. `"REDUCED"`, `"FLYAWAY"`) |
| `description` | string | Yes | Equipment description (e.g. `"HELICOPTER ATTACK"`) |
| `length` | string | Yes | Length in inches |
| `width` | string | Yes | Width in inches |
| `height` | string | Yes | Height in inches |
| `weight` | string | Yes | Weight in pounds |
| `kw` | integer | Yes | Power output in kilowatts (generators only) |
| `hz` | string | Yes | Frequency in hertz (generators only) |
| `family` | string | Yes | Equipment family classification |

#### Pagination Fields

| Field | Type | Description |
|-------|------|-------------|
| `count` | integer | Number of items on this page (≤ 40) |
| `page` | integer | Current page number |
| `total_pages` | integer | Total pages available |
| `is_last_page` | boolean | `true` if this is the final page |

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `page` is 0 or negative | `400` | `{"error": "Invalid page number"}` |
| `page` is not a valid integer | `400` | `{"error": "Invalid page number"}` |
| No equipment found (beyond last page) | `404` | `{"status": 404, "data": null, "message": "no item(s) found"}` |
| Database error | `500` | `{"status": 500, "data": null, "message": "internal Server Error"}` |

---

### 2. List Families

Returns all unique equipment family values. Use these values to populate a filter dropdown or category selector.

**`GET /equipment-details/families`**

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "families": [
      "aircraft",
      "armored wheeled vehicles",
      "armor tracked vehicles",
      "containers",
      "cranes",
      "engineer equipment",
      "heavy trailers",
      "light trailers",
      "material handling equipment",
      "miscellaneous equipment",
      "pallets and flatracks",
      "semitrailers",
      "towed artillery weapons",
      "trailer mounted generators",
      "trucks",
      "watercraft"
    ],
    "count": 16
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `families` | array of strings | Sorted alphabetically, lowercase |
| `count` | integer | Number of unique families |

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| Database error | `500` | `{"status": 500, "data": null, "message": "internal Server Error"}` |

---

### 3. Filter by Family

Returns a paginated list of equipment filtered to a specific family.

**`GET /equipment-details/family/:family`**

#### Path Parameters

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `family`  | string | Yes      | Family name. Must match a value from the families endpoint. URL-encode spaces (e.g. `armored%20wheeled%20vehicles`). Matching is **case-insensitive**. |

#### Query Parameters

| Parameter | Type    | Required | Default | Description |
|-----------|---------|----------|---------|-------------|
| `page`    | integer | No       | `1`     | Page number. Must be ≥ 1. |

#### Example Request

```
GET /api/v1/equipment-details/family/aircraft?page=1
```

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "id": 1,
        "model": "AH-1W",
        "lin": null,
        "mode": "REDUCED",
        "description": "HELICOPTER ATTACK",
        "length": "546",
        "width": "128",
        "height": "165",
        "weight": "10200",
        "kw": null,
        "hz": null,
        "family": "aircraft"
      }
    ],
    "count": 40,
    "page": 1,
    "total_pages": 2,
    "is_last_page": false
  }
}
```

Response shape is identical to the **List All Equipment** endpoint.

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `page` invalid | `400` | `{"error": "Invalid page number"}` |
| No equipment found for family | `404` | `{"status": 404, "data": null, "message": "no item(s) found"}` |
| Database error | `500` | `{"status": 500, "data": null, "message": "internal Server Error"}` |

---

### 4. Search by Model or LIN

Searches equipment by model designation or LIN. The search is **case-insensitive** and uses **partial matching** (substring).

**`GET /equipment-details/search`**

#### Query Parameters

| Parameter | Type    | Required | Default | Description |
|-----------|---------|----------|---------|-------------|
| `q`       | string  | **Yes**  | —       | Search term. Matched against both `model` and `lin` columns. |
| `page`    | integer | No       | `1`     | Page number. Must be ≥ 1. |

#### Example Requests

```
GET /api/v1/equipment-details/search?q=AH-64&page=1
GET /api/v1/equipment-details/search?q=R14216&page=1
GET /api/v1/equipment-details/search?q=m1037&page=1
```

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "id": 362,
        "model": "M1037",
        "lin": "R14216",
        "mode": null,
        "description": "TRUCK UTILITY 4X4 W/ GENERATOR POWER GROUP",
        "length": "185",
        "width": "85",
        "height": "80",
        "weight": "8655",
        "kw": null,
        "hz": null,
        "family": "trucks"
      }
    ],
    "count": 1,
    "page": 1,
    "total_pages": 1,
    "is_last_page": true
  }
}
```

Response shape is identical to the **List All Equipment** endpoint.

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `q` is missing or blank | `400` | `{"error": "Search query (q) is required"}` |
| `page` invalid | `400` | `{"error": "Invalid page number"}` |
| No results found | `404` | `{"status": 404, "data": null, "message": "no item(s) found"}` |
| Database error | `500` | `{"status": 500, "data": null, "message": "internal Server Error"}` |

---

## Image Endpoints

Equipment images are stored in Azure Blob Storage and organized by family. These endpoints let you browse the available image folders, list images within a family, and generate secure download URLs.

### 5. List Image Families

Returns the list of family folders that contain equipment images.

**`GET /equipment-details/images/families`**

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "families": [
      {
        "name": "aircraft",
        "full_path": "docs_equipment/images/aircraft/",
        "display_name": "AIRCRAFT"
      },
      {
        "name": "trucks",
        "full_path": "docs_equipment/images/trucks/",
        "display_name": "TRUCKS"
      }
    ],
    "count": 2
  }
}
```

#### Image Family Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Folder name — pass this to the family images endpoint |
| `full_path` | string | Full blob prefix (for reference only) |
| `display_name` | string | Uppercase, human-readable name for display |

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| Azure storage unavailable | `500` | `{"error": "Failed to list image families", "details": "..."}` |

---

### 6. List Images in a Family

Returns all images inside a specific family folder.

**`GET /equipment-details/images/family/:family`**

#### Path Parameters

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `family`  | string | Yes      | Family folder name from the image families endpoint. URL-encode spaces. |

#### Example Request

```
GET /api/v1/equipment-details/images/family/aircraft
```

#### Success Response — `200 OK`

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
      },
      {
        "name": "CH-47F.png",
        "blob_path": "docs_equipment/images/aircraft/CH-47F.png",
        "size_bytes": 98432,
        "last_modified": "2026-01-15T10:30:00Z"
      }
    ],
    "count": 2
  }
}
```

#### Image Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Filename of the image |
| `blob_path` | string | Full blob path — pass this to the download endpoint |
| `size_bytes` | integer | File size in bytes |
| `last_modified` | string | RFC 3339 timestamp of when the file was last modified in storage |

#### Supported Image Formats

Only files with these extensions are returned: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `family` is blank | `400` | `{"error": "Family parameter is required"}` |
| Azure storage unavailable | `500` | `{"error": "Failed to list images", "details": "..."}` |

> **Note:** If a family folder is empty or doesn't exist, the response will return successfully with an empty `images` array and `count: 0`.

---

### 7. Get All Image URLs for a Family (Batch)

Returns all images in a family folder **with pre-generated SAS download URLs** in a single call. This is the **recommended endpoint for displaying images** — it eliminates the need to call the download endpoint individually for each image.

All URLs share the same expiry time (**1 hour**). This endpoint is **rate-limited**.

**`GET /equipment-details/images/family/:family/urls`**

#### Path Parameters

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `family`  | string | Yes      | Family folder name from the image families endpoint. URL-encode spaces. |

#### Example Request

```
GET /api/v1/equipment-details/images/family/aircraft/urls
```

#### Success Response — `200 OK`

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
        "download_url": "https://storage.example.com/library/docs_equipment/images/aircraft/AH-64D.jpg?sv=...&se=...&sig=...",
        "size_bytes": 124567,
        "last_modified": "2026-01-15T10:30:00Z"
      },
      {
        "name": "CH-47F.png",
        "blob_path": "docs_equipment/images/aircraft/CH-47F.png",
        "download_url": "https://storage.example.com/library/docs_equipment/images/aircraft/CH-47F.png?sv=...&se=...&sig=...",
        "size_bytes": 98432,
        "last_modified": "2026-01-15T10:30:00Z"
      }
    ],
    "count": 2,
    "expires_at": "2026-03-03T22:00:00Z"
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `family` | string | The family that was queried |
| `images` | array | Array of image objects with download URLs |
| `count` | integer | Number of images returned |
| `expires_at` | string | RFC 3339 timestamp — all `download_url` values expire at this time |

#### Image URL Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Filename of the image |
| `blob_path` | string | Full blob path (for reference) |
| `download_url` | string | Direct HTTPS URL with SAS token — load directly in `Image` widget |
| `size_bytes` | integer | File size in bytes |
| `last_modified` | string | RFC 3339 timestamp |

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `family` is blank | `400` | `{"error": "Family parameter is required"}` |
| Azure AD or storage error | `500` | `{"error": "Failed to generate image URLs", "details": "..."}` |

> **Note:** If a family folder is empty or doesn't exist, the response will return successfully with an empty `images` array, `count: 0`, and empty `expires_at`.

#### When to Use This vs Endpoint 6 + 7

| Scenario | Use |
|----------|-----|
| **Display all images in a gallery** | ✅ This endpoint (one call, all URLs) |
| **Show image count/filenames only** | Use endpoint 6 (cheaper, no Azure AD call) |
| **Download a single specific image** | Use endpoint 8 (single download URL) |

---

### 8. Download Image

Returns a time-limited secure download URL for a specific image. The URL expires after **1 hour** and grants read-only access via HTTPS. This endpoint is **rate-limited**.

**`GET /equipment-details/images/download`**

#### Query Parameters

| Parameter   | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `blob_path` | string | **Yes**  | Full blob path of the image. Use the `blob_path` value from the family images endpoint. |

#### Example Request

```
GET /api/v1/equipment-details/images/download?blob_path=docs_equipment/images/aircraft/AH-64D.jpg
```

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "blob_path": "docs_equipment/images/aircraft/AH-64D.jpg",
    "download_url": "https://storage.example.com/library/docs_equipment/images/aircraft/AH-64D.jpg?sv=...&se=...&sig=...",
    "expires_at": "2026-03-02T22:30:00Z"
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `blob_path` | string | The blob path that was requested |
| `download_url` | string | Fully-formed HTTPS URL with embedded SAS token. Open directly to view or download the image — no extra headers required. |
| `expires_at` | string | RFC 3339 timestamp after which the URL expires |

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `blob_path` missing or blank | `400` | `{"error": "Invalid request", "details": "blob path cannot be empty"}` |
| `blob_path` doesn't start with `docs_equipment/images/` | `400` | `{"error": "Invalid request", "details": "invalid blob path: must start with docs_equipment/images/"}` |
| File is not a supported image type | `400` | `{"error": "Invalid request", "details": "invalid file type: only image files are allowed"}` |
| Image does not exist in storage | `404` | `{"error": "Image not found", "details": "The requested image does not exist"}` |
| URL generation fails | `500` | `{"error": "Failed to generate download URL", "details": "..."}` |

#### Download URL Usage Notes

- The `download_url` is a direct HTTPS link. Load it in an `Image` widget, `WebView`, or download with any HTTP client — **no additional headers or authentication required**.
- URLs are valid for **1 hour**. Cache the URL if you intend to use it within that window.
- If a cached URL has expired (check `expires_at`), call this endpoint again for a fresh one.
- Always source `blob_path` from the family images response. **Do not construct blob paths manually.**

---

## Recommended Mobile Workflow

### Equipment Data Flow

1. **Browse** — Call `GET /equipment-details?page=1` to show paginated equipment data.
2. **Filter** — Call `GET /equipment-details/families` to populate a category/family dropdown. When the user selects a family, call `GET /equipment-details/family/<family>?page=1`.
3. **Search** — Call `GET /equipment-details/search?q=<term>&page=1` for model or LIN search.
4. **Paginate** — Use `total_pages` and `is_last_page` to drive pagination UI.

### Equipment Images Flow

1. **Browse Families** — Call `GET /equipment-details/images/families` to list available image categories.
2. **Display All Images** *(recommended)* — Call `GET /equipment-details/images/family/<family>/urls` to get all images with ready-to-use download URLs in a single call. Cache the URLs using `expires_at`.
3. **Metadata Only** — If you just need filenames or counts (e.g. showing a badge), use `GET /equipment-details/images/family/<family>` instead — it's cheaper.
4. **Single Download** — For downloading one specific image, call `GET /equipment-details/images/download?blob_path=<blob_path>`.

---

## Pagination Notes

- Page size is fixed at **40 items per page** for all data endpoints.
- Requesting a page beyond `total_pages` returns a `404` response.
- Image endpoints are **not paginated** — all images in a family folder are returned at once.

---

## Error Handling Guidance

| Status | Recommended Behavior |
|--------|---------------------|
| `400` | Show the error message to help the user understand the issue |
| `404` | Show a "not found" message — the data may not exist for the given filter/search |
| `500` | Show a generic "something went wrong, try again" message — do not display `details` to end users |

---

## Query Examples

| Goal | Endpoint |
|------|----------|
| First page of all equipment | `GET /equipment-details?page=1` |
| Page 5 of all equipment | `GET /equipment-details?page=5` |
| All unique families | `GET /equipment-details/families` |
| All aircraft | `GET /equipment-details/family/aircraft?page=1` |
| All trucks, page 2 | `GET /equipment-details/family/trucks?page=2` |
| Armored wheeled vehicles | `GET /equipment-details/family/armored%20wheeled%20vehicles?page=1` |
| Search for AH-64 | `GET /equipment-details/search?q=AH-64&page=1` |
| Search by LIN | `GET /equipment-details/search?q=R14216&page=1` |
| List image families | `GET /equipment-details/images/families` |
| Aircraft images (metadata only) | `GET /equipment-details/images/family/aircraft` |
| Aircraft images with URLs | `GET /equipment-details/images/family/aircraft/urls` |
| Download specific image | `GET /equipment-details/images/download?blob_path=docs_equipment/images/aircraft/AH-64D.jpg` |
