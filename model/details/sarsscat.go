package details

import "miltechserver/prisma/db"

type Sarsscat struct {
	ArmySarsscat db.ArmySarsscatModel `json:"army_sarsscat"`
	MoeRule      []db.MoeRuleModel    `json:"moe_rule"`
	AmdfFreight  db.AmdfFreightModel  `json:"amdf_freight"`
}
