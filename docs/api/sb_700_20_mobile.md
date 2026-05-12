# SB 700-20 API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-05-12
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Overview

This document describes 26 public endpoints covering all 13 tables from SB 700-20 (Army Supply Bulletin). Each table exposes two endpoints: a paginated list and a LIN-based search. All endpoints require no authentication.

Base URL prefix: `/api/v1`

---

## Common Patterns

### Standard Response Wrapper

All successful responses (HTTP 200) use a consistent envelope:

```
{
  "status": 200,
  "message": "",
  "data": <object or pagination object — see per-table sections>
}
```

### Pagination

All `/list` endpoints return a pagination object as `data`:

| Field | Type | Description |
|-------|------|-------------|
| `items` | array | The records for the current page |
| `count` | integer | Number of records in this page (≤ 100) |
| `page` | integer | The requested page number |
| `total_pages` | integer | Total number of pages for the entire dataset |
| `is_last_page` | boolean | `true` when `page >= total_pages` |

Page size is fixed at **100 records per page**. Pages are 1-indexed — the first page is `?page=1`, which is also the default when the parameter is omitted.

### LIN Normalization

For all `/search/:lin` endpoints, the server normalizes the LIN before querying:
- Whitespace is trimmed from both ends
- The value is converted to uppercase

The search is an exact match after normalization. Sending a blank or whitespace-only value (e.g. URL-encoded spaces like `%20%20`) returns a 400 error.

### Nullable Fields

Fields marked as **nullable** in this document can appear as `null` in the JSON response. Non-nullable fields are always present as their stated type.

---

## Error Responses

### 400 Bad Request

Returned when the `page` parameter is non-numeric, zero, or negative, or when the `:lin` path parameter is blank/whitespace-only.

```json
{ "error": "Invalid page number" }
```
```json
{ "error": "lin parameter is required" }
```

> Note: 400 errors use a flat `{ "error": "..." }` shape, not the standard `status`/`data`/`message` wrapper.

### 404 Not Found

Returned when no records match the search, or when the requested page is beyond the last page.

```json
{
  "status": 404,
  "data": null,
  "message": "no item(s) found"
}
```

### 500 Internal Server Error

Returned when a database error occurs.

```json
{
  "status": 500,
  "data": null,
  "message": "internal Server Error"
}
```

---

## Endpoint Reference

---

### Appendix B — Reportable Items

**Paths:**
- `GET /api/v1/sb700-20/app-b/list`
- `GET /api/v1/sb700-20/app-b/search/:lin`

**Search key:** `lin`
**Search returns:** Array (a LIN may have multiple NSN records)

#### Data Model — App B Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `nsn` | string | No | National Stock Number — part of the composite key |
| `lin` | string | No | Line Item Number |
| `reportable_item_cont` | string | Yes | Reportable item continuance code |
| `chapter_code` | string | Yes | Chapter reference code |

#### List Response (`data` field)

```json
{
  "items": [ { "nsn": "...", "lin": "...", "reportable_item_cont": "...", "chapter_code": "..." } ],
  "count": 100,
  "page": 1,
  "total_pages": 12,
  "is_last_page": false
}
```

#### Search Response (`data` field)

Array of App B records sharing the same LIN.

---

### Appendix C — LIN Nomenclature

**Paths:**
- `GET /api/v1/sb700-20/app-c/list`
- `GET /api/v1/sb700-20/app-c/search/:lin`

**Search key:** `lin`
**Search returns:** Single object (one record per LIN)

#### Data Model — App C Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `nomenclature` | string | Yes | Item nomenclature / description |
| `chapter_code` | integer | Yes | Chapter reference code |

#### Search Response (`data` field)

Single App C record.

---

### Appendix D — Action Items

**Paths:**
- `GET /api/v1/sb700-20/app-d/list`
- `GET /api/v1/sb700-20/app-d/search/:lin`

**Search key:** `lin`
**Search returns:** Array (a LIN may have multiple NSN action records)

#### Data Model — App D Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — part of the composite key |
| `nsn` | string | No | National Stock Number — part of the composite key |
| `ric` | integer | Yes | Routing Identifier Code |
| `chapter_code` | integer | Yes | Chapter reference code |
| `type_of_action` | string | Yes | Action type code |

#### Search Response (`data` field)

Array of App D records sharing the same LIN.

---

### Appendix E — Deleted/Changed Items

**Paths:**
- `GET /api/v1/sb700-20/app-e/list`
- `GET /api/v1/sb700-20/app-e/search/:lin`

**Search key:** `lin`
**Search returns:** Array

#### Data Model — App E Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — part of the composite key |
| `nsn` | string | No | National Stock Number — part of the composite key |
| `cmc` | string | Yes | Controlled Medical Consumable code |
| `reason_for_deletion` | string | Yes | Reason the LIN was deleted or changed |
| `new_lin` | string | Yes | Replacement LIN, if applicable |
| `nomenclature` | string | Yes | Item nomenclature |
| `date_entered_into_ap` | string | Yes | Date the record was entered into the Army Program |
| `army_type_class` | string | Yes | Army type classification code |
| `ratio` | string | Yes | Ratio value |
| `zmm_appdx_dlw_and_or_zmm` | string | Yes | ZMM appendix / DLW code |
| `calendar_year_month` | string | Yes | Calendar year and month of change |
| `chapter_code` | integer | Yes | Chapter reference code |

#### Search Response (`data` field)

Array of App E records sharing the same LIN.

---

### Appendix F — LIN Actions

**Paths:**
- `GET /api/v1/sb700-20/app-f/list`
- `GET /api/v1/sb700-20/app-f/search/:lin`

**Search key:** `lin`
**Search returns:** Single object

#### Data Model — App F Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `type_of_action` | string | Yes | Action type code |
| `chapter_code` | string | Yes | Chapter reference code |

#### Search Response (`data` field)

Single App F record.

---

### Appendix G — LIN Cross-Reference

**Paths:**
- `GET /api/v1/sb700-20/app-g/list`
- `GET /api/v1/sb700-20/app-g/search/:lin`

**Search key:** `lin`
**Search returns:** Single object

#### Data Model — App G Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `tr` | string | Yes | Transfer reference code |
| `new_lin` | string | Yes | New LIN that this LIN maps to |

#### Search Response (`data` field)

Single App G record.

---

### Appendix H1 — LIN Sub-LIN Relationships (Parent)

**Paths:**
- `GET /api/v1/sb700-20/app-h1/list`
- `GET /api/v1/sb700-20/app-h1/search/:lin`

**Search key:** `lin_zmm_lin` (the `:lin` path value is matched against this field)
**Search returns:** Array (a parent LIN may have multiple sub-LIN records)

> App H1 and H2 use the `lin_zmm_lin` field as the search column rather than a plain `lin` column. The `:lin` path parameter is still used as the search value.

#### Data Model — App H1 Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin_zmm_lin` | string | No | Parent Line Item Number — part of the composite key |
| `lin_zmm_sublin` | string | No | Sub-LIN — part of the composite key |
| `nomenclature` | string | Yes | Parent LIN nomenclature |
| `sub_lin_nomenclature` | string | Yes | Sub-LIN nomenclature |

#### Search Response (`data` field)

Array of App H1 records where `lin_zmm_lin` matches the searched value.

---

### Appendix H2 — LIN Sub-LIN Relationships (Detail)

**Paths:**
- `GET /api/v1/sb700-20/app-h2/list`
- `GET /api/v1/sb700-20/app-h2/search/:lin`

**Search key:** `lin_zmm_lin` (same as H1 — the `:lin` path value is matched against this field)
**Search returns:** Array

#### Data Model — App H2 Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin_zmm_lin` | string | No | Parent Line Item Number — part of the composite key |
| `lin_zmmsublin` | string | No | Sub-LIN (note: single-word key, no underscore between `zmm` and `sublin`) — part of the composite key |
| `nomenclature` | string | Yes | Parent LIN nomenclature |
| `sub_lin_nomenclature` | string | Yes | Sub-LIN nomenclature |

#### Search Response (`data` field)

Array of App H2 records where `lin_zmm_lin` matches the searched value.

---

### Appendix I — Chapter Code Lookup

**Paths:**
- `GET /api/v1/sb700-20/app-i/list`
- `GET /api/v1/sb700-20/app-i/search/:lin`

**Search key:** `lin`
**Search returns:** Single object

#### Data Model — App I Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `chapter_code` | string | Yes | Chapter reference code |

#### Search Response (`data` field)

Single App I record.

---

### Appendix J — POMCUS LINs

**Paths:**
- `GET /api/v1/sb700-20/app-j/list`
- `GET /api/v1/sb700-20/app-j/search/:lin`

**Search key:** `lin`
**Search returns:** Single object

#### Data Model — App J Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `zlinum_pomcus` | string | Yes | POMCUS Z-LIN number |

#### Search Response (`data` field)

Single App J record.

---

### Chapter 4 — Reportable Items Master List

**Paths:**
- `GET /api/v1/sb700-20/chp-4/list`
- `GET /api/v1/sb700-20/chp-4/search/:lin`

**Search key:** `lin`
**Search returns:** Single object

#### Data Model — Chapter 4 Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — primary key |
| `control_item_code` | string | Yes | Control item code |
| `reportable_item_cont_zmmlricc` | integer | Yes | Reportable item continuance (ZMM LRICC) |
| `nomenclature` | string | Yes | Item nomenclature |
| `cmc` | string | Yes | Controlled Medical Consumable code |
| `ric` | string | Yes | Routing Identifier Code |
| `current_mcn` | string | Yes | Current Management Control Number |
| `supply_catof_material` | string | Yes | Supply category of material |
| `reportable_item_cont_zmmnricc` | string | Yes | Reportable item continuance (ZMM NRICC) |
| `nsn_nomenclature` | string | Yes | NSN-level nomenclature |
| `standard_price` | string | Yes | Standard price |
| `unit_of_issue` | string | Yes | Unit of issue code |
| `second_position_of_mara` | string | Yes | Second position of MARA code |
| `logistics_control_co` | string | Yes | Logistics control code |
| `army_type_class` | string | Yes | Army type classification |
| `reference_data` | string | Yes | Reference data |

#### Search Response (`data` field)

Single Chapter 4 record.

---

### Chapter 6 — Reportable Items by RIC

**Paths:**
- `GET /api/v1/sb700-20/chp-6/list`
- `GET /api/v1/sb700-20/chp-6/search/:lin`

**Search key:** `lin`
**Search returns:** Array (a LIN may appear under multiple MCNs)

#### Data Model — Chapter 6 Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — part of the composite key |
| `current_mcn` | string | No | Current Management Control Number — part of the composite key |
| `control_item_code` | string | Yes | Control item code |
| `reportable_item_cont_zmmlricc` | string | Yes | Reportable item continuance (ZMM LRICC) |
| `nomenclature` | string | Yes | Item nomenclature |
| `cmc` | string | Yes | Controlled Medical Consumable code |
| `ric` | string | Yes | Routing Identifier Code |
| `supply_catof_material` | string | Yes | Supply category of material |
| `reportable_item_cont_zmmnricc` | string | Yes | Reportable item continuance (ZMM NRICC) |
| `nsn_nomenclature` | string | Yes | NSN-level nomenclature |
| `standard_price` | string | Yes | Standard price |
| `unit_of_issue` | string | Yes | Unit of issue code |
| `second_position_of_mara` | string | Yes | Second position of MARA code |
| `logistics_control_co` | string | Yes | Logistics control code |
| `army_type_class` | string | Yes | Army type classification |
| `reference_data` | string | Yes | Reference data |

#### Search Response (`data` field)

Array of Chapter 6 records sharing the same LIN.

---

### Chapter 8 — Reportable Items by Supply Category

**Paths:**
- `GET /api/v1/sb700-20/chp-8/list`
- `GET /api/v1/sb700-20/chp-8/search/:lin`

**Search key:** `lin`
**Search returns:** Array (a LIN may appear under multiple MCNs)

#### Data Model — Chapter 8 Record

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `lin` | string | No | Line Item Number — part of the composite key |
| `current_mcn` | string | No | Current Management Control Number — part of the composite key |
| `control_item_code` | string | Yes | Control item code |
| `reportable_item_cont_zmmlricc` | string | Yes | Reportable item continuance (ZMM LRICC) |
| `nomenclature` | string | Yes | Item nomenclature |
| `cmc` | string | Yes | Controlled Medical Consumable code |
| `ric` | string | Yes | Routing Identifier Code |
| `supply_catof_material` | string | Yes | Supply category of material |
| `reportable_item_cont_zmmnricc` | string | Yes | Reportable item continuance (ZMM NRICC) |
| `nsn_nomenclature` | string | Yes | NSN-level nomenclature |
| `standard_price` | string | Yes | Standard price |
| `unit_of_issue` | string | Yes | Unit of issue code |
| `second_position_ofMara` | string | Yes | Second position of MARA code (note: mixed-case key) |
| `logistics_control_co` | string | Yes | Logistics control code |
| `army_type_class` | string | Yes | Army type classification |
| `reference_data` | string | Yes | Reference data |

> Note: Chapter 8's `second_position_ofMara` field has a mixed-case JSON key (`ofMara` not `of_mara`). This is a data quirk from the source schema — match the key exactly.

#### Search Response (`data` field)

Array of Chapter 8 records sharing the same LIN.

---

## Quick Reference — All Endpoints

| Table | List Endpoint | Search Endpoint | Search Returns |
|-------|--------------|-----------------|----------------|
| App B | `GET /api/v1/sb700-20/app-b/list` | `GET /api/v1/sb700-20/app-b/search/:lin` | Array |
| App C | `GET /api/v1/sb700-20/app-c/list` | `GET /api/v1/sb700-20/app-c/search/:lin` | Single |
| App D | `GET /api/v1/sb700-20/app-d/list` | `GET /api/v1/sb700-20/app-d/search/:lin` | Array |
| App E | `GET /api/v1/sb700-20/app-e/list` | `GET /api/v1/sb700-20/app-e/search/:lin` | Array |
| App F | `GET /api/v1/sb700-20/app-f/list` | `GET /api/v1/sb700-20/app-f/search/:lin` | Single |
| App G | `GET /api/v1/sb700-20/app-g/list` | `GET /api/v1/sb700-20/app-g/search/:lin` | Single |
| App H1 | `GET /api/v1/sb700-20/app-h1/list` | `GET /api/v1/sb700-20/app-h1/search/:lin` | Array |
| App H2 | `GET /api/v1/sb700-20/app-h2/list` | `GET /api/v1/sb700-20/app-h2/search/:lin` | Array |
| App I | `GET /api/v1/sb700-20/app-i/list` | `GET /api/v1/sb700-20/app-i/search/:lin` | Single |
| App J | `GET /api/v1/sb700-20/app-j/list` | `GET /api/v1/sb700-20/app-j/search/:lin` | Single |
| Chp 4 | `GET /api/v1/sb700-20/chp-4/list` | `GET /api/v1/sb700-20/chp-4/search/:lin` | Single |
| Chp 6 | `GET /api/v1/sb700-20/chp-6/list` | `GET /api/v1/sb700-20/chp-6/search/:lin` | Array |
| Chp 8 | `GET /api/v1/sb700-20/chp-8/list` | `GET /api/v1/sb700-20/chp-8/search/:lin` | Array |

## Quick Reference — Status Codes

| Status | Meaning | Response Shape |
|--------|---------|----------------|
| 200 | Success | `{ "status": 200, "message": "", "data": <...> }` |
| 400 | Bad request (invalid page or blank LIN) | `{ "error": "<message>" }` |
| 404 | No records found, or page out of range | `{ "status": 404, "data": null, "message": "no item(s) found" }` |
| 500 | Server/database error | `{ "status": 500, "data": null, "message": "internal Server Error" }` |
