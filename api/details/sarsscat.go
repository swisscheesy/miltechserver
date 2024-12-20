package details

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type Sarsscat struct {
	ArmySarsscat model.ArmySarsscat `json:"army_sarsscat"`
	MoeRule      []model.MoeRule    `json:"moe_rule"`
	AmdfFreight  model.AmdfFreight  `json:"amdf_freight"`
}
