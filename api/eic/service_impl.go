package eic

import (
	"strings"

	"miltechserver/api/response"
)

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func (svc *service) LookupByNIIN(niin string) ([]response.EICConsolidatedItem, error) {
	niinTrimmed := strings.TrimSpace(strings.ToUpper(niin))
	consolidatedData, err := svc.repository.GetByNIIN(niinTrimmed)
	if err != nil {
		return nil, err
	}

	return consolidatedData, nil
}

func (svc *service) LookupByLIN(lin string) ([]response.EICConsolidatedItem, error) {
	linTrimmed := strings.TrimSpace(strings.ToUpper(lin))
	consolidatedData, err := svc.repository.GetByLIN(linTrimmed)
	if err != nil {
		return nil, err
	}

	return consolidatedData, nil
}

func (svc *service) LookupByFSCPaginated(fsc string, page int) (response.EICPageResponse, error) {
	fscTrimmed := strings.TrimSpace(strings.ToUpper(fsc))
	eicData, err := svc.repository.GetByFSCPaginated(fscTrimmed, page)
	if err != nil {
		return response.EICPageResponse{}, err
	}

	return eicData, nil
}

func (svc *service) LookupAllPaginated(page int, search string) (response.EICPageResponse, error) {
	searchTrimmed := strings.TrimSpace(search)
	eicData, err := svc.repository.GetAllPaginated(page, searchTrimmed)
	if err != nil {
		return response.EICPageResponse{}, err
	}

	return eicData, nil
}
