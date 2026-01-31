package uoc

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
)

type Repository interface {
	SearchByPage(page int) (response.UOCPageResponse, error)
	SearchSpecific(uoc string) ([]model.LookupUoc, error)
	SearchByModel(model string) ([]model.LookupUoc, error)
}
