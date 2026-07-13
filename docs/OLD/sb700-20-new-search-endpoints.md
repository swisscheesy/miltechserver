# SB 700-20 New Search Endpoints

This document describes the 7 new search endpoints added to the SB 700-20 API. All endpoints are public (no authentication required) and are mounted under `/api/v1`.

All successful responses share the same envelope:

```json
{
  "status": 200,
  "message": "",
  "data": [ ... ]
}
```

All error responses follow one of two shapes:

**400 Bad Request**
```json
{ "error": "<param> parameter is required" }
```

**404 Not Found**
```json
{ "status": 404, "message": "No item found", "data": {} }
```

**500 Internal Server Error**
```json
{ "status": 500, "message": "Internal server error" }
```

> **Note on URL encoding:** Some `new_lin` values in the database use `#` as a sentinel for "no new LIN" entries. If you build URLs from database values, percent-encode any `#` as `%23` to prevent browsers and HTTP clients from treating it as a URL fragment separator.

---

## App E — Search by New LIN

**`GET /api/v1/sb700-20/app-e/search-new-lin/:new_lin`**

Returns all App E records whose `new_lin` field matches the given value. The value is matched case-insensitively (normalized to uppercase before querying).

**Path parameter:** `:new_lin` — the new LIN value to search for (e.g. `A12345`). Whitespace-only values return 400.

**Example request:**
```
GET /api/v1/sb700-20/app-e/search-new-lin/A12345
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "B99001",
      "cmc": "C",
      "reason_for_deletion": null,
      "new_lin": "A12345",
      "nsn": "1234-01-567-8901",
      "nomenclature": "RIFLE,5.56MM",
      "date_entered_into_ap": "20230101",
      "army_type_class": "W",
      "ratio": "1",
      "zmm_appdx_dlw_and_or_zmm": null,
      "calendar_year_month": "202301",
      "chapter_code": 4
    }
  ]
}
```

---

## App G — Search by New LIN

**`GET /api/v1/sb700-20/app-g/search-new-lin/:new_lin`**

Returns all App G records whose `new_lin` field matches the given value.

**Path parameter:** `:new_lin` — the new LIN value (e.g. `A12345`).

**Example request:**
```
GET /api/v1/sb700-20/app-g/search-new-lin/A12345
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "tr": "R",
      "new_lin": "A12345",
      "lin": "B99001"
    }
  ]
}
```

---

## App H1 — Search by Sublin

**`GET /api/v1/sb700-20/app-h1/search-sublin/:sublin`**

Returns all App H1 records whose `lin_zmm_sublin` field matches the given sublin value. A single sublin may appear under multiple parent LINs, so this can return multiple records.

**Path parameter:** `:sublin` — the sublin value (e.g. `AA`).

**Example request:**
```
GET /api/v1/sb700-20/app-h1/search-sublin/AA
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin_zmm_lin": "A00001",
      "nomenclature": "RIFLE SYSTEM",
      "lin_zmm_sublin": "AA",
      "sub_lin_nomenclature": "RIFLE,5.56MM"
    },
    {
      "lin_zmm_lin": "B00002",
      "nomenclature": "WEAPON SET",
      "lin_zmm_sublin": "AA",
      "sub_lin_nomenclature": "CARBINE,5.56MM"
    }
  ]
}
```

---

## App H2 — Search by Sublin

**`GET /api/v1/sb700-20/app-h2/search-sublin/:sublin`**

Returns all App H2 records whose `lin_zmmsublin` field matches the given sublin value. Note: App H2 uses `lin_zmmsublin` (no underscore between zmm and sublin) as its column name — this is distinct from App H1's `lin_zmm_sublin`.

**Path parameter:** `:sublin` — the sublin value (e.g. `AA`).

**Example request:**
```
GET /api/v1/sb700-20/app-h2/search-sublin/AA
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin_zmmsublin": "AA",
      "sub_lin_nomenclature": "RIFLE,5.56MM",
      "lin_zmm_lin": "A00001",
      "nomenclature": "RIFLE SYSTEM"
    }
  ]
}
```

---

## Chp 4 — Search by RIC

**`GET /api/v1/sb700-20/chp-4/search-ric/:ric`**

Returns all Chapter 4 records whose `ric` field matches the given RIC value. Multiple records can share the same RIC.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-4/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": 1,
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_of_mara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```

---

## Chp 6 — Search by RIC

**`GET /api/v1/sb700-20/chp-6/search-ric/:ric`**

Returns all Chapter 6 records whose `ric` field matches the given RIC value.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-6/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": "1",
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_of_mara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```

---

## Chp 8 — Search by RIC

**`GET /api/v1/sb700-20/chp-8/search-ric/:ric`**

Returns all Chapter 8 records whose `ric` field matches the given RIC value.

**Path parameter:** `:ric` — the Routing Identifier Code (e.g. `W62G`).

**Example request:**
```
GET /api/v1/sb700-20/chp-8/search-ric/W62G
```

**Example success response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "lin": "A00001",
      "control_item_code": "C",
      "reportable_item_cont_zmmlricc": "1",
      "nomenclature": "RIFLE,5.56MM",
      "cmc": "C",
      "ric": "W62G",
      "current_mcn": "1234567",
      "supply_catof_material": "9",
      "reportable_item_cont_zmmnricc": null,
      "nsn_nomenclature": "RIFLE,5.56MM,M16A2",
      "standard_price": "586.00",
      "unit_of_issue": "EA",
      "second_position_of_mara": "A",
      "logistics_control_co": "W",
      "army_type_class": "W",
      "reference_data": null
    }
  ]
}
```
