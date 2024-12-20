package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Packaging struct {
	FlisPackaging1   []model.FlisPackaging1 `json:"flis_packaging_1"`
	FlisPackaging2   []model.FlisPackaging2 `json:"flis_packaging_2"`
	CageAddress      []model.CageAddress    `json:"cage_addresses"`
	DssWeightAndCube model.DssWeightAndCube `json:"dss_weight_and_cube"`
}
