package response

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type UOCPLookupResponse struct {
	UOCs  []model.LookupUoc `json:"uocs"`
	Count int               `json:"count"`
}
