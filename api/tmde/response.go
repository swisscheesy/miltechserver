package tmde

import "miltechserver/.gen/miltech_ng/public/model"

type TmdePageResponse struct {
	Items      []model.TmdeIntervalMat `json:"items"`
	Count      int                     `json:"count"`
	Page       int                     `json:"page"`
	TotalPages int                     `json:"total_pages"`
	IsLastPage bool                    `json:"is_last_page"`
}
