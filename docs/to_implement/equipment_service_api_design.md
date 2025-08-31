# Equipment Service API Design Document

## Overview

This document provides detailed specifications for implementing remote API endpoints for the Equipment Service feature within the Shop system. The equipment service feature allows users to track maintenance schedules, service records, and equipment status across different vehicles and equipment within their shops.

## Current Local Implementation Analysis

### Data Model Structure

The equipment service feature uses the following core data model:

```dart
class ShopEquipmentService {
  final String id;               // UUID primary key
  final String shopId;           // Foreign key to shop
  final String equipmentId;      // Foreign key to shop vehicle/equipment
  final String listId;           // Foreign key to shop list
  final String description;      // Service description
  final String type;             // Service type (maintenance, repair, etc.)
  final String createdBy;        // User ID who created the service
  final String createdByUsername; // Username for display
  final DateTime createdAt;      // Creation timestamp (UTC)
  final DateTime updatedAt;      // Last update timestamp (UTC)
  final DateTime? serviceDate;   // Scheduled or completed service date
  final int? serviceHours;       // Service hours or meter reading
}
```

### Database Schema

The local database table `equipment_services` has the following structure:
- Primary key: `id` (TEXT)
- Foreign key: `shopId` → `shops.id` (CASCADE DELETE)
- Foreign key: `equipmentId` → `shop_vehicles.id` (CASCADE DELETE)  
- Foreign key: `listId` → `shop_lists.id` (CASCADE DELETE)
- All string fields are TEXT, dates are DATETIME, hours is INTEGER

### Current Local Operations

The LocalShopRepository provides these operations for equipment services:
1. **getShopEquipmentServicesStream()** - Real-time stream of all services for a shop
2. **getShopEquipmentServices()** - Get all services for a shop (one-time)
3. **getEquipmentServicesStream()** - Real-time stream of services for specific equipment
4. **getEquipmentServices()** - Get services for specific equipment (one-time)
5. **getServicesInDateRange()** - Get services within date range for calendar view
6. **getEquipmentServiceById()** - Get specific service by ID
7. **insertEquipmentService()** - Create new service
8. **updateEquipmentService()** - Update existing service
9. **deleteEquipmentService()** - Delete service
10. **getOverdueServices()** - Get overdue services for a shop
11. **getServicesDueSoon()** - Get services due within 7 days

### UI Components and User Interactions

The feature includes these main UI components:
- **EquipmentTrackingScreen** - Main screen with vehicles and calendar tabs
- **EquipmentVehiclesTab** - Lists vehicles/equipment with service status
- **EquipmentCalendarTab** - Calendar view of scheduled services
- **AddEquipmentServiceDialog** - Form to create/edit services
- **EquipmentTrackingCubit** - State management for the feature

## Required API Endpoints

### Base URL Structure
All endpoints should follow the existing pattern: `/api/v1/shops/{shopId}/equipment-services/`

### Authentication & Authorization
- All endpoints require valid JWT token
- User must be a member of the shop to access services
- Only shop members can view services
- Only admins can delete services created by other users
- Users can modify their own service records

### 1. Get All Equipment Services for Shop

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment-services`

**Purpose:** Retrieve all equipment services for a specific shop with optional filtering

**Query Parameters:**
- `equipmentId` (optional): Filter by specific equipment/vehicle
- `startDate` (optional): Filter services from this date (ISO 8601)
- `endDate` (optional): Filter services to this date (ISO 8601)
- `type` (optional): Filter by service type
- `status` (optional): Filter by status (overdue, due_soon, scheduled, completed)
- `limit` (optional): Limit number of results (default 100, max 500)
- `offset` (optional): Pagination offset (default 0)

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "services": [
      {
        "id": "uuid-string",
        "shop_id": "uuid-string",
        "equipment_id": "uuid-string", 
        "list_id": "uuid-string",
        "description": "Oil change and filter replacement",
        "type": "maintenance",
        "created_by": "user-uuid",
        "created_by_username": "john_doe",
        "created_at": "2025-08-29T10:00:00.000Z",
        "updated_at": "2025-08-29T10:00:00.000Z",
        "service_date": "2025-09-15T08:00:00.000Z",
        "service_hours": 5000
      }
    ],
    "total_count": 25,
    "has_more": false
  },
  "message": "Services retrieved successfully"
}
```

**Error Responses:**
- `403 Forbidden`: User not authorized to access this shop
- `404 Not Found`: Shop not found
- `400 Bad Request`: Invalid query parameters

### 2. Get Specific Equipment Service

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment-services/{serviceId}`

**Purpose:** Retrieve a specific equipment service by ID

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "id": "uuid-string",
    "shop_id": "uuid-string",
    "equipment_id": "uuid-string",
    "list_id": "uuid-string", 
    "description": "Brake inspection and replacement",
    "type": "maintenance",
    "created_by": "user-uuid",
    "created_by_username": "jane_smith",
    "created_at": "2025-08-29T14:30:00.000Z",
    "updated_at": "2025-08-29T14:30:00.000Z",
    "service_date": "2025-09-20T09:00:00.000Z",
    "service_hours": 12500
  },
  "message": "Service retrieved successfully"
}
```

**Error Responses:**
- `403 Forbidden`: User not authorized to access this shop
- `404 Not Found`: Service or shop not found

### 3. Create Equipment Service

**Endpoint:** `POST /api/v1/shops/{shopId}/equipment-services`

**Purpose:** Create a new equipment service record

**Request Body:**
```json
{
  "equipment_id": "uuid-string",
  "list_id": "uuid-string",
  "description": "Tire rotation and pressure check",
  "type": "maintenance",
  "service_date": "2025-10-01T10:00:00.000Z",
  "service_hours": 8000
}
```

**Required Fields:**
- `equipment_id`: Must reference existing shop vehicle/equipment
- `list_id`: Must reference existing shop list  
- `description`: Service description (max 500 characters)
- `type`: Service type (maintenance, repair, inspection, etc.)

**Optional Fields:**
- `service_date`: Scheduled service date (can be past for completed services)
- `service_hours`: Equipment hours/mileage at service time

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "id": "new-uuid-string",
    "shop_id": "uuid-string",
    "equipment_id": "uuid-string",
    "list_id": "uuid-string",
    "description": "Tire rotation and pressure check", 
    "type": "maintenance",
    "created_by": "current-user-uuid",
    "created_by_username": "current_username",
    "created_at": "2025-08-29T15:00:00.000Z",
    "updated_at": "2025-08-29T15:00:00.000Z",
    "service_date": "2025-10-01T10:00:00.000Z",
    "service_hours": 8000
  },
  "message": "Equipment service created successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid input data or missing required fields
- `403 Forbidden`: User not authorized to create services in this shop
- `404 Not Found`: Referenced equipment, list, or shop not found
- `409 Conflict`: Duplicate service (if business rules prevent duplicates)

### 4. Update Equipment Service

**Endpoint:** `PUT /api/v1/shops/{shopId}/equipment-services/{serviceId}`

**Purpose:** Update an existing equipment service record

**Request Body:**
```json
{
  "description": "Updated service description",
  "type": "repair",
  "service_date": "2025-10-02T11:00:00.000Z", 
  "service_hours": 8100
}
```

**Updatable Fields:**
- `description`: Service description
- `type`: Service type
- `service_date`: Scheduled/completed service date
- `service_hours`: Equipment hours at service

**Non-updatable Fields:**
- `id`, `shop_id`, `equipment_id`, `list_id` (cannot be changed)
- `created_by`, `created_by_username`, `created_at` (audit fields)
- `updated_at` (automatically updated by server)

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "id": "uuid-string",
    "shop_id": "uuid-string", 
    "equipment_id": "uuid-string",
    "list_id": "uuid-string",
    "description": "Updated service description",
    "type": "repair", 
    "created_by": "user-uuid",
    "created_by_username": "original_creator",
    "created_at": "2025-08-29T15:00:00.000Z",
    "updated_at": "2025-08-29T16:30:00.000Z",
    "service_date": "2025-10-02T11:00:00.000Z",
    "service_hours": 8100
  },
  "message": "Equipment service updated successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid input data
- `403 Forbidden`: User not authorized to update this service
- `404 Not Found`: Service or shop not found
- `409 Conflict`: Update would violate business rules

### 5. Delete Equipment Service

**Endpoint:** `DELETE /api/v1/shops/{shopId}/equipment-services/{serviceId}`

**Purpose:** Delete an equipment service record

**Response Structure:**
```json
{
  "success": true,
  "message": "Equipment service deleted successfully"
}
```

**Authorization Rules:**
- Service creator can always delete their own services
- Shop admins can delete any service in their shop
- Regular members cannot delete services created by others

**Error Responses:**
- `403 Forbidden`: User not authorized to delete this service
- `404 Not Found`: Service or shop not found

### 6. Get Services by Equipment

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment/{equipmentId}/services`

**Purpose:** Get all services for a specific piece of equipment/vehicle

**Query Parameters:**
- `limit` (optional): Limit results (default 50, max 200)
- `offset` (optional): Pagination offset
- `startDate` (optional): Filter from date
- `endDate` (optional): Filter to date

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "equipment_id": "uuid-string",
    "services": [
      {
        "id": "uuid-string",
        "shop_id": "uuid-string",
        "equipment_id": "uuid-string",
        "list_id": "uuid-string",
        "description": "Service description",
        "type": "maintenance",
        "created_by": "user-uuid", 
        "created_by_username": "username",
        "created_at": "2025-08-29T10:00:00.000Z",
        "updated_at": "2025-08-29T10:00:00.000Z",
        "service_date": "2025-09-15T08:00:00.000Z",
        "service_hours": 5000
      }
    ],
    "total_count": 12,
    "has_more": false
  },
  "message": "Equipment services retrieved successfully"
}
```

### 7. Get Services in Date Range (Calendar View)

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment-services/calendar`

**Purpose:** Get services within a specific date range for calendar display

**Query Parameters:**
- `start_date` (required): Start date in ISO 8601 format
- `end_date` (required): End date in ISO 8601 format
- `equipment_id` (optional): Filter by specific equipment

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "date_range": {
      "start_date": "2025-09-01T00:00:00.000Z",
      "end_date": "2025-09-30T23:59:59.999Z"
    },
    "services": [
      {
        "id": "uuid-string", 
        "shop_id": "uuid-string",
        "equipment_id": "uuid-string",
        "list_id": "uuid-string",
        "description": "Monthly maintenance check",
        "type": "maintenance",
        "created_by": "user-uuid",
        "created_by_username": "maintenance_user",
        "created_at": "2025-08-29T10:00:00.000Z",
        "updated_at": "2025-08-29T10:00:00.000Z", 
        "service_date": "2025-09-15T08:00:00.000Z",
        "service_hours": 5200
      }
    ],
    "total_count": 8
  },
  "message": "Calendar services retrieved successfully"
}
```

### 8. Get Overdue Services

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment-services/overdue`

**Purpose:** Get services that are past their scheduled date

**Query Parameters:**
- `equipment_id` (optional): Filter by specific equipment
- `limit` (optional): Limit results (default 50)

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "overdue_services": [
      {
        "id": "uuid-string",
        "shop_id": "uuid-string", 
        "equipment_id": "uuid-string",
        "list_id": "uuid-string",
        "description": "Overdue oil change",
        "type": "maintenance",
        "created_by": "user-uuid",
        "created_by_username": "service_tech",
        "created_at": "2025-07-15T10:00:00.000Z",
        "updated_at": "2025-07-15T10:00:00.000Z",
        "service_date": "2025-08-15T08:00:00.000Z",
        "service_hours": 4800,
        "days_overdue": 14
      }
    ],
    "total_count": 3
  },
  "message": "Overdue services retrieved successfully"  
}
```

### 9. Get Services Due Soon

**Endpoint:** `GET /api/v1/shops/{shopId}/equipment-services/due-soon`

**Purpose:** Get services due within the next 7 days

**Query Parameters:**
- `days_ahead` (optional): Number of days to look ahead (default 7, max 30)
- `equipment_id` (optional): Filter by specific equipment
- `limit` (optional): Limit results (default 50)

**Response Structure:**
```json
{
  "success": true,
  "data": {
    "due_soon_services": [
      {
        "id": "uuid-string",
        "shop_id": "uuid-string",
        "equipment_id": "uuid-string", 
        "list_id": "uuid-string",
        "description": "Brake fluid replacement",
        "type": "maintenance",
        "created_by": "user-uuid",
        "created_by_username": "brake_specialist",
        "created_at": "2025-08-20T14:00:00.000Z",
        "updated_at": "2025-08-20T14:00:00.000Z",
        "service_date": "2025-09-02T10:00:00.000Z",
        "service_hours": 5500,
        "days_until_due": 4
      }
    ],
    "total_count": 2
  },
  "message": "Due soon services retrieved successfully"
}
```

## Database Requirements

### Table: equipment_services

The server should create a table with the following structure:

```sql
CREATE TABLE equipment_services (
    id VARCHAR(36) PRIMARY KEY,
    shop_id VARCHAR(36) NOT NULL,
    equipment_id VARCHAR(36) NOT NULL, 
    list_id VARCHAR(36) NOT NULL,
    description TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    created_by VARCHAR(36) NOT NULL,
    created_by_username VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    service_date TIMESTAMP NULL,
    service_hours INTEGER NULL,
    
    FOREIGN KEY (shop_id) REFERENCES shops(id) ON DELETE CASCADE,
    FOREIGN KEY (equipment_id) REFERENCES shop_vehicles(id) ON DELETE CASCADE,
    FOREIGN KEY (list_id) REFERENCES shop_lists(id) ON DELETE CASCADE,
    
    INDEX idx_shop_id (shop_id),
    INDEX idx_equipment_id (equipment_id),  
    INDEX idx_service_date (service_date),
    INDEX idx_created_by (created_by),
    INDEX idx_type (type)
);
```

### Data Validation Rules

**Server-side validation must enforce:**
1. **ID Format**: All IDs must be valid UUIDs
2. **Description**: 1-500 characters, non-empty
3. **Type**: Must be one of predefined values (maintenance, repair, inspection, upgrade, etc.)
4. **Timestamps**: All timestamps must be in UTC ISO 8601 format
5. **Service Hours**: If provided, must be positive integer
6. **Foreign Keys**: All referenced entities must exist and user must have access
7. **Service Date**: Can be past, present, or future (no restrictions)

## Integration Points

### Authentication Integration
- Use existing JWT token validation
- Extract user ID from token for `created_by` field
- Validate user membership in shop before any operations

### Shop Membership Validation  
- Reuse existing shop membership validation logic
- Ensure user is member of shop before accessing services
- Check admin permissions for delete operations

### Equipment/Vehicle Validation
- Validate `equipment_id` references exist in `shop_vehicles` table
- Ensure equipment belongs to the same shop
- Consider equipment status (active/inactive) if applicable

### Shop List Integration
- Validate `list_id` references exist in `shop_lists` table
- Ensure list belongs to the same shop
- Consider list permissions if applicable

## Error Handling Standards

### HTTP Status Codes
- `200 OK`: Successful GET, PUT operations
- `201 Created`: Successful POST operations  
- `204 No Content`: Successful DELETE operations
- `400 Bad Request`: Invalid input data, malformed requests
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: Valid authentication but insufficient permissions
- `404 Not Found`: Requested resource doesn't exist
- `409 Conflict`: Business rule violations, duplicate data
- `422 Unprocessable Entity`: Valid JSON but business logic errors
- `500 Internal Server Error`: Unexpected server errors

### Error Response Format
```json
{
  "success": false,
  "error": {
    "code": "INVALID_SERVICE_DATE",
    "message": "Service date cannot be more than 2 years in the future",
    "details": {
      "field": "service_date",
      "provided_value": "2027-12-25T10:00:00.000Z",
      "max_allowed": "2025-12-31T23:59:59.999Z"
    }
  }
}
```

### Common Error Codes
- `SHOP_NOT_FOUND`: Shop doesn't exist
- `SERVICE_NOT_FOUND`: Service doesn't exist  
- `EQUIPMENT_NOT_FOUND`: Equipment doesn't exist
- `LIST_NOT_FOUND`: List doesn't exist
- `INSUFFICIENT_PERMISSIONS`: User lacks required permissions
- `INVALID_INPUT`: Input validation failed
- `DUPLICATE_SERVICE`: Service already exists (if applicable)
- `INVALID_DATE_RANGE`: Date range parameters are invalid

## Performance Considerations

### Indexing Strategy
- Index on `shop_id` for shop-wide queries
- Index on `equipment_id` for equipment-specific queries  
- Index on `service_date` for date range and overdue queries
- Index on `created_by` for user-specific queries
- Consider composite index on `(shop_id, service_date)` for calendar queries

### Query Optimization
- Use LIMIT and OFFSET for pagination to prevent large result sets
- Consider caching for frequently accessed data (overdue services, due soon)
- Use prepared statements for all database queries
- Implement connection pooling for database connections

### Rate Limiting
- Implement per-user rate limiting (e.g., 100 requests per minute)
- Consider stricter limits for write operations
- Monitor and alert on unusual access patterns

## Security Considerations

### Input Sanitization
- Sanitize all string inputs to prevent XSS attacks
- Validate all UUIDs to ensure proper format
- Escape special characters in database queries
- Use parameterized queries to prevent SQL injection

### Authorization Matrix

| Operation | Shop Member | Shop Admin | Service Creator |
|-----------|-------------|------------|----------------|
| View Services | ✓ | ✓ | ✓ |
| Create Service | ✓ | ✓ | ✓ |
| Edit Own Service | ✓ | ✓ | ✓ |
| Edit Others' Service | ✗ | ✓ | ✗ |
| Delete Own Service | ✓ | ✓ | ✓ |
| Delete Others' Service | ✗ | ✓ | ✗ |

### Data Privacy
- Never expose internal user IDs in error messages
- Log access attempts for audit purposes
- Implement request/response logging (excluding sensitive data)
- Consider GDPR compliance for user data handling

## Testing Requirements

### Unit Tests
- Test all CRUD operations with valid data
- Test input validation for all fields
- Test error handling for various failure scenarios
- Test authorization logic for different user roles

### Integration Tests  
- Test with actual database connections
- Test foreign key constraints and cascading deletes
- Test transaction rollbacks on failures
- Test with concurrent operations

### API Tests
- Test all endpoints with various HTTP methods
- Test query parameter combinations
- Test request/response formats
- Test error response formats
- Performance testing with large datasets

## Migration and Deployment

### Database Migration
```sql
-- Migration script to add equipment_services table
-- This should be part of the database migration system

ALTER TABLE equipment_services 
ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;

-- Create indexes for performance
CREATE INDEX idx_equipment_services_shop_id ON equipment_services(shop_id);
CREATE INDEX idx_equipment_services_equipment_id ON equipment_services(equipment_id);
CREATE INDEX idx_equipment_services_service_date ON equipment_services(service_date);
CREATE INDEX idx_equipment_services_created_by ON equipment_services(created_by);
```

### Feature Flag
- Implement feature flag to gradually roll out equipment service API
- Allow toggling between local-only and remote+local modes
- Monitor API performance and error rates during rollout

### Backward Compatibility
- Maintain existing local database functionality during transition
- Implement sync mechanism to migrate existing local data to server
- Provide fallback to local-only mode if API is unavailable

This comprehensive design provides all necessary details for implementing the equipment service API endpoints that mirror the existing local functionality while following established patterns and security best practices.