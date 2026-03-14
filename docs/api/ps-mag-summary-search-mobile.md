# PS Magazine Summary Search — Mobile Integration Guide

**Base URL:** `https://<host>/api/v1`
**Authentication:** None required — public endpoint
**Content-Type:** `application/json`

---

## Overview

The summary search endpoint lets users find PS Magazine issues by keyword. Rather than returning full issue summaries, the response returns only the specific lines within each matching summary that contain the search phrase. This keeps response payloads small and makes it easy to display highlighted snippets in the app.

Search is **case-insensitive**. The phrase is matched against pre-stored summaries in the database — not against PDF content directly.

---

## Endpoint

### Search PS Magazine Summaries

**`GET /library/ps-mag/search`**

Returns a paginated list of PS Magazine issues whose summaries contain the search phrase. Each result includes only the lines from that issue's summary that match.

---

### Query Parameters

| Parameter | Type    | Required | Default | Description |
|-----------|---------|----------|---------|-------------|
| `q`       | string  | Yes      | —       | The phrase to search for. Must be at least 3 characters. Leading and trailing whitespace is ignored. |
| `page`    | integer | No       | `1`     | Page number. Must be greater than 0. |

---

### Success Response — `200 OK`

The response envelope follows the standard API format. The `data` object contains paginated results.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "results": [
      {
        "file_name": "PS_Magazine_Issue_495_February_1994.pdf",
        "matching_lines": [
          "Check oil level before each operation.",
          "Oil filter must be replaced every 3,000 miles."
        ]
      },
      {
        "file_name": "PS_Magazine_Issue_312_June_1979.pdf",
        "matching_lines": [
          "Oil pressure warning light indicates low oil."
        ]
      }
    ],
    "count": 2,
    "total_count": 47,
    "page": 1,
    "total_pages": 2,
    "query": "oil"
  }
}
```

---

### Response Fields

#### Envelope

| Field     | Type    | Description |
|-----------|---------|-------------|
| `status`  | integer | HTTP status code mirrored in body |
| `message` | string  | Empty string on success |
| `data`    | object  | The search result payload (see below) |

#### `data` Object

| Field         | Type    | Description |
|---------------|---------|-------------|
| `results`     | array   | Array of matching issue objects for this page. Empty array if no matches. |
| `count`       | integer | Number of results returned on this page (≤ 30) |
| `total_count` | integer | Total number of issues with at least one matching summary line, across all pages |
| `page`        | integer | Current page number (1-indexed) |
| `total_pages` | integer | Total number of pages available. Always at least `1` even when results are empty. |
| `query`       | string  | The search phrase that was used, echoed back for display purposes |

#### Result Object (`results[]`)

| Field           | Type            | Description |
|-----------------|-----------------|-------------|
| `file_name`     | string          | Filename of the PS Magazine issue. Use this to display the issue title to the user. |
| `matching_lines`| array of strings| The lines from this issue's summary that contain the search phrase. Each line is trimmed of leading and trailing whitespace. Empty lines are excluded. |

---

### Empty Results Response — `200 OK`

When no issues match the query, the response is still `200 OK` with an empty results array.

```json
{
  "status": 200,
  "message": "",
  "data": {
    "results": [],
    "count": 0,
    "total_count": 0,
    "page": 1,
    "total_pages": 1,
    "query": "xyz_no_match"
  }
}
```

---

### Error Responses

All errors return a JSON object with `error` and `details` fields. The `details` value is safe to display in debug/development environments. In production, show a generic user-facing message instead.

#### `400 Bad Request`

| Condition | `details` value |
|-----------|-----------------|
| `q` parameter is missing or empty | `"search query must be at least 3 characters"` |
| `q` parameter is fewer than 3 characters | `"search query must be at least 3 characters"` |
| `page` is not a valid integer | `"page must be greater than 0"` |
| `page` is less than 1 | `"page must be greater than 0"` |

```json
{
  "error": "Invalid request",
  "details": "search query must be at least 3 characters"
}
```

#### `500 Internal Server Error`

```json
{
  "error": "Failed to search summaries",
  "details": "..."
}
```

---

## Pagination

- Page size is fixed at **30 results per page**.
- `total_pages` tells you how many pages exist. Use it to decide whether to offer a "load more" or next-page control.
- Requesting a page beyond `total_pages` returns an empty `results` array — it is not an error.
- `total_count` reflects the total number of matching issues, not the total number of matching lines. Use it to show a result count to the user (e.g. "47 issues found").

---

## Search Behavior Notes

- Matching is **case-insensitive** — searching `"OIL"`, `"oil"`, or `"Oil"` all return the same results.
- Only lines that contain the phrase are included in `matching_lines`. Lines from the summary that do not match are not returned.
- A single issue may have multiple matching lines. All of them are included in `matching_lines` for that result.
- Results are ordered alphabetically by filename.
- The search is a phrase match, not a full-text ranked search — the phrase must appear as a substring within a summary line.

---

## Recommended Mobile Workflow

1. **Search** — As the user types, debounce and call `GET /library/ps-mag/search?q=<phrase>` once they have entered at least 3 characters.
2. **Display snippets** — Show each result's `file_name` as the issue title and display the `matching_lines` as highlighted preview text beneath it.
3. **Paginate** — If `total_pages > 1`, offer a load-more or page control. Increment `page` to fetch the next batch using the same `q` value.
4. **Navigate to issue** — The `file_name` value matches the `name` field returned by the List Issues endpoint (`GET /library/ps-mag/issues`). Use it to look up the `blob_path` for download if the user wants to open the full PDF.
5. **Empty state** — When `total_count` is `0`, show a "no results" message. Do not treat this as an error.

---

## Error Handling Guidance

| Status | Recommended Behavior |
|--------|---------------------|
| `400` (query too short) | Enforce the 3-character minimum in the UI before calling the endpoint to avoid this entirely |
| `400` (invalid page) | Should not occur in normal use — validate page numbers client-side |
| `500` | Show a generic "search is unavailable, try again" message — do not display the `details` field to end users |

---

## Example Requests

| Goal | Request |
|------|---------|
| Search for "lubrication" | `GET /library/ps-mag/search?q=lubrication` |
| Search for "oil filter", page 2 | `GET /library/ps-mag/search?q=oil+filter&page=2` |
| Search for "track tension" | `GET /library/ps-mag/search?q=track+tension` |
