package response

import "miltechserver/prisma/db"

type LINPageResponse struct {
	Lins       []db.ArmyLineItemNumberModel `json:"lins"`
	Count      int                          `json:"count"`
	Page       int                          `json:"page"`
	TotalPages int                          `json:"total_pages"`
	IsLastPage bool                         `json:"is_last_page"`
}
