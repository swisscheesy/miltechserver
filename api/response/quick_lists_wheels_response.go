package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type QuickListsWheelsResponse struct {
	Wheels []model.QuickListWheelTires `json:"wheels"`
	Count  int                         `json:"count"`
}
