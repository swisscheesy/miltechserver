# Substitute LIN and CAGE Lookup API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-01-19
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Overview

This document describes two new public lookup endpoints for Substitute LIN and CAGE address data. These endpoints require no authentication and follow the existing standard response wrapper for successful responses and not-found responses.

Base URL prefix: /api/v1

---

## Endpoint 1: Substitute LIN Lookup (All Rows)

**Method:** GET
**Path:** /lookup/substitute-lin
**Auth:** None
**Purpose:** Retrieve all rows from the army_substitute_lin table. The dataset is approximately 250 rows and is intended to be filtered client-side.

### Request
- No path parameters
- No query parameters
- No request body

### Successful Response
Standard response wrapper with data as a list of Substitute LIN records.

Response fields:
- status: HTTP status code as integer (200)
- message: empty string
- data: list of Substitute LIN records

Substitute LIN record fields:
- lin: string, primary LIN
- substitute_lin: string, substitute LIN

Example response (200):
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "T12345",
      "substitute_lin": "T67890"
    },
    {
      "lin": "A11111",
      "substitute_lin": "A22222"
    }
  ]
}
```

### Not Found Response
If the table is empty, the response uses the standard not-found payload.

Not-found response fields:
- status: 404
- message: "no item(s) found"
- data: null

Example response (404):
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

### Error Response
If an unexpected server error occurs, the response uses the standard internal error payload.

Internal error response fields:
- status: 500
- message: "internal Server Error"
- data: null

Example response (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

## Endpoint 2: CAGE Lookup (Exact Match)

**Method:** GET
**Path:** /lookup/cage/:cage
**Auth:** None
**Purpose:** Retrieve all rows from the cage_address table that exactly match the provided CAGE code.

### Request
Path parameter:
- cage: string, required

Input normalization rules:
- The server trims whitespace from the path value and converts it to uppercase before searching.
- The search is an exact match after normalization.

### Successful Response
Standard response wrapper with data as a list of CAGE address records.

Response fields:
- status: HTTP status code as integer (200)
- message: empty string
- data: list of CAGE address records

CAGE address record fields (all columns):
- cage_code: string
- company_name: string or null
- company_name_2: string or null
- company_name_3: string or null
- company_name_4: string or null
- company_name_5: string or null
- street_address_1: string or null
- street_address_2: string or null
- po_box: string or null
- city: string or null
- state: string or null
- zip: string or null
- country: string or null
- date_est: string or null
- last_update: string or null
- former_name_1: string or null
- former_name_2: string or null
- former_name_3: string or null
- former_name_4: string or null
- frn_dom: string or null

Example response (200):
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "cage_code": "1A2B3",
      "company_name": "Example Co",
      "company_name_2": null,
      "company_name_3": null,
      "company_name_4": null,
      "company_name_5": null,
      "street_address_1": "123 Main St",
      "street_address_2": null,
      "po_box": null,
      "city": "Anytown",
      "state": "VA",
      "zip": "12345",
      "country": "US",
      "date_est": null,
      "last_update": null,
      "former_name_1": null,
      "former_name_2": null,
      "former_name_3": null,
      "former_name_4": null,
      "frn_dom": null
    }
  ]
}
```

### Validation Error Response
If the path parameter is missing or resolves to empty after trimming, the response is a simple error object (not the standard wrapper).

Validation error fields:
- error: "CAGE parameter is required"

Example response (400):
```json
{
  "error": "CAGE parameter is required"
}
```

### Not Found Response
If there are no matching rows, the response uses the standard not-found payload.

Not-found response fields:
- status: 404
- message: "no item(s) found"
- data: null

Example response (404):
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

### Error Response
If an unexpected server error occurs, the response uses the standard internal error payload.

Internal error response fields:
- status: 500
- message: "internal Server Error"
- data: null

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
- Substitute LIN results are not paginated; filter locally as needed.
- CAGE lookup is exact match after normalization to uppercase.
- The standard response wrapper is used for success and not-found cases. The 400 validation error for missing CAGE uses an error-only response.
