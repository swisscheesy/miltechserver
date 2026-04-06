# Shop Messages: parent_id Reply Support

**Date:** 2026-04-05  
**Status:** Approved

## Overview

Add support for direct message replies in shop chat by wiring the existing `parent_id` column (already present in the database and Jet-generated model) through the write path. All read endpoints already return `parent_id` via `AllColumns`.

## Background

The `shop_messages` table has a `parent_id` column (`*string`, nullable) that references another message's `id`. The Jet codegen already includes this field in `ShopMessages` model, `AllColumns`, and `MutableColumns`. The gap is entirely on the write path: `CreateShopMessageRequest`, the handler, and the `INSERT` statement do not yet include `parent_id`.

## Approach

Extend the existing `POST /shops/messages` endpoint with an optional `parent_id` field. No new endpoints. No server-side validation of parent existence — the client is trusted and the database nullable FK handles integrity.

## Changes

### 1. Request — `api/request/shops_request.go`

Add optional `ParentID *string` to `CreateShopMessageRequest`:

```go
type CreateShopMessageRequest struct {
    ShopID   string  `json:"shop_id" binding:"required"`
    Message  string  `json:"message" binding:"required"`
    ParentID *string `json:"parent_id"` // optional; nil = top-level message
}
```

### 2. Handler — `api/shops/messages/handler.go`

Map `ParentID` when building the model in `CreateShopMessage`:

```go
message := model.ShopMessages{
    ShopID:   req.ShopID,
    Message:  req.Message,
    ParentID: req.ParentID,
}
```

### 3. Repository — `api/shops/messages/repository_impl.go`

Add `ShopMessages.ParentID` to the INSERT column list in `CreateShopMessage`:

```go
stmt := ShopMessages.INSERT(
    ShopMessages.ID,
    ShopMessages.ShopID,
    ShopMessages.UserID,
    ShopMessages.Message,
    ShopMessages.CreatedAt,
    ShopMessages.UpdatedAt,
    ShopMessages.IsEdited,
    ShopMessages.ParentID, // added
).MODEL(message).RETURNING(ShopMessages.AllColumns)
```

When `ParentID` is `nil`, Jet writes `NULL`. No conditional logic needed.

## Read Endpoints — No Changes Required

All GET queries use `ShopMessages.AllColumns`, which already includes `ParentID` in the Jet-generated table. The field is already present in all message responses.

## Out of Scope

- Server-side validation that `parent_id` references an existing message in the same shop
- Nested/threaded response structures (responses return flat `[]model.ShopMessages`)
- Depth limiting or thread traversal logic
