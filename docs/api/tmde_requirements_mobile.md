# TMDE Requirements API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-05-07
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Overview

This document describes two new public endpoints for Test, Measurement, and Diagnostic Equipment (TMDE) calibration and service requirements. The data is keyed by NIIN. Both endpoints require no authentication and follow the existing standard response wrapper for successful responses.

Base URL prefix: `/api/v1`

---

## Data Model

A TMDE requirement record represents the calibration/service requirements for a single piece of equipment identified by its NIIN.

| Field | Type | Notes |
|-------|------|-------|
| `niin` | string | National Item Identification Number — primary key, one record per NIIN |
| `qty_skot` | integer or null | Quantity SKOT (Support Equipment/Tools) required |
| `component` | string or null | Single character flag — `"Y"` or `"N"` |
| `interval` | string or null | Calibration/service interval — free-form text (e.g. `"1800"`, `"360"`, `"CNR"`, `"N/A"`) |

---

## Endpoint 1: NIIN Lookup

**Method:** GET
**Path:** `/api/v1/tmde/niin/:niin`
**Auth:** None
**Purpose:** Look up a single TMDE requirement record by NIIN. At most one record is returned since NIIN is the primary key.

### Request

Path parameter:
- `niin`: string, required

Input normalization:
- The server trims whitespace and converts the value to uppercase before searching.
- The search is an exact match after normalization.
- Sending a blank or whitespace-only value (e.g. URL-encoded spaces) will return a 400 error.

### Successful Response (200)

Standard response wrapper with `data` as a single TMDE requirement record.

Response fields:
- `status`: 200
- `message`: empty string
- `data`: a single TMDE requirement object

TMDE requirement object fields:
- `niin`: string
- `qty_skot`: integer or null
- `component`: string or null
- `interval`: string or null

Example response (200):
```json
{
  "status": 200,
  "message": "",
  "data": {
    "niin": "012345678",
    "qty_skot": 4,
    "component": "Y",
    "interval": "1800"
  }
}
```

Example with null fields (200):
```json
{
  "status": 200,
  "message": "",
  "data": {
    "niin": "009876543",
    "qty_skot": null,
    "component": null,
    "interval": "N/A"
  }
}
```

### Validation Error Response (400)

Returned when the NIIN path parameter is blank or resolves to empty after trimming. Note: this uses a simple error object, not the standard response wrapper.

Fields:
- `error`: "NIIN parameter is required"

Example response (400):
```json
{
  "error": "NIIN parameter is required"
}
```

### Not Found Response (404)

Returned when no record exists for the given NIIN.

Fields:
- `status`: 404
- `message`: "no item(s) found"
- `data`: null

Example response (404):
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

### Error Response (500)

Returned when an unexpected server-side error occurs.

Fields:
- `status`: 500
- `message`: "internal Server Error"
- `data`: null

Example response (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

## Endpoint 2: Paginated Requirements List

**Method:** GET
**Path:** `/api/v1/tmde/requirements`
**Auth:** None
**Purpose:** Retrieve all TMDE requirement records, paginated. 100 records per page. Results are ordered by NIIN ascending for stable pagination.

### Request

Query parameter:
- `page`: integer, optional, defaults to `1` if omitted

Valid values: any integer greater than or equal to `1`. Non-numeric values and `0` or negative numbers return a 400 error. Requesting a page beyond the last page returns a 404.

### Successful Response (200)

Standard response wrapper with `data` as a pagination envelope.

Response fields:
- `status`: 200
- `message`: empty string
- `data`: pagination envelope

Pagination envelope fields:
- `items`: array of TMDE requirement objects (see data model above)
- `count`: integer — number of records in this page (will be less than 100 on the last page)
- `page`: integer — the page number returned
- `total_pages`: integer — total number of pages available
- `is_last_page`: boolean — `true` when `page >= total_pages`

Example response (200, first page of a multi-page result):
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "niin": "001234567",
        "qty_skot": 2,
        "component": "Y",
        "interval": "360"
      },
      {
        "niin": "001234568",
        "qty_skot": null,
        "component": "N",
        "interval": "CNR"
      }
    ],
    "count": 100,
    "page": 1,
    "total_pages": 5,
    "is_last_page": false
  }
}
```

Example response (200, last page):
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "niin": "009999999",
        "qty_skot": 1,
        "component": "Y",
        "interval": "N/A"
      }
    ],
    "count": 37,
    "page": 5,
    "total_pages": 5,
    "is_last_page": true
  }
}
```

### Validation Error Response (400)

Returned when the `page` parameter is non-numeric or less than `1`. Note: this uses a simple error object, not the standard response wrapper.

Fields:
- `error`: "Invalid page number"

Example response (400, non-numeric page):
```json
{
  "error": "Invalid page number"
}
```

Example response (400, page=0):
```json
{
  "error": "Invalid page number"
}
```

### Not Found Response (404)

Returned when the requested page is beyond the last page, or the table contains no records.

Fields:
- `status`: 404
- `message`: "no item(s) found"
- `data`: null

Example response (404):
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

### Error Response (500)

Returned when an unexpected server-side error occurs.

Fields:
- `status`: 500
- `message`: "internal Server Error"
- `data`: null

Example response (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

## Notes for Mobile Implementation

- Both endpoints are public and do not require Firebase authentication.
- Page size is fixed at 100 records per page and cannot be changed by the client.
- Results from the paginated list are ordered by NIIN ascending. This order is stable across pages — safe to accumulate locally page by page.
- Use `is_last_page: true` as the signal to stop fetching subsequent pages.
- The 400 validation errors use a simple `{"error": "..."}` shape, not the standard `status/message/data` wrapper used by 404 and 500 responses.
- NIIN lookup normalizes to uppercase on the server — sending lowercase NIINs will still match correctly.
- All nullable fields (`qty_skot`, `component`, `interval`) should be treated as optional and may be `null` for any record.
