package uoc

import "miltechserver/api/response"

type Service interface {
	LookupByPage(page int) (response.UOCPageResponse, error)
	LookupSpecific(uoc string) (response.UOCPageResponse, error)
	LookupByModel(model string) (response.UOCPageResponse, error)
}
