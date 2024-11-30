package details

import "miltechserver/prisma/db"

type Characteristics struct {
	Characteristics []db.FlisItemCharacteristicsModel `json:"characteristics"`
}
