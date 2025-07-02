package request

type CreateShopRequest struct {
	Name         string  `json:"name" binding:"required"`
	Details      *string `json:"details"`
	PasswordHash *string `json:"password_hash"`
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
	Niin    string `json:"niin" binding:"required"`
	Model   string `json:"model" binding:"required"`
	Serial  string `json:"serial" binding:"required"`
	Uoc     string `json:"uoc"`
	Mileage int32  `json:"mileage"`
	Hours   int32  `json:"hours"`
	Comment string `json:"comment"`
}

type UpdateShopVehicleRequest struct {
	VehicleID string `json:"vehicle_id" binding:"required"`
	Model     string `json:"model" binding:"required"`
	Serial    string `json:"serial" binding:"required"`
	Uoc       string `json:"uoc"`
	Mileage   int32  `json:"mileage"`
	Hours     int32  `json:"hours"`
	Comment   string `json:"comment"`
}

type CreateVehicleNotificationRequest struct {
	VehicleID   string `json:"vehicle_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Type        string `json:"type" binding:"required"` // M1, PM, MW
}

type UpdateVehicleNotificationRequest struct {
	NotificationID string `json:"notification_id" binding:"required"`
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
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
