# Shops API Routes Documentation

This document provides comprehensive documentation for all shop-related API endpoints in the military tech server application.

## Base URL
All routes are prefixed with `/api/v1/auth` and require Firebase JWT authentication.

## Authentication
- **Required**: Firebase JWT token in Authorization header
- **Format**: `Authorization: Bearer <jwt_token>`

---

## Shop Operations

### Create Shop
**`POST /shops`**

Creates a new shop with the authenticated user as the initial admin.

**Request Body:**
```json
{
  "name": "string (required)",
  "details": "string (optional)",
  "password_hash": "string (optional)"
}
```

**Response:**
- **201 Created**: Returns created shop data
- **400 Bad Request**: Invalid request format
- **401 Unauthorized**: Authentication required

**Example Response:**
```json
{
  "status": 201,
  "message": "Shop created successfully",
  "data": {
    "id": "shop-uuid",
    "name": "Alpha Maintenance Shop",
    "details": "Main vehicle maintenance facility",
    "created_by": "user-uuid",
    "created_at": "2023-01-01T12:00:00Z",
    "updated_at": "2023-01-01T12:00:00Z"
  }
}
```

---

### Get User Shops
**`GET /shops`**

Returns all shops the authenticated user is a member of.

**Request Body:** None

**Response:**
- **200 OK**: Array of user's shops
- **404 Not Found**: No shops found
- **401 Unauthorized**: Authentication required

**Example Response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "id": "shop-uuid",
      "name": "Alpha Maintenance Shop",
      "details": "Main vehicle maintenance facility",
      "created_by": "user-uuid",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
  ]
}
```

---

### Get User Data with Shops
**`GET /shops/user-data`**

Returns user profile data along with enriched shop information including statistics and admin status.

**Request Body:** None

**Response:**
- **200 OK**: User data with shop statistics
- **401 Unauthorized**: Authentication required

**Example Response:**
```json
{
  "status": 200,
  "message": "User data and shops retrieved successfully",
  "data": {
    "user": {
      "user_id": "user-uuid",
      "username": "john_doe",
      "email": "john@example.com",
      "role": "user"
    },
    "shops": [
      {
        "shop": {
          "id": "shop-uuid",
          "name": "Alpha Maintenance Shop",
          "details": "Main vehicle maintenance facility"
        },
        "member_count": 15,
        "vehicle_count": 8,
        "is_admin": true
      }
    ]
  }
}
```

---

### Get Shop by ID
**`GET /shops/:shop_id`**

Returns detailed information for a specific shop.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Shop details
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

---

### Update Shop
**`PUT /shops/:shop_id`**

Updates shop information (name and details).

**Authorization:** Shop admin permissions required

**Request Body:**
```json
{
  "name": "string (required)",
  "details": "string (optional)"
}
```

**Response:**
- **200 OK**: Updated shop data
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Admin permissions required

---

### Delete Shop
**`DELETE /shops/:shop_id`**

Permanently deletes a shop and all associated data.

**Authorization:** Shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Deletion confirmation
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Admin permissions required

---

## Shop Member Operations

### Join Shop via Invite Code
**`POST /shops/join`**

Allows a user to join a shop using a valid invite code.

**Request Body:**
```json
{
  "invite_code": "string (required)"
}
```

**Response:**
- **200 OK**: Successfully joined shop
- **400 Bad Request**: Invalid invite code
- **401 Unauthorized**: Authentication required

---

### Leave Shop
**`DELETE /shops/:shop_id/leave`**

Allows a user to leave a shop they are a member of.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Successfully left shop
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

**Note:** If the last member leaves, the shop is automatically deleted.

---

### Remove Member from Shop
**`DELETE /shops/members/remove`**

Allows shop admins to remove members from the shop.

**Authorization:** Shop admin permissions required

**Request Body:**
```json
{
  "shop_id": "string (required)",
  "target_user_id": "string (required)"
}
```

**Response:**
- **200 OK**: Member removed successfully
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Admin permissions required

---

### Get Shop Members
**`GET /shops/:shop_id/members`**

Returns all members of a specific shop with their roles and usernames.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of shop members
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

**Example Response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "id": "member-uuid",
      "shop_id": "shop-uuid",
      "user_id": "user-uuid",
      "role": "admin",
      "joined_at": "2023-01-01T12:00:00Z",
      "username": "john_doe"
    }
  ]
}
```

---

## Shop Invite Code Operations

### Generate Invite Code
**`POST /shops/invite-codes`**

Creates a new invite code for a shop.

**Authorization:** Shop admin permissions required

**Request Body:**
```json
{
  "shop_id": "string (required)",
  "max_uses": "int32 (optional)",
  "expires_at": "string (optional, ISO format)"
}
```

**Response:**
- **201 Created**: Created invite code data
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Admin permissions required

---

### Get Invite Codes by Shop
**`GET /shops/:shop_id/invite-codes`**

Returns all invite codes for a specific shop.

**Authorization:** Shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Array of invite codes
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Admin permissions required

---

### Deactivate Invite Code
**`DELETE /shops/invite-codes/:code_id`**

Deactivates an invite code (makes it unusable but keeps in database for audit).

**Authorization:** Shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Code deactivated successfully
- **400 Bad Request**: Missing code_id
- **401 Unauthorized**: Admin permissions required

---

### Delete Invite Code
**`DELETE /shops/invite-codes/:code_id/delete`**

Permanently deletes an invite code from the database.

**Authorization:** Shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Code deleted successfully
- **400 Bad Request**: Missing code_id
- **401 Unauthorized**: Admin permissions required

---

## Shop Message Operations

### Create Shop Message
**`POST /shops/messages`**

Creates a new message in the shop chat system.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "shop_id": "string (required)",
  "message": "string (required)"
}
```

**Response:**
- **201 Created**: Created message data
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Not a shop member

---

### Get Shop Messages
**`GET /shops/:shop_id/messages`**

Returns all messages for a shop's chat.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of messages
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

---

### Update Shop Message
**`PUT /shops/messages`**

Updates an existing shop message.

**Authorization:** Message author permissions required

**Request Body:**
```json
{
  "message_id": "string (required)",
  "message": "string (required)"
}
```

**Response:**
- **200 OK**: Message updated successfully
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Not message author

---

### Delete Shop Message
**`DELETE /shops/messages/:message_id`**

Deletes a shop message.

**Authorization:** Message author OR shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Message deleted successfully
- **400 Bad Request**: Missing message_id
- **401 Unauthorized**: Insufficient permissions

---

## Shop Vehicle Operations

### Create Shop Vehicle
**`POST /shops/vehicles`**

Creates a new vehicle record for a shop.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "shop_id": "string (required)",
  "niin": "string (optional)",
  "admin": "string (required)",
  "model": "string (optional)",
  "serial": "string (optional)",
  "uoc": "string (default: 'UNK')",
  "mileage": "int32 (default: 0)",
  "hours": "int32 (default: 0)",
  "comment": "string (optional)"
}
```

**Response:**
- **201 Created**: Created vehicle data
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Not a shop member

**Example Response:**
```json
{
  "status": 201,
  "message": "Vehicle created successfully",
  "data": {
    "id": "vehicle-uuid",
    "creator_id": "user-uuid",
    "niin": "123456789",
    "admin": "john_doe",
    "model": "HMMWV M998",
    "serial": "ABC123",
    "uoc": "UNK",
    "mileage": 15000,
    "hours": 500,
    "comment": "Primary patrol vehicle",
    "save_time": "2023-01-01T12:00:00Z",
    "last_updated": "2023-01-01T12:00:00Z",
    "shop_id": "shop-uuid"
  }
}
```

---

### Get Shop Vehicles
**`GET /shops/:shop_id/vehicles`**

Returns all vehicles for a specific shop.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of vehicles
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

---

### Get Shop Vehicle by ID
**`GET /shops/vehicles/:vehicle_id`**

Returns detailed information for a specific vehicle.

**Authorization:** Shop membership required (for vehicle's shop)

**Request Body:** None

**Response:**
- **200 OK**: Vehicle details
- **400 Bad Request**: Missing vehicle_id
- **401 Unauthorized**: No access to vehicle

---

### Update Shop Vehicle
**`PUT /shops/vehicles`**

Updates an existing shop vehicle.

**Authorization:** Vehicle creator OR shop admin permissions required

**Request Body:**
```json
{
  "vehicle_id": "string (required)",
  "admin": "string (required)",
  "niin": "string (optional)",
  "model": "string (optional)",
  "serial": "string (optional)",
  "uoc": "string (optional)",
  "mileage": "int32 (optional)",
  "hours": "int32 (optional)",
  "comment": "string (optional)"
}
```

**Response:**
- **200 OK**: Vehicle updated successfully
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Insufficient permissions

---

### Delete Shop Vehicle
**`DELETE /shops/vehicles/:vehicle_id`**

Deletes a shop vehicle and all associated notifications.

**Authorization:** Vehicle creator OR shop admin permissions required

**Request Body:** None

**Response:**
- **200 OK**: Vehicle deleted successfully
- **400 Bad Request**: Missing vehicle_id
- **401 Unauthorized**: Insufficient permissions

---

## Shop Vehicle Notification Operations

### Create Vehicle Notification
**`POST /shops/vehicles/notifications`**

Creates a new notification/work order for a vehicle.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "shop_id": "string (required)",
  "vehicle_id": "string (required)",
  "title": "string (required)",
  "description": "string (optional)",
  "type": "string (required, values: M1|PM|MW)"
}
```

**Response:**
- **201 Created**: Created notification data
- **400 Bad Request**: Invalid request or type
- **401 Unauthorized**: Not a shop member

**Note:** 
- **M1**: Maintenance
- **PM**: Preventive Maintenance  
- **MW**: Modification Work

---

### Get Vehicle Notifications
**`GET /shops/vehicles/:vehicle_id/notifications`**

Returns all notifications for a specific vehicle.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of notifications
- **400 Bad Request**: Missing vehicle_id
- **401 Unauthorized**: No access to vehicle

---

### Get Vehicle Notifications with Items
**`GET /shops/vehicles/:vehicle_id/notifications-with-items`**

Returns all notifications for a vehicle including associated items/parts.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of notifications with items
- **400 Bad Request**: Missing vehicle_id
- **401 Unauthorized**: No access to vehicle

**Example Response:**
```json
{
  "status": 200,
  "message": "",
  "data": [
    {
      "notification": {
        "id": "notification-uuid",
        "shop_id": "shop-uuid",
        "vehicle_id": "vehicle-uuid",
        "title": "Oil Change",
        "description": "Routine oil change maintenance",
        "type": "PM",
        "completed": false,
        "save_time": "2023-01-01T12:00:00Z",
        "last_updated": "2023-01-01T12:00:00Z"
      },
      "items": [
        {
          "id": "item-uuid",
          "shop_id": "shop-uuid",
          "notification_id": "notification-uuid",
          "niin": "987654321",
          "nomenclature": "Oil Filter",
          "quantity": 1,
          "save_time": "2023-01-01T12:00:00Z"
        }
      ]
    }
  ]
}
```

---

### Get Shop Notifications
**`GET /shops/:shop_id/notifications`**

Returns all notifications across all vehicles in a shop.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of notifications
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

---

### Get Vehicle Notification by ID
**`GET /shops/vehicles/notifications/:notification_id`**

Returns detailed information for a specific notification.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Notification details
- **400 Bad Request**: Missing notification_id
- **401 Unauthorized**: No access to notification

---

### Update Vehicle Notification
**`PUT /shops/vehicles/notifications`**

Updates an existing vehicle notification.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "notification_id": "string (required)",
  "title": "string (required)",
  "description": "string (optional)",
  "type": "string (required, values: M1|PM|MW)",
  "completed": "boolean (required)"
}
```

**Response:**
- **200 OK**: Notification updated successfully
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: Not a shop member

---

### Delete Vehicle Notification
**`DELETE /shops/vehicles/notifications/:notification_id`**

Deletes a vehicle notification and all associated items.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Notification deleted successfully
- **400 Bad Request**: Missing notification_id
- **401 Unauthorized**: Not a shop member

---

## Shop Notification Item Operations

### Add Notification Item
**`POST /shops/notifications/items`**

Adds a single item/part to a vehicle notification.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "notification_id": "string (required)",
  "niin": "string (required)",
  "nomenclature": "string (required)",
  "quantity": "int32 (required)"
}
```

**Response:**
- **201 Created**: Created item data
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: No access to notification

---

### Get Notification Items
**`GET /shops/notifications/:notification_id/items`**

Returns all items for a specific notification.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of items
- **400 Bad Request**: Missing notification_id
- **401 Unauthorized**: No access to notification

---

### Get Shop Notification Items
**`GET /shops/:shop_id/notification-items`**

Returns all notification items across all vehicles in a shop.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Array of items
- **400 Bad Request**: Missing shop_id
- **401 Unauthorized**: Not a shop member

---

### Add Notification Item List (Bulk)
**`POST /shops/notifications/items/bulk`**

Adds multiple items to a vehicle notification in a single request.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "notification_id": "string (required)",
  "items": [
    {
      "notification_id": "string (required)",
      "niin": "string (required)",
      "nomenclature": "string (required)",
      "quantity": "int32 (required)"
    }
  ]
}
```

**Response:**
- **201 Created**: Array of created items
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: No access to notification

---

### Remove Notification Item
**`DELETE /shops/notifications/items/:item_id`**

Removes a single item from a vehicle notification.

**Authorization:** Shop membership required

**Request Body:** None

**Response:**
- **200 OK**: Item removed successfully
- **400 Bad Request**: Missing item_id
- **401 Unauthorized**: No access to item

---

### Remove Notification Item List (Bulk)
**`DELETE /shops/notifications/items/bulk`**

Removes multiple items from vehicle notifications in a single request.

**Authorization:** Shop membership required

**Request Body:**
```json
{
  "item_ids": ["string", "string", "..."]
}
```

**Response:**
- **200 OK**: Items removed successfully with count
- **400 Bad Request**: Invalid request
- **401 Unauthorized**: No access to items

---

## Authorization Summary

### Permission Levels
1. **Public**: No additional permissions (just authentication)
2. **Shop Member**: User must be a member of the relevant shop
3. **Shop Admin**: User must be an admin of the relevant shop
4. **Creator**: User must be the creator of the resource (vehicles)
5. **Author**: User must be the author of the resource (messages)

### Permission Matrix

| Operation | Authentication | Shop Member | Shop Admin | Creator | Author |
|-----------|---------------|-------------|------------|---------|--------|
| Create Shop | ✓ | | | | |
| View Shop | ✓ | ✓ | | | |
| Update Shop | ✓ | | ✓ | | |
| Delete Shop | ✓ | | ✓ | | |
| Join Shop | ✓ | | | | |
| Leave Shop | ✓ | ✓ | | | |
| Remove Member | ✓ | | ✓ | | |
| View Members | ✓ | ✓ | | | |
| Generate Invite | ✓ | | ✓ | | |
| Manage Invites | ✓ | | ✓ | | |
| Create Message | ✓ | ✓ | | | |
| View Messages | ✓ | ✓ | | | |
| Edit Message | ✓ | | | | ✓ |
| Delete Message | ✓ | | ✓ | | ✓ |
| Create Vehicle | ✓ | ✓ | | | |
| View Vehicles | ✓ | ✓ | | | |
| Update Vehicle | ✓ | | ✓ | ✓ | |
| Delete Vehicle | ✓ | | ✓ | ✓ | |
| Manage Notifications | ✓ | ✓ | | | |
| Manage Items | ✓ | ✓ | | | |

---

## Data Models

### Military-Specific Fields
- **NIIN**: National Item Identification Number (9-digit military part number)
- **UOC**: Usable On Code (military equipment category)
- **Nomenclature**: Official military name/description for parts

### Notification Types
- **M1**: General maintenance and repairs
- **PM**: Preventive maintenance (scheduled)
- **MW**: Modification work (upgrades/changes)

### Timestamps
All timestamps are in UTC format (ISO 8601: `YYYY-MM-DDTHH:mm:ssZ`)

---

## Error Handling

### Standard Error Response Format
```json
{
  "status": 400,
  "message": "Error description",
  "data": null
}
```

### Common HTTP Status Codes
- **200 OK**: Successful operation
- **201 Created**: Resource created successfully
- **400 Bad Request**: Invalid request format or missing required fields
- **401 Unauthorized**: Authentication required or insufficient permissions
- **404 Not Found**: Resource not found or user lacks access

### Error Prevention
- All endpoints validate required parameters
- Authorization checks are performed before any data operations
- Input validation prevents SQL injection and malformed data
- Proper error messages help identify specific issues