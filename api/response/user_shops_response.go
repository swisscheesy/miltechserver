package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
	"time"
)

type ShopWithStats struct {
	Shop         model.Shops `json:"shop"`
	MemberCount  int64       `json:"member_count"`
	VehicleCount int64       `json:"vehicle_count"`
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
