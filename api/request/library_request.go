package request

// Currently no request validation needed for Phase 1 GET endpoints
// Future: Add pagination, filtering parameters for document listing
// Example:
// type ListDocumentsRequest struct {
//     VehicleName string `uri:"vehicle" binding:"required"`
//     Page        int    `form:"page" binding:"omitempty,min=1"`
//     PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
// }
