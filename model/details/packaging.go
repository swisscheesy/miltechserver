package details

import "miltechserver/prisma/db"

type Packaging struct {
	FlisPackaging1   db.FlisPackaging1Model   `json:"flis_packaging_1"`
	FlisPackaging2   db.FlisPackaging2Model   `json:"flis_packaging_2"`
	CageAddresses    []db.CageAddressModel    `json:"cage_addresses"`
	DssWeightAndCube db.DssWeightAndCubeModel `json:"dss_weight_and_cube"`
}
