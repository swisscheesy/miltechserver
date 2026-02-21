# Item Comments Design Document

**Date**: 2026-01-24  
**Author**: System Design  
**Status**: Design Phase  
**Feature**: Item comments by NIIN (public read, authenticated write)

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Current Implementation Analysis](#current-implementation-analysis)
3. [Requirements](#requirements)
4. [Proposed Solution](#proposed-solution)
5. [Data Model and Migration](#data-model-and-migration)
6. [API Contract](#api-contract)
7. [Application Flow](#application-flow)
8. [Implementation Plan](#implementation-plan)
9. [Risks and Tradeoffs](#risks-and-tradeoffs)
10. [Decisions](#decisions)

---

## Executive Summary

Add item comments tied to a NIIN. Any user can read comments for an item, while only authenticated users can post, edit, or delete their own comments. The existing comments table is no longer present, so this design introduces a new comments table (plus a flags table) and implements a dedicated feature stack to keep the comments functionality isolated and easier to maintain.

---

## Current Implementation Analysis

### Project Architecture
- Routes follow `route -> controller -> service -> repository`.
- Public routes are mounted at `/api/v1`.
- Authenticated routes are mounted at `/api/v1/auth` and use Firebase Auth middleware (`api/middleware/authentication.go`).
- Mixed route pattern exists (public read + authenticated write) in `api/route/material_images_route.go` and `api/route/library_route.go`.
- Database access uses Jet with generated types under `.gen/miltech_ng/public`.

### Existing Database Objects
- No existing item comments table (previous `user_item_comments` table has been deleted).
- Jet models for comments are stale and must be regenerated after new migrations.

### Gaps
No routes/controllers/services/repositories use `user_item_comments`. The table is empty and there are no API endpoints for item comments.

---

## Requirements

### Functional
1. Anyone can read comments for a given NIIN.
2. Only authenticated users can submit a comment.
3. Comments are stored by NIIN.
4. Each comment stores author ID and creation time.
5. Comments support replies (threaded via parent ID).
6. Only the author can edit or delete their own comment.
7. Comment text is required and max length is 255.
8. Comments are only allowed for valid NIINs.
9. Users can flag comments.

### Non-Functional
- Reads should be fast and indexed by NIIN.
- Writes must validate authentication and inputs.
- Responses follow existing `StandardResponse` patterns where practical.
- Order is oldest-first.
- No pagination is required.

---

## Proposed Solution

### Overview
Implement item comments with a new comments table and a comment flags table. Add mixed routes that live with the existing item query feature:
- Public GET to read by NIIN.
- Authenticated POST to create.
- Authenticated PUT to edit (author-only).
- Authenticated DELETE to remove (author-only).
- Authenticated POST to flag a comment.

### Architecture
```
Route -> Controller -> Service -> Repository -> DB (Jet)
```

### Code Placement (updated decision)
Implement a dedicated feature stack for item comments:
- Route: `api/route/item_comments_route.go`
- Controller: `api/controller/item_comments_controller.go`
- Service: `api/service/item_comments_service.go` + `api/service/item_comments_service_impl.go`
- Repository: `api/repository/item_comments_repository.go` + `api/repository/item_comments_repository_impl.go`
- Requests: `api/request/item_comment_create_request.go`, `api/request/item_comment_update_request.go`, `api/request/item_comment_flag_request.go` (if needed)
- Responses: `api/response/item_comments_response.go` (if needed)

### Routing
Public:
- `GET /api/v1/items/:niin/comments`

Authenticated:
- `POST /api/v1/auth/items/:niin/comments`
- `PUT /api/v1/auth/items/:niin/comments/:comment_id`
- `DELETE /api/v1/auth/items/:niin/comments/:comment_id`
- `POST /api/v1/auth/items/:niin/comments/:comment_id/flags`

---

## Data Model and Migration

### SQL (Tables and Indexes)
```sql
CREATE TABLE item_comments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_niin text NOT NULL REFERENCES nsn(niin),
    author_id text NOT NULL REFERENCES users(uid),
    text varchar(255) NOT NULL,
    parent_id uuid NULL REFERENCES item_comments(id),
    created_at timestamp NOT NULL DEFAULT now(),
    updated_at timestamp NULL,
);

CREATE TABLE item_comment_flags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id uuid NOT NULL REFERENCES item_comments(id),
    flagger_id text NOT NULL REFERENCES users(uid),
    created_at timestamp NOT NULL DEFAULT now(),
    CONSTRAINT item_comment_flags__uq_comment_flagger UNIQUE (comment_id, flagger_id)
);

CREATE INDEX item_comments__idx_niin_created
    ON item_comments (comment_niin, created_at ASC);

CREATE INDEX item_comments__idx_parent
    ON item_comments (parent_id);

CREATE INDEX item_comments__idx_author
    ON item_comments (author_id);

CREATE INDEX item_comment_flags__idx_comment
    ON item_comment_flags (comment_id);

CREATE INDEX item_comment_flags__idx_flagger
    ON item_comment_flags (flagger_id);
```

### Notes
- Use UUIDs to match existing usage in other tables.
- Hard deletes are used; no `deleted_at` column.
- UUIDs are generated by the database via `gen_random_uuid()`.
- Add FK indexes for faster joins and deletes.

---

## API Contract

### GET /api/v1/items/:niin/comments
**Auth**: None  
**Ordering**: Oldest first (by `created_at ASC`)

**Response (200)**:
```json
{
  "status": 200,
  "message": "Comments retrieved",
  "data": [
    {
      "id": 123,
      "author_id": "firebase_uid",
      "author_display_name": "DisplayName",
      "comment_niin": "123456789",
      "text": "Comment body",
      "parent_id": null,
      "created_at": "2026-01-24T12:34:56Z"
    }
  ]
}
```

### POST /api/v1/auth/items/:niin/comments
**Auth**: Firebase JWT  
**Body**:
```json
{
  "text": "Comment body",
  "parent_id": "uuid-or-null"
}
```

**Response (201)**:
```json
{
  "status": 201,
  "message": "Comment created",
  "data": {
    "id": "uuid",
    "author_id": "firebase_uid",
    "comment_niin": "123456789",
    "text": "Comment body",
    "parent_id": "uuid-or-null",
    "created_at": "2026-01-24T12:40:00Z"
  }
}
```

**Errors**:
- `400` invalid NIIN or missing/empty `text`
- `401` missing/invalid auth token
- `500` unexpected server error

### PUT /api/v1/auth/items/:niin/comments/:comment_id
**Auth**: Firebase JWT  
**Rules**: Only author can edit.  
**Body**:
```json
{
  "text": "Updated comment body"
}
```

**Response (200)**:
```json
{
  "status": 200,
  "message": "Comment updated",
  "data": {
    "id": "uuid",
    "author_id": "firebase_uid",
    "comment_niin": "123456789",
    "text": "Updated comment body",
    "parent_id": "uuid-or-null",
    "created_at": "2026-01-24T12:40:00Z",
    "updated_at": "2026-01-24T13:00:00Z"
  }
}
```

**Errors**:
- `401` missing/invalid auth token
- `403` user is not the comment author
- `404` comment not found

### DELETE /api/v1/auth/items/:niin/comments/:comment_id
**Auth**: Firebase JWT  
**Rules**: Only author can delete.  

**Response (200)**:
```json
{
  "status": 200,
  "message": "Comment deleted",
  "data": {}
}
```

### POST /api/v1/auth/items/:niin/comments/:comment_id/flags
**Auth**: Firebase JWT  

**Response (201)**:
```json
{
  "status": 201,
  "message": "Comment flagged",
  "data": {
    "comment_id": "uuid"
  }
}
```

**Errors**:
- `409` comment already flagged by this user

---

## Application Flow

### Read Comments (Public)
1. Normalize NIIN (trim + uppercase).
2. Validate NIIN format and existence (FK ensures validity).
3. Query `item_comments` by `comment_niin`, ordered by `created_at ASC`.
4. Join `users` to return `author_display_name`.
5. Return `StandardResponse`.

### Create Comment (Authenticated)
1. Auth middleware populates `bootstrap.User` in context.
2. Validate request body (`text` required, max 255, `parent_id` optional).
3. Normalize NIIN (trim + uppercase).
4. Insert into `item_comments` with `author_id` and `created_at`.
5. Return created comment.

### Edit Comment (Authenticated)
1. Auth middleware populates `bootstrap.User`.
2. Validate `text` required, max 255.
3. Verify comment exists and `author_id` matches current user.
4. Update `text` and set `updated_at`.
5. Return updated comment.

### Delete Comment (Authenticated)
1. Auth middleware populates `bootstrap.User`.
2. Verify comment exists and `author_id` matches current user.
3. Soft delete (set `deleted_at`) or hard delete (TBD).
4. Return empty response.

### Flag Comment (Authenticated)
1. Auth middleware populates `bootstrap.User`.
2. Insert into `item_comment_flags` (unique on `comment_id` + `flagger_id`).
3. Return flag confirmation.

---

## Implementation Plan

1. Confirm whether to reuse `user_item_comments` or create a new table.
2. Add migrations for `item_comments` and `item_comment_flags`.
3. Regenerate Jet models.
4. Implement repository methods in `item_comments_repository_impl.go`.
5. Implement service methods in `item_comments_service_impl.go`.
6. Implement controller handlers in `item_comments_controller.go`.
7. Register routes in `item_comments_route.go` (public + auth).
8. Wire the new router from `api/route/route.go`.
9. Return `author_display_name` by joining `users`.

---

## Risks and Tradeoffs

- **No pagination**: large comment sets could be expensive to return and render.
- **No rate limiting**: spam risk exists; flags are the only mitigation.
- **Threading**: `parent_id` requires client-side grouping in the mobile app.
- **Soft delete**: retaining deleted rows may complicate queries if not filtered.

---

## Decisions

1. Order is oldest-first with threaded replies via `parent_id`.
2. Create a new `item_comments` table and `item_comment_flags` table.
3. Comment text is required, max length 255.
4. Comments require valid NIINs (FK to `nsn.niin`).
5. API returns `author_id` and `author_display_name`.
6. Author-only edit and delete.
7. Users can flag comments; no rate limiting.
8. No pagination for now.
9. Use a dedicated feature stack for item comments (not item query).
