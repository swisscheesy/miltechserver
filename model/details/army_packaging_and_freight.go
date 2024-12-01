package details

import "miltechserver/prisma/db"

type ArmyPackagingAndFreight struct {
	ArmyPackagingAndFreight      db.ArmyPackagingAndFreightModel        `json:"army_packaging_and_freight"`
	ArmyPackaging1               db.ArmyPackaging1Model                 `json:"army_packaging_1"`
	ArmyPackaging2               db.ArmyPackaging2Model                 `json:"army_packaging_2"`
	ArmyPackSpecialInstruct      db.ArmyPackagingSpecialInstructModel   `json:"army_pack_special_instruct"`
	ArmyFreight                  db.ArmyFreightModel                    `json:"army_freight"`
	ArmyPackSupplementalInstruct []db.ArmyPackSupplementalInstructModel `json:"army_pack_supplemental_instruct"`
}
