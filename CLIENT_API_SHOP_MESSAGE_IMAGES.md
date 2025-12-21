# Shop Message Images - Client API Documentation

## Overview
This document describes the API endpoints for uploading and managing images in shop messages. The implementation uses a two-step process: first upload the image to get a URL, then create the message with that URL embedded in the message text.

## Authentication
All endpoints require authentication via Bearer token in the Authorization header:
```
Authorization: Bearer {FIREBASE_TOKEN}
```

## Base URL
```
{BASE_URL}/api/v1/auth
```

---

## Endpoints

### 1. Upload Message Image

Upload an image file to Azure Blob Storage and receive a URL that can be used in a shop message.

**Endpoint:**
```
POST /shops/messages/image/upload
```

**Request Format:**
- Content-Type: `multipart/form-data`

**Request Parameters:**

| Parameter | Type | Location | Required | Description |
|-----------|------|----------|----------|-------------|
| `file` | File | Form data | Yes | The image file to upload |
| `shop_id` | String (UUID) | Query parameter or form data | Yes | The UUID of the shop |

**Accepted File Types:**
- `image/jpeg` (.jpg, .jpeg)
- `image/png` (.png)
- `image/gif` (.gif)
- `image/webp` (.webp)


**Success Response:**
```json
{
  "status": 200,
  "message": "Image uploaded successfully",
  "data": {
    "message_id": "550e8400-e29b-41d4-a716-446655440000",
    "shop_id": "660e8400-e29b-41d4-a716-446655440001",
    "image_url": "https://{account}.blob.core.windows.net/shop-message-images/{shop_id}/{message_id}.jpg",
    "file_extension": ".jpg"
  }
}
```

**Success Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | String (UUID) | Pre-generated UUID for the message (used in blob naming) |
| `shop_id` | String (UUID) | The shop ID from the request |
| `image_url` | String (URL) | Full HTTPS URL to the uploaded image in Azure Blob Storage |
| `file_extension` | String | The file extension based on the detected MIME type |

**Error Responses:**

| Status Code | Message | Cause |
|-------------|---------|-------|
| 400 | "shop_id is required" | Missing shop_id parameter |
| 400 | "failed to get uploaded file" | No file in request or invalid multipart data |
| 401 | "unauthorized" | Missing or invalid authentication token |
| 403 | "access denied: user is not a member of this shop" | User is not a member of the specified shop |
| 500 | "failed to upload image: {error}" | Azure Blob Storage upload failure |
| 500 | "failed to read file data" | Error reading uploaded file data |

---

### 2. Create Shop Message (Existing Endpoint - Modified Usage)

Create a new message in the shop. To include an image, use the URL from the upload endpoint in the message text with the `[IMAGE:url]` tag format.

**Endpoint:**
```
POST /shops/messages
```

**Request Format:**
- Content-Type: `application/json`

**Request Body:**
```json
{
  "shop_id": "660e8400-e29b-41d4-a716-446655440001",
  "message": "[IMAGE:https://{account}.blob.core.windows.net/shop-message-images/{shop_id}/{message_id}.jpg]"
}
```

**Request Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `shop_id` | String (UUID) | Yes | The UUID of the shop |
| `message` | String | Yes | Message text. For image messages, use format: `[IMAGE:{url}]` |

**Message Format for Images:**
```
[IMAGE:{image_url}]
```

Where `{image_url}` is the `image_url` value returned from the upload endpoint.

**Success Response:**
```json
{
  "status": 201,
  "message": "Message created successfully",
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "shop_id": "660e8400-e29b-41d4-a716-446655440001",
    "user_id": "user-uid-from-firebase",
    "message": "[IMAGE:https://{account}.blob.core.windows.net/shop-message-images/{shop_id}/{message_id}.jpg]",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z",
    "is_edited": false
  }
}
```

**Error Responses:**

| Status Code | Message | Cause |
|-------------|---------|-------|
| 400 | "invalid request" | Missing required fields or malformed JSON |
| 401 | "unauthorized" | Missing or invalid authentication token |
| 403 | "access denied: user is not a member of this shop" | User is not a member of the specified shop |
| 500 | "failed to create shop message: {error}" | Database or server error |

---

### 3. Delete Message Image (Cleanup Endpoint)

Delete an uploaded image from Azure Blob Storage. This endpoint is provided for cleanup purposes but is **not typically used** by the client application.

**Endpoint:**
```
DELETE /shops/messages/image/{message_id}
```

**Request Format:**
- Content-Type: `application/json` (or none)

**URL Parameters:**

| Parameter | Type | Location | Required | Description |
|-----------|------|----------|----------|-------------|
| `message_id` | String (UUID) | Path | Yes | The message_id returned from upload endpoint |
| `shop_id` | String (UUID) | Query parameter | Yes | The UUID of the shop |

**Example Request:**
```
DELETE /shops/messages/image/550e8400-e29b-41d4-a716-446655440000?shop_id=660e8400-e29b-41d4-a716-446655440001
```

**Success Response:**
```json
{
  "message": "Image deleted successfully"
}
```

**Error Responses:**

| Status Code | Message | Cause |
|-------------|---------|-------|
| 400 | "message_id is required" | Missing message_id in URL path |
| 400 | "shop_id is required" | Missing shop_id query parameter |
| 401 | "unauthorized" | Missing or invalid authentication token |
| 403 | "access denied: user is not a member of this shop" | User is not a member of the specified shop |
| 500 | "failed to delete message image: {error}" | Azure Blob Storage deletion failure |

---

## Implementation Flow

### Sending an Image Message

**Step 1: Upload the Image**

```
POST {BASE_URL}/api/v1/auth/shops/messages/image/upload
Content-Type: multipart/form-data
Authorization: Bearer {FIREBASE_TOKEN}

Form Data:
- file: [image file]
- shop_id: "660e8400-e29b-41d4-a716-446655440001"
```

**Step 2: Create the Message**

Using the `image_url` from Step 1 response:

```
POST {BASE_URL}/api/v1/auth/shops/messages
Content-Type: application/json
Authorization: Bearer {FIREBASE_TOKEN}

Body:
{
  "shop_id": "660e8400-e29b-41d4-a716-446655440001",
  "message": "[IMAGE:https://{account}.blob.core.windows.net/shop-message-images/660e8400-e29b-41d4-a716-446655440001/550e8400-e29b-41d4-a716-446655440000.jpg]"
}
```

---

## Notes

### Image URL Storage
- Image URLs are stored directly in the `message` field using the `[IMAGE:url]` tag format
- Image-only messages contain only the `[IMAGE:url]` tag with no additional text
- The message text format is: `[IMAGE:{full_blob_url}]`

### File Extension Preservation
- The server automatically detects the image MIME type and assigns the appropriate file extension
- Supported extensions: `.jpg`, `.png`, `.gif`, `.webp`
- The extension is returned in the upload response for reference

### Blob Storage Structure
- Container: `shop-message-images`
- Blob path: `{shop_id}/{message_id}.{extension}`
- All URLs use HTTPS protocol

### Security
- User must be authenticated (Firebase Bearer token)
- User must be a member of the shop to upload images or create messages
- Only specific image MIME types are accepted

---

**Document Version:** 1.0
**Last Updated:** 2025-01-19
**API Version:** v1
