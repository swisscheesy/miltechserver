package details

import "miltechserver/prisma/db"

type Freight struct {
	FlisFreight db.FlisFreightModel `json:"flis_freight"`
}
