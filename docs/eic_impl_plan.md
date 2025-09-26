# EIC Implementation Plan

## Overview

This document outlines the implementation plan for the EIC (Equipment Identification Code) feature that provides query endpoints for military equipment catalog data. The implementation follows the established codebase patterns and provides read-only access to equipment information stored in the EIC database table.

## Database Analysis

### EIC Table Structure
The `eic` table contains comprehensive military equipment data with the following key fields:

**Primary Keys:**
- `niin` (National Item Identification Number) - Required
- `mrc` (Material Readiness Code) - Required
- `uoeic` (Unit of Equipment Identification Code) - Required

**Query Fields:**
- `lin` (Line Item Number) - Optional, used for equipment tracking
- `fsc` (Federal Supply Class) - Optional, categorization code
- `niin` - Primary equipment identifier
- `nomen` - Equipment nomenclature/name
- `model` - Equipment model information

**Additional Data:**
- Equipment characteristics (eic, eicc, ecc)
- Publication references (publvl1-7, pubno1-7, etc.)
- Equipment categories and classifications
- Status and management information

## Implementation Architecture

The EIC feature will follow the established 4-layer architecture pattern:

1. **Repository Layer** (`eic_repository.go` + `eic_repository_impl.go`) - Database queries using go-jet
2. **Service Layer** (`eic_service.go` + `eic_service_impl.go`) - Business logic and data processing
3. **Controller Layer** (`eic_controller.go`) - HTTP request handling
4. **Route Layer** (`eic_route.go`) - Endpoint registration and middleware

## Required Endpoints

### 1. Query by NIIN
- **Endpoint:** `GET /api/eic/niin/{niin}`
- **Purpose:** Retrieve EIC records by National Item Identification Number
- **Response:** Array of matching EIC records
- **Parameters:** NIIN (path parameter)

### 2. Query by LIN
- **Endpoint:** `GET /api/eic/lin/{lin}`
- **Purpose:** Retrieve EIC records by Line Item Number
- **Response:** Array of matching EIC records
- **Parameters:** LIN (path parameter)

### 3. Query by FSC (Paginated)
- **Endpoint:** `GET /api/eic/fsc/{fsc}`
- **Purpose:** Retrieve EIC records by Federal Supply Class
- **Response:** Paginated array of matching EIC records (40 per page)
- **Parameters:**
  - fsc (path parameter)
  - page (query parameter, optional, default: 1)

### 4. General Paginated Query
- **Endpoint:** `GET /api/eic/items`
- **Purpose:** Retrieve all EIC records with pagination
- **Response:** Paginated array of EIC records (40 per page)
- **Parameters:**
  - page (query parameter, optional, default: 1)
  - search (query parameter, optional) - general search across all text fields

## Implementation Details

### Repository Layer (`eic_repository.go` + `eic_repository_impl.go`)

**Interface Definition:**
```go
type EICRepository interface {
    GetByNIIN(niin string) ([]model.Eic, error)
    GetByLIN(lin string) ([]model.Eic, error)
    GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
    GetAllPaginated(page int, search string) (response.EICPageResponse, error)
}
```

**Implementation Requirements:**
- Use go-jet for type-safe SQL queries
- Implement proper error handling
- Use the existing `model.Eic` struct from go-jet generated models
- Include pagination logic with 40 items per page for `GetAllPaginated` and `GetByFSCPaginated`
- Implement general search across all text fields (no specific field targeting)
- FSC queries are now paginated to handle large result sets efficiently

### Service Layer (`eic_service.go` + `eic_service_impl.go`)

**Interface Definition:**
```go
type EICService interface {
    LookupByNIIN(niin string) ([]model.Eic, error)
    LookupByLIN(lin string) ([]model.Eic, error)
    LookupByFSCPaginated(fsc string, page int) (response.EICPageResponse, error)
    LookupAllPaginated(page int, search string) (response.EICPageResponse, error)
}
```

**Implementation Requirements:**
- Validate input parameters (trim whitespace, validate format)
- Transform repository responses to service responses
- Handle business logic for pagination (for `LookupAllPaginated` and `LookupByFSCPaginated`)
- Implement proper error handling and logging
- For non-paginated methods (NIIN, LIN): Return arrays directly for controller to wrap in `EICSearchResponse`
- For paginated methods (FSC, General): Return structured `EICPageResponse` with all pagination metadata

### Controller Layer (`eic_controller.go`)

**Structure:**
```go
type EICController struct {
    EICService service.EICService
}
```

**Methods:**
- `LookupByNIIN(c *gin.Context)`
- `LookupByLIN(c *gin.Context)`
- `LookupByFSCPaginated(c *gin.Context)`
- `LookupAllPaginated(c *gin.Context)`

**Implementation Requirements:**
- Follow existing controller patterns from `item_lookup_controller.go`
- Use standard HTTP status codes (200, 404, 500)
- Implement proper parameter validation
- Use clean response patterns based on endpoint type:
  - **NIIN, LIN endpoints**: Create `EICSearchResponse{Count: len(data), Items: data}` wrapped in `StandardResponse`
  - **FSC, General paginated endpoints**: Use service-returned `EICPageResponse` directly in `StandardResponse.Data`
- Handle errors consistently with existing controllers
- No unnecessary pagination metadata on non-paginated endpoints

### Response Structures

**Create new response file:** `api/response/eic_response.go`

Using improved response structures with clear separation between paginated and non-paginated responses:

```go
// For paginated responses (GET /api/eic/items and GET /api/eic/fsc/{fsc})
type EICPageResponse struct {
    Items      []model.Eic `json:"items"`
    Count      int         `json:"count"`
    Page       int         `json:"page"`
    TotalPages int         `json:"total_pages"`
    IsLastPage bool        `json:"is_last_page"`
}

// For non-paginated responses (NIIN, LIN queries only)
type EICSearchResponse struct {
    Count int         `json:"count"`
    Items []model.Eic `json:"items"`
    // No pagination fields since these endpoints don't paginate
}
```

**Response Format Standards:**
- NIIN, LIN queries: Use `EICSearchResponse` wrapped in `StandardResponse`
- FSC, General paginated queries: Use `EICPageResponse` wrapped in `StandardResponse`
- Clean separation eliminates confusing pagination metadata on non-paginated endpoints

**Design Rationale:**
This approach improves upon existing patterns by:
1. **API Clarity**: Clients immediately understand which endpoints support pagination
2. **Reduced Confusion**: No misleading `page: 1, total_pages: 1, is_last_page: true` on non-paginated responses
3. **Maintainability**: Each structure serves a specific purpose without unnecessary fields
4. **Future-Proof**: Easy to add fields specific to paginated vs non-paginated responses

### Route Layer (`eic_route.go`)

**Route Registration:**
```go
func NewEICRouter(db *sql.DB, group *gin.RouterGroup) {
    eicRepo := repository.NewEICRepositoryImpl(db)
    ec := &controller.EICController{
        EICService: service.NewEICServiceImpl(eicRepo),
    }

    group.GET("/eic/niin/:niin", ec.LookupByNIIN)
    group.GET("/eic/lin/:lin", ec.LookupByLIN)
    group.GET("/eic/fsc/:fsc", ec.LookupByFSCPaginated)
    group.GET("/eic/items", ec.LookupAllPaginated)
}
```

## Implementation Steps

### Phase 1: Repository Layer
1. Implement `EICRepository` interface in `eic_repository.go`
2. Implement `EICRepositoryImpl` struct in `eic_repository_impl.go`
3. Create `NewEICRepositoryImpl()` constructor
4. Implement database queries using go-jet:
   - `GetByNIIN()` - Simple WHERE niin = ? (returns all matches)
   - `GetByLIN()` - Simple WHERE lin = ? (returns all matches)
   - `GetByFSCPaginated()` - WHERE fsc = ? with pagination (40 per page)
   - `GetAllPaginated()` - General query with optional search and pagination (40 per page)
5. Add proper error handling and logging

### Phase 2: Service Layer
1. Implement `EICService` interface in `eic_service.go`
2. Implement `EICServiceImpl` struct in `eic_service_impl.go`
3. Create `NewEICServiceImpl()` constructor
4. Add input validation and sanitization
5. Transform repository responses to service responses
6. Implement pagination logic

### Phase 3: Response Structures
1. Create `eic_response.go` with required response structs
2. Follow existing patterns from `lin_page_response.go`
3. Ensure JSON serialization works correctly

### Phase 4: Controller Layer
1. Implement `EICController` struct and methods in `eic_controller.go`
2. Add parameter extraction and validation
3. Implement error handling following existing patterns
4. Create proper HTTP responses using standard patterns

### Phase 5: Route Registration
1. Implement `NewEICRouter()` function in `eic_route.go`
2. Register all endpoints with proper HTTP methods
3. Router integration will be handled separately (not part of this implementation)

### Phase 6: Testing and Validation
1. Test all endpoints manually
2. Verify error handling and edge cases
3. Ensure proper HTTP status codes
4. Validate response format consistency with existing endpoints

## Security Considerations

- **No Authentication Required**: Per requirements, endpoints are public
- **Input Validation**: Validate all parameters to prevent SQL injection
- **Rate Limiting**: Consider implementing rate limiting in future iterations
- **Data Sanitization**: Sanitize output to prevent XSS attacks

## Performance Considerations

- **Database Indexing**: Verify indexes exist on `niin`, `lin`, and `fsc` columns
- **Pagination**: Limit results to 40 items per page to manage response size
- **Caching**: Consider implementing caching for frequently accessed data
- **Query Optimization**: Use efficient SQL queries with proper WHERE clauses

## Testing Strategy

### Manual Testing
1. Test each endpoint with valid parameters
2. Test pagination functionality
3. Test error scenarios (invalid parameters, not found)
4. Verify response format consistency

### Edge Cases
1. Empty result sets
2. Invalid pagination parameters (negative numbers, zero) - for FSC and general paginated endpoints
3. Non-existent NIINs, LINs, FSCs
4. SQL injection attempts
5. Large result sets for FSC queries (now paginated to handle efficiently)
6. General search with very broad terms returning large datasets

## Error Handling

Follow existing patterns:
- **400 Bad Request**: Invalid parameters
- **404 Not Found**: No items found (use `response.NoItemFoundResponseMessage()`)
- **500 Internal Server Error**: Database or system errors (use `response.InternalErrorResponseMessage()`)

## File Structure

```
api/
├── controller/
│   └── eic_controller.go           # HTTP request handling
├── repository/
│   ├── eic_repository.go           # Repository interface
│   └── eic_repository_impl.go      # Repository implementation
├── response/
│   └── eic_response.go             # Response structures
├── route/
│   └── eic_route.go                # Route registration
└── service/
    ├── eic_service.go              # Service interface
    └── eic_service_impl.go         # Service implementation
```

## Future Enhancements

1. **Authentication**: Add API key or JWT authentication
2. **Advanced Search**: Full-text search across multiple fields
3. **Filtering**: Additional filter parameters (status, category, etc.)
4. **Caching**: Redis caching for improved performance
5. **Metrics**: Add monitoring and metrics collection
6. **Documentation**: OpenAPI/Swagger documentation
7. **Bulk Operations**: Batch query capabilities
8. **Export Features**: CSV/Excel export functionality

## Dependencies

- **Database**: PostgreSQL with existing `eic` table
- **ORM**: go-jet for type-safe database operations
- **Web Framework**: Gin for HTTP routing
- **Existing Models**: `.gen/miltech_ng/public/model/eic.go`

## Completion Criteria

- [ ] All four endpoints implemented and tested
- [ ] Pagination working correctly (40 items per page for FSC and general queries)
- [ ] Error handling following existing patterns
- [ ] Clean response format with proper separation of paginated vs non-paginated responses
- [ ] No authentication required (public endpoints)
- [ ] Database queries optimized and indexed
- [ ] NIIN, LIN queries return all results (non-paginated with clean `EICSearchResponse`)
- [ ] FSC queries use pagination (paginated with `EICPageResponse` for large result sets)
- [ ] General search implemented across all text fields
- [ ] Router integration ready (but not implemented in this phase)
- [ ] Interface/implementation pattern properly separated
- [ ] Response structures eliminate unnecessary pagination metadata on non-paginated endpoints

## Notes

- The EIC table appears to be a comprehensive military equipment catalog
- Primary key consists of three fields: `niin`, `mrc`, `uoeic`
- Most fields are optional (nullable) except primary key components
- The existing EIC files are empty stubs ready for implementation
- Implementation should follow established patterns from `item_lookup_*` files
- Files have been renamed from `gcss_*` to `eic_*` with interface/implementation separation

## Requirements Clarifications (Updated)

1. **Search Functionality**: General search parameter searches across ALL text fields, not specific fields
2. **Pagination**: The `GET /api/eic/fsc/{fsc}` and `GET /api/eic/items` endpoints use pagination (40 per page)
3. **Response Format**: Improved clean separation - non-paginated responses (NIIN, LIN) use `EICSearchResponse` (count + items only), paginated responses (FSC, General) use `EICPageResponse` (includes pagination metadata)
4. **Router Integration**: Implementation stops at route registration; main router integration handled separately
5. **File Naming**: All files use `eic_*` naming convention instead of `gcss_*`
6. **Architecture**: Uses interface/implementation pattern with separate `*_impl.go` files
7. **API Design**: Eliminates confusing pagination metadata on endpoints that don't paginate