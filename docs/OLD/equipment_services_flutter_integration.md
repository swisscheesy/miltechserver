# Equipment Services - Flutter Integration Guide

## Overview

The Equipment Services feature provides comprehensive service tracking and scheduling capabilities for shop equipment. This document provides all necessary information for Flutter application integration, including API endpoints, request/response formats, authentication requirements, and implementation guidelines.

## Table of Contents

1. [Authentication & Authorization](#authentication--authorization)
2. [API Base Configuration](#api-base-configuration)  
3. [Data Models](#data-models)
4. [API Endpoints](#api-endpoints)
5. [Implementation Examples](#implementation-examples)
6. [Error Handling](#error-handling)
7. [Flutter Integration Patterns](#flutter-integration-patterns)

## Authentication & Authorization

### Authentication Requirements
- **Bearer Token**: All endpoints require JWT authentication via Authorization header
- **User Context**: User information is automatically extracted from JWT token
- **Shop Access**: Users must have access to the specified shop_id

### Authorization Headers
```http
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

### User Validation
The API automatically validates:
- Valid JWT token in Authorization header
- User exists and is active
- User has appropriate permissions for the shop
- Equipment and list access permissions

## API Base Configuration

### Base URL Structure
```
{base_url}/api/v1/shops/{shop_id}/equipment-services
```

### Standard Response Format
All successful responses follow this structure:
```json
{
  "status": 200,
  "message": "Operation completed successfully", 
  "data": <response_data>
}
```

### Error Response Format
```json
{
  "status": 400|401|404|500,
  "message": "Error description",
  "data": {}
}
```

## Data Models

### EquipmentServiceResponse
```json
{
  "id": "string (UUID)",
  "shop_id": "string (UUID)",
  "equipment_id": "string (UUID)",
  "list_id": "string (UUID)", 
  "description": "string (1-500 chars)",
  "service_type": "string",
  "created_by": "string (User ID)",
  "created_by_username": "string (dynamically populated)",
  "is_completed": "boolean",
  "created_at": "string (ISO 8601 timestamp)",
  "updated_at": "string (ISO 8601 timestamp)",
  "service_date": "string|null (ISO 8601 timestamp)",
  "service_hours": "number|null (non-negative integer)",
  "completion_date": "string|null (ISO 8601 timestamp)"
}
```

### PaginatedEquipmentServicesResponse
```json
{
  "services": "EquipmentServiceResponse[]",
  "total_count": "number",
  "has_more": "boolean"
}
```

### CalendarServicesResponse
```json
{
  "date_range": {
    "start_date": "string (ISO 8601 timestamp)",
    "end_date": "string (ISO 8601 timestamp)"
  },
  "services": "EquipmentServiceResponse[]",
  "total_count": "number"
}
```

### OverdueServiceResponse
```json
{
  // Inherits all fields from EquipmentServiceResponse
  "days_overdue": "number"
}
```

### DueSoonServiceResponse  
```json
{
  // Inherits all fields from EquipmentServiceResponse
  "days_until_due": "number"
}
```

## API Endpoints

### 1. Create Equipment Service
**POST** `/shops/{shop_id}/equipment-services`

#### Request Body
```json
{
  "equipment_id": "string (required, UUID)",
  "list_id": "string (required, UUID)",
  "description": "string (required, 1-500 chars)",
  "service_type": "string (required)",
  "is_completed": "boolean (optional, default: false)",
  "service_date": "string|null (optional, ISO 8601)",
  "service_hours": "number|null (optional, non-negative)",
  "completion_date": "string|null (optional, ISO 8601)"
}
```

#### Response (201)
```json
{
  "status": 201,
  "message": "Equipment service created successfully",
  "data": <EquipmentServiceResponse>
}
```

### 2. Get Equipment Services (Paginated with Filters)
**GET** `/shops/{shop_id}/equipment-services`

#### Query Parameters
```
equipment_id: string (optional, UUID filter)
start_date: string (optional, ISO 8601 format)
end_date: string (optional, ISO 8601 format)  
service_type: string (optional, free-form filter)
is_completed: boolean (optional)
status: string (optional: "overdue", "due_soon", "scheduled", "completed")
limit: number (optional, 1-500, default: 100)
offset: number (optional, min: 0, default: 0)
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Services retrieved successfully",
  "data": <PaginatedEquipmentServicesResponse>
}
```

### 3. Get Equipment Service by ID
**GET** `/shops/{shop_id}/equipment-services/{service_id}`

#### Response (200)
```json
{
  "status": 200,
  "message": "",
  "data": <EquipmentServiceResponse>
}
```

### 4. Update Equipment Service  
**PUT** `/shops/{shop_id}/equipment-services/{service_id}`

#### Request Body
```json
{
  "description": "string (required, 1-500 chars)",
  "service_type": "string (required)",
  "is_completed": "boolean (optional)",
  "service_date": "string|null (optional, ISO 8601)",
  "service_hours": "number|null (optional, non-negative)",
  "completion_date": "string|null (optional, ISO 8601)"
}
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Equipment service updated successfully", 
  "data": <EquipmentServiceResponse>
}
```

### 5. Delete Equipment Service
**DELETE** `/shops/{shop_id}/equipment-services/{service_id}`

#### Response (200)
```json
{
  "message": "Equipment service deleted successfully"
}
```

### 6. Complete Equipment Service
**POST** `/shops/{shop_id}/equipment-services/{service_id}/complete`

#### Request Body
```json
{
  "completion_date": "string|null (optional, ISO 8601, defaults to current time)"
}
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Equipment service completed successfully",
  "data": <EquipmentServiceResponse>
}
```

### 7. Get Services by Equipment
**GET** `/shops/{shop_id}/equipment/{equipment_id}/services`

#### Query Parameters
```
start_date: string (optional, ISO 8601 format)
end_date: string (optional, ISO 8601 format)
limit: number (optional, 1-500, default: 100)
offset: number (optional, min: 0, default: 0)
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Services retrieved successfully",
  "data": <PaginatedEquipmentServicesResponse>
}
```

### 8. Get Services in Date Range (Calendar View)
**GET** `/shops/{shop_id}/equipment-services/calendar`

#### Query Parameters
```
start_date: string (required, ISO 8601 format)
end_date: string (required, ISO 8601 format)
equipment_id: string (optional, UUID filter)
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Calendar services retrieved successfully",
  "data": <CalendarServicesResponse>
}
```

### 9. Get Overdue Services
**GET** `/shops/{shop_id}/equipment-services/overdue`

#### Query Parameters
```
equipment_id: string (optional, UUID filter)
limit: number (optional, 1-200, default: 50)
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Overdue services retrieved successfully",
  "data": {
    "overdue_services": <OverdueServiceResponse[]>,
    "total_count": "number"
  }
}
```

### 10. Get Services Due Soon
**GET** `/shops/{shop_id}/equipment-services/due-soon`

#### Query Parameters
```
days_ahead: number (optional, 1-30, default: 7)
equipment_id: string (optional, UUID filter)
limit: number (optional, 1-200, default: 50)
```

#### Response (200)
```json
{
  "status": 200,
  "message": "Due soon services retrieved successfully", 
  "data": {
    "due_soon_services": <DueSoonServiceResponse[]>,
    "total_count": "number"
  }
}
```

## Implementation Examples

### Flutter HTTP Client Setup
```dart
class EquipmentServicesApi {
  final Dio _dio;
  final String baseUrl;
  
  EquipmentServicesApi({required this.baseUrl}) : _dio = Dio();
  
  void setAuthToken(String token) {
    _dio.options.headers['Authorization'] = 'Bearer $token';
  }
  
  String _buildUrl(String shopId, [String? path]) {
    final basePath = '/api/v1/shops/$shopId/equipment-services';
    return path != null ? '$baseUrl$basePath/$path' : '$baseUrl$basePath';
  }
}
```

### Create Service Example
```dart
Future<EquipmentService> createService({
  required String shopId,
  required String equipmentId,
  required String listId,
  required String description,
  required String serviceType,
  DateTime? serviceDate,
  int? serviceHours,
}) async {
  final response = await _dio.post(
    _buildUrl(shopId),
    data: {
      'equipment_id': equipmentId,
      'list_id': listId, 
      'description': description,
      'service_type': serviceType,
      'service_date': serviceDate?.toIso8601String(),
      'service_hours': serviceHours,
    },
  );
  
  return EquipmentService.fromJson(response.data['data']);
}
```

### Get Services with Filters Example
```dart
Future<PaginatedServicesResponse> getServices({
  required String shopId,
  String? equipmentId,
  DateTime? startDate,
  DateTime? endDate,
  String? serviceType,
  bool? isCompleted,
  String? status,
  int limit = 100,
  int offset = 0,
}) async {
  final queryParams = <String, dynamic>{
    'limit': limit,
    'offset': offset,
  };
  
  if (equipmentId != null) queryParams['equipment_id'] = equipmentId;
  if (startDate != null) queryParams['start_date'] = startDate.toIso8601String();
  if (endDate != null) queryParams['end_date'] = endDate.toIso8601String();
  if (serviceType != null) queryParams['service_type'] = serviceType;
  if (isCompleted != null) queryParams['is_completed'] = isCompleted;
  if (status != null) queryParams['status'] = status;
  
  final response = await _dio.get(
    _buildUrl(shopId),
    queryParameters: queryParams,
  );
  
  return PaginatedServicesResponse.fromJson(response.data['data']);
}
```

### Complete Service Example
```dart
Future<EquipmentService> completeService({
  required String shopId,
  required String serviceId,
  DateTime? completionDate,
}) async {
  final response = await _dio.post(
    _buildUrl(shopId, '$serviceId/complete'),
    data: {
      if (completionDate != null)
        'completion_date': completionDate.toIso8601String(),
    },
  );
  
  return EquipmentService.fromJson(response.data['data']);
}
```

## Error Handling

### Common Error Codes
- **400**: Invalid request parameters or validation errors
- **401**: Authentication required or invalid token
- **403**: Insufficient permissions for shop or equipment
- **404**: Service, equipment, or shop not found  
- **500**: Internal server error

### Error Response Handling
```dart
try {
  final service = await api.createService(/* params */);
  return service;
} on DioException catch (e) {
  switch (e.response?.statusCode) {
    case 400:
      throw ValidationException(e.response?.data['message'] ?? 'Invalid request');
    case 401:
      throw AuthenticationException('Authentication required');
    case 403:
      throw AuthorizationException('Insufficient permissions');
    case 404:
      throw NotFoundException(e.response?.data['message'] ?? 'Resource not found');
    default:
      throw ApiException('Service error: ${e.message}');
  }
}
```

## Flutter Integration Patterns

### State Management Example (using Bloc)
```dart
class EquipmentServicesBloc extends Bloc<EquipmentServicesEvent, EquipmentServicesState> {
  final EquipmentServicesApi _api;
  
  EquipmentServicesBloc(this._api) : super(EquipmentServicesInitial()) {
    on<LoadServices>(_onLoadServices);
    on<CreateService>(_onCreateService);
    on<CompleteService>(_onCompleteService);
    on<LoadOverdueServices>(_onLoadOverdueServices);
    on<LoadDueSoonServices>(_onLoadDueSoonServices);
  }
  
  Future<void> _onLoadServices(LoadServices event, Emitter<EquipmentServicesState> emit) async {
    emit(EquipmentServicesLoading());
    try {
      final services = await _api.getServices(
        shopId: event.shopId,
        equipmentId: event.equipmentId,
        startDate: event.startDate,
        endDate: event.endDate,
        limit: event.limit,
        offset: event.offset,
      );
      emit(EquipmentServicesLoaded(services));
    } catch (e) {
      emit(EquipmentServicesError(e.toString()));
    }
  }
}
```

### Calendar Integration Example
```dart
Future<List<CalendarEvent>> loadCalendarEvents({
  required String shopId,
  required DateTime startDate,
  required DateTime endDate,
}) async {
  final response = await _api.getServicesInDateRange(
    shopId: shopId,
    startDate: startDate,
    endDate: endDate,
  );
  
  return response.services.map((service) => CalendarEvent(
    id: service.id,
    title: '${service.serviceType}: ${service.description}',
    start: service.serviceDate ?? service.createdAt,
    isCompleted: service.isCompleted,
    equipmentId: service.equipmentId,
  )).toList();
}
```

### Dashboard Widgets Example
```dart
class ServicesDashboard extends StatelessWidget {
  final String shopId;
  
  const ServicesDashboard({Key? key, required this.shopId}) : super(key: key);
  
  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Overdue Services Card
        FutureBuilder<OverdueServicesResponse>(
          future: context.read<EquipmentServicesApi>().getOverdueServices(shopId: shopId),
          builder: (context, snapshot) {
            if (snapshot.hasData) {
              return OverdueServicesCard(
                count: snapshot.data!.totalCount,
                services: snapshot.data!.overdueServices,
              );
            }
            return const CircularProgressIndicator();
          },
        ),
        
        // Due Soon Services Card  
        FutureBuilder<DueSoonServicesResponse>(
          future: context.read<EquipmentServicesApi>().getServicesDueSoon(shopId: shopId),
          builder: (context, snapshot) {
            if (snapshot.hasData) {
              return DueSoonServicesCard(
                count: snapshot.data!.totalCount,
                services: snapshot.data!.dueSoonServices,
              );
            }
            return const CircularProgressIndicator();
          },
        ),
      ],
    );
  }
}
```

## Database Schema Reference

The equipment services are stored with the following database schema:

### Table: equipment_services
- **id**: VARCHAR(36) PRIMARY KEY (UUID)
- **shop_id**: VARCHAR(36) NOT NULL (FK to shops.id)
- **equipment_id**: VARCHAR(36) NOT NULL (FK to shop_vehicle.id)  
- **list_id**: VARCHAR(36) NOT NULL (FK to shop_lists.id)
- **description**: TEXT NOT NULL (1-500 chars)
- **service_type**: TEXT NOT NULL (free-form)
- **created_by**: VARCHAR(255) NOT NULL (FK to users.uid)
- **is_completed**: BOOLEAN NOT NULL DEFAULT FALSE
- **created_at**: TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
- **updated_at**: TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP  
- **service_date**: TIMESTAMP NULL
- **service_hours**: INTEGER NULL (non-negative)
- **completion_date**: TIMESTAMP NULL

### Foreign Key Relationships
- shop_id → shops.id (CASCADE DELETE)
- equipment_id → shop_vehicle.id (CASCADE DELETE)
- list_id → shop_lists.id (CASCADE DELETE)  
- created_by → users.uid (CASCADE DELETE)

### Performance Indexes
- idx_equipment_services_shop_id
- idx_equipment_services_equipment_id
- idx_equipment_services_service_date
- idx_equipment_services_created_by
- idx_equipment_services_service_type
- idx_equipment_services_is_completed
- idx_equipment_services_shop_date (composite)
- idx_equipment_services_equipment_date (composite)
- idx_equipment_services_shop_completed (composite)

## Security Considerations

### Data Validation
- All string inputs are validated for length and content
- Service hours must be non-negative integers
- Dates must be in valid ISO 8601 format
- UUIDs are validated for proper format

### Access Control  
- Users can only access services for shops they belong to
- Equipment and list access is validated before operations
- Service ownership is verified for update/delete operations
- Created by user ID is automatically set from JWT token

### Rate Limiting
Consider implementing client-side rate limiting for:
- Calendar date range queries (avoid excessive server load)
- Bulk service creation operations
- Frequent pagination requests

This documentation provides everything needed to implement the Equipment Services feature in the Flutter application, including comprehensive API coverage, implementation examples, and security considerations.