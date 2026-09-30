package request

import (
	"bytes"
	"encoding/json"
)

type NullableStringField struct {
	Set   bool
	Value *string
}

func (field *NullableStringField) UnmarshalJSON(data []byte) error {
	field.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		field.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	field.Value = &value
	return nil
}

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
	ShopID   string  `json:"shop_id" binding:"required"`
	Message  string  `json:"message" binding:"required"`
	ParentID *string `json:"parent_id"`
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

type AdjustShopVehicleUsageRequest struct {
	Operation         string `json:"operation"`
	MileageAdjustment *int32 `json:"mileage_adjustment"`
	HoursAdjustment   *int32 `json:"hours_adjustment"`
}

type CreateVehicleNotificationRequest struct {
	ShopID           string  `json:"shop_id" binding:"required"`
	VehicleID        string  `json:"vehicle_id" binding:"required"`
	Title            string  `json:"title" binding:"required"`
	Description      string  `json:"description"`
	Type             string  `json:"type" binding:"required"` // M1, PM, MW
	Completed        bool    `json:"completed"`
	AttachedShopList *string `json:"attached_shop_list"`
}

type UpdateVehicleNotificationRequest struct {
	NotificationID   string              `json:"notification_id" binding:"required"`
	Title            string              `json:"title" binding:"required"`
	Description      string              `json:"description"`
	Type             string              `json:"type" binding:"required"`
	Completed        bool                `json:"completed"`
	AttachedShopList NullableStringField `json:"attached_shop_list"`
}

type AddNotificationItemRequest struct {
	NotificationID string  `json:"notification_id" binding:"required"`
	Niin           string  `json:"niin" binding:"required"`
	Nomenclature   string  `json:"nomenclature" binding:"required"`
	Quantity       int32   `json:"quantity" binding:"required"`
	Nickname       *string `json:"nickname"`
	UnitOfMeasure  *string `json:"unit_of_measure"`
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
	ListID string               `json:"list_id" binding:"required"`
	Items  []AddListItemRequest `json:"items" binding:"required"`
}

type RemoveListItemBatchRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required"`
}

type GetShopMessagesPaginatedRequest struct {
	Page     int     `form:"page,default=1" binding:"omitempty,min=1"`
	Limit    int     `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	BeforeID *string `form:"before_id" binding:"omitempty"`
	AfterID  *string `form:"after_id" binding:"omitempty"`
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

// NotificationSaveRequest replaces the complete direct-item set. Linked-list
// items are represented solely by Attachment and never copied into Items.
type NotificationSaveRequest struct {
	OperationID    string                     `json:"operation_id"`
	ShopID         string                     `json:"shop_id"`
	VehicleID      string                     `json:"vehicle_id"`
	NotificationID *string                    `json:"notification_id"`
	Details        NotificationSaveDetails    `json:"details"`
	Attachment     NotificationSaveAttachment `json:"attachment"`
	Items          []NotificationSaveItem     `json:"items"`
}
type NotificationSaveDetails struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	IsCompleted bool   `json:"is_completed"`
}
type NotificationSaveAttachment struct {
	Intent string  `json:"intent"`
	ListID *string `json:"list_id"`
}
type NotificationSaveItem struct {
	ID           string `json:"id"`
	Niin         string `json:"niin"`
	Nomenclature string `json:"nomenclature"`
	Quantity     int32  `json:"quantity"`
	// Released clients omit these. Nil means "leave the stored value
	// unchanged", and omitempty keeps their operation fingerprints identical
	// to the pre-column contract so retries spanning a deploy still replay.
	Nickname      *string `json:"nickname,omitempty"`
	UnitOfMeasure *string `json:"unit_of_measure,omitempty"`
}
