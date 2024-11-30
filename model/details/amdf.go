package details

import "miltechserver/prisma/db"

type Amdf struct {
	ArmyMasterDataFile db.ArmyMasterDataFileModel `json:"army_master_data_file"`
	AmdfManagement     db.AmdfManagementModel     `json:"amdf_management"`
	AmdfCredit         db.AmdfCreditModel         `json:"amdf_credit"`
	AmdfBilling        db.AmdfBillingModel        `json:"amdf_billing"`
	AmdfMatcat         db.AmdfMatcatModel         `json:"amdf_matcat"`
	AmdfPhrases        db.AmdfPhraseModel         `json:"amdf_phrases"`
	AmdfIandS          db.AmdfIAndSModel          `json:"amdf_i_and_s"`
	ArmyLin            db.ArmyLineItemNumberModel `json:"army_lin"`
}
