package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/response"
)

type ItemLookupService interface {
	LookupLINByPage(page int) (response.LINPageResponse, error)
	LookupLINByNIIN(niin string) ([]model.LookupLinNiin, error)
	LookupNIINByLIN(niin string) ([]model.LookupLinNiin, error)

	LookupUOCByPage(page int) (response.UOCPageResponse, error)
	LookupSpecificUOC(uoc string) ([]model.LookupUoc, error)
	LookupUOCByModel(model string) ([]model.LookupUoc, error)
}
