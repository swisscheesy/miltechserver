package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
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
