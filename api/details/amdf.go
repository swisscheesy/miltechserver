package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Amdf struct {
	ArmyMasterDataFile model.ArmyMasterDataFile `json:"amdf"`
	AmdfManagement     model.AmdfManagement     `alias:"army_management" json:"amdf_management"`
	AmdfCredit         model.AmdfCredit         `json:"amdf_credit"`
	AmdfBilling        model.AmdfBilling        `json:"amdf_billing"`
	AmdfMatcat         model.AmdfMatcat         `json:"amdf_matcat"`
	AmdfPhrases        []model.AmdfPhrase       `json:"amdf_phrases"`
	AmdfIandS          []model.AmdfIAndS        `json:"amdf_i_and_s"`
	ArmyLin            model.ArmyLineItemNumber `json:"army_lin"`
}
