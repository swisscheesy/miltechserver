package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type ArmyPackagingAndFreight struct {
	ArmyPackagingAndFreight      model.ArmyPackagingAndFreight        `json:"army_packaging_and_freight"`
	ArmyPackaging1               model.ArmyPackaging1                 `json:"army_packaging_1"`
	ArmyPackaging2               model.ArmyPackaging2                 `json:"army_packaging_2"`
	ArmyPackSpecialInstruct      model.ArmyPackagingSpecialInstruct   `json:"army_pack_special_instruct"`
	ArmyFreight                  model.ArmyFreight                    `json:"army_freight"`
	ArmyPackSupplementalInstruct []model.ArmyPackSupplementalInstruct `json:"army_pack_supplemental_instruct"`
}
