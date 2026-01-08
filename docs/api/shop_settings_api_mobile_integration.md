# Shop Settings API - Mobile Integration Guide

**Version:** 1.0
**Date:** 2026-01-07
**Audience:** Mobile Development Team
**Status:** Ready for Integration

---

## Table of Contents
1. [Overview](#overview)
2. [What Changed and Why](#what-changed-and-why)
3. [New API Endpoints](#new-api-endpoints)
4. [Migration Path](#migration-path)
5. [Request/Response Examples](#requestresponse-examples)
6. [Error Handling](#error-handling)

---

## Overview

We've implemented a unified shop settings API that allows you to fetch and update all shop settings with a single endpoint call. This replaces the previous approach where each setting required its own dedicated endpoints.

**Benefits for Mobile:**
- **Fewer API calls**: Get all settings in one request instead of multiple requests
- **Atomic updates**: Update multiple settings in one call
- **Forward compatible**: New settings will automatically appear in responses
- **Simplified code**: Single settings model instead of multiple setting-specific methods
- **Backward compatible**: Existing endpoints continue to work during migration

---

## What Changed and Why

### Previous Approach

Before this update, getting shop settings required individual API calls:
- `GET /shops/:shop_id/settings/admin-only-lists` - To check one setting
- Future settings would have required additional endpoints

**Problems:**
- More network requests = slower loading
- More code to maintain
- Inconsistent state if calls fail partway through
- Difficult to expand with new settings

### New Approach

Now, one endpoint returns all settings:
- `GET /shops/:shop_id/settings` - Returns ALL settings
- `PUT /shops/:shop_id/settings` - Update one or multiple settings at once

**Improvements:**
- Single network request for all settings
- Atomic operations (all or nothing)
- Automatically includes new settings as they're added
- Cleaner, more maintainable client code

---

## New API Endpoints

### 1. Get All Shop Settings

**Endpoint:** `GET /api/v1/shops/:shop_id/settings`

**Purpose:** Retrieve all settings for a shop in a single call

**Authentication:** Required (Firebase JWT)

**Authorization:** User must be a member of the shop

**When to Use:**
- When loading shop details screen
- After updating settings (to get fresh state)
- When checking current shop configuration

**URL Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shop_id` | String (UUID) | Yes | The shop's unique identifier |

**Response (200 OK):**
```json
{
    "status": 200,
    "message": "Shop settings retrieved successfully",
    "data": {
        "admin_only_lists": true
    }
}
```

**Response Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `admin_only_lists` | Boolean | When `true`, only admins can create/modify lists in this shop |

> **Note:** As new settings are added to the backend, they will automatically appear in this response. Your app should handle unknown fields gracefully (ignore them or log for debugging).

---

### 2. Update Shop Settings

**Endpoint:** `PUT /api/v1/shops/:shop_id/settings`

**Purpose:** Update one or more shop settings

**Authentication:** Required (Firebase JWT)

**Authorization:** User must be a **shop administrator**

**When to Use:**
- When admin toggles a setting in shop management screen
- When bulk updating multiple settings

**URL Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shop_id` | String (UUID) | Yes | The shop's unique identifier |

**Request Body:**

All fields are **optional**. Only include the settings you want to update. Settings not included will remain unchanged.

```json
{
    "admin_only_lists": false
}
```

**Request Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `admin_only_lists` | Boolean | No | Set to `true` to restrict list management to admins only |

> **Important:** At least one setting must be provided in the request. Sending an empty object `{}` will return a 400 error.

**Response (200 OK):**
```json
{
    "status": 200,
    "message": "Shop settings updated successfully",
    "data": {
        "admin_only_lists": false
    }
}
```

The response returns the **complete** updated settings object, not just the fields you changed.

---

## Migration Path

### For New Code

Use the new unified endpoints immediately:
- ✅ `GET /shops/:shop_id/settings`
- ✅ `PUT /shops/:shop_id/settings`

### For Existing Code

**Legacy endpoints will continue to work:**
- `GET /shops/:shop_id/settings/admin-only-lists` (still active)
- `PUT /shops/:shop_id/settings/admin-only-lists` (still active)

---

## Request/Response Examples

### Example 1: Get All Settings

**HTTP Request:**
```
GET /api/v1/shops/{shop_id}/settings
Authorization: Bearer {firebase-jwt-token}
```

**Success Response (200):**
```json
{
    "status": 200,
    "message": "Shop settings retrieved successfully",
    "data": {
        "admin_only_lists": true
    }
}
```

---

### Example 2: Update Single Setting

**HTTP Request:**
```
PUT /api/v1/shops/{shop_id}/settings
Authorization: Bearer {firebase-jwt-token}
Content-Type: application/json
```

**Request Body:**
```json
{
    "admin_only_lists": false
}
```

**Success Response (200):**
```json
{
    "status": 200,
    "message": "Shop settings updated successfully",
    "data": {
        "admin_only_lists": false
    }
}
```

---

### Example 3: Update Multiple Settings (Future)

When additional settings are added, you can update multiple at once:

**HTTP Request:**
```
PUT /api/v1/shops/{shop_id}/settings
Authorization: Bearer {firebase-jwt-token}
Content-Type: application/json
```

**Request Body:**
```json
{
    "admin_only_lists": true
    // Future settings can be included here as additional fields
}
```

**Success Response (200):**
```json
{
    "status": 200,
    "message": "Shop settings updated successfully",
    "data": {
        "admin_only_lists": true
        // All current settings returned
    }
}
```

This is the main advantage: one call updates everything, atomically.

---

## Error Handling

### Common Error Responses

#### 401 Unauthorized
User is not authenticated.

```json
{
    "message": "unauthorized"
}
```

**Cause:** Missing or invalid Firebase JWT token
**Action:** Refresh authentication token and retry

---

#### 400 Bad Request

**Scenario 1: Missing shop_id**
```json
{
    "message": "shop_id is required"
}
```

**Cause:** shop_id parameter not provided in URL
**Action:** Ensure URL includes valid shop_id

**Scenario 2: No settings provided (PUT only)**
```json
{
    "message": "at least one setting must be provided"
}
```

**Cause:** Empty request body or all fields are null
**Action:** Include at least one setting to update

**Scenario 3: Invalid JSON**
```json
{
    "message": "invalid request"
}
```

**Cause:** Malformed JSON in request body
**Action:** Validate JSON structure before sending

---

#### 403 Forbidden

**Scenario 1: Not a shop member (GET)**
```json
{
    "message": "access denied: user is not a member of this shop"
}
```

**Cause:** User attempting to view settings for a shop they're not in
**Action:** Verify user is a shop member before calling

**Scenario 2: Not a shop admin (PUT)**
```json
{
    "message": "access denied: only shop administrators can modify settings"
}
```

**Cause:** Non-admin user attempting to update settings
**Action:** Check user's admin status before showing settings UI. Use `GET /shops/:shop_id/is-admin` to verify.

---

#### 404 Not Found

```json
{
    "message": "shop not found"
}
```

**Cause:** shop_id doesn't exist in database
**Action:** Verify shop_id is correct and shop hasn't been deleted

---

### Error Handling Best Practices

1. **Check HTTP Status Code First**
   - 200 → Success, parse response data
   - 401 → Refresh authentication token and retry
   - 403 → Show permission denied message
   - 404 → Shop not found, refresh shop list
   - 400 → Validation error, check request format
   - 500 → Server error, contact support

2. **Handle Network Errors**
   - Connection timeouts → Show retry option
   - No internet connection → Show offline message
   - Server unreachable → Check server status

3. **Display User-Friendly Messages**
   - Don't show technical error messages to users
   - Translate errors into actionable feedback
   - Examples:
     - "You need admin permissions to change shop settings"
     - "Unable to load settings. Please try again."
     - "Your session has expired. Please log in again."


## Appendix: Quick Reference

### Endpoints Summary

| Endpoint | Method | Auth | Permission | Purpose |
|----------|--------|------|------------|---------|
| `/shops/:shop_id/settings` | GET | Required | Shop Member | Get all settings |
| `/shops/:shop_id/settings` | PUT | Required | Shop Admin | Update settings |

### Status Codes

| Code | Meaning | Common Cause |
|------|---------|--------------|
| 200 | Success | Request completed successfully |
| 400 | Bad Request | Invalid input or missing required field |
| 401 | Unauthorized | Missing or invalid auth token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Shop doesn't exist |
| 500 | Server Error | Backend issue (rare, contact support) |

### Current Available Settings

| Setting Name | Type | Default | Description |
|--------------|------|---------|-------------|
| `admin_only_lists` | Boolean | `false` | When true, only admins can create/modify shop lists |

> **Note:** Additional settings will be added in future releases and will appear in API responses automatically.

---

**End of Mobile Integration Guide**
