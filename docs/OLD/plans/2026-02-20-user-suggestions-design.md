# User Suggestions Feature — Design Document

**Date:** 2026-02-20
**Status:** Approved
**Branch:** shop_refactor

---

## Overview

Add a user-facing feature suggestion system that allows authenticated users to submit, edit, delete, and vote on feature requests for the mobile application. Anyone (including unauthenticated users) can view all suggestions. Each suggestion carries a net vote score computed from individual upvotes (+1) and downvotes (-1). Authenticated users see their own vote status alongside each suggestion.

---

## Goals

- Allow authenticated users to submit feature suggestions with a title and description
- Allow anyone to list all suggestions (no pagination) with net vote scores
- Allow authenticated users to upvote or downvote suggestions (one vote per user per suggestion, toggleable)
- Include the current user's vote direction in the listing response when authenticated
- Allow users to edit or delete their own suggestions
- Track submission status with a default of "Submitted"

---

## Architecture

### Placement

The feature lives as a standalone domain package `api/user_suggestions/`, following the same flat-package pattern as `api/item_comments/`.

```
api/user_suggestions/
├── errors.go            ← sentinel errors
├── types.go             ← request/response types
├── repository.go        ← Repository interface
├── repository_impl.go   ← Jet/raw SQL queries
├── service.go           ← Service interface
├── service_impl.go      ← business logic, validation
├── service_impl_test.go ← unit tests with mock repo
├── route.go             ← Handler struct, RegisterRoutes, HTTP handlers
└── route_test.go        ← HTTP handler tests with service stub
```

### Dependency flow

```
route/route.go
  → user_suggestions.RegisterRoutes(deps, v1Route, authRoutes)
      → user_suggestions.NewRepository(db)
      → user_suggestions.NewService(repo)
      → registerHandlers(publicGroup, authGroup, authClient, svc)
```

The package receives `*sql.DB` and `*auth.Client`. The auth client is needed for the optional authentication middleware on the public GET endpoint.

---

## Database Schema

### `user_suggestions` table

```sql
CREATE TABLE user_suggestions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    title       TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 2000),
    status      TEXT NOT NULL DEFAULT 'Submitted',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ
);

CREATE INDEX idx_user_suggestions_user_id ON user_suggestions (user_id);
CREATE INDEX idx_user_suggestions_created_at ON user_suggestions (created_at);
```

### `user_suggestion_votes` table

```sql
CREATE TABLE user_suggestion_votes (
    suggestion_id UUID NOT NULL REFERENCES user_suggestions(id) ON DELETE CASCADE,
    voter_id      TEXT NOT NULL,
    direction     SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suggestion_id, voter_id)
);
```

**Design notes:**
- `SMALLINT` for direction allows direct `SUM(direction)` for net score computation — no CASE expression needed.
- `ON DELETE CASCADE` on the FK ensures votes are cleaned up when a suggestion is deleted.
- The composite PK `(suggestion_id, voter_id)` enforces one-vote-per-user at the database level and serves as the lookup index. No separate FK index needed since `suggestion_id` is the leading column.
- `user_id` / `voter_id` are Firebase UIDs stored as `TEXT`, consistent with the existing `users.uid` column and `item_comments.author_id`.

---

## API Routes

| Method   | Route                          | Auth     | Description                              |
|----------|--------------------------------|----------|------------------------------------------|
| `GET`    | `/suggestions`                 | Optional | List all suggestions with scores         |
| `POST`   | `/auth/suggestions`            | Required | Submit a new suggestion                  |
| `PUT`    | `/auth/suggestions/:id`        | Required | Edit own suggestion (title/description)  |
| `DELETE` | `/auth/suggestions/:id`        | Required | Delete own suggestion                    |
| `POST`   | `/auth/suggestions/:id/vote`   | Required | Cast or change vote                      |
| `DELETE` | `/auth/suggestions/:id/vote`   | Required | Remove vote                              |

### Optional Authentication Middleware

The `GET /suggestions` route uses a new `OptionalAuthMiddleware` applied at the route level (not group level). This middleware:
1. Checks for an `Authorization: Bearer <token>` header
2. If present and valid: sets `*bootstrap.User` in gin context, calls `c.Next()`
3. If missing or invalid: calls `c.Next()` without setting user (no abort)

This allows the handler to check for user context and conditionally include `my_vote` in the response.

---

## Request/Response Types (`types.go`)

### Requests

```go
type CreateSuggestionRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

type UpdateSuggestionRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

type VoteRequest struct {
    Direction int16 `json:"direction"` // 1 or -1
}
```

### Responses

```go
type SuggestionResponse struct {
    ID          string  `json:"id"`
    UserID      string  `json:"user_id"`
    Username    string  `json:"username"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Status      string  `json:"status"`
    Score       int     `json:"score"`
    MyVote      *int16  `json:"my_vote,omitempty"`
    CreatedAt   string  `json:"created_at"`
    UpdatedAt   *string `json:"updated_at,omitempty"`
}
```

**`MyVote` tri-state behavior:**
- Unauthenticated user: field omitted from JSON (`nil` + `omitempty`)
- Authenticated, no vote: `"my_vote": null`
- Authenticated, voted up: `"my_vote": 1`
- Authenticated, voted down: `"my_vote": -1`

---

## Service Interface (`service.go`)

```go
type Service interface {
    GetAllSuggestions(currentUser *bootstrap.User) ([]SuggestionResponse, error)
    CreateSuggestion(user *bootstrap.User, title, description string) (*SuggestionResponse, error)
    UpdateSuggestion(user *bootstrap.User, suggestionID, title, description string) (*SuggestionResponse, error)
    DeleteSuggestion(user *bootstrap.User, suggestionID string) error
    Vote(user *bootstrap.User, suggestionID string, direction int16) error
    RemoveVote(user *bootstrap.User, suggestionID string) error
}
```

---

## Repository Interface (`repository.go`)

```go
type Repository interface {
    GetAllWithScores(voterID string) ([]SuggestionWithScore, error)
    GetByID(id uuid.UUID) (*model.UserSuggestions, error)
    Create(suggestion model.UserSuggestions) (*model.UserSuggestions, error)
    Update(id uuid.UUID, title, description string) (*model.UserSuggestions, error)
    Delete(id uuid.UUID) error
    UpsertVote(suggestionID uuid.UUID, voterID string, direction int16) error
    DeleteVote(suggestionID uuid.UUID, voterID string) error
}
```

### Internal types

```go
type SuggestionWithScore struct {
    model.UserSuggestions
    Username *string
    Score    int
    MyVote   *int16
}
```

---

## Core Query — GetAllWithScores

```sql
SELECT
    s.id, s.user_id, s.title, s.description, s.status,
    s.created_at, s.updated_at,
    u.username,
    COALESCE(SUM(v.direction), 0)::INT AS score,
    uv.direction AS my_vote
FROM user_suggestions s
LEFT JOIN users u ON s.user_id = u.uid
LEFT JOIN user_suggestion_votes v ON s.id = v.suggestion_id
LEFT JOIN user_suggestion_votes uv ON s.id = uv.suggestion_id AND uv.voter_id = $1
GROUP BY s.id, u.username, uv.direction
ORDER BY s.created_at DESC
```

When unauthenticated, `$1` is set to empty string `""` (matches no voter_id), so `my_vote` is `NULL` for all rows.

**Query design notes:**
- Two separate LEFT JOINs on `user_suggestion_votes`: one (`v`) aggregates all votes for score, the other (`uv`) fetches the current user's vote. This avoids a correlated subquery.
- `COALESCE(SUM(...), 0)` ensures suggestions with zero votes return `0` instead of `NULL`.

---

## Vote Upsert Query

```sql
INSERT INTO user_suggestion_votes (suggestion_id, voter_id, direction)
VALUES ($1, $2, $3)
ON CONFLICT (suggestion_id, voter_id)
DO UPDATE SET direction = EXCLUDED.direction, created_at = now()
```

This handles both new votes and vote changes in a single statement. The composite PK serves as the conflict target.

---

## Sentinel Errors (`errors.go`)

```go
var (
    ErrUnauthorized       = errors.New("unauthorized user")
    ErrForbidden          = errors.New("not authorized to modify this suggestion")
    ErrSuggestionNotFound = errors.New("suggestion not found")
    ErrInvalidTitle       = errors.New("title must be between 1 and 200 characters")
    ErrInvalidDescription = errors.New("description must be between 1 and 2000 characters")
    ErrInvalidDirection   = errors.New("vote direction must be 1 or -1")
    ErrInvalidID          = errors.New("invalid suggestion ID")
)
```

---

## HTTP Error Mapping (`route.go`)

| Error Sentinel          | HTTP Status |
|-------------------------|-------------|
| `ErrSuggestionNotFound` | 404         |
| `ErrInvalidTitle`, `ErrInvalidDescription`, `ErrInvalidDirection`, `ErrInvalidID` | 400 |
| `ErrUnauthorized`       | 401         |
| `ErrForbidden`          | 403         |
| All others              | 500         |

Error mapping uses the same `errorCase` pattern from `item_comments`.

---

## Handler Logic

### `GET /suggestions` — listSuggestions

1. Attempt to get `*bootstrap.User` from context (set by optional auth middleware)
2. Call `service.GetAllSuggestions(user)` — `user` may be `nil`
3. Return `200 OK` with `StandardResponse{Data: suggestions}`

### `POST /auth/suggestions` — createSuggestion

1. Get user from context (required)
2. Bind JSON to `CreateSuggestionRequest`
3. Call `service.CreateSuggestion(user, req.Title, req.Description)`
4. Return `201 Created` with the created suggestion

### `PUT /auth/suggestions/:id` — updateSuggestion

1. Get user from context (required)
2. Parse `id` param
3. Bind JSON to `UpdateSuggestionRequest`
4. Call `service.UpdateSuggestion(user, id, req.Title, req.Description)`
5. Service verifies ownership before updating
6. Return `200 OK` with the updated suggestion

### `DELETE /auth/suggestions/:id` — deleteSuggestion

1. Get user from context (required)
2. Parse `id` param
3. Call `service.DeleteSuggestion(user, id)`
4. Service verifies ownership before deleting
5. Return `200 OK`

### `POST /auth/suggestions/:id/vote` — vote

1. Get user from context (required)
2. Parse `id` param
3. Bind JSON to `VoteRequest`
4. Call `service.Vote(user, id, req.Direction)`
5. Service validates direction is 1 or -1, verifies suggestion exists
6. Return `200 OK`

### `DELETE /auth/suggestions/:id/vote` — removeVote

1. Get user from context (required)
2. Parse `id` param
3. Call `service.RemoveVote(user, id)`
4. Return `200 OK`

---

## Service Validation Logic

### CreateSuggestion
- User must not be nil → `ErrUnauthorized`
- Title: 1–200 characters → `ErrInvalidTitle`
- Description: 1–2000 characters → `ErrInvalidDescription`

### UpdateSuggestion
- User must not be nil → `ErrUnauthorized`
- Parse suggestion ID → `ErrInvalidID`
- Fetch suggestion by ID → `ErrSuggestionNotFound`
- Verify `suggestion.user_id == user.UserID` → `ErrForbidden`
- Validate title and description (same as create)

### DeleteSuggestion
- User must not be nil → `ErrUnauthorized`
- Parse suggestion ID → `ErrInvalidID`
- Fetch suggestion by ID → `ErrSuggestionNotFound`
- Verify ownership → `ErrForbidden`

### Vote
- User must not be nil → `ErrUnauthorized`
- Parse suggestion ID → `ErrInvalidID`
- Direction must be 1 or -1 → `ErrInvalidDirection`
- Verify suggestion exists → `ErrSuggestionNotFound`
- Upsert vote

### RemoveVote
- User must not be nil → `ErrUnauthorized`
- Parse suggestion ID → `ErrInvalidID`
- Delete vote (idempotent — no error if vote doesn't exist)

---

## Tests

### `service_impl_test.go` — unit tests with mock repository

| Test | Description |
|------|-------------|
| `TestGetAllSuggestions_Unauthenticated` | Returns suggestions with scores, no my_vote |
| `TestGetAllSuggestions_Authenticated` | Returns suggestions with scores and my_vote |
| `TestCreateSuggestion_Success` | Valid input creates suggestion |
| `TestCreateSuggestion_Unauthorized` | Nil user returns ErrUnauthorized |
| `TestCreateSuggestion_InvalidTitle` | Empty/too-long title returns ErrInvalidTitle |
| `TestCreateSuggestion_InvalidDescription` | Empty/too-long description returns ErrInvalidDescription |
| `TestUpdateSuggestion_Success` | Owner can update title/description |
| `TestUpdateSuggestion_NotOwner` | Non-owner gets ErrForbidden |
| `TestUpdateSuggestion_NotFound` | Missing suggestion returns ErrSuggestionNotFound |
| `TestDeleteSuggestion_Success` | Owner can delete |
| `TestDeleteSuggestion_NotOwner` | Non-owner gets ErrForbidden |
| `TestVote_Upvote` | Cast +1 vote |
| `TestVote_Downvote` | Cast -1 vote |
| `TestVote_InvalidDirection` | Direction != 1 or -1 returns ErrInvalidDirection |
| `TestVote_SuggestionNotFound` | Non-existent suggestion returns error |
| `TestRemoveVote_Success` | Removes existing vote |
| `TestRemoveVote_Idempotent` | No error when no vote exists |

### `route_test.go` — HTTP handler tests with service stub

| Test | Description |
|------|-------------|
| `TestListSuggestions_Public` | 200 with suggestions, no my_vote |
| `TestListSuggestions_Authenticated` | 200 with my_vote included |
| `TestCreateSuggestion_201` | 201 with valid body |
| `TestCreateSuggestion_400_BadBody` | 400 for invalid JSON |
| `TestCreateSuggestion_400_Validation` | 400 for ErrInvalidTitle |
| `TestUpdateSuggestion_200` | 200 on successful update |
| `TestUpdateSuggestion_403` | 403 when not owner |
| `TestUpdateSuggestion_404` | 404 when not found |
| `TestDeleteSuggestion_200` | 200 on successful delete |
| `TestDeleteSuggestion_403` | 403 when not owner |
| `TestVote_200` | 200 on successful vote |
| `TestVote_400_InvalidDirection` | 400 for bad direction |
| `TestRemoveVote_200` | 200 on successful removal |

---

## Files Changed / Created

| File | Action |
|------|--------|
| `api/route/route.go` | Modified — add `user_suggestions.RegisterRoutes(...)` call |
| `api/middleware/optional_auth.go` | Created — optional authentication middleware |
| `api/user_suggestions/errors.go` | Created |
| `api/user_suggestions/types.go` | Created |
| `api/user_suggestions/repository.go` | Created |
| `api/user_suggestions/repository_impl.go` | Created |
| `api/user_suggestions/service.go` | Created |
| `api/user_suggestions/service_impl.go` | Created |
| `api/user_suggestions/service_impl_test.go` | Created |
| `api/user_suggestions/route.go` | Created |
| `api/user_suggestions/route_test.go` | Created |
| SQL migration | New migration file for both tables |

---

## Out of Scope

- Pagination (all suggestions returned in a single response)
- Admin endpoints for managing status (can be added later)
- Notification system for status changes
- Rate limiting on suggestion submission (global rate limiter already exists)
- Search/filter on suggestions
- Sorting by score (returns newest first; client can re-sort)
