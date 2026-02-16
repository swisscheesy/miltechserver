# Manual Testing Checklist - Post-Refactor Validation

**Purpose:** Validate that all application functionality works correctly after the bounded context refactoring completed on the `shop_refactor` branch.

**Last Updated:** 2026-01-31

**Refactoring Summary:** The codebase underwent a major Domain-Driven Design refactoring initiative using the Strangler Fig pattern. Monolithic domains were decomposed into focused bounded contexts while maintaining full backward API compatibility.


- POL
- Quick Lists
- PMCS
- LIN Lookup
- UOC Lookup
- MMDF
- Substitute LINS
- CAGE Lookup
- Reference Library

=== Shops ===
- Create Shop
- Create List
--- Empty Shop Audit List throws error:
flutter: \^[[38;5;196m┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────\^[[0m
flutter: \^[[38;5;196m│ ⛔ type 'Null' is not a subtype of type 'Map<String, dynamic>'\^[[0m
flutter: \^[[38;5;196m└───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────\^[[0m
flutter: \^[[38;5;196m┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────\^[[0m
flutter: \^[[38;5;196m│ ⛔ getShopNotificationChanges failed: Instance of 'RemoteUserSavesException'\^[[0m
flutter: \^[[38;5;196m└───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────\^[[0m

--- Vehicle Mileage Update
Only the user that updates the mileage sees the updated mileage.  Server still appears to send mileage and hours instead of tracked_mileage and tracked_hours
---
-- Created/Completed services are not tracked by auditing.
-- List Creation/Modification/Deletion isn't tracked by auditing.

---

## Table of Contents

1. [Pre-Testing Setup](#1-pre-testing-setup)
2. [Authentication & User Management](#2-authentication--user-management)
3. [Public Endpoints](#3-public-endpoints)
4. [User Saves](#4-user-saves)
5. [User Vehicles](#5-user-vehicles)
6. [Shops - Core Operations](#6-shops---core-operations)
7. [Shops - Membership & Invites](#7-shops---membership--invites)
8. [Shops - Messages](#8-shops---messages)
9. [Shops - Vehicles & Notifications](#9-shops---vehicles--notifications)
10. [Shops - Lists](#10-shops---lists)
11. [Equipment Services](#11-equipment-services)
12. [Material Images](#12-material-images)
13. [Item Comments](#13-item-comments)
14. [Library & Documents](#14-library--documents)
15. [Item Query (Short & Detailed)](#15-item-query-short--detailed)
16. [Item Lookup](#16-item-lookup)
17. [EIC Lookup](#17-eic-lookup)
18. [Quick Lists](#18-quick-lists)
19. [Integration Points](#19-integration-points)
20. [Edge Cases & Error Handling](#20-edge-cases--error-handling)
21. [Performance Validation](#21-performance-validation)

---

## 1. Pre-Testing Setup

### Environment Requirements
- [ ] PostgreSQL database running with latest migrations applied
- [ ] Azure Blob Storage connection configured
- [ ] Firebase Auth project configured
- [ ] Environment variables set (check `bootstrap/env.go`)
- [ ] Server running on port 8080

### Test User Accounts
- [x] Create at least 2 test users in Firebase Auth
- [x] Obtain valid Firebase ID tokens for both users
- [x] Note user IDs for ownership verification tests

### Test Data
- [x] Have valid NIIN values available for item queries
- [x] Have valid LIN values for lookup tests
- [x] Have valid CAGE codes for CAGE lookup tests

---

## 2. Authentication & User Management

### User Creation/Refresh
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Refresh/create user with valid token | `POST /auth/user/general/refresh` | 200 OK, user upserted | |
| Refresh with invalid token | `POST /auth/user/general/refresh` | 401 Unauthorized | |
| Refresh with expired token | `POST /auth/user/general/refresh` | 401 Unauthorized | |

### User Profile Updates
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Update display name | `POST /auth/user/general/dn_change` | 200 OK, name updated | |
| Update with empty display name | `POST /auth/user/general/dn_change` | 400 Bad Request | |

### User Deletion
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Delete user account | `DELETE /auth/user/general/delete_user` | 200 OK, user and related data deleted | |
| Verify cascading delete (saves, vehicles) | - | All user data removed | |

---

## 3. Public Endpoints

### General System
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get app version | `GET /api/v1/version` | 200 OK with version info | |
| Get database date | `GET /api/v1/general/db_date` | 200 OK with DB date | |

---

## 4. User Saves

### Quick Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get all quick items (empty) | `GET /auth/user/saves/quick_items` | 200 OK, empty array | |
| Add single quick item | `PUT /auth/user/saves/quick_items/add` | 200 OK, item created | |
| Add duplicate (upsert) | `PUT /auth/user/saves/quick_items/add` | 200 OK, item updated | |
| Add batch of quick items | `PUT /auth/user/saves/quick_items/addlist` | 200 OK, all items added | |
| Get all quick items | `GET /auth/user/saves/quick_items` | 200 OK, returns all items | |
| Delete single quick item | `DELETE /auth/user/saves/quick_items` | 200 OK, item deleted | |
| Delete all quick items | `DELETE /auth/user/saves/quick_items/all` | 200 OK, all items deleted | |

### Serialized Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get all serialized items (empty) | `GET /auth/user/saves/serialized_items` | 200 OK, empty array | |
| Add single serialized item | `PUT /auth/user/saves/serialized_items/add` | 200 OK, item created | |
| Add duplicate (upsert) | `PUT /auth/user/saves/serialized_items/add` | 200 OK, item updated | |
| Add batch of serialized items | `PUT /auth/user/saves/serialized_items/addlist` | 200 OK, all items added | |
| Get all serialized items | `GET /auth/user/saves/serialized_items` | 200 OK, returns all items | |
| Delete single serialized item | `DELETE /auth/user/saves/serialized_items` | 200 OK, item deleted | |
| Delete all serialized items | `DELETE /auth/user/saves/serialized_items/all` | 200 OK, all items deleted | |

### Categories
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get categories (empty) | `GET /auth/user/saves/item_category` | 200 OK, empty array | |
| Create category | `PUT /auth/user/saves/item_category` | 200 OK, category created | |
| Update category | `PUT /auth/user/saves/item_category` | 200 OK, category updated | |
| Delete category | `DELETE /auth/user/saves/item_category` | 200 OK, category deleted | |
| Delete all categories | `DELETE /auth/user/saves/item_category/all` | 200 OK, all deleted | |

### Categorized Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Add item to category | `PUT /auth/user/saves/categorized_items/add` | 200 OK, item added | |
| Add batch to category | `PUT /auth/user/saves/categorized_items/addlist` | 200 OK, items added | |
| Get items in category | `GET /auth/user/saves/categorized_items/category?category_id=X` | 200 OK, returns items | |
| Get all categorized items | `GET /auth/user/saves/categorized_items` | 200 OK, returns all | |
| Delete categorized item | `DELETE /auth/user/saves/categorized_items` | 200 OK, item deleted | |
| Delete all categorized items | `DELETE /auth/user/saves/categorized_items/all` | 200 OK, all deleted | |

### Item Images (Azure Blob)
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Upload image for quick item | `POST /auth/user/saves/items/image/upload/quick?item_id=X` | 200 OK, image stored | |
| Upload image for serialized item | `POST /auth/user/saves/items/image/upload/serialized?item_id=X` | 200 OK, image stored | |
| Get item image | `GET /auth/user/saves/items/image/quick?item_id=X` | 200 OK, image returned | |
| Delete item image | `DELETE /auth/user/saves/items/image/quick?item_id=X` | 200 OK, image deleted | |
| Verify Azure blob deleted | - | Blob removed from Azure | |

---

## 5. User Vehicles

### Vehicle CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get vehicles (empty) | `GET /auth/user/vehicles` | 200 OK, empty array | |
| Create vehicle | `PUT /auth/user/vehicles` | 200 OK, vehicle created | |
| Get vehicle by ID | `GET /auth/user/vehicles/:vehicleId` | 200 OK, returns vehicle | |
| Update vehicle | `PUT /auth/user/vehicles` | 200 OK, vehicle updated | |
| Delete vehicle by ID | `DELETE /auth/user/vehicles/:vehicleId` | 200 OK, vehicle deleted | |
| Delete all vehicles | `DELETE /auth/user/vehicles` | 200 OK, all deleted | |

### Vehicle Notifications
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get all notifications (empty) | `GET /auth/user/vehicle-notifications` | 200 OK, empty array | |
| Create notification for vehicle | `PUT /auth/user/vehicle-notifications` | 200 OK, notification created | |
| Get notification by ID | `GET /auth/user/vehicle-notifications/:notificationId` | 200 OK, returns notification | |
| Get notifications for vehicle | `GET /auth/user/vehicle-notifications/vehicle/:vehicleId` | 200 OK, returns list | |
| Update notification | `PUT /auth/user/vehicle-notifications` | 200 OK, updated | |
| Delete notification | `DELETE /auth/user/vehicle-notifications/:notificationId` | 200 OK, deleted | |
| Delete all for vehicle | `DELETE /auth/user/vehicle-notifications/vehicle/:vehicleId` | 200 OK, all deleted | |

### Notification Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get all notification items | `GET /auth/user/notification-items` | 200 OK | |
| Add item to notification | `PUT /auth/user/notification-items` | 200 OK, item added | |
| Add batch of items | `PUT /auth/user/notification-items/list` | 200 OK, items added | |
| Get items for notification | `GET /auth/user/notification-items/notification/:notificationId` | 200 OK, returns list | |
| Get item by ID | `GET /auth/user/notification-items/:itemId` | 200 OK, returns item | |
| Delete item | `DELETE /auth/user/notification-items/:itemId` | 200 OK, deleted | |
| Delete all for notification | `DELETE /auth/user/notification-items/notification/:notificationId` | 200 OK, all deleted | |

---

## 6. Shops - Core Operations

### Shop CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get shops (empty) | `GET /auth/shops` | 200 OK, empty array | |
| Create shop | `POST /auth/shops` | 200 OK, shop created, user is admin | |
| Get shop by ID | `GET /auth/shops/:shop_id` | 200 OK, includes stats | |
| Get user data with shops | `GET /auth/shops/user-data` | 200 OK, returns user + shops | |
| Update shop | `PUT /auth/shops/:shop_id` | 200 OK, shop updated | |
| Check if user is admin | `GET /auth/shops/:shop_id/is-admin` | 200 OK, returns boolean | |
| Delete shop | `DELETE /auth/shops/:shop_id` | 200 OK, cascades properly | |

### Shop Settings
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get shop settings | `GET /auth/shops/:shop_id/settings` | 200 OK, returns settings | |
| Update shop settings | `PUT /auth/shops/:shop_id/settings` | 200 OK, settings updated | |
| Get admin-only lists setting | `GET /auth/shops/:shop_id/settings/admin-only-lists` | 200 OK, returns boolean | |
| Toggle admin-only lists | `PUT /auth/shops/:shop_id/settings/admin-only-lists` | 200 OK, setting updated | |

---

## 7. Shops - Membership & Invites

### Invite Codes
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Generate invite code (admin) | `POST /auth/shops/invite-codes` | 200 OK, code generated | |
| Generate invite code (non-admin) | `POST /auth/shops/invite-codes` | 403 Forbidden | |
| Get invite codes | `GET /auth/shops/:shop_id/invite-codes` | 200 OK, returns codes | |
| Deactivate invite code | `DELETE /auth/shops/invite-codes/:code_id` | 200 OK, deactivated | |
| Delete invite code | `DELETE /auth/shops/invite-codes/:code_id/delete` | 200 OK, permanently deleted | |

### Membership
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Join shop with valid code (User 2) | `POST /auth/shops/join` | 200 OK, user joined | |
| Join with invalid code | `POST /auth/shops/join` | 404 Not Found | |
| Join with deactivated code | `POST /auth/shops/join` | 400 Bad Request | |
| Get shop members | `GET /auth/shops/:shop_id/members` | 200 OK, returns member list | |
| Promote member to admin | `PUT /auth/shops/members/promote` | 200 OK, user promoted | |
| Remove member (admin action) | `DELETE /auth/shops/members/remove` | 200 OK, member removed | |
| Remove member (non-admin) | `DELETE /auth/shops/members/remove` | 403 Forbidden | |
| Leave shop | `DELETE /auth/shops/:shop_id/leave` | 200 OK, user left | |

---

## 8. Shops - Messages

### Message CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Create message | `POST /auth/shops/messages` | 200 OK, message created | |
| Create message with image | `POST /auth/shops/messages` (multipart) | 200 OK, image uploaded | |
| Get all shop messages | `GET /auth/shops/:shop_id/messages` | 200 OK, returns messages | |
| Get paginated messages | `GET /auth/shops/:shop_id/messages/paginated?page=1&limit=20` | 200 OK, paginated | |
| Update message (author) | `PUT /auth/shops/messages` | 200 OK, updated | |
| Update message (non-author) | `PUT /auth/shops/messages` | 403 Forbidden | |
| Delete message | `DELETE /auth/shops/messages/:message_id` | 200 OK, cascades blob | |

### Message Images
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Upload message image | `POST /auth/shops/messages/image/upload` | 200 OK, image stored | |
| Delete message image | `DELETE /auth/shops/messages/image/:message_id` | 200 OK, blob deleted | |
| Verify blob deleted in Azure | - | Blob removed from storage | |

---

## 9. Shops - Vehicles & Notifications

### Shop Vehicles
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Create shop vehicle | `POST /auth/shops/vehicles` | 200 OK, vehicle created | |
| Get shop vehicles | `GET /auth/shops/:shop_id/vehicles` | 200 OK, returns list | |
| Get vehicle by ID | `GET /auth/shops/vehicles/:vehicle_id` | 200 OK, returns vehicle | |
| Update vehicle | `PUT /auth/shops/vehicles` | 200 OK, updated | |
| Delete vehicle | `DELETE /auth/shops/vehicles/:vehicle_id` | 200 OK, cascades | |

### Shop Notifications
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Create notification | `POST /auth/shops/vehicles/notifications` | 200 OK, created | |
| Get vehicle notifications | `GET /auth/shops/vehicles/:vehicle_id/notifications` | 200 OK, returns list | |
| Get notifications with items | `GET /auth/shops/vehicles/:vehicle_id/notifications-with-items` | 200 OK, includes items | |
| Get all shop notifications | `GET /auth/shops/:shop_id/notifications` | 200 OK, returns all | |
| Get notification by ID | `GET /auth/shops/vehicles/notifications/:notification_id` | 200 OK | |
| Update notification | `PUT /auth/shops/vehicles/notifications` | 200 OK, updated | |
| Delete notification | `DELETE /auth/shops/vehicles/notifications/:notification_id` | 200 OK, cascades | |

### Notification Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Add item to notification | `POST /auth/shops/notifications/items` | 200 OK, item added | |
| Add bulk items | `POST /auth/shops/notifications/items/bulk` | 200 OK, all added | |
| Get notification items | `GET /auth/shops/notifications/:notification_id/items` | 200 OK, returns list | |
| Get all shop notification items | `GET /auth/shops/:shop_id/notification-items` | 200 OK | |
| Delete item | `DELETE /auth/shops/notifications/items/:item_id` | 200 OK, deleted | |
| Delete bulk items | `DELETE /auth/shops/notifications/items/bulk` | 200 OK, all deleted | |

### Notification Audit Trail
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Update notification (triggers audit) | `PUT /auth/shops/vehicles/notifications` | Audit record created | |
| Get notification changes | `GET /auth/shops/notifications/:notification_id/changes` | 200 OK, returns history | |
| Get shop recent changes | `GET /auth/shops/:shop_id/notifications/changes?limit=100` | 200 OK, paginated | |
| Get vehicle notification changes | `GET /auth/shops/vehicles/:vehicle_id/notifications/changes` | 200 OK | |
| Verify 7 audit fields captured | - | old/new values for all tracked fields | |

---

## 10. Shops - Lists

### List CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Create list (admin) | `POST /auth/shops/lists` | 200 OK, list created | |
| Create list (member, admin-only=false) | `POST /auth/shops/lists` | 200 OK, list created | |
| Create list (member, admin-only=true) | `POST /auth/shops/lists` | 403 Forbidden | |
| Get shop lists | `GET /auth/shops/:shop_id/lists` | 200 OK, returns lists | |
| Get list by ID | `GET /auth/shops/lists/:list_id` | 200 OK | |
| Update list | `PUT /auth/shops/lists` | 200 OK, updated | |
| Delete list | `DELETE /auth/shops/lists` | 200 OK, cascades items | |

### List Items
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Add item to list | `POST /auth/shops/lists/items` | 200 OK, item added | |
| Add bulk items | `POST /auth/shops/lists/items/bulk` | 200 OK, all added | |
| Get list items | `GET /auth/shops/lists/:list_id/items` | 200 OK, returns items | |
| Update list item | `PUT /auth/shops/lists/items` | 200 OK, updated | |
| Delete item | `DELETE /auth/shops/lists/items` | 200 OK, deleted | |
| Delete bulk items | `DELETE /auth/shops/lists/items/bulk` | 200 OK, all deleted | |

---

## 11. Equipment Services

### Service CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Create equipment service | `POST /auth/shops/:shop_id/equipment-services` | 200 OK, created | |
| Get service by ID | `GET /auth/shops/:shop_id/equipment-services/:service_id` | 200 OK | |
| Update service | `PUT /auth/shops/:shop_id/equipment-services/:service_id` | 200 OK, updated | |
| Delete service | `DELETE /auth/shops/:shop_id/equipment-services/:service_id` | 200 OK, deleted | |

### Service Queries
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get by shop with filters | `GET /auth/shops/:shop_id/equipment-services?status=pending` | 200 OK, filtered | |
| Get by equipment with date range | `GET /auth/shops/:shop_id/equipment/:equipment_id/services?start_date=...` | 200 OK, filtered | |
| Get calendar view | `GET /auth/shops/:shop_id/equipment-services/calendar?month=2026-01` | 200 OK, grouped | |
| Get overdue services | `GET /auth/shops/:shop_id/equipment-services/overdue` | 200 OK, returns overdue | |
| Get due soon | `GET /auth/shops/:shop_id/equipment-services/due-soon?days=7` | 200 OK, returns upcoming | |

### Service Completion
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Mark service complete | `POST /auth/shops/:shop_id/equipment-services/:service_id/complete` | 200 OK, marked | |
| Complete with notes | `POST .../:service_id/complete` (with body) | 200 OK, notes saved | |

---

## 12. Material Images

### Image CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get images by NIIN (empty) | `GET /api/v1/material-images/niin/:niin` | 200 OK, empty array | |
| Upload image for NIIN | `POST /api/v1/material-images/upload` (multipart) | 200 OK, uploaded | |
| Get images by NIIN | `GET /api/v1/material-images/niin/:niin?page=1` | 200 OK, paginated | |
| Get image by ID | `GET /api/v1/material-images/:image_id` | 200 OK | |
| Get user's images | `GET /api/v1/material-images/user/:user_id` | 200 OK, paginated | |
| Delete image (author) | `DELETE /api/v1/material-images/:image_id` | 200 OK, blob deleted | |
| Delete image (non-author) | `DELETE /api/v1/material-images/:image_id` | 403 Forbidden | |

### Rate Limiting
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Upload 3 images in 1 hour | `POST /api/v1/material-images/upload` x3 | All succeed | |
| Upload 4th image in same hour | `POST /api/v1/material-images/upload` | 429 Too Many Requests | |

### Voting
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Upvote image | `POST /api/v1/material-images/:image_id/vote` | 200 OK, vote recorded | |
| Change vote to downvote | `POST /api/v1/material-images/:image_id/vote` | 200 OK, vote updated | |
| Remove vote | `DELETE /api/v1/material-images/:image_id/vote` | 200 OK, vote removed | |

### Flagging
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Flag image | `POST /api/v1/material-images/:image_id/flag` | 200 OK, flag created | |
| Get flag details | `GET /api/v1/material-images/:image_id/flags` | 200 OK, returns flags | |

---

## 13. Item Comments

### Comment CRUD
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get comments for NIIN (empty) | `GET /api/v1/items/:niin/comments` | 200 OK, empty array | |
| Create comment | `POST /api/v1/items/:niin/comments` | 200 OK, comment created | |
| Create threaded reply | `POST /api/v1/items/:niin/comments` (with parent_id) | 200 OK, threaded | |
| Update comment (author) | `PUT /api/v1/items/:niin/comments/:comment_id` | 200 OK, updated | |
| Update comment (non-author) | `PUT /api/v1/items/:niin/comments/:comment_id` | 403 Forbidden | |
| Delete comment (soft delete) | `DELETE /api/v1/items/:niin/comments/:comment_id` | 200 OK, soft deleted | |
| Flag comment | `POST /api/v1/items/:niin/comments/:comment_id/flags` | 200 OK, flag created | |

---

## 14. Library & Documents

### PMCS Documents
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| List vehicle folders | `GET /api/v1/library/pmcs/vehicles` | 200 OK, returns folders | |
| List documents in folder | `GET /api/v1/library/pmcs/:vehicle/documents` | 200 OK, returns PDFs | |
| Download document (get SAS URL) | `GET /api/v1/library/download?blob_path=...` | 200 OK, returns URL | |
| Verify SAS URL works | Open URL in browser | PDF downloads | |
| Verify SAS URL expires (1 hour) | Wait >1 hour, retry URL | 403 Forbidden | |

---

## 15. Item Query (Short & Detailed)

### Short Query
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Search by NIIN | `GET /api/v1/queries/items/initial?method=niin&value=123456789` | 200 OK, results | |
| Search by part number | `GET /api/v1/queries/items/initial?method=part&value=ABC123` | 200 OK, results | |
| Search with invalid NIIN | `GET /api/v1/queries/items/initial?method=niin&value=invalid` | 200 OK, empty or 404 | |
| Verify analytics tracked | Check analytics table | Search event recorded | |

### Detailed Query
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get detailed item data | `GET /api/v1/queries/items/detailed?niin=123456789` | 200 OK, comprehensive data | |
| Verify all data sections present | - | AMDF, characteristics, disposition, freight, etc. | |
| Verify response time | - | <2 seconds (post-optimization) | |
| Query non-existent NIIN | `GET /api/v1/queries/items/detailed?niin=000000000` | 404 or empty sections | |

---

## 16. Item Lookup

### LIN Lookup
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| List LIN records | `GET /lookup/lin?page=1` | 200 OK, paginated | |
| Get LIN by NIIN | `GET /lookup/lin/by-niin/:niin` | 200 OK, returns LIN | |
| Legacy route (LIN by NIIN) | `GET /lookup/lin/lin/:niin` | 200 OK, same result | |
| Get NIIN by LIN | `GET /lookup/niin/by-lin/:lin` | 200 OK, returns NIIN | |
| Legacy route (NIIN by LIN) | `GET /lookup/lin/niin/:lin` | 200 OK, same result | |

### UOC Lookup
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| List UOC records | `GET /lookup/uoc?page=1` | 200 OK, paginated | |
| Get UOC by code | `GET /lookup/uoc/:uoc` | 200 OK, returns record | |
| Get UOC by model | `GET /lookup/uoc/by-model/:model` | 200 OK, returns record | |
| Legacy route (by model) | `GET /lookup/uoc/model/:model` | 200 OK, same result | |

### CAGE Lookup
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Lookup CAGE address | `GET /lookup/cage/:cage` | 200 OK, returns address | |
| Invalid CAGE code | `GET /lookup/cage/INVALID` | 404 Not Found | |

### Substitute LIN
| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get substitute LIN records | `GET /lookup/substitute-lin` | 200 OK, returns all | |

---

## 17. EIC Lookup

| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Lookup EIC by NIIN | `GET /api/v1/eic/niin/:niin` | 200 OK, returns EIC | |
| Lookup EIC by LIN | `GET /api/v1/eic/lin/:lin` | 200 OK, returns EIC | |
| Lookup EIC by FSC (paginated) | `GET /api/v1/eic/fsc/:fsc?page=1` | 200 OK, paginated | |
| Search EIC items | `GET /api/v1/eic/items?page=1&search=query` | 200 OK, filtered results | |

---

## 18. Quick Lists

| Test Case | Endpoint | Expected Result | Pass/Fail |
|-----------|----------|-----------------|-----------|
| Get clothing list | `GET /api/v1/quick-lists/clothing` | 200 OK, returns list | |
| Get wheels list | `GET /api/v1/quick-lists/wheels` | 200 OK, returns list | |
| Get batteries list | `GET /api/v1/quick-lists/batteries` | 200 OK, returns list | |

---

## 19. Integration Points

### Azure Blob Storage
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Upload image (any endpoint) | Image stored in Azure | |
| Delete entity with image | Blob deleted from Azure | |
| Generate SAS URL | Valid, time-limited URL returned | |
| SAS URL expiration | URL fails after 1 hour | |

### Firebase Auth
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Valid token accepted | Request succeeds | |
| Expired token rejected | 401 Unauthorized | |
| Invalid token rejected | 401 Unauthorized | |
| User ID extracted correctly | Correct user in context | |

### Database (PostgreSQL)
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Connection pool working | No connection errors under load | |
| Cascading deletes work | Related data deleted properly | |
| Transactions commit properly | Data persisted correctly | |
| Transactions rollback on error | No partial data | |

### Analytics
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Item search tracked | Event counter incremented | |
| PMCS download tracked | Event counter incremented | |
| Analytics non-blocking | Main request not delayed | |

---

## 20. Edge Cases & Error Handling

### Authorization Edge Cases
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Access shop without membership | 403 Forbidden | |
| Admin action by non-admin | 403 Forbidden | |
| Delete another user's resource | 403 Forbidden | |
| Access deleted shop | 404 Not Found | |

### Input Validation
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Empty required fields | 400 Bad Request | |
| Invalid UUID format | 400 Bad Request | |
| Invalid pagination params | Default values used | |
| SQL injection attempt | Properly escaped, no error | |

### Concurrent Operations
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Simultaneous shop joins | All succeed or proper conflict | |
| Concurrent message posts | All messages created | |
| Concurrent item updates | Last write wins, no corruption | |

### Resource Cleanup
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Delete shop → cascades members | Members removed | |
| Delete shop → cascades messages | Messages + blobs removed | |
| Delete shop → cascades vehicles | Vehicles + notifications removed | |
| Delete shop → cascades lists | Lists + items removed | |
| Delete vehicle → cascades notifications | Notifications + items removed | |

---

## 21. Performance Validation

### Detailed Item Query (Post-Optimization)
| Test Case | Target | Actual | Pass/Fail |
|-----------|--------|--------|-----------|
| Single detailed query | <2 seconds | | |
| 10 concurrent queries | <5 seconds each | | |
| Query with cache hit | <100ms | | |

### General Response Times
| Endpoint Type | Target | Actual | Pass/Fail |
|---------------|--------|--------|-----------|
| Simple GET (list) | <200ms | | |
| Create/Update | <500ms | | |
| Complex query | <2s | | |
| Image upload | <5s | | |

### Connection Pool
| Test Case | Expected Result | Pass/Fail |
|-----------|-----------------|-----------|
| Max connections (50) | No errors at limit | |
| Connection recycling (5 min) | Connections refreshed | |
| Idle connection cleanup | Resources freed | |

---

## Test Execution Log

| Date | Tester | Sections Tested | Issues Found | Notes |
|------|--------|-----------------|--------------|-------|
| | | | | |
| | | | | |
| | | | | |

---

## Issues Tracker

| Issue # | Section | Description | Severity | Status | Resolution |
|---------|---------|-------------|----------|--------|------------|
| | | | | | |
| | | | | | |

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Developer | | | |
| QA | | | |
| Tech Lead | | | |

---

## Notes

- All authenticated endpoints require `Authorization: Bearer <firebase_token>` header
- Use separate test users for ownership/authorization tests
- Clean up test data after each test run or use isolated test database
- Document any deviations from expected behavior in Issues Tracker
- Performance tests should be run on production-like environment
