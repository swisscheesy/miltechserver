# PS Magazine API — Mobile Integration Guide

**Base URL:** `https://<host>/api/v1`
**Authentication:** None required — all endpoints are public
**Content-Type:** `application/json`

---

## Overview

The PS Magazine API provides access to the PS Magazine digital archive. Issues are browsable via a paginated listing endpoint and downloadable via a time-limited secure URL endpoint. All metadata (issue number, month, year) is derived from the filename — no separate metadata call is needed.

---

## Endpoints

### 1. List Issues

Returns a paginated list of PS Magazine issues, with optional filters by year or issue number.

**`GET /library/ps-mag/issues`**

---

#### Query Parameters

| Parameter | Type    | Required | Default | Description |
|-----------|---------|----------|---------|-------------|
| `page`    | integer | No       | `1`     | Page number. Must be greater than 0. |
| `order`   | string  | No       | `asc`   | Sort direction by issue number. Accepted values: `asc`, `desc`. |
| `year`    | integer | No       | —       | Filter to issues from a specific year (e.g. `1994`). Omit to return all years. |
| `issue`   | integer | No       | —       | Filter to a specific issue number (e.g. `495`). Omit to return all issues. |

---

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "issues": [
      {
        "name": "PS_Magazine_Issue_495_February_1994.pdf",
        "blob_path": "ps-mag/PS_Magazine_Issue_495_February_1994.pdf",
        "issue_number": 495,
        "month": "February",
        "year": 1994,
        "size_bytes": 4821032,
        "last_modified": "2024-01-15T10:30:00Z"
      }
    ],
    "count": 1,
    "total_count": 1,
    "page": 1,
    "total_pages": 1,
    "order": "asc"
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code mirrored in body |
| `message` | string | Empty on success |
| `data.issues` | array | Array of issue objects for this page |
| `data.count` | integer | Number of issues returned on this page (≤ 50) |
| `data.total_count` | integer | Total matching issues across all pages |
| `data.page` | integer | Current page number |
| `data.total_pages` | integer | Total number of pages available |
| `data.order` | string | Sort direction used (`asc` or `desc`) |

#### Issue Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Filename of the issue |
| `blob_path` | string | Full blob path — pass this to the download endpoint |
| `issue_number` | integer | Issue number extracted from the filename |
| `month` | string | Publication month extracted from the filename (e.g. `"February"`) |
| `year` | integer | Publication year extracted from the filename (e.g. `1994`) |
| `size_bytes` | integer | File size in bytes |
| `last_modified` | string | RFC 3339 timestamp of when the file was last modified in storage |

---

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `page` is less than 1 | `400` | `{"error": "Invalid request", "details": "page must be greater than 0"}` |
| `page` is not a valid integer | `400` | `{"error": "Invalid request", "details": "page must be greater than 0"}` |
| `order` is not `asc` or `desc` | `400` | `{"error": "Invalid request", "details": "order must be 'asc' or 'desc'"}` |
| `year` is not a valid integer | `400` | `{"error": "Invalid request", "details": "year must be a valid integer"}` |
| `issue` is not a valid integer | `400` | `{"error": "Invalid request", "details": "issue must be a valid integer"}` |
| Azure storage unavailable | `500` | `{"error": "Failed to list issues", "details": "..."}` |

---

#### Pagination Notes

- Page size is fixed at **50 issues per page**.
- When a filter returns fewer than 50 results, `total_pages` will be `1` regardless.
- Requesting a page beyond `total_pages` returns an empty `issues` array — it is not an error.
- Use `total_count` to display result counts to users before they paginate.

---

### 2. Generate Download URL

Returns a time-limited secure download URL for a specific issue. The URL expires after **1 hour** and grants read-only access to the PDF file via HTTPS.

**`GET /library/ps-mag/download`**

---

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `blob_path` | string | Yes | The full blob path of the issue. This value comes directly from the `blob_path` field returned by the List Issues endpoint. |

---

#### Success Response — `200 OK`

```json
{
  "status": 200,
  "message": "",
  "data": {
    "blob_path": "ps-mag/PS_Magazine_Issue_495_February_1994.pdf",
    "download_url": "https://storage.example.com/library/ps-mag/PS_Magazine_Issue_495_February_1994.pdf?sv=...&se=...&sig=...",
    "expires_at": "2026-02-20T11:30:00Z"
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | integer | HTTP status code mirrored in body |
| `message` | string | Empty on success |
| `data.blob_path` | string | The blob path that was requested |
| `data.download_url` | string | A fully-formed HTTPS URL with an embedded SAS token. Open this URL directly to download the PDF. |
| `data.expires_at` | string | RFC 3339 timestamp after which the URL will no longer be valid |

---

#### Error Responses

| Condition | Status | Response Body |
|-----------|--------|---------------|
| `blob_path` is missing or blank | `400` | `{"error": "Invalid request", "details": "blob path cannot be empty"}` |
| `blob_path` does not start with `ps-mag/` | `400` | `{"error": "Invalid request", "details": "invalid blob path: must start with ps-mag/"}` |
| `blob_path` does not end in `.pdf` | `400` | `{"error": "Invalid request", "details": "invalid file type: only PDF files can be downloaded"}` |
| The requested issue does not exist | `404` | `{"error": "Issue not found", "details": "The requested issue does not exist or is not accessible"}` |
| URL generation fails | `500` | `{"error": "Failed to generate download URL", "details": "..."}` |

---

#### Download URL Usage Notes

- The `download_url` is a direct, authenticated HTTPS link. Open it in a browser, `WKWebView`, or download it with any HTTP client — no additional headers or authentication required.
- URLs are valid for **1 hour** from the time of generation. Cache the URL only if you intend to use it within that window.
- If a cached URL has expired (indicated by `expires_at`), call this endpoint again to get a fresh one.
- Always source `blob_path` from the List Issues response. Do not construct blob paths manually.

---

## Recommended Mobile Workflow

1. **Browse** — Call `GET /library/ps-mag/issues` with `page=1` and `order=desc` to show the newest issues first.
2. **Filter** — Allow users to filter by `year` or jump to a specific `issue` number using the optional query parameters.
3. **Paginate** — Use `total_pages` and `total_count` to drive pagination UI. Increment `page` to load the next batch.
4. **Download** — When a user selects an issue, call `GET /library/ps-mag/download?blob_path=<blob_path>` using the `blob_path` from the listing response.
5. **Open** — Pass the returned `download_url` directly to a PDF viewer or in-app browser. No further authentication is required.

---

## Error Handling Guidance

| Status | Recommended Behavior |
|--------|---------------------|
| `400` | Show the `details` field to help the user understand the issue (e.g. invalid page number) |
| `404` | Show a "not found" message — the issue may have been removed from storage |
| `500` | Show a generic "something went wrong, try again" message — do not display `details` to end users |

---

## Filtering Examples

| Goal | Query |
|------|-------|
| First 50 issues, oldest first | `?page=1&order=asc` |
| First 50 issues, newest first | `?page=1&order=desc` |
| All issues from 1994 | `?year=1994` |
| Specific issue #495 | `?issue=495` |
| Issues from 1994, newest first | `?year=1994&order=desc` |
| Page 2 of all issues | `?page=2` |
