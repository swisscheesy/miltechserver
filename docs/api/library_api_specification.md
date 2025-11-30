# PMCS Library API Specification

**Version**: 1.0
**Last Updated**: 2025-11-14
**Status**: Production Ready
**Base URL**: `{BASE_URL}/api/v1`

---

## Overview

The PMCS Library API provides access to military equipment Preventive Maintenance Checks and Services (PMCS) documentation stored in Azure Blob Storage. The API allows mobile applications to browse vehicle categories, list available documents, and generate secure time-limited download URLs.

### Key Features

- **Browse Vehicle Categories**: List all available PMCS vehicle types
- **List Documents**: View PDF documents available for each vehicle
- **Secure Downloads**: Generate time-limited download URLs (1-hour expiry)
- **No Authentication Required**: All endpoints are publicly accessible
- **PDF Only**: All documents are in PDF format

### API Characteristics

- **Protocol**: HTTPS (recommended for production)
- **Response Format**: JSON
- **Authentication**: None required
- **Rate Limiting**: None currently implemented
- **CORS**: Enabled for cross-origin requests

---

## Environment Configuration

### Base URLs

| Environment | Base URL | Notes |
|-------------|----------|-------|
| Development | `http://localhost:8080/api/v1` | Local development server |
| Staging | `https://staging-api.yourapp.com/api/v1` | Testing environment |
| Production | `https://api.yourapp.com/api/v1` | Production environment |

**Note**: Replace `{BASE_URL}` in all endpoint URLs with your environment's base URL.

---

## Common HTTP Headers

### Request Headers

```http
Content-Type: application/json
Accept: application/json
```

### Response Headers

All responses include:

```http
Content-Type: application/json
```

---

## HTTP Status Codes

All endpoints may return these standard HTTP status codes:

| Status Code | Meaning | Description |
|-------------|---------|-------------|
| `200 OK` | Success | Request completed successfully |
| `400 Bad Request` | Client Error | Invalid or missing request parameters |
| `404 Not Found` | Not Found | Requested resource doesn't exist |
| `500 Internal Server Error` | Server Error | Server-side error occurred |

---

## Endpoints

### 1. List Vehicle Categories

Returns a list of all available vehicle categories in the PMCS library.

#### HTTP Request

```
GET {BASE_URL}/library/pmcs/vehicles
```

#### Request Parameters

**None**

#### Request Example

```http
GET https://api.yourapp.com/api/v1/library/pmcs/vehicles HTTP/1.1
Content-Type: application/json
Accept: application/json
```

#### Success Response

**Status Code**: `200 OK`

**Response Body**:

```json
{
  "vehicles": [
    {
      "name": "GENERATOR",
      "full_path": "pmcs/GENERATOR/",
      "display_name": "GENERATOR"
    },
    {
      "name": "HEMTT_LET",
      "full_path": "pmcs/HEMTT_LET/",
      "display_name": "HEMTT LET"
    },
    {
      "name": "HMMWV",
      "full_path": "pmcs/HMMWV/",
      "display_name": "HMMWV"
    },
    {
      "name": "TRACK",
      "full_path": "pmcs/TRACK/",
      "display_name": "TRACK"
    }
  ],
  "count": 4
}
```

#### Response Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `vehicles` | Array | List of vehicle category objects | See Vehicle Object below |
| `count` | Integer | Total number of vehicle categories | `11` |

**Vehicle Object**:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `name` | String | Internal folder name (use for API requests) | `"TRACK"` |
| `full_path` | String | Full blob storage path | `"pmcs/TRACK/"` |
| `display_name` | String | Human-readable name for display | `"TRACK"` |

#### Display Name Rules

- Underscores (`_`) are replaced with spaces
- All letters are uppercase
- Examples:
  - `HEMTT_LET` → `"HEMTT LET"`
  - `MATERIAL_HANDLING_EQUIPMENT` → `"MATERIAL HANDLING EQUIPMENT"`

#### Known Vehicle Categories

As of 2025-11-14, these vehicle categories exist:

1. `GENERATOR` → "GENERATOR"
2. `HEMTT_LET` → "HEMTT LET"
3. `HMMWV` → "HMMWV"
4. `LMTV_MTV` → "LMTV MTV"
5. `MATERIAL_HANDLING_EQUIPMENT` → "MATERIAL HANDLING EQUIPMENT"
6. `MISCELLANEOUS` → "MISCELLANEOUS"
7. `OTHER_VEHICLES` → "OTHER VEHICLES"
8. `RECOVERY` → "RECOVERY"
9. `TRACK` → "TRACK"
10. `TRAILER` → "TRAILER"
11. `WEAPONS_AND_ELECTRONICS` → "WEAPONS AND ELECTRONICS"

#### Error Responses

**Status Code**: `500 Internal Server Error`

```json
{
  "error": "Failed to retrieve PMCS vehicles",
  "details": "connection to Azure Blob Storage failed"
}
```

#### Usage Notes

- Use the `name` field (not `display_name`) when calling other endpoints
- The `display_name` field is formatted for UI display
- All vehicle categories are returned, even if empty

---

### 2. List Documents for Vehicle

Returns all PDF documents available for a specific vehicle category.

#### HTTP Request

```
GET {BASE_URL}/library/pmcs/{vehicle}/documents
```

#### URL Parameters

| Parameter | Type | Required | Description | Example |
|-----------|------|----------|-------------|---------|
| `vehicle` | String | **Yes** | Vehicle category name from vehicle list | `TRACK` |

#### Request Examples

```http
GET https://api.yourapp.com/api/v1/library/pmcs/TRACK/documents HTTP/1.1

GET https://api.yourapp.com/api/v1/library/pmcs/HMMWV/documents HTTP/1.1

GET https://api.yourapp.com/api/v1/library/pmcs/MATERIAL_HANDLING_EQUIPMENT/documents HTTP/1.1
```

#### Success Response

**Status Code**: `200 OK`

**Response Body**:

```json
{
  "vehicle_name": "TRACK",
  "documents": [
    {
      "name": "m1-abrams-pmcs.pdf",
      "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
      "size_bytes": 2457600,
      "last_modified": "2024-11-10T14:30:00Z"
    },
    {
      "name": "m2-bradley-pmcs.pdf",
      "blob_path": "pmcs/TRACK/m2-bradley-pmcs.pdf",
      "size_bytes": 1843200,
      "last_modified": "2024-10-25T09:15:00Z"
    }
  ],
  "count": 2
}
```

#### Response Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `vehicle_name` | String | Vehicle category name (matches URL parameter) | `"TRACK"` |
| `documents` | Array | List of PDF document objects | See Document Object below |
| `count` | Integer | Total number of PDF documents found | `2` |

**Document Object**:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `name` | String | File name | `"m1-abrams-pmcs.pdf"` |
| `blob_path` | String | Full blob path (use for download endpoint) | `"pmcs/TRACK/m1-abrams-pmcs.pdf"` |
| `size_bytes` | Integer | File size in bytes | `2457600` |
| `last_modified` | String | Last modified timestamp (ISO 8601 UTC) | `"2024-11-10T14:30:00Z"` |

#### Empty Folder Response

When a vehicle category exists but contains no PDF documents:

**Status Code**: `200 OK`

```json
{
  "vehicle_name": "EMPTY_VEHICLE",
  "documents": [],
  "count": 0
}
```

#### Error Responses

**Status Code**: `500 Internal Server Error`

```json
{
  "error": "Failed to retrieve PMCS documents",
  "details": "connection to Azure Blob Storage failed"
}
```

#### Field Notes

**`size_bytes`**:
- Value is in bytes
- To convert to megabytes: `size_bytes ÷ 1024 ÷ 1024`
- Example: `2457600` bytes = 2.34 MB

**`last_modified`**:
- Format: ISO 8601 timestamp
- Timezone: Always UTC (indicated by `Z` suffix)
- Example: `"2024-11-10T14:30:00Z"`

**`blob_path`**:
- Use this exact value when requesting download URLs
- Do not modify or construct manually

#### Usage Notes

- Only PDF files are returned (other file types filtered out)
- PDF filtering is case-insensitive (`.pdf`, `.PDF` both recognized)
- Non-existent vehicles return empty array (not 404 error)
- Empty folders return empty array with `count: 0`

---

### 3. Generate Download URL

Generates a time-limited, secure download URL for a specific PDF document.

#### HTTP Request

```
GET {BASE_URL}/library/download?blob_path={blob_path}
```

#### Query Parameters

| Parameter | Type | Required | Description | Example |
|-----------|------|----------|-------------|---------|
| `blob_path` | String | **Yes** | Full blob path from document list | `pmcs/TRACK/m1-abrams-pmcs.pdf` |

#### URL Encoding

Special characters in `blob_path` must be URL-encoded:

| Character | Encoded | Example |
|-----------|---------|---------|
| Space | `%20` | `forklift manual.pdf` → `forklift%20manual.pdf` |
| Plus | `%2B` | `manual+guide.pdf` → `manual%2Bguide.pdf` |

#### Request Example

```http
GET https://api.yourapp.com/api/v1/library/download?blob_path=pmcs/TRACK/m1-abrams-pmcs.pdf HTTP/1.1
```

#### Success Response

**Status Code**: `200 OK`

**Response Body**:

```json
{
  "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
  "download_url": "https://miltechng.blob.core.windows.net/library/pmcs/TRACK/m1-abrams-pmcs.pdf?sv=2023-11-03&se=2024-11-14T15:30:00Z&sr=b&sp=r&sig=ABCD1234567890EFGH...",
  "expires_at": "2024-11-14T15:30:00Z"
}
```

#### Response Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `blob_path` | String | Original blob path from request | `"pmcs/TRACK/m1-abrams-pmcs.pdf"` |
| `download_url` | String | Time-limited HTTPS URL for download | See Download URL Format below |
| `expires_at` | String | Expiry timestamp (ISO 8601 UTC) | `"2024-11-14T15:30:00Z"` |

#### Download URL Format

The `download_url` is a complete HTTPS URL with SAS (Shared Access Signature) query parameters:

```
https://miltechng.blob.core.windows.net/library/{blob_path}?{sas_params}
```

**SAS Query Parameters** (included automatically):

| Parameter | Description |
|-----------|-------------|
| `sv` | Storage service version |
| `se` | Expiry time (ISO 8601) |
| `sr` | Resource type (always `b` for blob) |
| `sp` | Permissions (always `r` for read-only) |
| `sig` | Cryptographic signature |

**Important**: Use the complete URL as provided. Do not modify or parse it.

#### Download URL Properties

| Property | Value | Description |
|----------|-------|-------------|
| **Expiry Time** | 1 hour | URL is valid for exactly 1 hour from generation |
| **Protocol** | HTTPS only | HTTP requests will fail |
| **Permissions** | Read-only | Cannot modify or delete the file |
| **Sharing** | Public | Anyone with the URL can download (until expiry) |

#### Error Responses

**Missing Parameter** - `400 Bad Request`:

```json
{
  "error": "blob_path query parameter is required"
}
```

**Invalid Path Prefix** - `400 Bad Request`:

```json
{
  "error": "Invalid request",
  "details": "invalid blob path: must start with pmcs/ or bii/"
}
```

**Non-PDF File** - `400 Bad Request`:

```json
{
  "error": "Invalid request",
  "details": "invalid file type: only PDF files can be downloaded"
}
```

**Document Not Found** - `404 Not Found`:

```json
{
  "error": "Document not found",
  "details": "The requested document does not exist or is not accessible"
}
```

**Server Error** - `500 Internal Server Error`:

```json
{
  "error": "Failed to generate download URL",
  "details": "SAS token generation failed"
}
```

#### Path Validation Rules

The API validates `blob_path` against these rules:

1. **Must start with**: `pmcs/` (future: `bii/` will be supported)
2. **Must end with**: `.pdf` (case-insensitive)
3. **Must exist**: File must be present in Azure Blob Storage

**Valid Examples**:
- ✅ `pmcs/TRACK/manual.pdf`
- ✅ `pmcs/HMMWV/guide.PDF`
- ✅ `pmcs/GENERATOR/operators-manual.pdf`

**Invalid Examples**:
- ❌ `unauthorized/file.pdf` (wrong prefix)
- ❌ `pmcs/TRACK/image.jpg` (not a PDF)
- ❌ `pmcs/TRACK/nonexistent.pdf` (file doesn't exist)

#### Usage Notes

**URL Expiry**:
- Download URLs expire exactly 1 hour after generation
- Check `expires_at` field for exact expiry time
- After expiry, Azure returns HTTP `403 Forbidden`
- Generate a new URL if expired

**Security**:
- URLs work over HTTPS only (HTTP will fail)
- URLs are read-only (cannot modify or delete files)
- URLs are public (anyone with URL can download until expiry)
- Do not share URLs if documents are sensitive

**When to Generate**:
- Generate URLs on-demand when user initiates download
- Do not pre-generate and cache (URLs expire after 1 hour)
- Generate fresh URL if download fails with 403 error

---

## Complete Workflow Example

This example shows the complete user flow for browsing and downloading a document.

### Step 1: Get Vehicle Categories

**Request**:
```http
GET {BASE_URL}/library/pmcs/vehicles
```

**Response**:
```json
{
  "vehicles": [
    {"name": "TRACK", "full_path": "pmcs/TRACK/", "display_name": "TRACK"},
    {"name": "HMMWV", "full_path": "pmcs/HMMWV/", "display_name": "HMMWV"}
  ],
  "count": 2
}
```

### Step 2: User Selects "TRACK", Get Documents

**Request**:
```http
GET {BASE_URL}/library/pmcs/TRACK/documents
```

**Response**:
```json
{
  "vehicle_name": "TRACK",
  "documents": [
    {
      "name": "m1-abrams-pmcs.pdf",
      "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
      "size_bytes": 2457600,
      "last_modified": "2024-11-10T14:30:00Z"
    }
  ],
  "count": 1
}
```

### Step 3: User Selects Document, Generate Download URL

**Request**:
```http
GET {BASE_URL}/library/download?blob_path=pmcs/TRACK/m1-abrams-pmcs.pdf
```

**Response**:
```json
{
  "blob_path": "pmcs/TRACK/m1-abrams-pmcs.pdf",
  "download_url": "https://miltechng.blob.core.windows.net/library/pmcs/TRACK/m1-abrams-pmcs.pdf?sv=2023-11-03&se=2024-11-14T15:30:00Z&sr=b&sp=r&sig=ABC123...",
  "expires_at": "2024-11-14T15:30:00Z"
}
```

### Step 4: Download File Using Generated URL

**Request**:
```http
GET {download_url from Step 3}
```

**Response**: Binary PDF file data

---

## Testing

### Test Commands

Use these curl commands to test the API:

**List Vehicles**:
```bash
curl https://api.yourapp.com/api/v1/library/pmcs/vehicles
```

**List Documents**:
```bash
curl https://api.yourapp.com/api/v1/library/pmcs/TRACK/documents
```

**Generate Download URL**:
```bash
curl "https://api.yourapp.com/api/v1/library/download?blob_path=pmcs/TRACK/test.pdf"
```

**Download File**:
```bash
curl "PASTE_DOWNLOAD_URL_HERE" --output document.pdf
```

### Mock Data

For UI development before API integration, use this mock data:

**Mock Vehicle List**:
```json
{
  "vehicles": [
    {"name": "TRACK", "full_path": "pmcs/TRACK/", "display_name": "TRACK"},
    {"name": "HMMWV", "full_path": "pmcs/HMMWV/", "display_name": "HMMWV"}
  ],
  "count": 2
}
```

**Mock Document List**:
```json
{
  "vehicle_name": "TRACK",
  "documents": [
    {
      "name": "test-manual.pdf",
      "blob_path": "pmcs/TRACK/test-manual.pdf",
      "size_bytes": 2500000,
      "last_modified": "2024-11-14T12:00:00Z"
    }
  ],
  "count": 1
}
```

---

## Data Types

### Field Type Reference

| Field Name | JSON Type | Format/Constraints | Example |
|------------|-----------|-------------------|---------|
| `name` | String | Alphanumeric with underscores | `"TRACK"` |
| `full_path` | String | Path with trailing slash | `"pmcs/TRACK/"` |
| `display_name` | String | Uppercase, spaces | `"TRACK"` |
| `vehicle_name` | String | Same as `name` | `"TRACK"` |
| `blob_path` | String | Full path to file | `"pmcs/TRACK/file.pdf"` |
| `size_bytes` | Integer | Positive number | `2457600` |
| `last_modified` | String | ISO 8601 UTC timestamp | `"2024-11-10T14:30:00Z"` |
| `expires_at` | String | ISO 8601 UTC timestamp | `"2024-11-14T15:30:00Z"` |
| `download_url` | String | Complete HTTPS URL | See examples above |
| `count` | Integer | Non-negative number | `11` |

### Timestamp Format

All timestamps use **ISO 8601 format** in **UTC timezone**:

```
YYYY-MM-DDTHH:mm:ssZ
```

**Examples**:
- `"2024-11-10T14:30:00Z"` - November 10, 2024 at 2:30 PM UTC
- `"2024-12-25T00:00:00Z"` - December 25, 2024 at midnight UTC

**Note**: The `Z` suffix indicates UTC timezone (zero offset).

---

## Error Response Format

All error responses follow this consistent format:

```json
{
  "error": "Brief error message",
  "details": "Detailed error information"
}
```

### Error Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `error` | String | Short, user-friendly error message |
| `details` | String | Technical details for debugging |

### Common Error Messages

| HTTP Status | `error` Field | `details` Field Examples |
|-------------|---------------|-------------------------|
| 400 | `"blob_path query parameter is required"` | N/A (error field only) |
| 400 | `"Invalid request"` | `"invalid blob path: must start with pmcs/ or bii/"` |
| 400 | `"Invalid request"` | `"invalid file type: only PDF files can be downloaded"` |
| 404 | `"Document not found"` | `"The requested document does not exist or is not accessible"` |
| 500 | `"Failed to retrieve PMCS vehicles"` | `"connection to Azure Blob Storage failed"` |
| 500 | `"Failed to retrieve PMCS documents"` | `"connection to Azure Blob Storage failed"` |
| 500 | `"Failed to generate download URL"` | `"SAS token generation failed"` |

---

## API Versioning

### Current Version: v1

- **Base URL**: `{BASE_URL}/api/v1`
- **Status**: Production Ready
- **Stability**: Stable - No breaking changes planned
- **Release Date**: 2025-11-14

### Deprecation Policy

- **Notice Period**: 6 months minimum for any breaking changes
- **Version Support**: When v2 is released, v1 will remain available for at least 6 months
- **Migration Path**: Migration guide will be provided before v1 deprecation

### Future Versions

Planned features for future API versions:
- BII (Basic Issue Items) document support
- User authentication and authorization
- Favorites and bookmarks
- Download history tracking
- Search and filtering

---

## Frequently Asked Questions

### Authentication & Access

**Q: Is authentication required?**
A: No. All endpoints are currently public and require no authentication.

**Q: Are there rate limits?**
A: No rate limits are currently enforced.

### Download URLs

**Q: How long are download URLs valid?**
A: Exactly 1 hour from generation time. Check the `expires_at` field.

**Q: What happens after a URL expires?**
A: Azure Blob Storage returns HTTP `403 Forbidden`. Generate a new URL and retry.

**Q: Can download URLs be reused across users?**
A: Yes. Download URLs are public once generated. Anyone with the URL can download the file until it expires.

**Q: Can I request a longer expiry time?**
A: No. The 1-hour expiry is fixed for security reasons.

### File Types

**Q: Can I download non-PDF files?**
A: No. The API only supports PDF file downloads. Other file types return `400 Bad Request`.

**Q: What's the maximum file size?**
A: No maximum size is enforced. Typical PMCS manuals range from 1 MB to 100 MB.

### Special Cases

**Q: How are special characters in file names handled?**
A: URL-encode special characters in the `blob_path` parameter (spaces as `%20`, etc.).

**Q: What if a vehicle category has no documents?**
A: The API returns `200 OK` with an empty `documents` array and `count: 0`.

**Q: Can the API work offline?**
A: No. All endpoints require internet connectivity to access Azure Blob Storage.

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-11-14 | Initial release - Vehicle categories, document listing, download URL generation |

---

## Appendix: Complete Response Examples

### Full Vehicle List Response

All 11 vehicle categories as returned by the API:

```json
{
  "vehicles": [
    {"name": "GENERATOR", "full_path": "pmcs/GENERATOR/", "display_name": "GENERATOR"},
    {"name": "HEMTT_LET", "full_path": "pmcs/HEMTT_LET/", "display_name": "HEMTT LET"},
    {"name": "HMMWV", "full_path": "pmcs/HMMWV/", "display_name": "HMMWV"},
    {"name": "LMTV_MTV", "full_path": "pmcs/LMTV_MTV/", "display_name": "LMTV MTV"},
    {"name": "MATERIAL_HANDLING_EQUIPMENT", "full_path": "pmcs/MATERIAL_HANDLING_EQUIPMENT/", "display_name": "MATERIAL HANDLING EQUIPMENT"},
    {"name": "MISCELLANEOUS", "full_path": "pmcs/MISCELLANEOUS/", "display_name": "MISCELLANEOUS"},
    {"name": "OTHER_VEHICLES", "full_path": "pmcs/OTHER_VEHICLES/", "display_name": "OTHER VEHICLES"},
    {"name": "RECOVERY", "full_path": "pmcs/RECOVERY/", "display_name": "RECOVERY"},
    {"name": "TRACK", "full_path": "pmcs/TRACK/", "display_name": "TRACK"},
    {"name": "TRAILER", "full_path": "pmcs/TRAILER/", "display_name": "TRAILER"},
    {"name": "WEAPONS_AND_ELECTRONICS", "full_path": "pmcs/WEAPONS_AND_ELECTRONICS/", "display_name": "WEAPONS AND ELECTRONICS"}
  ],
  "count": 11
}
```

### Document List with Multiple Files

Example response showing various file sizes:

```json
{
  "vehicle_name": "TRACK",
  "documents": [
    {
      "name": "m1-abrams-operators-manual.pdf",
      "blob_path": "pmcs/TRACK/m1-abrams-operators-manual.pdf",
      "size_bytes": 52428800,
      "last_modified": "2024-11-10T14:30:00Z"
    },
    {
      "name": "m1-abrams-pmcs-checklist.pdf",
      "blob_path": "pmcs/TRACK/m1-abrams-pmcs-checklist.pdf",
      "size_bytes": 1048576,
      "last_modified": "2024-11-10T14:35:00Z"
    },
    {
      "name": "m2-bradley-operators-manual.pdf",
      "blob_path": "pmcs/TRACK/m2-bradley-operators-manual.pdf",
      "size_bytes": 45678901,
      "last_modified": "2024-10-25T09:15:00Z"
    }
  ],
  "count": 3
}
```

### File Size Reference

Common PMCS document sizes:

| Size (bytes) | Size (MB) | Typical Content |
|--------------|-----------|-----------------|
| 1,048,576 | 1 MB | Quick reference checklist |
| 5,242,880 | 5 MB | Basic operator manual |
| 26,214,400 | 25 MB | Complete technical manual |
| 52,428,800 | 50 MB | Comprehensive maintenance guide |
| 104,857,600 | 100 MB | Full illustrated parts manual |
