# Shop Message Image Deletion Implementation

## Overview
This document describes the implementation that ensures Azure Blob Storage images are automatically deleted when shop messages containing images are deleted.

## Problem Statement
When users delete shop messages that contain images, the image blobs in Azure Blob Storage were not being deleted, causing:
- Orphaned blobs accumulating in storage
- Unnecessary storage costs
- Data management issues

## Solution Design

### Architecture Flow

```
User Deletes Message
        ↓
Controller: DeleteShopMessage
        ↓
Service: DeleteShopMessage
        ↓
    ┌───────────────────────────────────┐
    │ 1. Get message by ID              │
    │ 2. Delete database record         │
    │ 3. Extract [IMAGE:url] from text  │
    │ 4. Delete blob from Azure         │
    └───────────────────────────────────┘
```

### Implementation Details

#### 1. Repository Layer ([shops_repository_impl.go](api/repository/shops_repository_impl.go))

**New Methods Added:**

**`GetShopMessageByID(user, messageID)`**
- Retrieves a single message before deletion
- Returns the full message model including text
- Used to extract image URL before deletion

**`DeleteBlobByURL(messageText)`**
- Takes message text as input
- Extracts image URL from `[IMAGE:url]` tag
- Parses blob name from Azure URL
- Deletes blob from Azure Blob Storage
- Returns nil (graceful failure) if:
  - No image URL found in message
  - URL parsing fails
  - Blob doesn't exist
  - Azure deletion fails

**New Helper Functions:**

**`extractImageURLFromMessage(messageText string)`**
- Uses regex to extract URL from `[IMAGE:https://...]` tag
- Pattern: `\[IMAGE:(https://[^\]]+)\]`
- Returns empty string if no image tag found
- Handles text-only messages gracefully

**`parseBlobNameFromURL(url, container)`**
- Extracts blob path from full Azure URL
- Input: `https://{account}.blob.core.windows.net/shop-message-images/{shopID}/{messageID}.jpg`
- Output: `{shopID}/{messageID}.jpg`
- Validates container name in URL
- Returns error if URL format is invalid

#### 2. Service Layer ([shops_service_impl.go](api/service/shops_service_impl.go))

**Updated Method: `DeleteShopMessage(user, messageID)`**

**New Logic:**
```go
1. Get message by ID (to retrieve message text with image URL)
2. Delete database record (primary operation)
3. Attempt to delete blob from Azure (secondary operation)
   - If blob deletion fails, log warning but don't fail operation
   - Graceful degradation ensures message deletion always succeeds
```

**Key Design Decision:**
- Database deletion happens FIRST
- Blob deletion happens AFTER
- Blob deletion failures don't cause operation to fail

**Rationale:**
- Prevents orphaned database records (worse than orphaned blobs)
- Message deletion is the primary user intent
- Blob cleanup is secondary/housekeeping
- Failed blob deletions are logged for monitoring

#### 3. Repository Interface ([shops_repository.go](api/repository/shops_repository.go))

**New Interface Methods:**
```go
GetShopMessageByID(user *bootstrap.User, messageID string) (*model.ShopMessages, error)
DeleteBlobByURL(messageText string) error
```

## Technical Implementation

### Message Format Handling

**Image Message Example:**
```
[IMAGE:https://account.blob.core.windows.net/shop-message-images/shop-uuid/message-uuid.jpg]
```

**Extraction Process:**
1. Regex matches `[IMAGE:(url)]` pattern
2. Captures URL between `[IMAGE:` and `]`
3. Returns extracted URL for blob deletion

### Blob URL Parsing

**URL Structure:**
```
https://{account}.blob.core.windows.net/{container}/{blob_path}
                                        ↓
                            shop-message-images/{shopID}/{messageID}.{ext}
```

**Parsing Logic:**
1. Find container name in URL (`/shop-message-images/`)
2. Extract everything after container prefix
3. Result: `{shopID}/{messageID}.{ext}`

### Error Handling Strategy

**Graceful Degradation:**
- All blob deletion operations fail silently
- Errors are logged but not returned
- Primary operation (message deletion) never fails due to blob issues

**Logged Scenarios:**
- No image URL found in message (informational)
- Failed to parse blob URL (warning)
- Azure blob deletion failed (warning)

**Success Logging:**
- Blob deleted successfully with blob_name and image_url

## Testing Considerations

### Test Cases

1. **Delete message with image**
   - Verify database record deleted
   - Verify blob deleted from Azure
   - Check logs for successful blob deletion

2. **Delete message without image**
   - Verify database record deleted
   - Verify no errors logged
   - Confirm graceful handling of no image

3. **Delete message with malformed URL**
   - Verify database record deleted
   - Check warning logged for URL parsing failure
   - Confirm operation completes successfully

4. **Delete message when blob already deleted**
   - Verify database record deleted
   - Check warning logged for blob not found
   - Confirm operation completes successfully

5. **Delete message when Azure is unavailable**
   - Verify database record deleted
   - Check warning logged for Azure failure
   - Confirm operation completes successfully

### Manual Testing Steps

1. Upload an image using the upload endpoint
2. Create a message with the returned image URL
3. Delete the message
4. Check Azure Blob Storage to confirm blob is deleted
5. Verify message no longer appears in shop messages

## Performance Impact

**Additional Operations:**
- 1 additional database SELECT (to get message before deletion)
- 1 Azure Blob Storage DELETE call (async operation)

**Expected Overhead:**
- ~10-50ms for message retrieval
- ~50-200ms for Azure blob deletion
- Total: ~60-250ms additional latency

**Mitigation:**
- Blob deletion happens after database deletion (non-blocking for user)
- Failures don't impact user experience
- Could be moved to async background job if needed

## Future Enhancements

### Potential Improvements

1. **Async Blob Deletion**
   - Move blob deletion to background job queue
   - Reduce deletion latency to ~10-50ms
   - Better for high-volume operations

2. **Bulk Deletion Support**
   - When deleting shop, delete all shop message images
   - Batch deletion for efficiency
   - Reduce API calls to Azure

3. **Orphaned Blob Cleanup Job**
   - Scheduled job to find and delete orphaned blobs
   - Safety net for failed deletions
   - Query blobs older than 24 hours with no corresponding message

## Dependencies

**Required Packages:**
- `regexp` - For extracting image URLs from message text
- `strings` - For URL parsing and manipulation
- `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` - Azure Blob Storage client

**No New Dependencies Added** - all packages already in use

## Security Considerations

**Authorization:**
- User must have permission to access message (existing check)
- Blob deletion only after successful message retrieval
- No direct blob manipulation without message ownership

**Data Integrity:**
- Database deletion happens first (prevents orphaned records)
- Blob deletion is best-effort (graceful degradation)
- No risk of deleting wrong blobs (URL extracted from actual message)

## Monitoring & Logging

**Success Logs:**
```
INFO: Blob deleted successfully from Azure
      blob_name={shopID}/{messageID}.{ext}
      image_url={full_url}
```

**Warning Logs:**
```
WARN: Failed to parse blob name from URL
      url={image_url}
      error={error_message}

WARN: Failed to delete blob from Azure
      blob_name={blob_path}
      error={error_message}

WARN: Failed to delete blob during message deletion
      message_id={uuid}
      user_id={user_id}
      error={error_message}
```

**Recommended Monitoring:**
- Track blob deletion failure rate
- Alert if failure rate > 5%
- Monitor orphaned blob count growth
- Dashboard for storage usage trends

---

**Implementation Date:** 2025-01-19
**Status:** ✅ Completed & Built Successfully
**Breaking Changes:** None
**Backward Compatibility:** Full
