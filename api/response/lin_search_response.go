package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

// LinSearchResponse represents the response structure for LIN search.
// \param Lins - the LIN data retrieved from the database.
// \param Count - the total count of LINs.
type LinSearchResponse struct {
	Lins       []model.LookupLinNiin `json:"lins"`
	Count      int                   `json:"count"`
	Page       int                   `json:"page"`
	TotalPages int                   `json:"total_pages"`
	IsLastPage bool                  `json:"is_last_page"`
}
