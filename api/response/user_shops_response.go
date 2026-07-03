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
	Pagination *PaginationMetadata  `json:"pagination,omitempty"`
	NextCursor *string              `json:"next_cursor,omitempty"`
}

// ShopDetailResponse includes shop data with calculated statistics
// Used by GetShopByID endpoint to provide comprehensive shop information
type ShopDetailResponse struct {
	model.Shops         // Embed base shop model for backwards compatibility
	TotalMessages int64 `json:"total_messages"` // Total count of messages in shop
	MemberCount   int64 `json:"member_count"`   // Total count of shop members
	VehicleCount  int64 `json:"vehicle_count"`  // Total count of shop vehicles
	IsAdmin       bool  `json:"is_admin"`       // Whether current user is admin
}

type ShopEquipmentOverviewResponse struct {
	Shops []ShopEquipmentOverview `json:"shops"`
}

type ShopEquipmentOverview struct {
	ID             string                 `sql:"primary_key" alias:"shops.id" json:"id"`
	Name           string                 `alias:"shops.name" json:"name"`
	Details        *string                `alias:"shops.details" json:"details"`
	Role           string                 `alias:"shop_members.role" json:"role"`
	EquipmentCount int                    `json:"equipment_count"`
	Equipment      []ShopEquipmentSummary `alias:"shop_vehicle" json:"equipment"`
}

type ShopEquipmentSummary struct {
	ID     string `sql:"primary_key" alias:"id" json:"id"`
	Admin  string `alias:"admin" json:"admin"`
	Model  string `alias:"model" json:"model"`
	Serial string `alias:"serial" json:"serial"`
	Niin   string `alias:"niin" json:"niin"`
}

type ShopListsWithItemsResponse struct {
	Lists []ShopListWithItems `json:"lists"`
}

type ShopListWithItems struct {
	ShopListWithUsername
	Items []ShopListItemWithUsername `json:"items"`
}

type ShopAggregateSettings struct {
	AdminOnlyLists bool `json:"admin_only_lists"`
}

type ShopAggregateCounts struct {
	Members           int64 `json:"members"`
	Vehicles          int64 `json:"vehicles"`
	Lists             int64 `json:"lists"`
	Messages          int64 `json:"messages"`
	Notifications     int64 `json:"notifications"`
	NotificationItems int64 `json:"notification_items,omitempty"`
	OpenServices      int64 `json:"open_services"`
	Services          int64 `json:"services,omitempty"`
	RecentChanges     int64 `json:"recent_changes,omitempty"`
}

type ShopSnapshotResponse struct {
	Shop          ShopSnapshotSummary              `json:"shop"`
	Vehicles      []model.ShopVehicle              `json:"vehicles"`
	Lists         []ShopListWithItems              `json:"lists"`
	Notifications []VehicleNotificationWithItems   `json:"notifications"`
	Messages      []model.ShopMessages             `json:"messages"`
	Services      []EquipmentServiceResponse       `json:"services"`
	RecentChanges []NotificationChangeWithUsername `json:"recent_changes"`
}

type ShopSnapshotSummary struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Details  *string               `json:"details"`
	Role     string                `json:"role"`
	IsAdmin  bool                  `json:"is_admin"`
	Settings ShopAggregateSettings `json:"settings"`
	Counts   ShopAggregateCounts   `json:"counts"`
}

type VehicleMaintenanceSnapshotResponse struct {
	Vehicle       model.ShopVehicle                `json:"vehicle"`
	Notifications []VehicleNotificationWithItems   `json:"notifications"`
	RecentChanges []NotificationChangeWithUsername `json:"recent_changes"`
	Services      []EquipmentServiceResponse       `json:"services"`
	Counts        ShopAggregateCounts              `json:"counts"`
}

type ShopsBootstrapResponse struct {
	Shops []ShopBootstrapSummary `json:"shops"`
}

type ShopBootstrapSummary struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Details   *string                `json:"details"`
	Role      string                 `json:"role"`
	IsAdmin   bool                   `json:"is_admin"`
	Settings  ShopAggregateSettings  `json:"settings"`
	Counts    ShopAggregateCounts    `json:"counts"`
	Equipment []ShopEquipmentSummary `json:"equipment"`
}
