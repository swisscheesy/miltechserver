# Shops API Documentation

## Overview
The Shops API provides comprehensive management functionality for shops, including member management, vehicle tracking, notifications, lists, and messaging. All endpoints require Firebase authentication via Bearer token in the Authorization header.

## Authentication
All endpoints require authentication using Firebase JWT tokens:
```
Authorization: Bearer <firebase-jwt-token>
```

## Endpoints

### Shop Operations

#### 1. Create Shop
- **Endpoint**: `POST /shops`
- **Description**: Creates a new shop
- **Request Body**:
  ```json
  {
    "name": "string (required)",
    "details": "string (optional)",
    "password_hash": "string (optional)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Shop created successfully",
    "data": {
      "id": "string",
      "name": "string",
      "details": "string",
      "created_by": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  }
  ```

#### 2. Get User's Shops
- **Endpoint**: `GET /shops`
- **Description**: Returns all shops for the authenticated user
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "shop": {
          "id": "string",
          "name": "string",
          "details": "string",
          "created_by": "string",
          "created_at": "timestamp",
          "updated_at": "timestamp"
        },
        "member_count": "number",
        "vehicle_count": "number",
        "is_admin": "boolean"
      }
    ]
  }
  ```

#### 3. Get User Data with Shops
- **Endpoint**: `GET /shops/user-data`
- **Description**: Returns user data along with all shops they are a part of
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "User data and shops retrieved successfully",
    "data": {
      "user": {
        "user_id": "string",
        "email": "string"
      },
      "shops": [
        {
          "shop": "shop_object",
          "member_count": "number",
          "vehicle_count": "number",
          "is_admin": "boolean"
        }
      ]
    }
  }
  ```

#### 4. Get Shop by ID
- **Endpoint**: `GET /shops/{shop_id}`
- **Description**: Returns a specific shop by ID
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": {
      "id": "string",
      "name": "string",
      "details": "string",
      "created_by": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp"
    }
  }
  ```

#### 5. Update Shop
- **Endpoint**: `PUT /shops/{shop_id}`
- **Description**: Updates an existing shop
- **Path Parameters**: `shop_id` (string, required)
- **Request Body**:
  ```json
  {
    "name": "string (required)",
    "details": "string (optional)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "Shop updated successfully",
    "data": "updated_shop_object"
  }
  ```

#### 6. Delete Shop
- **Endpoint**: `DELETE /shops/{shop_id}`
- **Description**: Deletes a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Shop deleted successfully"
  }
  ```

### Shop Member Operations

#### 7. Join Shop via Invite Code
- **Endpoint**: `POST /shops/join`
- **Description**: Allows a user to join a shop using an invite code
- **Request Body**:
  ```json
  {
    "invite_code": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Successfully joined shop"
  }
  ```

#### 8. Leave Shop
- **Endpoint**: `DELETE /shops/{shop_id}/leave`
- **Description**: Allows a user to leave a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Successfully left shop"
  }
  ```

#### 9. Remove Member from Shop
- **Endpoint**: `DELETE /shops/members/remove`
- **Description**: Allows admins to remove members from a shop
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "target_user_id": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Member removed successfully"
  }
  ```

#### 10. Get Shop Members
- **Endpoint**: `GET /shops/{shop_id}/members`
- **Description**: Returns all members of a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "id": "string",
        "shop_id": "string",
        "user_id": "string",
        "role": "string",
        "joined_at": "timestamp",
        "username": "string"
      }
    ]
  }
  ```

### Shop Invite Code Operations

#### 11. Generate Invite Code
- **Endpoint**: `POST /shops/invite-codes`
- **Description**: Creates a new invite code for a shop
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "max_uses": "number (optional)",
    "expires_at": "string (optional, ISO format)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Invite code generated successfully",
    "data": {
      "id": "string",
      "shop_id": "string",
      "code": "string",
      "created_by": "string",
      "is_active": "boolean",
      "created_at": "timestamp"
    }
  }
  ```

#### 12. Get Invite Codes by Shop
- **Endpoint**: `GET /shops/{shop_id}/invite-codes`
- **Description**: Returns all invite codes for a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "id": "string",
        "shop_id": "string",
        "code": "string",
        "created_by": "string",
        "is_active": "boolean",
        "created_at": "timestamp"
      }
    ]
  }
  ```

#### 13. Deactivate Invite Code
- **Endpoint**: `DELETE /shops/invite-codes/{code_id}`
- **Description**: Deactivates an invite code
- **Path Parameters**: `code_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Invite code deactivated successfully"
  }
  ```

#### 14. Delete Invite Code
- **Endpoint**: `DELETE /shops/invite-codes/{code_id}/delete`
- **Description**: Permanently deletes an invite code
- **Path Parameters**: `code_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Invite code deleted successfully"
  }
  ```

### Shop Message Operations

#### 15. Create Shop Message
- **Endpoint**: `POST /shops/messages`
- **Description**: Creates a new message in the shop chat
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "message": "string (required)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Message created successfully",
    "data": "message_object"
  }
  ```

#### 16. Get Shop Messages
- **Endpoint**: `GET /shops/{shop_id}/messages`
- **Description**: Returns all messages for a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["message_objects"]
  }
  ```

#### 17. Update Shop Message
- **Endpoint**: `PUT /shops/messages`
- **Description**: Updates an existing shop message
- **Request Body**:
  ```json
  {
    "message_id": "string (required)",
    "message": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Message updated successfully"
  }
  ```

#### 18. Delete Shop Message
- **Endpoint**: `DELETE /shops/messages/{message_id}`
- **Description**: Deletes a shop message
- **Path Parameters**: `message_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Message deleted successfully"
  }
  ```

### Shop Vehicle Operations

#### 19. Create Shop Vehicle
- **Endpoint**: `POST /shops/vehicles`
- **Description**: Creates a new vehicle for a shop
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "niin": "string (optional)",
    "admin": "string (required)",
    "model": "string (optional)",
    "serial": "string (optional)",
    "uoc": "string (optional)",
    "mileage": "number (optional)",
    "hours": "number (optional)",
    "comment": "string (optional)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Vehicle created successfully",
    "data": {
      "id": "string",
      "creator_id": "string",
      "niin": "string",
      "admin": "string",
      "model": "string",
      "serial": "string",
      "uoc": "string",
      "mileage": "number",
      "hours": "number",
      "comment": "string",
      "save_time": "timestamp",
      "last_updated": "timestamp",
      "shop_id": "string"
    }
  }
  ```

#### 20. Get Shop Vehicles
- **Endpoint**: `GET /shops/{shop_id}/vehicles`
- **Description**: Returns all vehicles for a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["vehicle_objects"]
  }
  ```

#### 21. Get Shop Vehicle by ID
- **Endpoint**: `GET /shops/vehicles/{vehicle_id}`
- **Description**: Returns a specific vehicle by ID
- **Path Parameters**: `vehicle_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": "vehicle_object"
  }
  ```

#### 22. Update Shop Vehicle
- **Endpoint**: `PUT /shops/vehicles`
- **Description**: Updates an existing shop vehicle
- **Request Body**:
  ```json
  {
    "vehicle_id": "string (required)",
    "admin": "string (required)",
    "niin": "string (optional)",
    "model": "string (optional)",
    "serial": "string (optional)",
    "uoc": "string (optional)",
    "mileage": "number (optional)",
    "hours": "number (optional)",
    "comment": "string (optional)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Vehicle updated successfully"
  }
  ```

#### 23. Delete Shop Vehicle
- **Endpoint**: `DELETE /shops/vehicles/{vehicle_id}`
- **Description**: Deletes a shop vehicle
- **Path Parameters**: `vehicle_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Vehicle deleted successfully"
  }
  ```

### Shop Vehicle Notification Operations

#### 24. Create Vehicle Notification
- **Endpoint**: `POST /shops/vehicles/notifications`
- **Description**: Creates a new notification for a vehicle
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "vehicle_id": "string (required)",
    "title": "string (required)",
    "description": "string (optional)",
    "type": "string (required)" // M1, PM, MW
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Notification created successfully",
    "data": {
      "id": "string",
      "shop_id": "string",
      "vehicle_id": "string",
      "title": "string",
      "description": "string",
      "type": "string",
      "completed": "boolean",
      "save_time": "timestamp",
      "last_updated": "timestamp"
    }
  }
  ```

#### 25. Get Vehicle Notifications
- **Endpoint**: `GET /shops/vehicles/{vehicle_id}/notifications`
- **Description**: Returns all notifications for a vehicle
- **Path Parameters**: `vehicle_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["notification_objects"]
  }
  ```

#### 26. Get Vehicle Notifications with Items
- **Endpoint**: `GET /shops/vehicles/{vehicle_id}/notifications-with-items`
- **Description**: Returns all notifications for a vehicle with their items
- **Path Parameters**: `vehicle_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "notification": "notification_object",
        "items": ["item_objects"]
      }
    ]
  }
  ```

#### 27. Get Shop Notifications
- **Endpoint**: `GET /shops/{shop_id}/notifications`
- **Description**: Returns all notifications for a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["notification_objects"]
  }
  ```

#### 28. Get Vehicle Notification by ID
- **Endpoint**: `GET /shops/vehicles/notifications/{notification_id}`
- **Description**: Returns a specific notification by ID
- **Path Parameters**: `notification_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": "notification_object"
  }
  ```

#### 29. Update Vehicle Notification
- **Endpoint**: `PUT /shops/vehicles/notifications`
- **Description**: Updates an existing vehicle notification
- **Request Body**:
  ```json
  {
    "notification_id": "string (required)",
    "title": "string (required)",
    "description": "string (optional)",
    "type": "string (required)",
    "completed": "boolean (optional)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Notification updated successfully"
  }
  ```

#### 30. Delete Vehicle Notification
- **Endpoint**: `DELETE /shops/vehicles/notifications/{notification_id}`
- **Description**: Deletes a vehicle notification
- **Path Parameters**: `notification_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Notification deleted successfully"
  }
  ```

### Shop Notification Item Operations

#### 31. Add Notification Item
- **Endpoint**: `POST /shops/notifications/items`
- **Description**: Adds an item to a vehicle notification
- **Request Body**:
  ```json
  {
    "notification_id": "string (required)",
    "niin": "string (required)",
    "nomenclature": "string (required)",
    "quantity": "number (required)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Item added successfully",
    "data": "item_object"
  }
  ```

#### 32. Get Notification Items
- **Endpoint**: `GET /shops/notifications/{notification_id}/items`
- **Description**: Returns all items for a notification
- **Path Parameters**: `notification_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["item_objects"]
  }
  ```

#### 33. Get Shop Notification Items
- **Endpoint**: `GET /shops/{shop_id}/notification-items`
- **Description**: Returns all notification items for a shop
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": ["item_objects"]
  }
  ```

#### 34. Add Notification Item List (Bulk)
- **Endpoint**: `POST /shops/notifications/items/bulk`
- **Description**: Adds multiple items to a vehicle notification
- **Request Body**:
  ```json
  {
    "notification_id": "string (required)",
    "items": [
      {
        "notification_id": "string (required)",
        "niin": "string (required)",
        "nomenclature": "string (required)",
        "quantity": "number (required)"
      }
    ]
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Items added successfully",
    "data": ["item_objects"]
  }
  ```

#### 35. Remove Notification Item
- **Endpoint**: `DELETE /shops/notifications/items/{item_id}`
- **Description**: Removes an item from a vehicle notification
- **Path Parameters**: `item_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "message": "Item removed successfully"
  }
  ```

#### 36. Remove Notification Item List (Bulk)
- **Endpoint**: `DELETE /shops/notifications/items/bulk`
- **Description**: Removes multiple items from vehicle notifications
- **Request Body**:
  ```json
  {
    "item_ids": ["string (required)"]
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Items removed successfully",
    "count": "number"
  }
  ```

### Shop List Operations

#### 37. Create Shop List
- **Endpoint**: `POST /shops/lists`
- **Description**: Creates a new list for a shop
- **Request Body**:
  ```json
  {
    "shop_id": "string (required)",
    "description": "string (required)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "List created successfully",
    "data": "list_object"
  }
  ```

#### 38. Get Shop Lists
- **Endpoint**: `GET /shops/{shop_id}/lists`
- **Description**: Returns all lists for a shop with creator usernames
- **Path Parameters**: `shop_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "id": "string",
        "shop_id": "string",
        "created_by": "string",
        "created_by_username": "string",
        "description": "string",
        "created_at": "timestamp",
        "updated_at": "timestamp"
      }
    ]
  }
  ```

#### 39. Get Shop List by ID
- **Endpoint**: `GET /shops/lists/{list_id}`
- **Description**: Returns a specific list by ID
- **Path Parameters**: `list_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": "list_object"
  }
  ```

#### 40. Update Shop List
- **Endpoint**: `PUT /shops/lists`
- **Description**: Updates an existing shop list
- **Request Body**:
  ```json
  {
    "list_id": "string (required)",
    "description": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "List updated successfully"
  }
  ```

#### 41. Delete Shop List
- **Endpoint**: `DELETE /shops/lists`
- **Description**: Deletes a shop list
- **Request Body**:
  ```json
  {
    "list_id": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "List deleted successfully"
  }
  ```

### Shop List Item Operations

#### 42. Add List Item
- **Endpoint**: `POST /shops/lists/items`
- **Description**: Adds an item to a shop list
- **Request Body**:
  ```json
  {
    "list_id": "string (required)",
    "niin": "string (required)",
    "nomenclature": "string (required)",
    "quantity": "number (required)"
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Item added successfully",
    "data": "item_object"
  }
  ```

#### 43. Get List Items
- **Endpoint**: `GET /shops/lists/{list_id}/items`
- **Description**: Returns all items for a list with added by usernames
- **Path Parameters**: `list_id` (string, required)
- **Response**: 200 OK
  ```json
  {
    "status": 200,
    "message": "",
    "data": [
      {
        "id": "string",
        "list_id": "string",
        "niin": "string",
        "nomenclature": "string",
        "quantity": "number",
        "added_by": "string",
        "added_by_username": "string",
        "created_at": "timestamp",
        "updated_at": "timestamp"
      }
    ]
  }
  ```

#### 44. Update List Item
- **Endpoint**: `PUT /shops/lists/items`
- **Description**: Updates an existing list item
- **Request Body**:
  ```json
  {
    "item_id": "string (required)",
    "niin": "string (required)",
    "nomenclature": "string (required)",
    "quantity": "number (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Item updated successfully"
  }
  ```

#### 45. Remove List Item
- **Endpoint**: `DELETE /shops/lists/items`
- **Description**: Removes an item from a list
- **Request Body**:
  ```json
  {
    "item_id": "string (required)"
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Item removed successfully"
  }
  ```

#### 46. Add List Item Batch
- **Endpoint**: `POST /shops/lists/items/bulk`
- **Description**: Adds multiple items to a list
- **Request Body**:
  ```json
  {
    "list_id": "string (required)",
    "items": [
      {
        "list_id": "string (required)",
        "niin": "string (required)",
        "nomenclature": "string (required)",
        "quantity": "number (required)"
      }
    ]
  }
  ```
- **Response**: 201 Created
  ```json
  {
    "status": 201,
    "message": "Items added successfully",
    "data": ["item_objects"]
  }
  ```

#### 47. Remove List Item Batch
- **Endpoint**: `DELETE /shops/lists/items/bulk`
- **Description**: Removes multiple items from lists
- **Request Body**:
  ```json
  {
    "item_ids": ["string (required)"]
  }
  ```
- **Response**: 200 OK
  ```json
  {
    "message": "Items removed successfully",
    "count": "number"
  }
  ```

## Data Models

### Shop Object
```json
{
  "id": "string",
  "name": "string",
  "details": "string",
  "created_by": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Shop List Object
```json
{
  "id": "string",
  "shop_id": "string",
  "created_by": "string",
  "created_by_username": "string",
  "description": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Shop List Item Object
```json
{
  "id": "string",
  "list_id": "string",
  "niin": "string",
  "nomenclature": "string",
  "quantity": "number",
  "added_by": "string",
  "added_by_username": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Shop Member Object
```json
{
  "id": "string",
  "shop_id": "string",
  "user_id": "string",
  "role": "string",
  "joined_at": "timestamp"
}
```

### Shop Vehicle Object
```json
{
  "id": "string",
  "creator_id": "string",
  "niin": "string",
  "admin": "string",
  "model": "string",
  "serial": "string",
  "uoc": "string",
  "mileage": "number",
  "hours": "number",
  "comment": "string",
  "save_time": "timestamp",
  "last_updated": "timestamp",
  "shop_id": "string"
}
```

### Shop Vehicle Notification Object
```json
{
  "id": "string",
  "shop_id": "string",
  "vehicle_id": "string",
  "title": "string",
  "description": "string",
  "type": "string", // M1, PM, MW
  "completed": "boolean",
  "save_time": "timestamp",
  "last_updated": "timestamp"
}
```

### Shop Invite Code Object
```json
{
  "id": "string",
  "shop_id": "string",
  "code": "string",
  "created_by": "string",
  "is_active": "boolean",
  "created_at": "timestamp"
}
```

## Error Responses

### 401 Unauthorized
```json
{
  "message": "unauthorized"
}
```

### 400 Bad Request
```json
{
  "message": "invalid request"
}
```

### 404 Not Found
```json
{
  "status": 404,
  "message": "No item found",
  "data": {}
}
```

## Authorization Rules

### Shop Operations
- **Create Shop**: Any authenticated user
- **Update/Delete Shop**: Shop creator only
- **View Shop**: Shop members only

### Shop Members
- **Add Members**: Via invite codes (any member can use)
- **Remove Members**: Shop admins only
- **View Members**: Shop members only

### Shop Lists
- **Create List**: Any shop member
- **Update/Delete List**: List creator or shop admin
- **View Lists**: Any shop member

### Shop List Items
- **Add/Update/Remove Items**: Any shop member
- **View Items**: Any shop member

### Vehicle Operations
- **Create/Update/Delete Vehicles**: Any shop member
- **View Vehicles**: Any shop member

### Notifications & Messages
- **Create/Update/Delete**: Any shop member
- **View**: Any shop member

## Notes

1. All endpoints require Firebase authentication
2. The API uses Firebase JWT tokens for user identification
3. Shop roles include member/admin functionality
4. Vehicle notification types are: M1 (Maintenance), PM (Preventive Maintenance), MW (Modification)
5. All timestamps are in ISO format
6. NIIN stands for National Item Identification Number
7. UOC stands for Usable On Code
8. The system tracks both vehicle mileage and hours for maintenance scheduling
9. Bulk operations are available for notification items and list items
10. Shop membership is managed through invite codes
11. Lists can only be deleted by the creator or shop admins
12. All other list operations are available to any shop member

This API provides a complete shop management system with vehicle tracking, maintenance notifications, inventory lists, and team collaboration features.