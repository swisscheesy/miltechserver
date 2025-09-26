package service

import (
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"strings"
)

type EICServiceImpl struct {
	EICRepository repository.EICRepository
}

func NewEICServiceImpl(eicRepository repository.EICRepository) *EICServiceImpl {
	return &EICServiceImpl{EICRepository: eicRepository}
}

// LookupByNIIN looks up consolidated EIC records by National Item Identification Number.
// Duplicate records are consolidated with UOEIC and MRC values aggregated into arrays.
// \param niin - the NIIN to search for.
// \return a slice of EICConsolidatedItem containing the consolidated EIC data.
// \return an error if the operation fails.
func (service *EICServiceImpl) LookupByNIIN(niin string) ([]response.EICConsolidatedItem, error) {
	niinTrimmed := strings.TrimSpace(strings.ToUpper(niin))
	consolidatedData, err := service.EICRepository.GetByNIIN(niinTrimmed)

	if err != nil {
		return nil, err
	}

	return consolidatedData, nil
}

// LookupByLIN looks up consolidated EIC records by Line Item Number.
// Duplicate records are consolidated with UOEIC and MRC values aggregated into arrays.
// \param lin - the LIN to search for.
// \return a slice of EICConsolidatedItem containing the consolidated EIC data.
// \return an error if the operation fails.
func (service *EICServiceImpl) LookupByLIN(lin string) ([]response.EICConsolidatedItem, error) {
	linTrimmed := strings.TrimSpace(strings.ToUpper(lin))
	consolidatedData, err := service.EICRepository.GetByLIN(linTrimmed)

	if err != nil {
		return nil, err
	}

	return consolidatedData, nil
}

// LookupByFSCPaginated looks up EIC records by Federal Supply Class with pagination.
// \param fsc - the FSC to search for.
// \param page - the page number to retrieve.
// \return an EICPageResponse containing the EIC data with pagination metadata.
// \return an error if the operation fails.
func (service *EICServiceImpl) LookupByFSCPaginated(fsc string, page int) (response.EICPageResponse, error) {
	fscTrimmed := strings.TrimSpace(strings.ToUpper(fsc))
	eicData, err := service.EICRepository.GetByFSCPaginated(fscTrimmed, page)

	if err != nil {
		return response.EICPageResponse{}, err
	}

	return eicData, nil
}

// LookupAllPaginated looks up all EIC records with optional search and pagination.
// \param page - the page number to retrieve.
// \param search - optional search term to filter across all text fields.
// \return an EICPageResponse containing the EIC data with pagination metadata.
// \return an error if the operation fails.
func (service *EICServiceImpl) LookupAllPaginated(page int, search string) (response.EICPageResponse, error) {
	searchTrimmed := strings.TrimSpace(search)
	eicData, err := service.EICRepository.GetAllPaginated(page, searchTrimmed)

	if err != nil {
		return response.EICPageResponse{}, err
	}

	return eicData, nil
}
