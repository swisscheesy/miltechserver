# Shops API Endpoints Reference

This document describes all endpoints for the Shops feature, including expected request bodies and response formats.

---

## Shop Operations

### 1. Create Shop
**POST** `/shops`
#### Request
```json
{
  "name": "",
  "details": "",          // optional
  "password_hash": ""     // optional
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Shop created successfully",
  "data": {
    "id": "",
    "name": "",
    "details": "",
    "created_by": "",
    "created_at": "",
    "updated_at": ""
  }
}
```

### 2. Get User Shops
**GET** `/shops`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopResponse... } ]
}
```

### 3. Get Shop by ID
**GET** `/shops/{shop_id}`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": { ...ShopResponse... }
}
```

### 4. Delete Shop
**DELETE** `/shops/{shop_id}`
#### Response (200)
```json
{ "message": "Shop deleted successfully" }
```

---

## Shop Member Operations

### 5. Join Shop via Invite Code
**POST** `/shops/join`
#### Request
```json
{ "invite_code": "" }
```
#### Response (200)
```json
{ "message": "Successfully joined shop" }
```

### 6. Leave Shop
**DELETE** `/shops/{shop_id}/leave`
#### Response (200)
```json
{ "message": "Successfully left shop" }
```

### 7. Remove Member from Shop
**DELETE** `/shops/members/remove`
#### Request
```json
{
  "shop_id": "",
  "target_user_id": ""
}
```
#### Response (200)
```json
{ "message": "Member removed successfully" }
```

### 8. Get Shop Members
**GET** `/shops/{shop_id}/members`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopMemberResponse... } ]
}
```

---

## Shop Invite Code Operations

### 9. Generate Invite Code
**POST** `/shops/invite-codes`
#### Request
```json
{
  "shop_id": "",
  "max_uses": 0,        // optional
  "expires_at": ""      // ISO date, optional
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Invite code generated successfully",
  "data": { ...ShopInviteCodeResponse... }
}
```

### 10. Get Invite Codes by Shop
**GET** `/shops/{shop_id}/invite-codes`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopInviteCodeResponse... } ]
}
```

### 11. Deactivate Invite Code
**DELETE** `/shops/invite-codes/{code_id}`
#### Response (200)
```json
{ "message": "Invite code deactivated successfully" }
```

---

## Shop Message Operations

### 12. Create Shop Message
**POST** `/shops/messages`
#### Request
```json
{
  "shop_id": "",
  "message": ""
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Message created successfully",
  "data": { ...ShopMessageResponse... }
}
```

### 13. Get Shop Messages
**GET** `/shops/{shop_id}/messages`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopMessageResponse... } ]
}
```

### 14. Update Shop Message
**PUT** `/shops/messages`
#### Request
```json
{
  "message_id": "",
  "message": ""
}
```
#### Response (200)
```json
{ "message": "Message updated successfully" }
```

### 15. Delete Shop Message
**DELETE** `/shops/messages/{message_id}`
#### Response (200)
```json
{ "message": "Message deleted successfully" }
```

---

## Shop Vehicle Operations

### 16. Create Shop Vehicle
**POST** `/shops/vehicles`
#### Request
```json
{
  "shop_id": "",
  "niin": "",
  "model": "",
  "serial": "",
  "uoc": "",
  "mileage": 0,
  "hours": 0,
  "comment": ""
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Vehicle created successfully",
  "data": { ...ShopVehicleResponse... }
}
```

### 17. Get Shop Vehicles
**GET** `/shops/{shop_id}/vehicles`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopVehicleResponse... } ]
}
```

### 18. Get Shop Vehicle by ID
**GET** `/shops/vehicles/{vehicle_id}`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": { ...ShopVehicleResponse... }
}
```

### 19. Update Shop Vehicle
**PUT** `/shops/vehicles`
#### Request
```json
{
  "vehicle_id": "",
  "model": "",
  "serial": "",
  "uoc": "",
  "mileage": 0,
  "hours": 0,
  "comment": ""
}
```
#### Response (200)
```json
{ "message": "Vehicle updated successfully" }
```

### 20. Delete Shop Vehicle
**DELETE** `/shops/vehicles/{vehicle_id}`
#### Response (200)
```json
{ "message": "Vehicle deleted successfully" }
```

---

## Vehicle Notification Operations

### 21. Create Vehicle Notification
**POST** `/shops/vehicles/notifications`
#### Request
```json
{
  "vehicle_id": "",
  "title": "",
  "description": "",
  "type": ""         // M1, PM, MW
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Notification created successfully",
  "data": { ...ShopVehicleNotificationResponse... }
}
```

### 22. Get Vehicle Notifications
**GET** `/shops/vehicles/{vehicle_id}/notifications`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopVehicleNotificationResponse... } ]
}
```

### 23. Get Vehicle Notification by ID
**GET** `/shops/vehicles/notifications/{notification_id}`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": { ...ShopVehicleNotificationResponse... }
}
```

### 24. Update Vehicle Notification
**PUT** `/shops/vehicles/notifications`
#### Request
```json
{
  "notification_id": "",
  "title": "",
  "description": "",
  "type": "",
  "completed": false
}
```
#### Response (200)
```json
{ "message": "Notification updated successfully" }
```

### 25. Delete Vehicle Notification
**DELETE** `/shops/vehicles/notifications/{notification_id}`
#### Response (200)
```json
{ "message": "Notification deleted successfully" }
```

---

## Notification Item Operations

### 26. Add Notification Item
**POST** `/shops/notifications/items`
#### Request
```json
{
  "notification_id": "",
  "niin": "",
  "nomenclature": "",
  "quantity": 0
}
```
#### Response (201)
```json
{
  "status": 201,
  "message": "Item added successfully",
  "data": { ...ShopNotificationItemResponse... }
}
```

### 27. Get Notification Items
**GET** `/shops/notifications/{notification_id}/items`
#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": [ { ...ShopNotificationItemResponse... } ]
}
```

### 28. Add Notification Item List (Bulk)
**POST** `/shops/notifications/items/bulk`
#### Request
```json
{
  "notification_id": "",
  "items": [
    {
      "notification_id": "",
      "niin": "",
      "nomenclature": "",
      "quantity": 0
    }
  ]
}
```
#### Response (201)
```json
{ "message": "Items added successfully", "count": 0 }
```

### 29. Remove Notification Item
**DELETE** `/shops/notifications/items/{item_id}`
#### Response (200)
```json
{ "message": "Item removed successfully" }
```

### 30. Remove Notification Item List (Bulk)
**DELETE** `/shops/notifications/items/bulk`
#### Request
```json
{ "item_ids": ["", ""] }
```
#### Response (200)
```json
{ "message": "Items removed successfully", "count": 0 }
```

---

## Response Types Reference

- **ShopResponse**: See `api/response/shops_response.go` for fields.
- **ShopMemberResponse**: See `api/response/shops_response.go` for fields.
- **ShopInviteCodeResponse**: See `api/response/shops_response.go` for fields.
- **ShopMessageResponse**: See `api/response/shops_response.go` for fields.
- **ShopVehicleResponse**: See `api/response/shops_response.go` for fields.
- **ShopVehicleNotificationResponse**: See `api/response/shops_response.go` for fields.
- **ShopNotificationItemResponse**: See `api/response/shops_response.go` for fields.

All responses are wrapped in a StandardResponse:
```json
{
  "status": <int>,
  "message": <string>,
  "data": <object|array|null>
}
``` 