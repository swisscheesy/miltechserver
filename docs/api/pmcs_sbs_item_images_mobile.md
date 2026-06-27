# PMCS SBS Item Images Mobile API

This document covers the public PMCS SBS item image endpoint for mobile clients.

## Endpoint

Base URL: `/api/v1`

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/library/pmcs-sbs/image` | Download one PNG image referenced by a PMCS SBS guide item. |

## Authentication

No authentication is required.

This endpoint is part of the public PMCS SBS library API, alongside folders, files, and guide content.

## Request Parameters

### Query Parameters

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `guide_blob_path` | string | Yes | Full PMCS SBS guide JSON blob path. Use the selected guide's `blob_path` value from `GET /library/pmcs-sbs/:folder/files`. |
| `image_name` | string | Yes | Extensionless image name from the loaded guide item. Do not include `.png` or any folder path. |

## Request Example

`GET /api/v1/library/pmcs-sbs/image?guide_blob_path=pmcs_sbs/HMMWV/HMMWV%20NoArmor%20(SEPT13).json&image_name=Before_12`

## Image Identity

Each image request is identified by:

| Field | Source |
|-------|--------|
| `guide_blob_path` | Selected guide JSON blob path. |
| `image_name` | Extensionless image name from the guide item's image metadata. |

The server derives the PNG blob path from those two values.

Example:

| Input | Value |
|-------|-------|
| `guide_blob_path` | `pmcs_sbs/HMMWV/HMMWV NoArmor (SEPT13).json` |
| `image_name` | `Before_12` |
| Derived image blob path | `pmcs_sbs/HMMWV/images/HMMWV NoArmor (SEPT13)/Before_12.png` |

Mobile clients do not send the derived image blob path.

## Success Response

Status: `200 OK`

The response body is raw PNG bytes. It is not wrapped in the standard JSON response envelope.

Expected headers:

| Header | Value |
|--------|-------|
| `Content-Type` | `image/png` |
| `Content-Length` | PNG byte length when known. |
| `Content-Disposition` | Attachment or inline filename metadata for `<image_name>.png`. |

Example response metadata:

```text
HTTP/1.1 200 OK
Content-Type: image/png
Content-Length: 184392
Content-Disposition: inline; filename="Before_12.png"
```

Response body:

```text
<raw PNG bytes>
```

## Validation Rules

### `guide_blob_path`

| Rule | Notes |
|------|-------|
| Must not be blank | Whitespace-only values are rejected. |
| Must start with `pmcs_sbs/` | Any other prefix is rejected. |
| Must end with `.json` | Case-insensitive. |
| Must use forward slashes only | Backslashes are rejected. |
| Must be a clean path | Traversal or normalization such as `../`, `./`, or duplicate path separators is rejected. |

### `image_name`

| Rule | Notes |
|------|-------|
| Must not be blank | Whitespace-only values are rejected. |
| Must be extensionless | `Before_12` is valid; `Before_12.png` is rejected. |
| Must be a basename only | Folder paths are rejected. |
| Must not contain `/` or `\` | Path separators are rejected. |
| Must not contain `.` | Dot characters and traversal tokens are rejected. |

## Error Responses

Error responses are JSON.

| Condition | HTTP status | Response |
|-----------|-------------|----------|
| Missing or blank `guide_blob_path` | `400` | `{"error":"guide_blob_path query parameter is required"}` |
| Invalid `guide_blob_path` prefix, separators, or path traversal | `400` | `{"error":"Invalid request","details":"invalid blob path: must start with pmcs_sbs/"}` |
| `guide_blob_path` does not end in `.json` | `400` | `{"error":"Invalid request","details":"invalid file type: only JSON files are accessible"}` |
| Missing or blank `image_name` | `400` | `{"error":"image_name query parameter is required"}` |
| Invalid `image_name` | `400` | `{"error":"Invalid request","details":"invalid image name: must be an extensionless PNG basename"}` |
| Image does not exist or is not accessible | `404` | `{"error":"Image not found","details":"The requested image does not exist or is not accessible"}` |
| Rate limit exceeded | `429` | No guaranteed JSON body. |
| Azure storage unavailable or read failure | `500` | `{"error":"Failed to retrieve image"}` |

## JSON Error Examples

### Invalid Guide Path

Request:

`GET /api/v1/library/pmcs-sbs/image?guide_blob_path=pmcs/other/file.json&image_name=Before_12`

Response:

```json
{
  "error": "Invalid request",
  "details": "invalid blob path: must start with pmcs_sbs/"
}
```

### Image Name Includes Extension

Request:

`GET /api/v1/library/pmcs-sbs/image?guide_blob_path=pmcs_sbs/HMMWV/HMMWV%20NoArmor%20(SEPT13).json&image_name=Before_12.png`

Response:

```json
{
  "error": "Invalid request",
  "details": "invalid image name: must be an extensionless PNG basename"
}
```

### Image Not Found

Request:

`GET /api/v1/library/pmcs-sbs/image?guide_blob_path=pmcs_sbs/HMMWV/HMMWV%20NoArmor%20(SEPT13).json&image_name=Missing_99`

Response:

```json
{
  "error": "Image not found",
  "details": "The requested image does not exist or is not accessible"
}
```

## Mobile Behavior Notes

- Use the guide `blob_path` returned by the PMCS SBS List Files endpoint as `guide_blob_path`.
- Use the extensionless image name from the loaded guide item as `image_name`.
- Do not append `.png` to `image_name`.
- Do not send folder paths or derived Azure blob paths for item images.
- Treat `200 OK` as binary image data, not JSON.
- Cache image bytes by `guide_blob_path` plus `image_name` if local image caching is needed.
- If the endpoint returns `404`, show the item without that image and keep the rest of the guide usable.
