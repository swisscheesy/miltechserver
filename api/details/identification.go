package details

import "miltechserver/prisma/db"

type Identification struct {
	FlisManagementId    db.FlisManagementIDModel      `json:"flis_management_id"`
	ColloquialNames     []db.ColloquialNameModel      `json:"colloquial_names"`
	FlisStandardization []db.FlisStandardizationModel `json:"flis_standardization"`
	FlisCancelledNiin   []db.FlisCancelledNiinModel   `json:"flis_cancelled_niin"`
}
