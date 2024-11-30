package details

import "miltechserver/prisma/db"

type Management struct {
	FLisManagement        db.FlisManagementModel        `json:"flis_management"`
	FlisPhrase            db.FlisPhraseModel            `json:"flis_phrase"`
	ComponentEndItem      db.ComponentEndItemModel      `json:"component_end_item"`
	ArmyManagement        db.ArmyManagementModel        `json:"army_management"`
	AirForceManagement    db.AirForceManagementModel    `json:"air_force_management"`
	MarineCorpsManagement db.MarineCorpsManagementModel `json:"marine_corps_management"`
	NavyManagement        db.NavyManagementModel        `json:"navy_management"`
	FaaManagement         db.FaaManagementModel         `json:"faa_management"`
}
