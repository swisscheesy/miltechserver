# Shop Message Image Upload - Design Document

## Overview
This document outlines the design for adding image upload capability to shop messages. Users will be able to upload images that are stored in Azure Blob Storage and referenced in shop messages using a URL tag format.

## Current Implementation Analysis

### Existing Shop Messages System
**Location**: [shops_service_impl.go:464-493](api/service/shops_service_impl.go#L464-L493)

**Current Message Flow**:
1. User sends JSON request with `shopID` and `message` text
2. Service validates user is a shop member
3. Message saved to `shop_messages` table
4. Returns created message with metadata

**ShopMessages Model** ([model/shop_messages.go](/.gen/miltech_ng/public/model/shop_messages.go)):
```go
type ShopMessages struct {
    ID        string      // UUID, primary key
    ShopID    string      // Foreign key to shops
    UserID    string      // Message author
    Message   string      // Message content (will contain [IMAGE:url] tags)
    CreatedAt *time.Time
    UpdatedAt *time.Time
    IsEdited  *bool
}
```

### Existing Image Upload Pattern (User Saves)
**Reference**: [user_saves_repository_impl.go:809-838](api/repository/user_saves_repository_impl.go#L809-L838)

**Pattern Used**:
```
Controller (multipart form-data)
    → Service Layer
    → Repository (Azure Blob Upload)
    → Database Update (URL stored)
    → Return blob URL
```

**Key Implementation Details**:
- Container: `user-item-images`
- Blob naming: `{userID}/{itemID}.jpg`
- URL format: `https://{accountName}.blob.core.windows.net/{container}/{blobName}`
- File received as multipart form-data with "file" field
- Image data read into byte array for upload

## Proposed Solution

### Design Decision: Separate Upload Endpoint

**Chosen Approach**: Create a dedicated image upload endpoint separate from message creation.

**Rationale**:
1. **Flexibility**: Allows client to upload image first, get URL, then include in message
2. **Consistency**: Matches the user_saves pattern already established in codebase
3. **Error Handling**: Easier to handle upload failures separately from message creation
4. **Reusability**: Upload endpoint could potentially be used for editing messages with images later
5. **Client Control**: Client can validate upload success before creating message

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Application                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Step 1: Upload Image
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  POST /api/v1/auth/shops/messages/image/upload              │
│  Content-Type: multipart/form-data                          │
│  Body: { file: <image>, shop_id: "<uuid>" }                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              ShopsController.UploadMessageImage             │
│  - Validates user authentication                            │
│  - Validates shop membership                                │
│  - Extracts file from multipart form                        │
│  - Generates message ID for blob naming                     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│           ShopsService.UploadMessageImage                   │
│  - Business logic validation                                │
│  - Forwards to repository                                   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│         ShopsRepository.UploadMessageImage                  │
│  - Upload to Azure Blob Storage                             │
│  - Container: "shop-message-images"                         │
│  - Blob: {shopID}/{messageID}.jpg                           │
│  - Returns blob URL                                         │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Returns: blob URL
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Client Application                        │
│  Receives: { image_url: "https://..." }                     │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ Step 2: Create Message with Image URL
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  POST /api/v1/auth/shops/:shop_id/messages                  │
│  Body: {                                                     │
│    "shop_id": "<uuid>",                                      │
│    "message": "[IMAGE:https://azure.url/image.jpg]"         │
│  }                                                           │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│           Existing CreateShopMessage Flow                   │
│  - Stores message with embedded [IMAGE:url] tag             │
│  - Returns ShopMessages model                               │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Design Specifications

### 1. New Endpoint

**Route**: `POST /api/v1/auth/shops/messages/image/upload`

**Request Format**:
- Content-Type: `multipart/form-data`
- Form Fields:
  - `file`: The image file (required)
  - `shop_id`: UUID of the shop (required, query parameter or form field)

**Response Format**:
```json
{
  "status": 200,
  "message": "Image uploaded successfully",
  "data": {
    "message_id": "uuid-generated-for-blob-naming",
    "shop_id": "original-shop-id",
    "image_url": "https://{account}.blob.core.windows.net/shop-message-images/{shopID}/{messageID}.{ext}",
    "file_extension": ".jpg"
  }
}
```

**Additional Endpoint for Cleanup**:

**Route**: `DELETE /api/v1/auth/shops/messages/image/:message_id`

**Purpose**: Cleanup orphaned images when message creation fails or is cancelled

**Request Parameters**:
- `message_id`: UUID of the pre-generated message ID (URL parameter)
- `shop_id`: UUID of the shop (query parameter)

**Response Format**:
```json
{
  "status": 200,
  "message": "Image deleted successfully"
}
```

**Error Responses** (for cleanup endpoint):
- `401 Unauthorized`: User not authenticated
- `403 Forbidden`: User not a member of the shop
- `404 Not Found`: Image not found
- `500 Internal Server Error`: Deletion failed

**Error Responses** (for upload endpoint):
- `401 Unauthorized`: User not authenticated
- `403 Forbidden`: User not a member of the shop
- `400 Bad Request`: Missing file or shop_id, invalid file type, file too large, multiple images
- `500 Internal Server Error`: Azure upload failure

### 2. Azure Blob Storage Configuration

**Container Name**: `shop-message-images`

**Blob Naming Convention**: `{shopID}/{messageID}.{extension}`
- **shopID**: UUID of the shop (provides organization by shop)
- **messageID**: Pre-generated UUID for the message (ensures unique blob names)
- **Extension**: Preserved from original file based on MIME type
  - `image/jpeg` → `.jpg`
  - `image/png` → `.png`
  - `image/gif` → `.gif`
  - `image/webp` → `.webp`

**Rationale for Naming**:
- Organizes blobs by shop (makes it easier to manage/delete shop-related images)
- Uses message ID (not user ID) since the message "owns" the image
- Pre-generating message ID ensures blob name is available before message creation
- **Preserving extension** maintains image quality (PNG transparency, GIF animation)
- Allows for easy blob cleanup if message creation fails

### 3. Image Validation Requirements

**File Size Limits**:
- **Maximum**: 5 MB per image (configurable)
- Prevents storage abuse and ensures reasonable upload times

**Accepted File Types**:
- `image/jpeg` (.jpg, .jpeg)
- `image/png` (.png)
- `image/gif` (.gif)
- `image/webp` (.webp)

**Security Validations**:
1. Verify user is authenticated
2. Verify user is member of the specified shop
3. Validate file MIME type (not just extension)
4. Validate file size before upload
5. Generate unique blob names to prevent overwrites

### 4. Message Format Specification

**Image Tag Format**: `[IMAGE:{url}]`

**Examples**:
```
// Text only
"Check out this new part we need to order"

// Image only
"[IMAGE:https://account.blob.core.windows.net/shop-message-images/shop-123/msg-456.jpg]"

// Text with image
"Here's the damaged component: [IMAGE:https://account.blob.core.windows.net/shop-message-images/shop-123/msg-789.png]"
```

**Limitations**:
- **Single image per message**: Only one `[IMAGE:url]` tag supported per message
- Multiple images not supported in this implementation

**Client Responsibilities**:
- Parse single `[IMAGE:...]` tag from message text
- Display image inline or as attachment
- Handle image loading failures gracefully
- Prevent users from adding multiple image tags

### 5. Database Schema

**No Changes Required** to `shop_messages` table.

The `message` column (string) will store the text with embedded `[IMAGE:url]` tags. This approach:
- Requires no schema migration
- Allows for flexible message formats (text, images, or both)
- Maintains backward compatibility with existing messages
- Supports multiple images per message in the future

## Implementation Components

### New Files to Create

1. **Controller Method**: `shops_controller.go`
   - `UploadMessageImage(c *gin.Context)` - Upload endpoint
   - `DeleteMessageImage(c *gin.Context)` - Cleanup endpoint for orphaned images
   - Handles multipart form-data
   - Extracts file, shop_id, and determines file extension
   - Validates inputs (file type, size, single image)
   - Returns uploaded image URL and message ID

2. **Service Interface**: `shops_service.go`
   - Add: `UploadMessageImage(user *User, shopID string, imageData []byte, fileExtension string) (string, string, error)`
     - Returns: messageID, imageURL, error
   - Add: `DeleteMessageImage(user *User, shopID, messageID string) error`
     - For cleanup of orphaned images

3. **Service Implementation**: `shops_service_impl.go`
   - Implement `UploadMessageImage`
     - Verify shop membership
     - Generate message ID
     - Validate file extension
     - Delegate to repository
   - Implement `DeleteMessageImage`
     - Verify shop membership
     - Delete blob from Azure

4. **Repository Interface**: `shops_repository.go`
   - Add: `UploadMessageImage(user *User, shopID, messageID string, imageData []byte, fileExtension string) (string, error)`
     - Returns: imageURL, error
   - Add: `DeleteMessageImageBlob(shopID, messageID, fileExtension string) error`
     - Delete blob from Azure

5. **Repository Implementation**: `shops_repository_impl.go`
   - Implement Azure blob upload
   - Use container: `shop-message-images`
   - Blob naming: `{shopID}/{messageID}.{extension}`
   - Return constructed URL
   - Implement blob deletion helper

6. **Route Registration**: `shops_route.go`
   - `POST /api/v1/auth/shops/messages/image/upload`
   - `DELETE /api/v1/auth/shops/messages/image/:message_id` (cleanup endpoint)

### Modified Files

1. **shops_service_impl.go** - `DeleteShopMessage` method
   - Before deleting message, parse message text for `[IMAGE:url]` tag
   - If found, extract blob path and delete from Azure
   - Handle cases where blob may not exist (graceful failure)
   - Then proceed with existing message deletion logic

2. **shops_service_impl.go** - `DeleteShop` method
   - Before deleting shop, enumerate all blobs in `shop-message-images/{shopID}/`
   - Delete all blobs for that shop
   - Then proceed with existing shop deletion logic (messages cascade automatically)

## API Usage Flow

### Complete Client Flow Example

```javascript
// Step 1: User selects an image
const imageFile = document.getElementById('imageInput').files[0];
const shopId = 'current-shop-uuid';

// Step 2: Upload image to get URL
const formData = new FormData();
formData.append('file', imageFile);
formData.append('shop_id', shopId);

const uploadResponse = await fetch('/api/v1/auth/shops/messages/image/upload', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: formData
});

const { data } = await uploadResponse.json();
const imageUrl = data.image_url;

// Step 3: Create message with image URL
const messageResponse = await fetch(`/api/v1/auth/shops/${shopId}/messages`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    shop_id: shopId,
    message: `[IMAGE:${imageUrl}]`
  })
});

// Result: Message created with image
```

## Edge Cases and Error Handling

### 1. Image Upload Succeeds, Message Creation Fails
**Problem**: Orphaned blob in Azure storage
**Solution**: **Delete orphaned blobs**

**Implementation Strategy**:
- Track uploaded blob URL/path during upload
- If message creation fails on client side, client should call cleanup endpoint
- Alternatively, implement background job to delete blobs older than 24 hours with no associated message in database
- This keeps storage clean and prevents accumulation of unused blobs

**Technical Approach**:
1. Upload endpoint returns both `message_id` and `image_url`
2. Client stores these temporarily
3. If message creation fails or is cancelled, client calls: `DELETE /api/v1/auth/shops/messages/image/{message_id}`
4. Server deletes blob from Azure storage
5. Background job provides safety net for missed cleanups

### 2. User Uploads Image but Never Creates Message
**Problem**: Storage waste from abandoned uploads
**Solution**: Background cleanup job
- Runs daily/hourly
- Queries for blobs uploaded >24 hours ago
- Checks if blob path exists in any `shop_messages.message` field
- Deletes orphaned blobs
- Logs cleanup statistics

### 3. File Type Spoofing
**Problem**: User renames .exe to .jpg
**Solution**: Validate actual file content (magic bytes) not just extension

### 4. Large File DoS
**Problem**: User uploads very large files repeatedly
**Solution**:
- Enforce strict file size limits (5 MB)
- Implement rate limiting on upload endpoint
- Monitor per-user upload metrics

### 5. Shop Deletion
**Problem**: What happens to message images when shop is deleted?
**Solution**: Cascade blob deletion on shop deletion
- When shop is deleted, enumerate all blobs in `shop-message-images/{shopID}/`
- Delete all blobs for that shop directory
- Database already handles message deletion via CASCADE
- Implement in `DeleteShop` service method

### 6. Message Deletion
**Problem**: What happens to images when message is deleted?
**Solution**: Delete blob from Azure when message deleted
- Extract `[IMAGE:url]` from message text before deletion
- Parse URL to get blob path
- Delete blob from Azure storage
- Then delete message from database
- Implement in `DeleteShopMessage` service method
- Handle cases where blob may already be deleted (graceful failure)

## Security Considerations

### Authentication & Authorization
1. **Endpoint Protection**: Upload endpoint requires authentication (via middleware)
2. **Shop Membership**: Verify user is member of shop before allowing upload
3. **Ownership**: Message images are scoped to shop, not individual user

### Storage Security
1. **Access Control**: Azure blob container should have appropriate access policies
2. **Blob Names**: UUIDs prevent predictable blob names / enumeration attacks
3. **HTTPS Only**: All Azure blob URLs should use HTTPS
4. **No Public Write**: Container configured for authenticated write only

### Input Validation
1. **File Type Validation**: Check both extension and MIME type
2. **File Size Validation**: Enforce maximum size before upload
3. **Shop ID Validation**: Ensure shop exists and user has access
4. **Malware Scanning**: Consider Azure blob malware scanning for production

## Performance Considerations

### Upload Performance
- **Expected**: ~1-2 seconds for typical image (1-2 MB)
- **Network**: Directly upload to Azure, minimal server processing
- **Concurrency**: Azure SDK handles connection pooling

### Storage Costs
- **Estimate**: ~$0.02 per GB/month (Azure Blob Storage Standard)
- **Example**: 10,000 images @ 1MB avg = 10GB = ~$0.20/month

### Client Experience
- Upload progress indication recommended
- Consider image compression on client before upload
- Lazy loading for images in message feed

## Testing Requirements

### Unit Tests
1. Controller validation logic
2. Service shop membership checks
3. Repository blob upload (mock Azure client)
4. File extension extraction from MIME type
5. Message deletion with image cleanup
6. Shop deletion with bulk blob cleanup
7. Single image tag validation

### Integration Tests
1. End-to-end upload flow
2. Error handling (invalid shop, non-member, etc.)
3. File size validation
4. File type validation
5. Message deletion triggers blob deletion
6. Shop deletion triggers all shop blobs deletion
7. Orphaned image cleanup via DELETE endpoint
8. Multiple image tag rejection

### Manual Testing Scenarios
1. Upload various image formats (JPEG, PNG, GIF, WebP) - verify extension preserved
2. Upload at size limits (just under and just over 5MB)
3. Upload with invalid shop ID
4. Upload as non-shop-member
5. Create message with uploaded image URL
6. Verify image displays correctly in message feed
7. Delete message, verify blob deleted from Azure
8. Delete shop, verify all shop message images deleted
9. Upload image, don't create message, call cleanup endpoint
10. Try to create message with multiple [IMAGE:] tags - should fail validation
11. Upload image with correct extension based on actual file type

## Configuration

### Environment Variables (Existing)
- `BLOB_ACCOUNT_NAME`: Azure storage account name
- `BLOB_ACCOUNT_KEY`: Azure storage account key

### New Configuration Constants
```go
const (
    ShopMessageImageContainer = "shop-message-images"
    MaxImageSizeBytes        = 5 * 1024 * 1024  // 5 MB
    AllowedImageTypes        = map[string]string{
        "image/jpeg": ".jpg",
        "image/png":  ".png",
        "image/gif":  ".gif",
        "image/webp": ".webp",
    }
    OrphanedBlobCleanupAge   = 24 * time.Hour  // Delete orphaned blobs after 24 hours
)
```

## Migration Plan

### Phase 1: Implementation
1. Create new endpoint for image upload
2. Implement service and repository layers
3. Add route registration
4. Unit tests

### Phase 2: Testing
1. Integration testing in development environment
2. Create Azure container `shop-message-images`
3. Manual testing with various image types/sizes

### Phase 3: Deployment
1. Deploy to staging environment
2. User acceptance testing
3. Production deployment
4. Monitor upload metrics and errors

### Phase 4: Cleanup & Monitoring
1. Implement background job for orphaned blob cleanup (runs daily)
   - Query blobs older than 24 hours
   - Check if URL exists in any shop_messages record
   - Delete orphaned blobs
2. Add monitoring for storage usage metrics
3. Monitor cleanup job execution and results
4. (Future) Consider image optimization if storage becomes concern

## Design Decisions (Finalized)

### Decision 1: Image Limits ✓
**Decision**: No limits on images per user or shop
- Single image per message only
- No aggregate limits enforced
- Rely on storage monitoring and costs
- Can add limits later if abuse occurs

### Decision 2: Image Processing ✓
**Decision**: No server-side processing
- Store images as-is (original format, size, quality)
- Simpler implementation, faster uploads
- Lower server resource usage
- Client responsible for any pre-upload optimization
- Future enhancement: can add thumbnail generation later if needed

### Decision 3: Image Deletion ✓
**Decision**: Delete blobs when messages deleted
- When `DeleteShopMessage` is called, extract image URL from message
- Delete blob from Azure storage before deleting message record
- Implement graceful handling if blob already deleted
- Keeps storage clean, prevents orphaned blobs

### Decision 4: File Extensions ✓
**Decision**: Preserve original file extensions
- Maintain image quality characteristics (PNG transparency, GIF animation)
- Blob naming: `{shopID}/{messageID}.{original-extension}`
- Accepted extensions: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`
- Extension determined from uploaded file's MIME type

### Decision 5: Multiple Images ✓
**Decision**: Single image per message only
- Do NOT support multiple `[IMAGE:url]` tags per message
- Client should validate and prevent multiple image tags
- Server validation: reject messages with more than one `[IMAGE:` tag
- Future enhancement: can add multi-image support later if requested

### Decision 6: Orphaned Blob Cleanup ✓
**Decision**: Implement cleanup mechanism for orphaned blobs
- Delete blobs when upload succeeds but message creation fails/is cancelled
- Provide optional cleanup endpoint: `DELETE /api/v1/auth/shops/messages/image/{message_id}`
- Implement background job to delete blobs >24 hours old with no associated message
- Helps manage storage costs and keeps container clean

## Success Metrics

### Implementation Success
- Upload endpoint successfully stores images in Azure
- Image URLs correctly formatted and accessible
- Messages with [IMAGE:url] tags created successfully
- No degradation of existing message functionality

### User Experience
- Upload completes in < 3 seconds for typical images
- Error messages are clear and actionable
- Images display correctly in message feed
- No broken image links

### System Health
- Upload endpoint response time < 2s (p95)
- Error rate < 1%
- Azure blob storage costs within budget
- No orphaned blob accumulation issues

## References

### Related Code Files
- [shops_service_impl.go:464-493](api/service/shops_service_impl.go#L464-L493) - Current message creation
- [user_saves_controller.go:498-553](api/controller/user_saves_controller.go#L498-L553) - Image upload reference
- [user_saves_repository_impl.go:809-838](api/repository/user_saves_repository_impl.go#L809-L838) - Azure upload pattern
- [azure_blob.go](bootstrap/azure_blob.go) - Azure client setup
- [model/shop_messages.go](/.gen/miltech_ng/public/model/shop_messages.go) - ShopMessages model

### Azure Documentation
- [Azure Blob Storage Go SDK](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob)
- [Blob Storage Best Practices](https://docs.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction)

---

**Document Version**: 1.0
**Last Updated**: 2025-12-15
**Status**: Design Phase - Awaiting Approval
