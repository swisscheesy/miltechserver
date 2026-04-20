# Shop Messages — Reply Support API Changes

**Date:** 2026-04-06  
**Feature:** Message replies via `parent_id`  
**Base path:** `/api/v1/auth`

---

## Overview

The shop messages API now supports direct message replies. A message can reference another message in the same shop by including a `parent_id` in the create request. All message responses — across every existing endpoint — now include a `parent_id` field. It is `null` for top-level messages and a message ID string for replies.

No new endpoints were added. Only `POST /shops/messages` has a changed request shape. All GET endpoints return `parent_id` automatically with no query parameter required.

---

## Changed Endpoint

### POST /shops/messages

Creates a new shop message. Now accepts an optional `parent_id` to mark the message as a reply to an existing message.

#### Request

**Headers:**
- `Content-Type: application/json`
- Authentication header (Firebase token)

**Body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `shop_id` | string | Yes | The ID of the shop to post the message in |
| `message` | string | Yes | The message text |
| `parent_id` | string | No | The ID of the message being replied to. Omit or pass `null` for a top-level message |

**Example — top-level message (no change from before):**
```json
{
  "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Who has the torque wrench?"
}
```

**Example — reply to an existing message:**
```json
{
  "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "I have it, returning it in 10 minutes.",
  "parent_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543210"
}
```

#### Response

**Status:** `201 Created`

The response data is the newly created message object. It includes the `parent_id` field reflecting whatever was sent in the request.

**Example — reply created:**
```json
{
  "status": 201,
  "message": "Message created successfully",
  "data": {
    "id": "11223344-5566-7788-99aa-bbccddeeff00",
    "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user_id": "uid-user-123",
    "message": "I have it, returning it in 10 minutes.",
    "created_at": "2026-04-06T14:30:00Z",
    "updated_at": "2026-04-06T14:30:00Z",
    "is_edited": false,
    "parent_id": "f9e8d7c6-b5a4-3210-fedc-ba9876543210"
  }
}
```

**Example — top-level message (parent_id is null):**
```json
{
  "status": 201,
  "message": "Message created successfully",
  "data": {
    "id": "aabbccdd-eeff-0011-2233-445566778899",
    "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user_id": "uid-user-123",
    "message": "Who has the torque wrench?",
    "created_at": "2026-04-06T14:29:00Z",
    "updated_at": "2026-04-06T14:29:00Z",
    "is_edited": false,
    "parent_id": null
  }
}
```

---

## Unchanged Endpoints — Updated Response Shape

The following endpoints are **unchanged in their request parameters** but now include `parent_id` on every message object in their responses.

---

### GET /shops/:shop_id/messages

Returns all messages for a shop in ascending chronological order.

**Example response (mixed top-level and reply):**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "id": "aabbccdd-eeff-0011-2233-445566778899",
      "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "user_id": "uid-user-123",
      "message": "Who has the torque wrench?",
      "created_at": "2026-04-06T14:29:00Z",
      "updated_at": "2026-04-06T14:29:00Z",
      "is_edited": false,
      "parent_id": null
    },
    {
      "id": "11223344-5566-7788-99aa-bbccddeeff00",
      "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "user_id": "uid-user-456",
      "message": "I have it, returning it in 10 minutes.",
      "created_at": "2026-04-06T14:30:00Z",
      "updated_at": "2026-04-06T14:30:00Z",
      "is_edited": false,
      "parent_id": "aabbccdd-eeff-0011-2233-445566778899"
    }
  ]
}
```

---

### GET /shops/:shop_id/messages/paginated

Returns paginated messages. Supports both offset-based (`page`, `limit`) and cursor-based (`before_id`, `after_id`) pagination. All message objects in the response now include `parent_id`.

**Example response (cursor-based, includes reply):**
```json
{
  "status": 200,
  "message": "",
  "data": {
    "messages": [
      {
        "id": "11223344-5566-7788-99aa-bbccddeeff00",
        "shop_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "user_id": "uid-user-456",
        "message": "I have it, returning it in 10 minutes.",
        "created_at": "2026-04-06T14:30:00Z",
        "updated_at": "2026-04-06T14:30:00Z",
        "is_edited": false,
        "parent_id": "aabbccdd-eeff-0011-2233-445566778899"
      }
    ],
    "next_cursor": null
  }
}
```

---

## Message Object Reference

All endpoints return messages using this shape:

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique message ID (UUID) |
| `shop_id` | string | ID of the shop this message belongs to |
| `user_id` | string | ID of the user who posted the message |
| `message` | string | Message text content |
| `created_at` | string (ISO 8601) | When the message was created |
| `updated_at` | string (ISO 8601) | When the message was last updated |
| `is_edited` | boolean | Whether the message has been edited after creation |
| `parent_id` | string \| null | ID of the parent message if this is a reply; `null` for top-level messages |

---

## Implementation Notes

- **`parent_id` is optional on create.** Omitting the field and sending `"parent_id": null` are equivalent — both result in a top-level message.
- **The server does not validate that `parent_id` refers to an existing message.** If an invalid ID is sent, the server will return a `500` error. The mobile app should only send `parent_id` values obtained from previously fetched messages.
- **There is no depth limit.** A reply can itself be a reply. The API returns a flat list — threading structure must be assembled client-side using `id` and `parent_id`.
- **Deleting a parent message does not delete its replies.** The replies remain in the chat with their `parent_id` preserved, pointing to the now-deleted message ID.
