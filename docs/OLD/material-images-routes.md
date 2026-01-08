# Material Images API Routes Documentation

## Overview
The Material Images API provides endpoints for users to upload, view, vote on, and manage images associated with specific NIINs (National Item Identification Numbers). The API supports image upload to Azure Blob Storage, voting system, flagging for moderation, and rate limiting.

**Image Data Format:** All image retrieval endpoints return image data as base64-encoded binary data in the `image_data` field. Clients must decode this base64 string to get the raw image bytes for display.

**Client Usage Examples:**
- **JavaScript**: `const imageBytes = atob(response.image_data); const blob = new Blob([imageBytes], {type: response.mime_type});`
- **Python**: `import base64; image_bytes = base64.b64decode(response['image_data'])`
- **Go**: `imageBytes, err := base64.StdEncoding.DecodeString(response.ImageData)`

## Base URL
All endpoints are prefixed with: `/api/v1`

## Authentication
- **Public Routes**: No authentication required
- **Protected Routes**: Require Firebase authentication token in `Authorization` header as `Bearer <token>`

---

## Public Routes

### 1. Get Images by NIIN
**GET** `/material-images/niin/{niin}`

Retrieves all active images for a specific NIIN with pagination, sorted by vote score and upload date. Images are returned as base64-encoded binary data.

**Parameters:**
- `niin` (path, required): 9-character NIIN
- `page` (query, optional): Page number (default: 1, min: 1)
- `page_size` (query, optional): Items per page (default: 20, min: 1, max: 100)

**Query Example:**
```
GET /api/v1/material-images/niin/123456789?page=1&page_size=20
```

**Response:**
```json
{
  "images": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "niin": "123456789",
      "user_id": "user123",
      "username": "john_doe",
      "image_data": "/9j/4AAQSkZJRgABAQEAYABgAAD...base64encodedimagedata...",
      "original_filename": "part_photo.jpg",
      "file_size_bytes": 1048576,
      "mime_type": "image/jpeg",
      "upload_date": "2024-01-15T10:30:00Z",
      "upvote_count": 5,
      "downvote_count": 1,
      "net_votes": 4,
      "is_flagged": false,
      "can_delete": false
    }
  ],
  "total_count": 25,
  "page": 1,
  "page_size": 20,
  "total_pages": 2
}
```

### 2. Get Image by ID
**GET** `/material-images/{image_id}`

Retrieves detailed information about a specific image including base64-encoded binary data.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Query Example:**
```
GET /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "niin": "123456789",
  "user_id": "user123",
  "username": "john_doe",
  "image_data": "/9j/4AAQSkZJRgABAQEAYABgAAD...base64encodedimagedata...",
  "original_filename": "part_photo.jpg",
  "file_size_bytes": 1048576,
  "mime_type": "image/jpeg",
  "upload_date": "2024-01-15T10:30:00Z",
  "upvote_count": 5,
  "downvote_count": 1,
  "net_votes": 4,
  "is_flagged": false,
  "can_delete": false
}
```

---

## Protected Routes (Authentication Required)

### 3. Upload Image
**POST** `/material-images/upload`

Uploads a new image for a specific NIIN. Rate limited to 1 upload per NIIN per user per 1 hour.

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `niin` (required): 9-character NIIN
- `image` (required): Image file (max 10MB, formats: JPEG, PNG, WebP)

**Request Example:**
```bash
curl -X POST /api/v1/material-images/upload \
  -H "Authorization: Bearer <token>" \
  -F "niin=123456789" \
  -F "image=@photo.jpg"
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Image uploaded successfully",
  "image": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "niin": "123456789",
    "user_id": "user123",
    "username": "john_doe",
    "image_data": "/9j/4AAQSkZJRgABAQEAYABgAAD...base64encodedimagedata...",
    "original_filename": "photo.jpg",
    "file_size_bytes": 1048576,
    "mime_type": "image/jpeg",
    "upload_date": "2024-01-15T10:30:00Z",
    "upvote_count": 0,
    "downvote_count": 0,
    "net_votes": 0,
    "is_flagged": false,
    "can_delete": true
  }
}
```

**Error Responses:**
- `400`: Invalid NIIN, file too large, unsupported format, rate limit exceeded
- `401`: Authentication required

### 4. Delete Image
**DELETE** `/material-images/{image_id}`

Deletes an image (soft delete). Only the uploader can delete their own images.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Request Example:**
```bash
curl -X DELETE /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

**Response (Success):**
```json
{
  "success": true,
  "message": "Image deleted successfully"
}
```

**Error Responses:**
- `403`: Unauthorized (not the uploader)
- `404`: Image not found
- `401`: Authentication required

### 5. Get Images by User
**GET** `/material-images/user/{user_id}`

Retrieves all images uploaded by a specific user with pagination. Images are returned as base64-encoded binary data.

**Parameters:**
- `user_id` (path, required): User ID
- `page` (query, optional): Page number (default: 1, min: 1)
- `page_size` (query, optional): Items per page (default: 20, min: 1, max: 100)

**Query Example:**
```
GET /api/v1/material-images/user/user123?page=1&page_size=20
```

**Response:** Same format as "Get Images by NIIN" with `can_delete: true` for all images. Images include base64-encoded `image_data` field instead of `blob_url`.

### 6. Vote on Image
**POST** `/material-images/{image_id}/vote`

Casts or updates a vote on an image. Users can change their vote type.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Request Body:**
```json
{
  "vote_type": "upvote"  // "upvote" or "downvote"
}
```

**Request Example:**
```bash
curl -X POST /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000/vote \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type": "upvote"}'
```

**Response:**
```json
{
  "success": true,
  "message": "Vote recorded successfully",
  "upvote_count": 6,
  "downvote_count": 1,
  "net_votes": 5
}
```

### 7. Remove Vote
**DELETE** `/material-images/{image_id}/vote`

Removes the user's vote from an image.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Request Example:**
```bash
curl -X DELETE /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000/vote \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "success": true,
  "message": "Vote removed successfully",
  "upvote_count": 5,
  "downvote_count": 1,
  "net_votes": 4
}
```

### 8. Flag Image
**POST** `/material-images/{image_id}/flag`

Flags an image for moderation review. Each user can only flag an image once.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Request Body:**
```json
{
  "reason": "inappropriate",  // "incorrect_item", "inappropriate", "poor_quality", "duplicate", "other"
  "description": "Optional detailed description of the issue"
}
```

**Request Example:**
```bash
curl -X POST /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000/flag \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "poor_quality", "description": "Image is too blurry to be useful"}'
```

**Response:**
```json
{
  "success": true,
  "message": "Image flagged successfully",
  "flag_count": 3,
  "is_flagged": false
}
```

**Error Responses:**
- `409`: User has already flagged this image
- `400`: Invalid flag reason

### 9. Get Image Flags (Admin Only)
**GET** `/material-images/{image_id}/flags`

Retrieves all flags for a specific image. Intended for admin/moderation use.

**Parameters:**
- `image_id` (path, required): UUID of the image

**Request Example:**
```bash
curl -X GET /api/v1/material-images/550e8400-e29b-41d4-a716-446655440000/flags \
  -H "Authorization: Bearer <token>"
```

**Response:**
```json
{
  "flags": [
    {
      "id": "flag-uuid-1",
      "image_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "user456",
      "reason": "poor_quality",
      "description": "Image is too blurry to be useful",
      "created_at": "2024-01-15T11:00:00Z"
    },
    {
      "id": "flag-uuid-2",
      "image_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "user789",
      "reason": "incorrect_item",
      "description": "This part doesn't match the NIIN",
      "created_at": "2024-01-15T12:00:00Z"
    }
  ]
}
```

---

## Business Rules & Constraints

### Upload Limits
- **File Size**: Maximum 10MB per image
- **File Types**: JPEG, PNG, WebP only
- **Rate Limiting**: 1 upload per NIIN per user per 1 hour
- **Storage**: Images stored in Azure Blob Storage container `material-images`

### Voting System
- Users can vote "upvote" or "downvote" on any image
- Users can change their vote type
- Users can remove their vote completely
- Net votes = upvotes - downvotes
- Images are sorted by net votes (descending) then upload date (newest first)

### Flagging System
- Each user can flag an image only once
- Valid flag reasons: `incorrect_item`, `inappropriate`, `poor_quality`, `duplicate`, `other`
- Images with e5 flags are automatically marked as `is_flagged: true`
- Flagged images remain visible but are marked for moderation review

### Authorization Rules
- **Upload**: Authenticated users only
- **Delete**: Only the original uploader
- **Vote**: Authenticated users only (cannot vote on own images - optional restriction)
- **Flag**: Authenticated users only
- **View Flags**: Admin users only (TODO: implement admin middleware)

### NIIN Validation
- Must be exactly 9 characters
- Should exist in NSN table (optional validation can be added)

---

## Error Responses

### Common HTTP Status Codes
- `200`: Success
- `201`: Created (successful upload)
- `400`: Bad Request (validation errors, rate limits)
- `401`: Unauthorized (missing/invalid token)
- `403`: Forbidden (insufficient permissions)
- `404`: Not Found
- `409`: Conflict (duplicate flag)
- `500`: Internal Server Error

### Error Response Format
```json
{
  "error": "Descriptive error message"
}
```

---

## Implementation Notes

### Current Limitations
1. **Username Resolution**: Currently returns "Unknown" for usernames. Future enhancement will add JOIN queries to fetch actual usernames.
2. **Admin Middleware**: Flag viewing endpoint needs admin authorization middleware.
3. **Blob Cleanup**: Deleted images remain in blob storage for audit purposes.
4. **Image Processing**: No thumbnail generation or image optimization currently implemented.

### Future Enhancements
1. Add username resolution via database JOINs
2. Implement admin middleware for flag management
3. Add image thumbnail generation
4. Add duplicate image detection
5. Add batch operations for admin users
6. Add image analytics and metrics
7. Implement SAS token generation for secure blob access

### Performance Considerations
1. All list endpoints use pagination
2. Database indexes on `niin`, `user_id`, `upload_date`, and `net_votes`
3. Soft deletes preserve data integrity
4. Rate limiting prevents abuse
5. **Image Data Overhead**: Base64 encoding increases response size by ~33%
6. **Large Payloads**: Consider pagination limits when dealing with multiple large images
7. **Blob Download**: Each image request triggers a blob storage download

### Security Measures
1. File type validation by content type and extension
2. File size limits prevent storage abuse
3. Rate limiting prevents spam
4. User authorization for sensitive operations
5. Input sanitization and validation
6. Firebase authentication integration