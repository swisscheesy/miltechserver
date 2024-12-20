package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Management struct {
	FLisManagement        []model.FlisManagement        `json:"flis_management"`
	FlisPhrase            []model.FlisPhrase            `json:"flis_phrase"`
	ComponentEndItem      []model.ComponentEndItem      `json:"component_end_item"`
	ArmyManagement        []model.ArmyManagement        `json:"army_management"`
	AirForceManagement    model.AirForceManagement      `json:"air_force_management"`
	MarineCorpsManagement []model.MarineCorpsManagement `json:"marine_corps_management"`
	NavyManagement        model.NavyManagement          `json:"navy_management"`
	FaaManagement         []model.FaaManagement         `json:"faa_management"`
}
