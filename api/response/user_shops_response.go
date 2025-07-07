package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type UserShopsResponse struct {
	User  *bootstrap.User `json:"user"`
	Shops []model.Shops   `json:"shops"`
}
