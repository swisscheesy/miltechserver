# Material Images Feature Implementation Plan

## Overview
This document provides a comprehensive implementation plan for the Material Images feature, allowing logged-in users to upload, view, and manage images for specific NIINs (National Item Identification Numbers). The images will be stored in Azure Blob Storage container 'material-images' with metadata tracked in PostgreSQL.

## Feature Requirements Summary
1. **Upload**: Logged-in users can upload images for specific NIINs (rate-limited to 1 image per NIIN per 24 hours)
2. **View**: All users can view material images
3. **Delete**: Only the uploader can delete their own images
4. **Vote/Flag**: Logged-in users can downvote/flag images for review
5. **Storage**: Images stored in Azure Blob container 'material-images'
6. **Multiple Images**: Each NIIN can have multiple associated images

## Database Schema Design

### New Tables Required

#### 1. material_images
Primary table for tracking uploaded images.

```sql
CREATE TABLE material_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    niin VARCHAR(9) NOT NULL,
    user_id TEXT NOT NULL,
    blob_name TEXT NOT NULL UNIQUE,
    blob_url TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    upload_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_flagged BOOLEAN NOT NULL DEFAULT false,
    flag_count INTEGER NOT NULL DEFAULT 0,
    downvote_count INTEGER NOT NULL DEFAULT 0,
    upvote_count INTEGER NOT NULL DEFAULT 0,
    net_votes INTEGER GENERATED ALWAYS AS (upvote_count - downvote_count) STORED,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE,
    CONSTRAINT valid_niin CHECK (length(niin) = 9)
);

-- Indexes for performance
CREATE INDEX idx_material_images_niin ON material_images(niin) WHERE is_active = true;
CREATE INDEX idx_material_images_user_id ON material_images(user_id);
CREATE INDEX idx_material_images_upload_date ON material_images(upload_date DESC);
CREATE INDEX idx_material_images_net_votes ON material_images(net_votes DESC) WHERE is_active = true;
CREATE INDEX idx_material_images_flagged ON material_images(is_flagged) WHERE is_flagged = true;
```

#### 2. material_images_votes
Track user votes (upvotes/downvotes) on images.

```sql
CREATE TABLE material_images_votes (
    image_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (image_id, user_id),
    CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES material_images(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE
);

-- Index for user vote lookups
CREATE INDEX idx_material_images_votes_user ON material_images_votes(user_id);
```

#### 3. material_images_flags
Track user flags/reports on images with reasons.

```sql
CREATE TABLE material_images_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    reason VARCHAR(50) NOT NULL CHECK (reason IN ('incorrect_item', 'inappropriate', 'poor_quality', 'duplicate', 'other')),
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_image FOREIGN KEY (image_id) REFERENCES material_images(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE,
    CONSTRAINT unique_user_image_flag UNIQUE (image_id, user_id)
);

-- Indexes for flag management
CREATE INDEX idx_material_images_flags_image ON material_images_flags(image_id);
CREATE INDEX idx_material_images_flags_user ON material_images_flags(user_id);
```

#### 4. material_images_upload_limits
Track upload rate limiting per user per NIIN.

```sql
CREATE TABLE material_images_upload_limits (
    user_id TEXT NOT NULL,
    niin VARCHAR(9) NOT NULL,
    last_upload_time TIMESTAMP NOT NULL,
    upload_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, niin),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE
);

-- Index for cleanup operations
CREATE INDEX idx_upload_limits_time ON material_images_upload_limits(last_upload_time);
```

## Implementation Tasks

### Phase 1: Database Setup and Model Generation

**Actions:**
1. Create the four tables defined above
2. Add necessary indexes
3. Create rollback scripts

#### Task 1.2: Generate Jet Models
**Command to run:**
```bash
jet -dsn="postgresql://postgres:potato123@192.168.20.70:5432/miltech_ng?sslmode=disable" -schema=public -path=./.gen
```

**Generated files (automatic):**
- `.gen/miltech_ng/public/model/material_images.go`
- `.gen/miltech_ng/public/model/material_images_votes.go`
- `.gen/miltech_ng/public/model/material_images_flags.go`
- `.gen/miltech_ng/public/model/material_images_upload_limits.go`
- `.gen/miltech_ng/public/table/material_images.go`
- `.gen/miltech_ng/public/table/material_images_votes.go`
- `.gen/miltech_ng/public/table/material_images_flags.go`
- `.gen/miltech_ng/public/table/material_images_upload_limits.go`

### Phase 2: Repository Layer Implementation

#### Task 2.1: Create Repository Interface
**File to create:** `api/repository/material_images_repository.go`

```go
package repository

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
    "time"
)

type MaterialImagesRepository interface {
    // Image operations
    CreateImage(user *bootstrap.User, image model.MaterialImages) (*model.MaterialImages, error)
    GetImageByID(imageID string) (*model.MaterialImages, error)
    GetImagesByNIIN(niin string, limit int, offset int) ([]model.MaterialImages, int64, error)
    GetImagesByUser(userID string, limit int, offset int) ([]model.MaterialImages, int64, error)
    UpdateImageFlags(imageID string, flagCount int, isFlagged bool) error
    DeleteImage(imageID string) error
    
    // Vote operations
    UpsertVote(vote model.MaterialImagesVotes) error
    DeleteVote(imageID string, userID string) error
    GetUserVoteForImage(imageID string, userID string) (*model.MaterialImagesVotes, error)
    UpdateImageVoteCounts(imageID string) error
    
    // Flag operations
    CreateFlag(flag model.MaterialImagesFlags) error
    GetFlagsByImage(imageID string) ([]model.MaterialImagesFlags, error)
    
    // Rate limiting
    CheckUploadLimit(userID string, niin string) (bool, *time.Time, error)
    UpdateUploadLimit(userID string, niin string) error
    CleanupOldLimits(olderThan time.Time) error
}
```

#### Task 2.2: Create Repository Implementation
**File to create:** `api/repository/material_images_repository_impl.go`

Implement all interface methods using Jet ORM patterns similar to existing repositories.

### Phase 3: Service Layer Implementation

#### Task 3.1: Create Service Interface
**File to create:** `api/service/material_images_service.go`

```go
package service

import (
    "mime/multipart"
    "miltechserver/.gen/miltech_ng/public/model"
    "miltechserver/bootstrap"
)

type MaterialImagesService interface {
    // Image operations
    UploadImage(user *bootstrap.User, niin string, file multipart.File, header *multipart.FileHeader) (*model.MaterialImages, error)
    GetImagesByNIIN(niin string, page int, pageSize int) ([]model.MaterialImages, int64, error)
    GetImagesByUser(userID string, page int, pageSize int) ([]model.MaterialImages, int64, error)
    DeleteImage(user *bootstrap.User, imageID string) error
    
    // Vote operations
    VoteOnImage(user *bootstrap.User, imageID string, voteType string) error
    RemoveVote(user *bootstrap.User, imageID string) error
    
    // Flag operations
    FlagImage(user *bootstrap.User, imageID string, reason string, description string) error
    GetImageFlags(imageID string) ([]model.MaterialImagesFlags, error)
    
}
```

#### Task 3.2: Create Service Implementation
**File to create:** `api/service/material_images_service_impl.go`

Key implementation details:
1. Validate file types (accept only JPEG, PNG, WebP)
2. Limit file size (e.g., 10MB max)
3. Generate unique blob names using UUID + original extension
4. Handle Azure Blob upload with proper error handling
5. Implement rate limiting logic (24-hour window per NIIN per user)
6. Update vote counts atomically
7. Auto-flag images with high flag counts (e.g., >5 flags)

### Phase 4: Controller Layer Implementation

#### Task 4.1: Create Controller
**File to create:** `api/controller/material_images_controller.go`

```go
package controller

import (
    "miltechserver/api/service"
    "github.com/gin-gonic/gin"
)

type MaterialImagesController struct {
    MaterialImagesService service.MaterialImagesService
}

// Methods to implement:
// - UploadImage(c *gin.Context)
// - GetImagesByNIIN(c *gin.Context) 
// - GetImagesByUser(c *gin.Context)
// - DeleteImage(c *gin.Context)
// - VoteOnImage(c *gin.Context)
// - RemoveVote(c *gin.Context)
// - FlagImage(c *gin.Context)
```

### Phase 5: Request/Response DTOs

#### Task 5.1: Create Request DTOs
**File to create:** `api/request/material_images_request.go`

```go
package request

type UploadImageRequest struct {
    NIIN string `form:"niin" binding:"required,len=9"`
}

type VoteImageRequest struct {
    VoteType string `json:"vote_type" binding:"required,oneof=upvote downvote"`
}

type FlagImageRequest struct {
    Reason      string `json:"reason" binding:"required,oneof=incorrect_item inappropriate poor_quality duplicate copyright other"`
    Description string `json:"description" binding:"max=500"`
}

```

#### Task 5.2: Create Response DTOs
**File to create:** `api/response/material_images_response.go`

```go
package response

import (
    "miltechserver/.gen/miltech_ng/public/model"
    "time"
)

type MaterialImageResponse struct {
    ID               string    `json:"id"`
    NIIN            string    `json:"niin"`
    UserID          string    `json:"user_id"`
    Username        string    `json:"username"`
    BlobURL         string    `json:"blob_url"`
    OriginalFilename string    `json:"original_filename"`
    FileSizeBytes   int64     `json:"file_size_bytes"`
    MimeType        string    `json:"mime_type"`
    UploadDate      time.Time `json:"upload_date"`
    UpvoteCount     int       `json:"upvote_count"`
    DownvoteCount   int       `json:"downvote_count"`
    NetVotes        int       `json:"net_votes"`
    IsFlagged       bool      `json:"is_flagged"`
    UserVote        *string   `json:"user_vote,omitempty"`
    CanDelete       bool      `json:"can_delete"`
}

type PaginatedImagesResponse struct {
    Images      []MaterialImageResponse `json:"images"`
    TotalCount  int64                  `json:"total_count"`
    Page        int                    `json:"page"`
    PageSize    int                    `json:"page_size"`
    TotalPages  int                    `json:"total_pages"`
}
```

### Phase 6: Routes Configuration

#### Task 6.1: Create Routes
**File to create:** `api/route/material_images_route.go`

```go
package route

import (
    "database/sql"
    "miltechserver/api/controller"
    "miltechserver/api/repository"
    "miltechserver/api/service"
    "miltechserver/bootstrap"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/gin-gonic/gin"
)

func NewMaterialImagesRouter(db *sql.DB, blobClient *azblob.Client, env *bootstrap.Env, group *gin.RouterGroup) {
    repo := repository.NewMaterialImagesRepositoryImpl(db)
    svc := service.NewMaterialImagesServiceImpl(repo, blobClient, env)
    ctrl := controller.NewMaterialImagesController(svc)
    
    // Public routes (no auth required)
    group.GET("/material-images/niin/:niin", ctrl.GetImagesByNIIN)
    
    // Protected routes (auth required)
    protected := group.Group("")
    protected.Use(AuthenticationMiddleware()) // Use existing auth middleware
    {
        // Upload and management
        protected.POST("/material-images/upload", ctrl.UploadImage)
        protected.DELETE("/material-images/:image_id", ctrl.DeleteImage)
        protected.GET("/material-images/user/:user_id", ctrl.GetImagesByUser)
        
        // Voting
        protected.POST("/material-images/:image_id/vote", ctrl.VoteOnImage)
        protected.DELETE("/material-images/:image_id/vote", ctrl.RemoveVote)
        
        // Flagging
        protected.POST("/material-images/:image_id/flag", ctrl.FlagImage)
        
    }
}
```

#### Task 6.2: Update Main Router
**File to modify:** `api/route/route.go`

Add the new MaterialImagesRouter to the main route setup.

### Phase 7: Azure Blob Integration

#### Task 7.1: Create Blob Helper Service
**File to create:** `api/service/blob_storage_service.go`

```go
package service

import (
    "context"
    "fmt"
    "io"
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type BlobStorageService interface {
    UploadImage(containerName string, blobName string, data io.Reader, contentType string) (string, error)
    DeleteImage(containerName string, blobName string) error
    GetImageURL(containerName string, blobName string) string
}

type BlobStorageServiceImpl struct {
    client *azblob.Client
}

// Implementation details:
// - Use container "material-images"
// - Generate URLs with SAS tokens if needed
// - Handle retries and errors appropriately
```

### Phase 8: Validation and Security

#### Task 8.1: Input Validation
1. **File validation:**
   - Max size: 10MB
   - Allowed types: image/jpeg, image/png, image/webp
   - Validate actual file content, not just extension

2. **NIIN validation:**
   - Must be exactly 9 characters
   - Should exist in the NSN table (optional check)

3. **Rate limiting:**
   - 1 upload per NIIN per user per 24 hours
   - Consider global rate limits per user

#### Task 8.2: Security Measures
1. **Authentication:** Verify user is logged in for all protected endpoints
2. **Authorization:** Only image uploader can delete their image
3. **Sanitization:** Clean all user inputs
4. **Blob security:** Use SAS tokens with expiration for blob URLs
5. **CORS:** Configure appropriate CORS settings

### Phase 9: Testing Implementation

#### Task 9.1: Unit Tests
**Files to create:**
- `api/repository/material_images_repository_test.go`
- `api/service/material_images_service_test.go`
- `api/controller/material_images_controller_test.go`

#### Task 9.2: Integration Tests
**File to create:** `tests/integration/material_images_test.go`

Test scenarios:
1. Upload image successfully
2. Rate limiting enforcement
3. Vote counting accuracy
4. Delete authorization
5. Pagination

### Phase 10: Background Jobs (Optional Enhancement)


## API Endpoints Summary

### Public Endpoints
- `GET /api/material-images/niin/:niin` - Get images for a NIIN

### Authenticated Endpoints
- `POST /api/material-images/upload` - Upload image for NIIN
- `DELETE /api/material-images/:image_id` - Delete own image
- `GET /api/material-images/user/:user_id` - Get user's uploads
- `POST /api/material-images/:image_id/vote` - Vote on image
- `DELETE /api/material-images/:image_id/vote` - Remove vote
- `POST /api/material-images/:image_id/flag` - Flag image


## Configuration Requirements

### Environment Variables
Add to `.env`:
```
BLOB_CONTAINER_MATERIAL_IMAGES=material-images
MAX_IMAGE_SIZE_MB=10
IMAGE_UPLOAD_RATE_LIMIT_HOURS=24
AUTO_FLAG_THRESHOLD=5
```

### Azure Blob Container Setup
1. Create container named "material-images" in Azure Storage Account
2. Set container access level to "Private"
3. Configure CORS if needed for direct browser uploads
4. Set up lifecycle management for old images (optional)

## Migration Strategy

### Step 1: Database Migration
1. Review and test migration scripts in development
2. Backup production database
3. Run migrations during maintenance window
4. Verify table creation and indexes

### Step 2: Code Deployment
1. Deploy backend with feature flag disabled
2. Test with limited users
3. Enable feature flag gradually
4. Monitor for issues

### Step 3: Monitoring
1. Set up alerts for high flag counts
2. Monitor blob storage usage
3. Track upload patterns
4. Review performance metrics

## Performance Considerations

1. **Database:**
   - Use pagination for all list queries
   - Index on niin, user_id, and timestamps
   - Consider partitioning if data grows large

2. **Blob Storage:**
   - Use CDN for serving images
   - Implement client-side caching
   - Consider image compression

3. **Rate Limiting:**
   - Use Redis for distributed rate limiting if scaling horizontally
   - Implement cleanup job for old limit records

## Security Considerations

1. **Input Validation:**
   - Validate file headers, not just extensions
   - Scan for malware if possible
   - Limit file sizes strictly

2. **Access Control:**
   - Verify user ownership before delete
   - Use secure blob URLs with SAS tokens
   - Implement proper CORS policies

3. **Data Protection:**
   - Don't expose internal IDs in URLs
   - Log all delete operations
   - Implement soft deletes initially

## Future Enhancements

1. **Image Processing:**
   - Auto-generate thumbnails
   - Image optimization
   - OCR for text extraction

2. **AI Features:**
   - Auto-categorization
   - Duplicate detection
   - Quality scoring

3. **Social Features:**
   - Comments on images
   - Image collections
   - User reputation system

4. **Admin Features:**
   - Bulk operations
   - Moderation queue
   - Analytics dashboard

## Success Metrics

1. **Usage Metrics:**
   - Number of images uploaded per day
   - Active uploaders per week
   - Images per NIIN distribution

2. **Quality Metrics:**
   - Flag rate percentage
   - Resolution time for flags
   - User satisfaction scores

3. **Performance Metrics:**
   - Upload success rate
   - Average upload time
   - Image load times

## Rollback Plan

If issues arise:
1. Disable feature flag immediately
2. Stop accepting new uploads
3. Keep existing images accessible (read-only)
4. Fix issues and re-deploy
5. If critical, rollback database migrations

## Documentation Updates Required

1. **API Documentation:** Add new endpoints to API docs
2. **User Guide:** Create upload guidelines
3. **Admin Guide:** Document moderation process
4. **Developer Guide:** Document architecture and patterns

## Estimated Timeline

- **Phase 1-2 (Database & Repository):** 2 days
- **Phase 3-4 (Service & Controller):** 3 days  
- **Phase 5-6 (DTOs & Routes):** 1 day
- **Phase 7-8 (Azure & Security):** 2 days
- **Phase 9 (Testing):** 2 days
- **Total:** ~10 days of development

## Notes

- Ensure backward compatibility with existing systems
- Consider feature flags for gradual rollout
- Plan for data migration if replacing existing system
- Review with security team before production deployment