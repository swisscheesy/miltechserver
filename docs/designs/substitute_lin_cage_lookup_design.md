# Substitute LIN and CAGE Lookup Design Document

**Date**: 2026-01-19
**Author**: System Design
**Status**: Design Phase
**Feature**: Public Substitute LIN and CAGE Lookup

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

This design introduces two new public lookup endpoints under the existing item lookup router:
- Substitute LIN lookup that returns the full `army_substitute_lin` table (approximately 250 rows).
- CAGE lookup by exact CAGE code, with input normalization to uppercase.

Both endpoints follow existing architecture (route → controller → service → repository), use Jet for database access, and return data using the standard response wrapper.

---

## Current Implementation Analysis

### Existing Patterns
- Public item lookup routes are defined in `api/route/item_lookup_route.go`.
- Handlers follow the controller/service/repository pattern.
- Responses use `response.StandardResponse` for successful results and `response.NoItemFoundResponseMessage()` for missing data.
- Jet is used for queries via generated table/model types in `.gen/miltech_ng/public`.

### Relevant Models
- `model.ArmySubstituteLin` maps `army_substitute_lin` with columns:
  - `lin`
  - `substitute_lin`
- `model.CageAddress` maps `cage_address` with columns:
  - `cage_code`, `company_name`, `company_name_2`, `company_name_3`, `company_name_4`, `company_name_5`,
    `street_address_1`, `street_address_2`, `po_box`, `city`, `state`, `zip`, `country`, `date_est`,
    `last_update`, `former_name_1`, `former_name_2`, `former_name_3`, `former_name_4`, `frn_dom`

---

## Requirements

### Functional
1. Add a Substitute LIN lookup endpoint returning all rows from `army_substitute_lin`.
2. Add a CAGE lookup endpoint that:
   - Accepts a CAGE code path parameter.
   - Normalizes the input by trimming and converting to uppercase.
   - Performs an exact match search on `cage_address.cage_code`.
   - Returns all columns from `cage_address`.
3. Both endpoints are public (no authentication).
4. Responses must use the standard response structure.

### Non-Functional
- No pagination is required for Substitute LIN (approx. 250 rows).
- No performance or rate-limiting constraints for these endpoints.

---

## Proposed Solution

### Endpoints
- `GET /api/v1/lookup/substitute-lin`
- `GET /api/v1/lookup/cage/:cage`

### Architecture
Reuse the existing item lookup stack:

```
Route (api/route/item_lookup_route.go)
  -> Controller (api/controller/item_lookup_controller.go)
    -> Service (api/service/item_lookup_service_impl.go)
      -> Repository (api/repository/item_lookup_repository_impl.go)
        -> Jet Query (table.ArmySubstituteLin / table.CageAddress)
```

---

## Technical Design

### 1. Route Layer
Add two new routes to `api/route/item_lookup_route.go`:
- `group.GET("/lookup/substitute-lin", pc.LookupSubstituteLINAll)`
- `group.GET("/lookup/cage/:cage", pc.LookupCAGEByCode)`

### 2. Controller Layer
Add controller methods in `api/controller/item_lookup_controller.go`:

- `LookupSubstituteLINAll`
  - Calls service to fetch all substitute LIN rows.
  - Returns `StandardResponse` on success.
  - Returns 404 via `NoItemFoundResponseMessage` if empty.
  - Returns 500 on unexpected errors.

- `LookupCAGEByCode`
  - Reads `:cage` path parameter.
  - Validates non-empty after trimming.
  - Normalizes to uppercase.
  - Calls service for exact match search.
  - Returns standard responses as above.

### 3. Service Layer
Add methods to `api/service/item_lookup_service.go` and implementation:
- `LookupSubstituteLINAll() ([]model.ArmySubstituteLin, error)`
- `LookupCAGEByCode(cage string) ([]model.CageAddress, error)`

Responsibilities:
- Normalize CAGE input to uppercase in service.
- Pass through data or errors from repository.

### 4. Repository Layer
Add methods to `api/repository/item_lookup_repository.go` and implementation:
- `SearchSubstituteLINAll() ([]model.ArmySubstituteLin, error)`
- `SearchCAGEByCode(cage string) ([]model.CageAddress, error)`

Jet queries:
- Substitute LIN:
  - `SELECT(table.ArmySubstituteLin.AllColumns).FROM(table.ArmySubstituteLin)`
- CAGE exact match:
  - `SELECT(table.CageAddress.AllColumns).FROM(table.CageAddress).WHERE(table.CageAddress.CageCode.EQ(String(cage)))`

### 5. Response Structures
Use the existing standard response wrapper:

```json
{
  "status": 200,
  "message": "",
  "data": [ ... ]
}
```

- Substitute LIN: data array of `model.ArmySubstituteLin`.
- CAGE lookup: data array of `model.CageAddress` (matching rows, typically one).

### 6. Error Handling
- Empty result set: return `404` with `response.NoItemFoundResponseMessage()`.
- Invalid parameters (empty CAGE): return `400` with `{"error":"CAGE parameter is required"}`.
- Other errors: return `500` with `response.InternalErrorResponseMessage()`.

---

## API Specification

### 1. Substitute LIN Lookup
**Endpoint:** `GET /api/v1/lookup/substitute-lin`

**Authentication:** None (public)

**Response (200):**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "XXXX",
      "substitute_lin": "YYYY"
    }
  ]
}
```

**Response (404):**
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

### 2. CAGE Lookup
**Endpoint:** `GET /api/v1/lookup/cage/:cage`

**Authentication:** None (public)

**Path Parameter:**
- `cage` (string, required) — normalized to uppercase before exact match.

**Response (200):**
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

**Response (400):**
```json
{
  "error": "CAGE parameter is required"
}
```

**Response (404):**
```json
{
  "status": 404,
  "message": "no item(s) found",
  "data": null
}
```

---

## Implementation Plan

1. Add new route registrations in `api/route/item_lookup_route.go`.
2. Add controller methods in `api/controller/item_lookup_controller.go`.
3. Extend service interface and implementation in `api/service/item_lookup_service.go` and `api/service/item_lookup_service_impl.go`.
4. Extend repository interface and implementation in `api/repository/item_lookup_repository.go` and `api/repository/item_lookup_repository_impl.go`.
5. Ensure inputs are normalized and error paths match existing patterns.
6. Run tests or manual verification against the database.

---

## Risks and Tradeoffs

- **Large response size:** Returning all substitute LIN rows is acceptable at current size (~250), but may need pagination if the table grows significantly.
- **Exact match only:** Users must provide a precise CAGE code after normalization; no partial match support is included by design.

---

## Future Considerations

- Add optional filtering or pagination if `army_substitute_lin` grows beyond current size.
- Consider a `GET /lookup/cage?cage=...` query parameter variant for consistency with other search endpoints.
- Add caching headers for public lookup endpoints to reduce load if needed.
