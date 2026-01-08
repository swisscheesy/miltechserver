package request

type CreateShopRequest struct {
	Name           string  `json:"name" binding:"required"`
	Details        *string `json:"details"`
	PasswordHash   *string `json:"password_hash"`
	AdminOnlyLists *bool   `json:"admin_only_lists"`
}

type JoinShopRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

type GenerateInviteCodeRequest struct {
	ShopID    string  `json:"shop_id" binding:"required"`
	MaxUses   *int32  `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"` // ISO format date string
}

type RemoveMemberRequest struct {
	ShopID       string `json:"shop_id" binding:"required"`
	TargetUserID string `json:"target_user_id" binding:"required"`
}

type PromoteMemberRequest struct {
	ShopID       string `json:"shop_id" binding:"required"`
	TargetUserID string `json:"target_user_id" binding:"required"`
}

type CreateShopMessageRequest struct {
	ShopID  string `json:"shop_id" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type UpdateShopMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

type CreateShopVehicleRequest struct {
	ShopID  string `json:"shop_id" binding:"required"`
	Niin    string `json:"niin"`
	Admin   string `json:"admin" binding:"required"`
	Model   string `json:"model"`
	Serial  string `json:"serial"`
	Uoc     string `json:"uoc"`
	Mileage int32  `json:"mileage"`
	Hours   int32  `json:"hours"`
	Comment string `json:"comment"`
}

type UpdateShopVehicleRequest struct {
	VehicleID      string `json:"vehicle_id" binding:"required"`
	Admin          string `json:"admin" binding:"required"`
	Niin           string `json:"niin"`
	Model          string `json:"model"`
	Serial         string `json:"serial"`
	Uoc            string `json:"uoc"`
	Mileage        int32  `json:"mileage"`
	Hours          int32  `json:"hours"`
	Comment        string `json:"comment"`
	TrackedMileage *int32 `json:"tracked_mileage"`
	TrackedHours   *int32 `json:"tracked_hours"`
}

type CreateVehicleNotificationRequest struct {
	ShopID      string `json:"shop_id" binding:"required"`
	VehicleID   string `json:"vehicle_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type" binding:"required"` // M1, PM, MW
}

type UpdateVehicleNotificationRequest struct {
	NotificationID string `json:"notification_id" binding:"required"`
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description"`
	Type           string `json:"type" binding:"required"`
	Completed      bool   `json:"completed"`
}

type AddNotificationItemRequest struct {
	NotificationID string `json:"notification_id" binding:"required"`
	Niin           string `json:"niin" binding:"required"`
	Nomenclature   string `json:"nomenclature" binding:"required"`
	Quantity       int32  `json:"quantity" binding:"required"`
}

type AddNotificationItemListRequest struct {
	NotificationID string                       `json:"notification_id" binding:"required"`
	Items          []AddNotificationItemRequest `json:"items" binding:"required"`
}

type RemoveNotificationItemListRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}

type UpdateShopRequest struct {
	Name    string  `json:"name" binding:"required"`
	Details *string `json:"details"`
}

// Shop List Operations

type CreateShopListRequest struct {
	ShopID      string `json:"shop_id" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UpdateShopListRequest struct {
	ListID      string `json:"list_id" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// Shop List Item Operations

type AddListItemRequest struct {
	ListID        string  `json:"list_id" binding:"required"`
	Niin          string  `json:"niin" binding:"required"`
	Nomenclature  string  `json:"nomenclature" binding:"required"`
	Quantity      int32   `json:"quantity" binding:"required"`
	Nickname      *string `json:"nickname"`
	UnitOfMeasure *string `json:"unit_of_measure"`
}

type UpdateListItemRequest struct {
	ItemID        string  `json:"item_id" binding:"required"`
	Niin          string  `json:"niin" binding:"required"`
	Nomenclature  string  `json:"nomenclature" binding:"required"`
	Quantity      int32   `json:"quantity" binding:"required"`
	Nickname      *string `json:"nickname"`
	UnitOfMeasure *string `json:"unit_of_measure"`
}

type DeleteShopListRequest struct {
	ListID string `json:"list_id" binding:"required"`
}

type RemoveListItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

type AddListItemBatchRequest struct {
	ListID string                `json:"list_id" binding:"required"`
	Items  []AddListItemRequest  `json:"items" binding:"required"`
}

type RemoveListItemBatchRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}

type GetShopMessagesPaginatedRequest struct {
	Page  int `form:"page,default=1" binding:"omitempty,min=1"`
	Limit int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
}

type UpdateAdminOnlyListsRequest struct {
	AdminOnlyLists bool `json:"admin_only_lists" binding:"required"`
}

// Unified Shop Settings

// ShopSettings represents all settings for a shop
type ShopSettings struct {
	AdminOnlyLists bool `json:"admin_only_lists"`
	// Future settings will be added here
}

// UpdateShopSettingsRequest is used for partial updates to shop settings
// All fields are optional pointers to support partial updates
type UpdateShopSettingsRequest struct {
	AdminOnlyLists *bool `json:"admin_only_lists,omitempty"`
	// Future settings will be added here as optional pointers
}
