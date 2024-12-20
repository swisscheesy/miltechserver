package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Identification struct {
	FlisManagementId    model.FlisManagementID      `json:"flis_management_id"`
	ColloquialName      []model.ColloquialName      `json:"colloquial_names"`
	FlisStandardization []model.FlisStandardization `json:"flis_standardization"`
	FlisCancelledNiin   []model.FlisCancelledNiin   `json:"flis_cancelled_niin"`
}
