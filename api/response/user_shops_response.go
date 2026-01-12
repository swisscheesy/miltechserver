package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"time"
)

type ShopWithStats struct {
	Shop             model.Shops `json:"shop"`
	MemberCount      int64       `json:"member_count"`
	VehicleCount     int64       `json:"vehicle_count"`
	IsAdmin          bool        `json:"is_admin"`
	IsListsAdminOnly bool        `json:"is_lists_admin_only"`
}

type UserShopsResponse struct {
	User  *bootstrap.User `json:"user"`
	Shops []ShopWithStats `json:"shops"`
}

type ShopMemberWithUsername struct {
	ID       string     `json:"id"`
	ShopID   string     `json:"shop_id"`
	UserID   string     `json:"user_id"`
	Role     string     `json:"role"`
	JoinedAt *time.Time `json:"joined_at"`
	Username *string    `json:"username"`
}

type ShopListWithUsername struct {
	ID                string     `json:"id"`
	ShopID            string     `json:"shop_id"`
	CreatedBy         string     `json:"created_by"`
	CreatedByUsername *string    `json:"created_by_username"`
	Description       string     `json:"description"`
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
}

type ShopListItemWithUsername struct {
	ID              string     `json:"id"`
	ListID          string     `json:"list_id"`
	Niin            string     `json:"niin"`
	Nomenclature    string     `json:"nomenclature"`
	Quantity        int32      `json:"quantity"`
	AddedBy         string     `json:"added_by"`
	AddedByUsername *string    `json:"added_by_username"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	Nickname        *string    `json:"nickname"`
	UnitOfMeasure   *string    `json:"unit_of_measure"`
}

type PaginationMetadata struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type PaginatedShopMessagesResponse struct {
	Messages   []model.ShopMessages `json:"messages"`
	Pagination PaginationMetadata   `json:"pagination"`
}

// ShopDetailResponse includes shop data with calculated statistics
// Used by GetShopByID endpoint to provide comprehensive shop information
type ShopDetailResponse struct {
	model.Shops              // Embed base shop model for backwards compatibility
	TotalMessages int64 `json:"total_messages"` // Total count of messages in shop
	MemberCount   int64 `json:"member_count"`   // Total count of shop members
	VehicleCount  int64 `json:"vehicle_count"`  // Total count of shop vehicles
	IsAdmin       bool  `json:"is_admin"`       // Whether current user is admin
}
