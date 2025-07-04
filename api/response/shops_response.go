package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"time"
)

type ShopResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Details   *string    `json:"details"`
	CreatedBy string     `json:"created_by"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	// Exclude password_hash for security
}

type ShopMemberResponse struct {
	ID       string     `json:"id"`
	ShopID   string     `json:"shop_id"`
	UserID   string     `json:"user_id"`
	Role     string     `json:"role"`
	JoinedAt *time.Time `json:"joined_at"`
}

type ShopInviteCodeResponse struct {
	ID          string     `json:"id"`
	ShopID      string     `json:"shop_id"`
	Code        string     `json:"code"`
	CreatedBy   string     `json:"created_by"`
	ExpiresAt   *time.Time `json:"expires_at"`
	MaxUses     *int32     `json:"max_uses"`
	CurrentUses *int32     `json:"current_uses"`
	IsActive    *bool      `json:"is_active"`
	CreatedAt   *time.Time `json:"created_at"`
}

type ShopMessageResponse struct {
	ID        string     `json:"id"`
	ShopID    string     `json:"shop_id"`
	UserID    string     `json:"user_id"`
	Message   string     `json:"message"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	IsEdited  *bool      `json:"is_edited"`
}

type ShopVehicleResponse struct {
	ID          string    `json:"id"`
	CreatorID   string    `json:"creator_id"`
	Niin        string    `json:"niin"`
	Admin       string    `json:"admin"`
	Model       string    `json:"model"`
	Serial      string    `json:"serial"`
	Uoc         string    `json:"uoc"`
	Mileage     int32     `json:"mileage"`
	Hours       int32     `json:"hours"`
	Comment     string    `json:"comment"`
	SaveTime    time.Time `json:"save_time"`
	LastUpdated time.Time `json:"last_updated"`
	ShopID      string    `json:"shop_id"`
}

type ShopVehicleNotificationResponse struct {
	ID          string    `json:"id"`
	ShopID      string    `json:"shop_id"`
	VehicleID   string    `json:"vehicle_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Completed   bool      `json:"completed"`
	SaveTime    time.Time `json:"save_time"`
	LastUpdated time.Time `json:"last_updated"`
}

type ShopNotificationItemResponse struct {
	ID             string    `json:"id"`
	ShopID         string    `json:"shop_id"`
	NotificationID string    `json:"notification_id"`
	Niin           string    `json:"niin"`
	Nomenclature   string    `json:"nomenclature"`
	Quantity       int32     `json:"quantity"`
	SaveTime       time.Time `json:"save_time"`
}

// Conversion functions to safely convert models to responses
func ToShopResponse(shop model.Shops) ShopResponse {
	return ShopResponse{
		ID:        shop.ID,
		Name:      shop.Name,
		Details:   shop.Details,
		CreatedBy: shop.CreatedBy,
		CreatedAt: shop.CreatedAt,
		UpdatedAt: shop.UpdatedAt,
	}
}

func ToShopMemberResponse(member model.ShopMembers) ShopMemberResponse {
	return ShopMemberResponse{
		ID:       member.ID,
		ShopID:   member.ShopID,
		UserID:   member.UserID,
		Role:     member.Role,
		JoinedAt: member.JoinedAt,
	}
}

func ToShopInviteCodeResponse(code model.ShopInviteCodes) ShopInviteCodeResponse {
	return ShopInviteCodeResponse{
		ID:        code.ID,
		ShopID:    code.ShopID,
		Code:      code.Code,
		CreatedBy: code.CreatedBy,
		IsActive:  code.IsActive,
		CreatedAt: code.CreatedAt,
	}
}

func ToShopMessageResponse(message model.ShopMessages) ShopMessageResponse {
	return ShopMessageResponse{
		ID:        message.ID,
		ShopID:    message.ShopID,
		UserID:    message.UserID,
		Message:   message.Message,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
		IsEdited:  message.IsEdited,
	}
}

func ToShopVehicleResponse(vehicle model.ShopVehicle) ShopVehicleResponse {
	return ShopVehicleResponse{
		ID:          vehicle.ID,
		CreatorID:   vehicle.CreatorID,
		Niin:        vehicle.Niin,
		Admin:       vehicle.Admin,
		Model:       vehicle.Model,
		Serial:      vehicle.Serial,
		Uoc:         vehicle.Uoc,
		Mileage:     vehicle.Mileage,
		Hours:       vehicle.Hours,
		Comment:     vehicle.Comment,
		SaveTime:    vehicle.SaveTime,
		LastUpdated: vehicle.LastUpdated,
		ShopID:      vehicle.ShopID,
	}
}

func ToShopVehicleNotificationResponse(notification model.ShopVehicleNotifications) ShopVehicleNotificationResponse {
	return ShopVehicleNotificationResponse{
		ID:          notification.ID,
		ShopID:      notification.ShopID,
		VehicleID:   notification.VehicleID,
		Title:       notification.Title,
		Description: notification.Description,
		Type:        notification.Type,
		Completed:   notification.Completed,
		SaveTime:    notification.SaveTime,
		LastUpdated: notification.LastUpdated,
	}
}

func ToShopNotificationItemResponse(item model.ShopNotificationItems) ShopNotificationItemResponse {
	return ShopNotificationItemResponse{
		ID:             item.ID,
		ShopID:         item.ShopID,
		NotificationID: item.NotificationID,
		Niin:           item.Niin,
		Nomenclature:   item.Nomenclature,
		Quantity:       item.Quantity,
		SaveTime:       item.SaveTime,
	}
}
