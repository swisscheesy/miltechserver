package details

import "miltechserver/prisma/db"

type Reference struct {
	FlisReference          *db.FlisIdentificationModel `json:"flis_reference"`            // This isn't a mistake
	ReferenceAndPartNumber []db.FlisReferenceModel     `json:"reference_and_part_number"` // Not a mistake
	CageAddresses          []db.CageAddressModel       `json:"cage_addresses"`
	CageStatusAndType      []db.CageStatusAndTypeModel `json:"cage_status"`
}
