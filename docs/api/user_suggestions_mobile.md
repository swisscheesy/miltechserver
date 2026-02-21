# User Suggestions API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-02-20
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Overview

This document describes the user suggestions feature. Users can submit feature requests and ideas, upvote or downvote submissions, and edit or delete their own entries. The list is publicly readable; all write operations require authentication.

Base URL prefixes:
- Public: `/api/v1`
- Authenticated: `/api/v1/auth`

---

## Authentication

Authenticated endpoints require a Firebase ID token in the `Authorization` header using the Bearer scheme.

Header:
- Authorization: Bearer {firebase-id-token}

The list endpoint (`GET /api/v1/suggestions`) accepts an optional token. Including a valid token causes the response to include the `my_vote` field per suggestion. Omitting the token or providing an invalid one returns suggestions without `my_vote`.

---

## Data Formatting and Rules

Suggestion ID:
- UUIDs only (e.g., `"3fa85f64-5717-4562-b3fc-2c963f66afa6"`)
- Passing a non-UUID string returns 400 with message `invalid suggestion ID`

Title:
- Required, 1–200 characters
- Leading and trailing whitespace is trimmed by the server

Description:
- Required, 1–2000 characters
- Leading and trailing whitespace is trimmed by the server

Vote direction:
- Integer, must be exactly `1` (upvote) or `-1` (downvote)
- Any other value returns 400 with message `invalid vote direction`
- Voting on a suggestion that does not exist returns 404

Ownership:
- Only the user who created a suggestion may edit or delete it
- Attempting to modify another user's suggestion returns 403

Score:
- The `score` field is the sum of all votes: `(number of upvotes) - (number of downvotes)`
- Score can be negative

Status:
- All suggestions are created with status `Submitted`
- Status is managed server-side and is not settable by the client

Timestamps:
- `created_at` is always present, formatted as RFC3339 (e.g., `"2026-02-20T15:04:05Z"`)
- `updated_at` is `null` until the suggestion has been edited

---

## Response Shapes

### Success wrapper (all 200/201 responses)
Fields:
- status: integer HTTP status code
- message: string
- data: response payload (object, array, or omitted on delete/vote)

### Error responses
Errors return a JSON object with a single `message` field, except internal server errors which use the full wrapper with `data: null`.

---

## Suggestion Object

Fields:
- id: string (UUID)
- user_id: string (Firebase UID of the author)
- username: string (from `users.username`, or `"Unknown"` if not found)
- title: string
- description: string
- status: string (e.g., `"Submitted"`)
- score: integer (sum of all vote directions)
- my_vote: integer or omitted — `1`, `-1`, or `null` (only present when the request was authenticated; omitted entirely for unauthenticated requests)
- created_at: RFC3339 timestamp
- updated_at: RFC3339 timestamp or `null`

---

## Endpoints

### 1) List all suggestions
**Method:** GET
**Path:** `/api/v1/suggestions`
**Auth:** Optional

No path or query parameters.

If a valid Bearer token is provided, `my_vote` is included in each suggestion object. If no token or an invalid token is provided, `my_vote` is omitted entirely from all objects.

Success (200):
- Uses success wrapper
- data: array of Suggestion Objects (newest first)

Example response — unauthenticated (200):
```json
{
  "status": 200,
  "message": "Suggestions retrieved",
  "data": [
    {
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "user_id": "firebase_uid_1",
      "username": "Alex Rivera",
      "title": "Add dark mode",
      "description": "A dark mode option would reduce eye strain during night shifts.",
      "status": "Submitted",
      "score": 12,
      "created_at": "2026-02-20T15:04:05Z",
      "updated_at": null
    },
    {
      "id": "9c8a2e17-3b4d-4f2a-9e11-5d7c6b8a1f3e",
      "user_id": "firebase_uid_2",
      "username": "Jordan Lee",
      "title": "Export to PDF",
      "description": "Allow exporting item queries to PDF for offline use.",
      "status": "Submitted",
      "score": -2,
      "created_at": "2026-02-19T10:30:00Z",
      "updated_at": "2026-02-19T11:00:00Z"
    }
  ]
}
```

Example response — authenticated (200):
```json
{
  "status": 200,
  "message": "Suggestions retrieved",
  "data": [
    {
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "user_id": "firebase_uid_1",
      "username": "Alex Rivera",
      "title": "Add dark mode",
      "description": "A dark mode option would reduce eye strain during night shifts.",
      "status": "Submitted",
      "score": 12,
      "my_vote": 1,
      "created_at": "2026-02-20T15:04:05Z",
      "updated_at": null
    },
    {
      "id": "9c8a2e17-3b4d-4f2a-9e11-5d7c6b8a1f3e",
      "user_id": "firebase_uid_2",
      "username": "Jordan Lee",
      "title": "Export to PDF",
      "description": "Allow exporting item queries to PDF for offline use.",
      "status": "Submitted",
      "score": -2,
      "my_vote": null,
      "created_at": "2026-02-19T10:30:00Z",
      "updated_at": "2026-02-19T11:00:00Z"
    }
  ]
}
```

Errors:
- 500: internal error payload

Example error (500):
```json
{
  "status": 500,
  "message": "internal Server Error",
  "data": null
}
```

---

### 2) Create suggestion
**Method:** POST
**Path:** `/api/v1/auth/suggestions`
**Auth:** Required

Request body fields:
- title: string, required (1–200 characters)
- description: string, required (1–2000 characters)

Success (201):
- Uses success wrapper
- data: Suggestion Object

Example request:
```json
{
  "title": "Add dark mode",
  "description": "A dark mode option would reduce eye strain during night shifts."
}
```

Example response (201):
```json
{
  "status": 201,
  "message": "Suggestion created",
  "data": {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "user_id": "firebase_uid_1",
    "username": "Alex Rivera",
    "title": "Add dark mode",
    "description": "A dark mode option would reduce eye strain during night shifts.",
    "status": "Submitted",
    "score": 0,
    "created_at": "2026-02-20T15:04:05Z",
    "updated_at": null
  }
}
```

Errors:
- 400: message `invalid title`
- 400: message `invalid description`
- 400: message `invalid request body`
- 401: message `unauthorized`
- 500: internal error payload

Example error (400 invalid title):
```json
{
  "message": "invalid title"
}
```

Example error (400 invalid description):
```json
{
  "message": "invalid description"
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

### 3) Update suggestion
**Method:** PUT
**Path:** `/api/v1/auth/suggestions/:id`
**Auth:** Required

Path parameters:
- id: string, required (UUID of the suggestion)

Request body fields:
- title: string, required (1–200 characters)
- description: string, required (1–2000 characters)

Both `title` and `description` must be provided. There is no partial update — both fields are replaced.

Success (200):
- Uses success wrapper
- data: Suggestion Object (with `updated_at` populated)

Example request:
```json
{
  "title": "Add dark mode support",
  "description": "A dark mode option would reduce eye strain during night and low-light operations."
}
```

Example response (200):
```json
{
  "status": 200,
  "message": "Suggestion updated",
  "data": {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "user_id": "firebase_uid_1",
    "username": "Alex Rivera",
    "title": "Add dark mode support",
    "description": "A dark mode option would reduce eye strain during night and low-light operations.",
    "status": "Submitted",
    "score": 0,
    "created_at": "2026-02-20T15:04:05Z",
    "updated_at": "2026-02-20T16:00:00Z"
  }
}
```

Errors:
- 400: message `invalid suggestion ID`
- 400: message `invalid title`
- 400: message `invalid description`
- 400: message `invalid request body`
- 401: message `unauthorized`
- 403: message `forbidden`
- 404: message `suggestion not found`
- 500: internal error payload

Example error (400 invalid ID):
```json
{
  "message": "invalid suggestion ID"
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
  "message": "suggestion not found"
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

### 4) Delete suggestion
**Method:** DELETE
**Path:** `/api/v1/auth/suggestions/:id`
**Auth:** Required

Path parameters:
- id: string, required (UUID of the suggestion)

No request body.

Success (200):
- Uses success wrapper
- No `data` field

Example response (200):
```json
{
  "status": 200,
  "message": "Suggestion deleted"
}
```

Errors:
- 400: message `invalid suggestion ID`
- 401: message `unauthorized`
- 403: message `forbidden`
- 404: message `suggestion not found`
- 500: internal error payload

Example error (403):
```json
{
  "message": "forbidden"
}
```

Example error (404):
```json
{
  "message": "suggestion not found"
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

### 5) Vote on a suggestion
**Method:** POST
**Path:** `/api/v1/auth/suggestions/:id/vote`
**Auth:** Required

Path parameters:
- id: string, required (UUID of the suggestion)

Request body fields:
- direction: integer, required — must be `1` (upvote) or `-1` (downvote)

If the user has already voted on this suggestion, the existing vote is replaced with the new direction. There is no separate "change vote" endpoint.

Success (200):
- Uses success wrapper
- No `data` field

Example request (upvote):
```json
{
  "direction": 1
}
```

Example request (downvote):
```json
{
  "direction": -1
}
```

Example response (200):
```json
{
  "status": 200,
  "message": "Vote recorded"
}
```

Errors:
- 400: message `invalid suggestion ID`
- 400: message `invalid vote direction`
- 400: message `invalid request body`
- 401: message `unauthorized`
- 404: message `suggestion not found`
- 500: internal error payload

Example error (400 invalid direction):
```json
{
  "message": "invalid vote direction"
}
```

Example error (404):
```json
{
  "message": "suggestion not found"
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

### 6) Remove vote from a suggestion
**Method:** DELETE
**Path:** `/api/v1/auth/suggestions/:id/vote`
**Auth:** Required

Path parameters:
- id: string, required (UUID of the suggestion)

No request body.

This removes the current user's vote from the suggestion. If the user has not voted, this is a no-op and still returns 200.

Success (200):
- Uses success wrapper
- No `data` field

Example response (200):
```json
{
  "status": 200,
  "message": "Vote removed"
}
```

Errors:
- 400: message `invalid suggestion ID`
- 401: message `unauthorized`
- 500: internal error payload

Example error (400):
```json
{
  "message": "invalid suggestion ID"
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

## Client Notes

- The list endpoint is not paginated; all suggestions are returned in a single response ordered newest first.
- The `my_vote` field is omitted entirely (not `null`) for unauthenticated list responses. Check for key presence rather than null-checking to determine whether the user is authenticated.
- When authenticated, `my_vote` is `null` if the user has not voted, `1` if upvoted, and `-1` if downvoted.
- Vote on a suggestion you already voted on to change direction — no separate "change vote" endpoint exists.
- Delete vote is always safe to call; if the user has no vote on record it returns 200 with no change.
- The `score` field can be negative if a suggestion has more downvotes than upvotes.
- `username` falls back to `"Unknown"` if the user record cannot be found; do not treat this as an error.
- `updated_at` is `null` until a suggestion has been edited at least once; display the edit timestamp only when non-null.
- A user cannot vote on their own suggestion at the API level; the server does not enforce this restriction. Enforce it client-side based on comparing `user_id` to the authenticated user's UID if desired.
