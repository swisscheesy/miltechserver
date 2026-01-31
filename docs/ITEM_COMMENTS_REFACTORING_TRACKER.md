# Item Comments Domain Refactor Tracker

**Created:** 2026-01-30
**Owner:** swisscheese
**Status:** Complete (legacy cleanup done)

## Executive Summary

The `item_comments` domain handles user comments on military items (NIIN-based). It provides CRUD operations for comments with reply threading support and a comment flagging system for moderation. This is a well-structured domain with proper typed errors already in place, making it a good candidate for colocation without major structural changes.

### Current Architecture Assessment

**Strengths:**
1. **Typed Error Handling**: Already uses `errors.Is()` pattern with typed errors (ErrInvalidNiin, ErrInvalidText, etc.) - considered best practice
2. **Clean Separation**: Service layer handles business logic, repository handles data access
3. **Proper Validation**: NIIN validation, comment length validation, parent comment validation
4. **Good Security**: Checks user ownership before update/delete operations
5. **Threaded Comments**: Supports parent-child comment relationships

**Weaknesses:**
1. **Scattered Files**: Code spread across 6 directories (`controller`, `service`, `repository`, `route`, `response`, `request`)
2. **Inconsistent Pattern**: Uses legacy `New*Router()` pattern instead of `domain.RegisterRoutes()`
3. **Mixed Route Groups**: Requires both public (`v1Route`) and authenticated (`authRoutes`) groups
4. **Raw SQL**: GetCommentsByNiin uses raw SQL instead of Jet ORM (for JOIN)

### Refactoring Priority Assessment

| Criteria | Score | Notes |
|----------|-------|-------|
| Code Size | Medium | 721 LOC total |
| Duplication | Low | Minimal duplication present |
| Pattern Violation | Low | Already uses typed errors correctly |
| Complexity | Medium | Auth logic, threading, flagging |
| Risk | Low | Well-tested patterns in controller |
| Effort | Low | Straightforward migration |

**Recommendation**: Priority 3 - This domain is already well-organized. Refactoring provides colocation benefits but limited code quality improvements. Consider refactoring for consistency with other domains.

## Current State Analysis

### File Inventory (Old Pattern)

| File | Location | LOC | Purpose |
|------|----------|-----|---------|
| item_comments_controller.go | api/controller/ | 193 | HTTP handlers for all comment operations |
| item_comments_service.go | api/service/ | 15 | Service interface (5 methods) |
| item_comments_service_impl.go | api/service/ | 244 | Service implementation with validation |
| item_comments_repository.go | api/repository/ | 21 | Repository interface + custom type |
| item_comments_repository_impl.go | api/repository/ | 204 | Database queries |
| item_comments_route.go | api/route/ | 27 | Route registration |
| item_comments_response.go | api/response/ | 14 | Response type definition |
| item_comments_request.go | api/request/ | 11 | Request type definitions |
| **Total** | | **729** | |

### Current API Endpoints

```
# Public
GET /api/v1/items/:niin/comments              # Get all comments for NIIN

# Authenticated
POST /api/v1/auth/items/:niin/comments        # Create comment
PUT /api/v1/auth/items/:niin/comments/:comment_id    # Update comment
DELETE /api/v1/auth/items/:niin/comments/:comment_id # Soft delete (set text to "Deleted by user")
POST /api/v1/auth/items/:niin/comments/:comment_id/flags # Flag comment for moderation
```

### Database Dependencies

| Context | Table | Description |
|---------|-------|-------------|
| comments | item_comments | Comment storage with author, text, threading |
| comments | item_comment_flags | Flag tracking for moderation |
| comments | users | JOIN for author display names |

### Error Handling Analysis

**Typed Errors Already Defined** (in service layer):
```go
var (
    ErrInvalidNiin     = errors.New("invalid NIIN")
    ErrInvalidText     = errors.New("invalid comment text")
    ErrCommentNotFound = errors.New("comment not found")
    ErrUnauthorized    = errors.New("unauthorized user")
    ErrForbidden       = errors.New("user not authorized")
    ErrInvalidParent   = errors.New("invalid parent comment")
)
```

**Controller Error Handling** (already using `errors.Is()`):
```go
switch {
case errors.Is(err, service.ErrInvalidNiin):
    c.JSON(http.StatusBadRequest, gin.H{"message": "invalid NIIN"})
case errors.Is(err, service.ErrCommentNotFound):
    c.JSON(http.StatusNotFound, gin.H{"message": "comment not found"})
// ... etc
}
```

### Interface Analysis

**Current Repository Interface** (5 methods):
```go
type ItemCommentsRepository interface {
    GetCommentsByNiin(niin string) ([]ItemCommentWithAuthor, error)
    GetCommentByID(commentID uuid.UUID) (*model.ItemComments, error)
    CreateComment(comment model.ItemComments) (*model.ItemComments, error)
    UpdateCommentText(commentID uuid.UUID, text string) (*model.ItemComments, error)
    FlagComment(flag model.ItemCommentFlags) error
}
```

**Current Service Interface** (5 methods):
```go
type ItemCommentsService interface {
    GetCommentsByNiin(niin string) ([]response.ItemCommentResponse, error)
    CreateComment(user *bootstrap.User, niin string, text string, parentID *string) (*response.ItemCommentResponse, error)
    UpdateComment(user *bootstrap.User, niin string, commentID string, text string) (*response.ItemCommentResponse, error)
    DeleteComment(user *bootstrap.User, niin string, commentID string) (*response.ItemCommentResponse, error)
    FlagComment(user *bootstrap.User, niin string, commentID string) error
}
```

## Proposed New Structure

### Option A: Simple Bounded Context (Recommended)

All operations relate to item comments. A simple colocated structure is appropriate:

```
api/
  item_comments/
    route.go           # HTTP handlers and route registration
    service.go         # Service interface
    service_impl.go    # Service implementation with validation
    repository.go      # Repository interface + ItemCommentWithAuthor type
    repository_impl.go # Database queries
    errors.go          # Typed error definitions (moved from service)
    types.go           # Request/response types (consolidated)
```

**Key Changes:**
- Errors moved to dedicated `errors.go` (from service file)
- Request/response types consolidated into `types.go`
- Controller logic moved into `route.go`

## Implementation Checklist

### Phase 1: Foundation

- [x] 1.1 Create directory structure
  ```bash
  mkdir -p api/item_comments
  ```

- [x] 1.2 Create errors.go
  ```go
  package item_comments

  import "errors"

  var (
      ErrInvalidNiin     = errors.New("invalid NIIN")
      ErrInvalidText     = errors.New("invalid comment text")
      ErrCommentNotFound = errors.New("comment not found")
      ErrUnauthorized    = errors.New("unauthorized user")
      ErrForbidden       = errors.New("user not authorized")
      ErrInvalidParent   = errors.New("invalid parent comment")
  )
  ```

- [x] 1.3 Create types.go
  ```go
  package item_comments

  import (
      "time"
      "miltechserver/.gen/miltech_ng/public/model"
  )

  // Request types
  type CreateRequest struct {
      Text     string  `json:"text"`
      ParentID *string `json:"parent_id"`
  }

  type UpdateRequest struct {
      Text string `json:"text"`
  }

  // Response types
  type CommentResponse struct {
      ID                string    `json:"id"`
      CommentNiin       string    `json:"comment_niin"`
      AuthorID          string    `json:"author_id"`
      AuthorDisplayName string    `json:"author_display_name"`
      Text              string    `json:"text"`
      ParentID          *string   `json:"parent_id"`
      CreatedAt         time.Time `json:"created_at"`
  }

  // Internal types
  type CommentWithAuthor struct {
      model.ItemComments
      AuthorDisplayName *string
  }
  ```

### Phase 2: Core Implementation

- [x] 2.1 Create repository.go (interface)
- [x] 2.2 Create repository_impl.go (migrate existing logic)
- [x] 2.3 Create service.go (interface)
- [x] 2.4 Create service_impl.go (migrate existing logic, update error imports)
- [x] 2.5 Create route.go (handlers + registration)

### Phase 3: Wiring

- [x] 3.1 Create Dependencies struct and RegisterRoutes function
  ```go
  package item_comments

  import (
      "database/sql"
      "github.com/gin-gonic/gin"
  )

  type Dependencies struct {
      DB *sql.DB
  }

  func RegisterRoutes(deps Dependencies, publicGroup, authGroup *gin.RouterGroup) {
      repo := NewRepository(deps.DB)
      svc := NewService(repo)
      registerHandlers(svc, publicGroup, authGroup)
  }
  ```

- [x] 3.2 Update main route registration in api/route/route.go
  ```go
  // Replace:
  NewItemCommentsRouter(db, v1Route, authRoutes)

  // With:
  item_comments.RegisterRoutes(item_comments.Dependencies{DB: db}, v1Route, authRoutes)
  ```

### Phase 4: Verification & Cleanup

- [ ] 4.1 Manual API testing
- [x] 4.1a Add unit + integration tests
  - Test GET /api/v1/items/:niin/comments
  - Test POST /api/v1/auth/items/:niin/comments
  - Test PUT /api/v1/auth/items/:niin/comments/:comment_id
  - Test DELETE /api/v1/auth/items/:niin/comments/:comment_id
  - Test POST /api/v1/auth/items/:niin/comments/:comment_id/flags

- [x] 4.2 Remove legacy files:
  - api/controller/item_comments_controller.go
  - api/service/item_comments_service.go
  - api/service/item_comments_service_impl.go
  - api/repository/item_comments_repository.go
  - api/repository/item_comments_repository_impl.go
  - api/route/item_comments_route.go
  - api/response/item_comments_response.go
  - api/request/item_comments_request.go

## Expected Metrics Improvement

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Total LOC | 729 | 730 | -0.1% (flat) |
| Files | 8 (scattered) | 7 (colocated) | Better organization |
| Directories | 6 | 1 | 83% fewer directories |
| Error handling | Already typed | Typed (unchanged) | Good baseline |

## Risk Assessment

### Low Risk
- Already well-structured code
- Typed errors already in place
- Clear separation of concerns
- Well-defined API contract

### Medium Risk
- Mixed route groups (public + auth) need careful handling
- Raw SQL for GetCommentsByNiin needs to be preserved

### Mitigation
- Keep dual route group pattern from existing implementation
- Preserve raw SQL query for author JOIN (Jet ORM limitation)
- Comprehensive endpoint testing before removing legacy code

## Comparison with Other Refactored Domains

| Domain | Bounded Contexts | Complexity | Pattern | Auth |
|--------|------------------|------------|---------|------|
| shops | 7 | High | Full decomposition | Required |
| equipment_services | 5 | Medium | Full decomposition | Required |
| user_saves | 5 | Medium | Feature-based | Required |
| item_lookup | 4 | Low | Data-domain | None |
| eic | 1 | Low | Colocated | None |
| library | 1 | Low | Simple colocated | Partial |
| **item_comments** | 1 | Low-Medium | Simple colocated | Partial |

Key difference: `item_comments` has mixed public/authenticated routes and already follows best practices for error handling.

## Progress Log

- 2026-01-30: Initial analysis and planning complete
- 2026-01-31: Colocated item_comments module implemented, routes rewired, unit + integration tests added and passing
- 2026-01-31: Legacy item_comments files removed after validation

## Notes

- Domain is already well-organized - refactoring is primarily for consistency
- Typed errors are a model pattern that other domains should follow
- Consider adding integration tests during refactor
- The raw SQL for GetCommentsByNiin could potentially be converted to Jet with LEFT_JOIN
- DeleteComment performs soft delete (sets text to "Deleted by user")
- Comment flagging uses ON_CONFLICT DO_NOTHING (user can only flag once)
