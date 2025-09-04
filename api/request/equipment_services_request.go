package request

import "time"

// CreateEquipmentServiceRequest represents the request to create a new equipment service
type CreateEquipmentServiceRequest struct {
	EquipmentID string `json:"equipment_id" binding:"required"`
	//ListID         string     `json:"list_id" binding:"required"`
	ListID         string     `json:"list_id"`
	Description    string     `json:"description" binding:"required,min=1,max=500"`
	ServiceType    string     `json:"service_type" binding:"required"`
	IsCompleted    bool       `json:"is_completed"`
	ServiceDate    *time.Time `json:"service_date"`
	ServiceHours   *int32     `json:"service_hours" binding:"omitempty,min=0"`
	CompletionDate *time.Time `json:"completion_date"`
}

// UpdateEquipmentServiceRequest represents the request to update an existing equipment service
type UpdateEquipmentServiceRequest struct {
	ServiceID      string     `json:"service_id" binding:"required"`
	Description    string     `json:"description" binding:"required,min=1,max=500"`
	ServiceType    string     `json:"service_type" binding:"required"`
	IsCompleted    bool       `json:"is_completed"`
	ServiceDate    *time.Time `json:"service_date"`
	ServiceHours   *int32     `json:"service_hours" binding:"omitempty,min=0"`
	CompletionDate *time.Time `json:"completion_date"`
}

// GetEquipmentServicesRequest represents the query parameters for getting equipment services
type GetEquipmentServicesRequest struct {
	EquipmentID *string `form:"equipment_id"`
	StartDate   *string `form:"start_date"`   // ISO 8601 format
	EndDate     *string `form:"end_date"`     // ISO 8601 format
	ServiceType *string `form:"service_type"` // Free-form service type filter
	IsCompleted *bool   `form:"is_completed"` // Filter by completion status
	Status      *string `form:"status"`       // overdue, due_soon, scheduled, completed
	Limit       int     `form:"limit,default=100" binding:"omitempty,min=1,max=500"`
	Offset      int     `form:"offset,default=0" binding:"omitempty,min=0"`
}

// GetCalendarServicesRequest represents the request to get services in a date range
type GetCalendarServicesRequest struct {
	StartDate   string  `form:"start_date" binding:"required"` // ISO 8601 format
	EndDate     string  `form:"end_date" binding:"required"`   // ISO 8601 format
	EquipmentID *string `form:"equipment_id"`
}

// GetDueSoonServicesRequest represents the request to get services due soon
type GetDueSoonServicesRequest struct {
	DaysAhead   int     `form:"days_ahead,default=7" binding:"omitempty,min=1,max=30"`
	EquipmentID *string `form:"equipment_id"`
	Limit       int     `form:"limit,default=50" binding:"omitempty,min=1,max=200"`
}

// GetOverdueServicesRequest represents the request to get overdue services
type GetOverdueServicesRequest struct {
	EquipmentID *string `form:"equipment_id"`
	Limit       int     `form:"limit,default=50" binding:"omitempty,min=1,max=200"`
}

// CompleteEquipmentServiceRequest represents the request to mark a service as completed
type CompleteEquipmentServiceRequest struct {
	CompletionDate *time.Time `json:"completion_date"` // Optional: defaults to current time
}
