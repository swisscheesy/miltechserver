package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type UOCPLookupResponse struct {
	UOCs       []model.LookupUoc `json:"uocs"`
	Count      int               `json:"count"`
	Page       int               `json:"page"`
	TotalPages int               `json:"total_pages"`
	IsLastPage bool              `json:"is_last_page"`
}
