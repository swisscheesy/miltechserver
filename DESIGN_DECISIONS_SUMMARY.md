# Shop Message Images - Design Decisions Summary

## Overview
This document summarizes the finalized design decisions for the shop message image upload feature based on client requirements.

## Finalized Decisions

### ✓ Decision 1: Image Deletion on Message Delete
**Question**: Should images be deleted from Azure when a message is deleted?
**Answer**: **YES**

**Implementation**:
- Modify `DeleteShopMessage` in [shops_service_impl.go:606-618](api/service/shops_service_impl.go#L606-L618)
- Before deleting message from database:
  1. Parse message text for `[IMAGE:url]` pattern
  2. Extract blob path from URL
  3. Delete blob from Azure storage
  4. Handle gracefully if blob doesn't exist
  5. Then proceed with database deletion

**Benefits**:
- Clean storage management
- No orphaned blobs accumulating
- Predictable behavior for users

---

### ✓ Decision 2: No Image Processing
**Question**: Should images be resized/optimized server-side?
**Answer**: **NO - Store as-is**

**Implementation**:
- Accept uploaded images without any transformation
- Store in original format, size, and quality
- No thumbnail generation
- No compression

**Benefits**:
- Simpler implementation
- Faster upload times
- Lower server resource usage
- Client maintains control over quality

**Client Responsibility**:
- Users should compress/resize before upload if desired
- 5 MB max file size enforced

---

### ✓ Decision 3: Single Image Per Message
**Question**: Should messages support multiple images?
**Answer**: **NO - Single image only**

**Implementation**:
- Only one `[IMAGE:url]` tag allowed per message
- Server-side validation: reject messages with multiple `[IMAGE:` tags
- Client-side: prevent users from adding multiple images

**Validation Logic**:
```go
func validateSingleImage(message string) error {
    count := strings.Count(message, "[IMAGE:")
    if count > 1 {
        return errors.New("only one image per message is allowed")
    }
    return nil
}
```

**Benefits**:
- Simpler UI/UX
- Easier to implement
- Clear usage pattern
- Can add multi-image support later if needed

---

### ✓ Decision 4: Preserve File Extensions
**Question**: Should we preserve original extensions or standardize?
**Answer**: **Preserve original extensions**

**Implementation**:
- Detect MIME type from uploaded file
- Map to appropriate extension:
  - `image/jpeg` → `.jpg`
  - `image/png` → `.png`
  - `image/gif` → `.gif`
  - `image/webp` → `.webp`
- Blob naming: `{shopID}/{messageID}.{extension}`

**Benefits**:
- Maintains PNG transparency
- Preserves GIF animation
- Better image quality retention
- More flexible for different use cases

**Example URLs**:
```
https://account.blob.core.windows.net/shop-message-images/shop-123/msg-456.jpg
https://account.blob.core.windows.net/shop-message-images/shop-789/msg-abc.png
https://account.blob.core.windows.net/shop-message-images/shop-def/msg-xyz.gif
```

---

### ✓ Decision 5: Delete Orphaned Blobs
**Question**: How to handle orphaned blobs (upload succeeds, message creation fails)?
**Answer**: **Implement cleanup mechanism**

**Implementation - Two-Pronged Approach**:

**1. Client-Initiated Cleanup Endpoint**:
```
DELETE /api/v1/auth/shops/messages/image/:message_id?shop_id={shopID}
```
- Client calls this if message creation fails or is cancelled
- Server deletes the orphaned blob immediately
- Provides immediate cleanup

**2. Background Cleanup Job** (Safety Net):
- Runs daily (configurable interval)
- Finds blobs uploaded >24 hours ago
- Checks if blob URL exists in any `shop_messages.message` field
- Deletes orphaned blobs
- Logs cleanup statistics

**Benefits**:
- Clean storage
- Cost management
- Automatic recovery from client failures
- Manual cleanup option for clients

---

### ✓ Decision 6: No Aggregate Limits
**Question**: Should there be limits on total images per user/shop?
**Answer**: **NO limits**

**Implementation**:
- No enforcement of per-user image counts
- No enforcement of per-shop image counts
- Rely on:
  - 5 MB max per image
  - Storage monitoring
  - Cost tracking

**Monitoring**:
- Track storage usage metrics
- Alert if costs exceed thresholds
- Can add limits later if abuse detected

**Benefits**:
- Simpler implementation
- More flexible for users
- No arbitrary restrictions
- Easy to add limits later if needed

---

## Additional Implementation Requirements

### Shop Deletion Cascade
**Requirement**: Delete all shop images when shop is deleted

**Implementation**:
- Modify `DeleteShop` in [shops_service_impl.go:82-105](api/service/shops_service_impl.go#L82-L105)
- Before deleting shop from database:
  1. Enumerate all blobs in `shop-message-images/{shopID}/`
  2. Delete all blobs in that directory
  3. Log count of deleted blobs
  4. Then proceed with shop deletion (messages cascade via database)

---

### Message Format Validation
**Requirement**: Enforce single image per message

**Validation Points**:
1. **Client-Side** (recommended):
   - Prevent adding multiple image tags in UI
   - Show error if user attempts multiple images

2. **Server-Side** (required):
   - In `CreateShopMessage` handler
   - Before saving to database:
     ```go
     if strings.Count(message.Message, "[IMAGE:") > 1 {
         return errors.New("only one image per message allowed")
     }
     ```

---

## New Endpoints Summary

### 1. Upload Image
```
POST /api/v1/auth/shops/messages/image/upload
Content-Type: multipart/form-data

Fields:
- file: <image file>
- shop_id: <shop UUID>

Response:
{
  "status": 200,
  "data": {
    "message_id": "generated-uuid",
    "image_url": "https://.../{shopID}/{messageID}.{ext}",
    "file_extension": ".jpg"
  }
}
```

### 2. Delete Orphaned Image
```
DELETE /api/v1/auth/shops/messages/image/:message_id?shop_id={shopID}

Response:
{
  "status": 200,
  "message": "Image deleted successfully"
}
```

---

## Modified Methods

### 1. DeleteShopMessage
**File**: `api/service/shops_service_impl.go`
**Current**: Lines 606-618

**New Logic**:
```go
func (service *ShopsServiceImpl) DeleteShopMessage(user *User, messageID string) error {
    // Get message first
    message, err := service.GetMessageByID(user, messageID)
    if err != nil {
        return err
    }

    // Extract and delete image blob if present
    if strings.Contains(message.Message, "[IMAGE:") {
        imageURL := extractImageURL(message.Message)
        if imageURL != "" {
            // Delete from Azure (graceful failure)
            _ = service.deleteImageBlob(imageURL)
        }
    }

    // Proceed with message deletion
    return service.ShopsRepository.DeleteShopMessage(user, messageID)
}
```

### 2. DeleteShop
**File**: `api/service/shops_service_impl.go`
**Current**: Lines 82-105

**New Logic**:
```go
func (service *ShopsServiceImpl) DeleteShop(user *User, shopID string) error {
    // Check admin permissions (existing logic)
    // ...

    // Delete all shop message images before deleting shop
    err := service.deleteAllShopMessageImages(shopID)
    if err != nil {
        slog.Warn("Failed to delete some shop images", "error", err)
        // Continue with shop deletion even if blob deletion fails
    }

    // Proceed with existing shop deletion
    return service.ShopsRepository.DeleteShop(user, shopID)
}
```

---

## Configuration Constants

```go
const (
    ShopMessageImageContainer = "shop-message-images"
    MaxImageSizeBytes        = 5 * 1024 * 1024  // 5 MB
    OrphanedBlobCleanupAge   = 24 * time.Hour
)

var AllowedImageTypes = map[string]string{
    "image/jpeg": ".jpg",
    "image/png":  ".png",
    "image/gif":  ".gif",
    "image/webp": ".webp",
}
```

---

## Testing Checklist

### Critical Test Cases
- ✓ Upload each supported format (JPEG, PNG, GIF, WebP)
- ✓ Verify correct extension preserved for each format
- ✓ Delete message → blob deleted from Azure
- ✓ Delete shop → all shop blobs deleted
- ✓ Upload image, cancel message creation → call cleanup endpoint
- ✓ Try multiple [IMAGE:] tags → rejected with error
- ✓ Upload 4.9 MB image → success
- ✓ Upload 5.1 MB image → rejected
- ✓ Upload as non-member → 403 error
- ✓ Background job deletes 25+ hour old orphaned blobs

---

## Migration Steps

1. **Create Azure Container**:
   ```bash
   az storage container create \
     --name shop-message-images \
     --account-name {account_name}
   ```

2. **Deploy Code**:
   - New upload/delete endpoints
   - Modified DeleteShopMessage method
   - Modified DeleteShop method

3. **Deploy Background Job**:
   - Orphaned blob cleanup task
   - Schedule: daily at 2 AM

4. **Monitor**:
   - Upload success rate
   - Storage usage
   - Cleanup job results
   - User feedback

---

## Success Metrics

### Technical
- Upload endpoint p95 < 2 seconds
- Error rate < 1%
- Zero orphaned blob accumulation
- Message deletion successfully removes blobs

### User Experience
- Images display correctly in message feed
- Upload process is intuitive
- Error messages are clear

### Business
- Storage costs remain under budget
- No user complaints about image functionality
- Usage metrics show adoption

---

**Document Status**: Finalized
**Approved**: 2025-12-15
**Ready for Implementation**: YES
