package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
)

type ItemLookupRepository interface {
	SearchLINByPage(page int) (response.LINPageResponse, error)
	SearchLINByNIIN(niin string) ([]model.LookupLinNiin, error)
	SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error)

	SearchSubstituteLINAll() ([]model.ArmySubstituteLin, error)
	SearchCAGEByCode(cage string) ([]model.CageAddress, error)

	SearchUOCByPage(page int) (response.UOCPageResponse, error)
	SearchSpecificUOC(uoc string) ([]model.LookupUoc, error)
	SearchUOCByModel(model string) ([]model.LookupUoc, error)
}
