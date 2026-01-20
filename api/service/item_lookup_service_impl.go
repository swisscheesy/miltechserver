package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/repository"
	"miltechserver/api/response"
	"strings"
)

type ItemLookupServiceImpl struct {
	ItemLookupService repository.ItemLookupRepository
}

func NewItemLookupServiceImpl(itemLookupRepository repository.ItemLookupRepository) *ItemLookupServiceImpl {
	return &ItemLookupServiceImpl{ItemLookupService: itemLookupRepository}
}

// LookupLINByPage looks up LIN (Line Item Number) by page.
// \param page - the page number to retrieve.
// \return a LINPageResponse containing the LIN data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupLINByPage(page int) (response.LINPageResponse, error) {
	linData, err := service.ItemLookupService.SearchLINByPage(page)

	if err != nil {
		return response.LINPageResponse{}, err
	}

	return linData, nil

}

// LookupLINByNIIN looks up LIN (Line Item Number) by NIIN (National Item Identification Number).
// \param niin - the NIIN to search for.
// \return a slice of LookupLinNiin containing the LIN data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupLINByNIIN(niin string) ([]model.LookupLinNiin, error) {
	linData, err := service.ItemLookupService.SearchLINByNIIN(niin)

	if err != nil {
		return nil, err
	}

	return linData, nil

}

// LookupNIINByLIN looks up NIIN (National Item Identification Number) by LIN (Line Item Number).
// \param lin - the LIN to search for.
// \return a slice of LookupLinNiin containing the NIIN data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	niinData, err := service.ItemLookupService.SearchNIINByLIN(strings.ToUpper(lin))

	if err != nil {
		return nil, err
	}

	return niinData, nil
}

// LookupSubstituteLINAll looks up all substitute LIN records.
// \return a slice of ArmySubstituteLin containing all substitute LIN data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupSubstituteLINAll() ([]model.ArmySubstituteLin, error) {
	substituteData, err := service.ItemLookupService.SearchSubstituteLINAll()
	if err != nil {
		return nil, err
	}

	return substituteData, nil
}

// LookupCAGEByCode looks up CAGE address records by CAGE code.
// \param cage - the CAGE code to search for.
// \return a slice of CageAddress containing the matching CAGE data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupCAGEByCode(cage string) ([]model.CageAddress, error) {
	cageData, err := service.ItemLookupService.SearchCAGEByCode(strings.ToUpper(cage))
	if err != nil {
		return nil, err
	}

	return cageData, nil
}

// LookupUOCByPage looks up UOC (Unit of Consumption) by page.
// \param page - the page number to retrieve.
// \return a UOCPageResponse containing the UOC data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupUOCByPage(page int) (response.UOCPageResponse, error) {
	uocData, err := service.ItemLookupService.SearchUOCByPage(page)

	if err != nil {
		return response.UOCPageResponse{}, err
	}

	return uocData, nil
}

// LookupSpecificUOC looks up a specific UOC (Unit of Consumption).
// \param uoc - the UOC to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupSpecificUOC(uoc string) ([]model.LookupUoc, error) {
	uocData, err := service.ItemLookupService.SearchSpecificUOC(strings.ToUpper(uoc))

	if err != nil {
		return nil, err
	}

	return uocData, nil
}

// LookupUOCByModel looks up UOC (Unit of Consumption) by vehicle model.
// \param model - the vehicle model to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (service *ItemLookupServiceImpl) LookupUOCByModel(model string) ([]model.LookupUoc, error) {
	uocData, err := service.ItemLookupService.SearchUOCByModel(strings.ToUpper(model))

	if err != nil {
		return nil, err
	}

	return uocData, nil
}
