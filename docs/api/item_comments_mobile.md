# Item Comments API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-01-24
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Overview

This document describes the item comments feature for NIIN-based items. The API supports public read access and authenticated write/edit/delete/flag actions. Comments are returned oldest-first and support threaded replies via parent IDs.

Base URL prefixes:
- Public: `/api/v1`
- Authenticated: `/api/v1/auth`

---

## Authentication

Authenticated endpoints require a Firebase ID token in the `Authorization` header using the Bearer scheme.

Header:
- Authorization: Bearer {firebase-id-token}

---

## Data Formatting and Rules

NIIN:
- Exactly 9 digits
- Non-numeric or non-9-digit values return 400 with message `invalid NIIN`

Comment text:
- Required for create and update
- Length: 1–255 characters
- Whitespace is accepted and not trimmed

Replies:
- `parent_id` is optional
- If provided, it must reference an existing comment with the same `comment_niin`
- If invalid, returns 400 with message `invalid parent comment`

Deletion:
- Hard delete is not used
- Delete replaces the comment text with `Deleted by user`
- Comment remains in the list and replies are preserved

Ordering:
- All lists are oldest-first by `created_at`
- Client should render threads using `parent_id`

Flags:
- Boolean-only flagging (no reason)
- Re-flagging the same comment by the same user returns 200 with no change

Timestamps:
- `created_at` is returned as RFC3339 (Go `time.Time` JSON format)

---

## Response Shapes

### Success wrapper (most 200/201 responses)
Fields:
- status: integer HTTP status code
- message: string
- data: response payload

### Error responses
Errors are returned in two formats depending on the case:
- Simple message-only errors: JSON object with `message` only
- Internal server errors: JSON object with `status`, `message`, and `data` (data is null)

---

## Comment Object

Fields:
- id: string (UUID)
- comment_niin: string (9 digits)
- author_id: string (Firebase UID)
- author_display_name: string (from `users.username`, or `Unknown`)
- text: string
- parent_id: string or null (UUID)
- created_at: RFC3339 timestamp

---

## Endpoints

### 1) Get comments by NIIN
**Method:** GET
**Path:** `/api/v1/items/:niin/comments`
**Auth:** None

Path parameters:
- niin: string, required, 9 digits

Success (200):
- Uses success wrapper
- data: list of Comment Objects

Example response (200):
```json
{
  "status": 200,
  "message": "Comments retrieved",
  "data": [
    {
      "id": "8a01a5d7-5a7f-4a7f-8e11-1f2f9c9a3d1c",
      "comment_niin": "123456789",
      "author_id": "firebase_uid_1",
      "author_display_name": "Alex Rivera",
      "text": "First comment",
      "parent_id": null,
      "created_at": "2026-01-24T12:00:00Z"
    },
    {
      "id": "c4cc7f1a-7a9e-4c88-9a4f-0b7d8b1d2e3f",
      "comment_niin": "123456789",
      "author_id": "firebase_uid_2",
      "author_display_name": "Jordan Lee",
      "text": "Reply to first comment",
      "parent_id": "8a01a5d7-5a7f-4a7f-8e11-1f2f9c9a3d1c",
      "created_at": "2026-01-24T12:05:00Z"
    }
  ]
}
```

Errors:
- 400: message `invalid NIIN`
- 500: internal error payload

Example error (400):
```json
{
  "message": "invalid NIIN"
}
```

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

### 2) Create comment
**Method:** POST
**Path:** `/api/v1/auth/items/:niin/comments`
**Auth:** Required

Path parameters:
- niin: string, required, 9 digits

Request body fields:
- text: string, required (1–255)
- parent_id: string, optional (UUID)

Success (201):
- Uses success wrapper
- data: Comment Object

Example request:
```json
{
  "text": "New comment",
  "parent_id": null
}
```

Example response (201):
```json
{
  "status": 201,
  "message": "Comment created",
  "data": {
    "id": "b7e6e1c4-2b4c-4a2f-8b27-3d11a8c5f2d9",
    "comment_niin": "123456789",
    "author_id": "firebase_uid_1",
    "author_display_name": "Alex Rivera",
    "text": "New comment",
    "parent_id": null,
    "created_at": "2026-01-24T12:10:00Z"
  }
}
```

Errors:
- 400: message `invalid NIIN`
- 400: message `invalid comment text`
- 400: message `invalid parent comment`
- 401: message `unauthorized`
- 500: internal error payload

Example error (400 invalid NIIN):
```json
{
  "message": "invalid NIIN"
}
```

Example error (400 invalid comment text):
```json
{
  "message": "invalid comment text"
}
```

Example error (400 invalid parent comment):
```json
{
  "message": "invalid parent comment"
}
```

Example error (401):
```json
{
  "message": "unauthorized"
}
```

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

### 3) Update comment
**Method:** PUT
**Path:** `/api/v1/auth/items/:niin/comments/:comment_id`
**Auth:** Required

Path parameters:
- niin: string, required, 9 digits
- comment_id: string, required (UUID)

Request body fields:
- text: string, required (1–255)

Success (200):
- Uses success wrapper
- data: Comment Object

Example request:
```json
{
  "text": "Updated comment"
}
```

Example response (200):
```json
{
  "status": 200,
  "message": "Comment updated",
  "data": {
    "id": "b7e6e1c4-2b4c-4a2f-8b27-3d11a8c5f2d9",
    "comment_niin": "123456789",
    "author_id": "firebase_uid_1",
    "author_display_name": "Alex Rivera",
    "text": "Updated comment",
    "parent_id": null,
    "created_at": "2026-01-24T12:10:00Z"
  }
}
```

Errors:
- 400: message `invalid NIIN`
- 400: message `invalid comment text`
- 401: message `unauthorized`
- 403: message `forbidden`
- 404: message `comment not found`
- 500: internal error payload

Example error (400 invalid NIIN):
```json
{
  "message": "invalid NIIN"
}
```

Example error (400 invalid comment text):
```json
{
  "message": "invalid comment text"
}
```

Example error (401):
```json
{
  "message": "unauthorized"
}
```

Example error (403):
```json
{
  "message": "forbidden"
}
```

Example error (404):
```json
{
  "message": "comment not found"
}
```

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

### 4) Delete comment
**Method:** DELETE
**Path:** `/api/v1/auth/items/:niin/comments/:comment_id`
**Auth:** Required

Path parameters:
- niin: string, required, 9 digits
- comment_id: string, required (UUID)

Success (200):
- Uses success wrapper
- data: Comment Object with text set to `Deleted by user`

Example response (200):
```json
{
  "status": 200,
  "message": "Comment deleted",
  "data": {
    "id": "b7e6e1c4-2b4c-4a2f-8b27-3d11a8c5f2d9",
    "comment_niin": "123456789",
    "author_id": "firebase_uid_1",
    "author_display_name": "Alex Rivera",
    "text": "Deleted by user",
    "parent_id": null,
    "created_at": "2026-01-24T12:10:00Z"
  }
}
```

Errors:
- 400: message `invalid NIIN`
- 401: message `unauthorized`
- 403: message `forbidden`
- 404: message `comment not found`
- 500: internal error payload

Example error (400 invalid NIIN):
```json
{
  "message": "invalid NIIN"
}
```

Example error (401):
```json
{
  "message": "unauthorized"
}
```

Example error (403):
```json
{
  "message": "forbidden"
}
```

Example error (404):
```json
{
  "message": "comment not found"
}
```

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

### 5) Flag comment
**Method:** POST
**Path:** `/api/v1/auth/items/:niin/comments/:comment_id/flags`
**Auth:** Required

Path parameters:
- niin: string, required, 9 digits
- comment_id: string, required (UUID)

Request body:
- None

Success (200):
- Uses success wrapper
- data: object with field `comment_id`

Example response (200):
```json
{
  "status": 200,
  "message": "Comment flagged",
  "data": {
    "comment_id": "b7e6e1c4-2b4c-4a2f-8b27-3d11a8c5f2d9"
  }
}
```

Errors:
- 400: message `invalid NIIN`
- 401: message `unauthorized`
- 404: message `comment not found`
- 500: internal error payload

Example error (400 invalid NIIN):
```json
{
  "message": "invalid NIIN"
}
```

Example error (401):
```json
{
  "message": "unauthorized"
}
```

Example error (404):
```json
{
  "message": "comment not found"
}
```

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

## Client Notes

- The API does not paginate; expect all comments for a NIIN in one response.
- Replies are indicated by `parent_id` only; build the thread tree client-side.
- Deleted comments remain in the list with text replaced by `Deleted by user`.
- If `author_display_name` is `Unknown`, no username was found for that UID.
