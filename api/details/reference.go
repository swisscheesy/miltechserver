package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Reference struct {
	FlisReference          model.FlisIdentification  `json:"flis_reference"`            // This isn't a mistake
	ReferenceAndPartNumber []model.FlisReference     `json:"reference_and_part_number"` // Not a mistake
	CageAddresses          []model.CageAddress       `json:"cage_addresses"`
	CageStatusAndType      []model.CageStatusAndType `json:"cage_status"`
}
