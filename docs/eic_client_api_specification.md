# EIC (Equipment Identification Code) Client API Specification

## Overview

The EIC API provides access to military equipment catalog data through four REST endpoints. This specification is designed for client application developers who need to integrate with the EIC service to query military equipment information by various identifiers.

## Base Information

- **Base URL**: `{server}/api`
- **Authentication**: None required (public endpoints)
- **Content Type**: `application/json`
- **Response Format**: All responses follow a standardized structure

## Standard Response Wrapper

All successful API responses are wrapped in a standard response structure:

```json
{
  "status": 200,
  "message": "",
  "data": {
    // Response-specific data structure
  }
}
```

## Data Model

### EIC (Equipment Identification Code) Object

The core data structure returned by all endpoints:

```json
{
  "inc": "string|null",           // Increment
  "fsc": "string|null",           // Federal Supply Class
  "niin": "string",               // National Item Identification Number (Primary Key)
  "eic": "string|null",           // Equipment Identification Code
  "uoeic": "string",              // Unit of Equipment Identification Code (Primary Key)
  "lin": "string|null",           // Line Item Number
  "nomen": "string|null",         // Equipment Nomenclature/Name
  "model": "string|null",         // Equipment Model
  "eicc": "string|null",          // Equipment Identification Code Category
  "ecc": "string|null",           // Equipment Category Code
  "cmdtycd": "string|null",       // Commodity Code
  "reported": "string|null",      // Reported Status
  "dahr": "string|null",          // Date of Approval for High Risk
  "publvl1": "string|null",       // Publication Level 1
  "pubno1": "string|null",        // Publication Number 1
  "pubdate1": "string|null",      // Publication Date 1
  "pubchg1": "string|null",       // Publication Change 1
  "pubcgdt1": "string|null",      // Publication Change Date 1
  // ... Additional publication fields (publvl2-7, pubno2-7, etc.)
  "pubremks": "string|null",      // Publication Remarks
  "eqpmcsa": "string|null",       // Equipment Characteristics A
  "eqpmcsb": "string|null",       // Equipment Characteristics B
  // ... Additional equipment characteristics (eqpmcsc-l)
  "wpnrec": "string|null",        // Weapon Record
  "sernotrk": "string|null",      // Serial Number Tracking
  "orf": "string|null",           // Operational Readiness Float
  "aoap": "string|null",          // Army Oil Analysis Program
  "gainloss": "string|null",      // Gain/Loss
  "usage": "string|null",         // Usage
  "urm1": "string|null",          // Unit of Issue Reporting Metric 1
  "urm2": "string|null",          // Unit of Issue Reporting Metric 2
  "uom1": "string|null",          // Unit of Measure 1
  "uom2": "string|null",          // Unit of Measure 2
  "uom3": "string|null",          // Unit of Measure 3
  "mau1": "string|null",          // Maintenance Authorization Unit 1
  "uom4": "string|null",          // Unit of Measure 4
  "mau2": "string|null",          // Maintenance Authorization Unit 2
  "warranty": "string|null",      // Warranty Information
  "rbm": "string|null",           // Reliability and Maintainability
  "mrc": "string",                // Material Readiness Code (Primary Key)
  "sos": "string|null",           // Source of Supply
  "erc": "string|null",           // Equipment Readiness Code
  "eslvl": "string|null",         // Equipment Support Level
  "oslin": "string|null",         // Organization Stock List Item Number
  "lcc": "string|null",           // Life Cycle Code
  "nounabb": "string|null",       // Noun Abbreviation
  "curfmc": "string|null",        // Current Full Mission Capable
  "prevfmc": "string|null",       // Previous Full Mission Capable
  "bstat1": "string|null",        // Battle Status 1
  "bstat2": "string|null",        // Battle Status 2
  "matcat": "string|null",        // Material Category
  "itemmgr": "string|null",       // Item Manager
  "eos": "string|null",           // End of Service
  "sorts": "string|null",         // Status of Resources and Training System
  "status": "string|null",        // Status
  "lstUpdt": "string|null"        // Last Updated
}
```

### Key Fields for Client Applications

**Primary Identifiers:**
- `niin`: National Item Identification Number (always present)
- `uoeic`: Unit of Equipment Identification Code (always present)
- `mrc`: Material Readiness Code (always present)

**Search Fields:**
- `lin`: Line Item Number (used for equipment tracking)
- `fsc`: Federal Supply Class (used for categorization)
- `nomen`: Equipment name/nomenclature
- `model`: Equipment model information

## API Endpoints

### 1. Query by NIIN

Retrieve EIC records by National Item Identification Number.

**Endpoint:** `GET /api/eic/niin/{niin}`

**Parameters:**
- `niin` (path, required): National Item Identification Number

**Response Structure:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "count": 2,
    "items": [
      {
        // EIC object
      },
      {
        // EIC object
      }
    ]
  }
}
```

**Example Request:**
```bash
GET /api/eic/niin/012345678
```

**Example Response:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "count": 1,
    "items": [
      {
        "inc": null,
        "fsc": "2320",
        "niin": "012345678",
        "eic": "ABC",
        "uoeic": "A01234",
        "lin": "M12345",
        "nomen": "TRUCK, CARGO: 2 1/2 TON, 6X6",
        "model": "M35A3",
        "eicc": "A",
        "ecc": "12",
        "mrc": "H",
        "status": "ACTIVE"
        // ... additional fields
      }
    ]
  }
}
```

### 2. Query by LIN

Retrieve EIC records by Line Item Number.

**Endpoint:** `GET /api/eic/lin/{lin}`

**Parameters:**
- `lin` (path, required): Line Item Number

**Response Structure:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "count": 1,
    "items": [
      {
        // EIC object
      }
    ]
  }
}
```

**Example Request:**
```bash
GET /api/eic/lin/M12345
```

### 3. Query by FSC (Paginated)

Retrieve EIC records by Federal Supply Class with pagination support.

**Endpoint:** `GET /api/eic/fsc/{fsc}?page={page}`

**Parameters:**
- `fsc` (path, required): Federal Supply Class code
- `page` (query, optional): Page number (default: 1)

**Response Structure:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        // EIC objects
      }
    ],
    "count": 40,              // Number of items in current page
    "page": 1,                // Current page number
    "total_pages": 15,        // Total number of pages
    "is_last_page": false     // Whether this is the last page
  }
}
```

**Example Request:**
```bash
GET /api/eic/fsc/2320?page=1
```

**Example Response:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        "fsc": "2320",
        "niin": "012345678",
        "lin": "M12345",
        "nomen": "TRUCK, CARGO: 2 1/2 TON, 6X6",
        "model": "M35A3"
        // ... additional fields
      }
      // ... up to 39 more items
    ],
    "count": 40,
    "page": 1,
    "total_pages": 15,
    "is_last_page": false
  }
}
```

### 4. General Query (Paginated)

Retrieve all EIC records with optional search and pagination.

**Endpoint:** `GET /api/eic/items?page={page}&search={search}`

**Parameters:**
- `page` (query, optional): Page number (default: 1)
- `search` (query, optional): Search term to filter across all text fields

**Response Structure:**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "items": [
      {
        // EIC objects
      }
    ],
    "count": 25,              // Number of items in current page
    "page": 2,                // Current page number
    "total_pages": 8,         // Total number of pages
    "is_last_page": false     // Whether this is the last page
  }
}
```

**Example Requests:**
```bash
# Get page 1 of all items
GET /api/eic/items

# Get page 2 of all items
GET /api/eic/items?page=2

# Search for items containing "truck"
GET /api/eic/items?search=truck

# Search with pagination
GET /api/eic/items?page=2&search=M35
```

**Search Behavior:**
- The search parameter performs a case-insensitive search across multiple fields:
  - `niin`, `lin`, `fsc`, `nomen`, `model`, `eic`, `uoeic`
- Search uses partial matching (contains logic)
- Empty or whitespace-only search returns all items with pagination

## Error Responses

### 400 Bad Request
Returned when request parameters are invalid.

```json
{
  "error": "Invalid page number"
}
```

Common causes:
- Missing required path parameters (NIIN, LIN, FSC)
- Invalid page numbers (negative or zero)
- Malformed request parameters

### 404 Not Found
Returned when no items match the query criteria.

```json
{
  "status": 404,
  "data": null,
  "message": "no item(s) found"
}
```

### 500 Internal Server Error
Returned when server encounters an error processing the request.

```json
{
  "status": 500,
  "data": null,
  "message": "internal Server Error"
}
```

## Pagination Details

### Paginated Endpoints
- `/api/eic/fsc/{fsc}`
- `/api/eic/items`

### Pagination Parameters
- **Page Size**: Fixed at 40 items per page
- **Page Numbers**: Start at 1 (not 0-indexed)
- **Default Page**: 1 when page parameter is omitted

### Pagination Response Fields
```json
{
  "count": 40,              // Items in current response (≤ 40)
  "page": 1,                // Current page number
  "total_pages": 15,        // Total pages available
  "is_last_page": false     // True if no more pages after this one
}
```

### Navigation Logic
```javascript
// Check if more pages available
if (!response.data.is_last_page) {
  // Can request page + 1
}

// Calculate total items (approximate)
const approximateTotal = response.data.total_pages * 40;

// Navigate to specific page
const targetPage = 5;
if (targetPage <= response.data.total_pages) {
  // Request valid
}
```

## Best Practices

### 1. Error Handling
- Always handle 404 responses gracefully (no items found is not an error)
- Implement retry logic for 500 errors with exponential backoff
- Validate input parameters before making requests

### 2. Pagination
- Use the `is_last_page` field to determine when to stop pagination
- Don't assume all pages will have exactly 40 items
- Consider implementing page size limits in your client to avoid excessive requests

### 3. Performance
- Implement request debouncing for search functionality

### 4. Search Optimization
- Trim whitespace from search terms
- Implement minimum character length (2-3 chars) before searching
- Use appropriate delays between search requests to avoid overwhelming the server

### 5. Data Handling
- Many fields can be `null` - always check before using
- Primary key fields (`niin`, `uoeic`, `mrc`) are always present
- Implement proper null checking in your data models

## Rate Limiting and Usage

Currently, there are no rate limits enforced, but clients should:
- Avoid excessive concurrent requests
- Implement reasonable delays between bulk operations
- Cache results when possible to reduce server load


## Changelog

### Version 1.0.0 (Initial Release)
- Four endpoint implementation (NIIN, LIN, FSC, General)
- Pagination support for FSC and general queries
- Comprehensive EIC data model with 100+ fields
- No authentication required
- Standard error response format