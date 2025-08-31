package response

import "time"

// EquipmentServiceResponse represents a single equipment service in responses
type EquipmentServiceResponse struct {
	ID                string     `json:"id"`
	ShopID            string     `json:"shop_id"`
	EquipmentID       string     `json:"equipment_id"`
	ListID            string     `json:"list_id"`
	Description       string     `json:"description"`
	ServiceType       string     `json:"service_type"`
	CreatedBy         string     `json:"created_by"`
	CreatedByUsername string     `json:"created_by_username"`  // Dynamically populated from users table
	IsCompleted       bool       `json:"is_completed"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ServiceDate       *time.Time `json:"service_date"`
	ServiceHours      *int32     `json:"service_hours"`
	CompletionDate    *time.Time `json:"completion_date"`
}

// PaginatedEquipmentServicesResponse represents a paginated list of equipment services
type PaginatedEquipmentServicesResponse struct {
	Services   []EquipmentServiceResponse `json:"services"`
	TotalCount int64                      `json:"total_count"`
	HasMore    bool                       `json:"has_more"`
}

// CalendarServicesResponse represents services within a date range for calendar view
type CalendarServicesResponse struct {
	DateRange struct {
		StartDate time.Time `json:"start_date"`
		EndDate   time.Time `json:"end_date"`
	} `json:"date_range"`
	Services   []EquipmentServiceResponse `json:"services"`
	TotalCount int64                      `json:"total_count"`
}

// OverdueServiceResponse represents an overdue equipment service with additional metadata
type OverdueServiceResponse struct {
	EquipmentServiceResponse
	DaysOverdue int `json:"days_overdue"`
}

// DueSoonServiceResponse represents an equipment service due soon with additional metadata
type DueSoonServiceResponse struct {
	EquipmentServiceResponse  
	DaysUntilDue int `json:"days_until_due"`
}

// OverdueServicesResponse represents a list of overdue services
type OverdueServicesResponse struct {
	OverdueServices []OverdueServiceResponse `json:"overdue_services"`
	TotalCount      int64                    `json:"total_count"`
}

// DueSoonServicesResponse represents a list of services due soon
type DueSoonServicesResponse struct {
	DueSoonServices []DueSoonServiceResponse `json:"due_soon_services"`
	TotalCount      int64                    `json:"total_count"`
}