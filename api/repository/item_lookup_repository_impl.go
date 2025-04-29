package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/response"

	. "github.com/go-jet/jet/v2/postgres"
)

var returnCount = int64(20)

type ItemLookupRepositoryImpl struct {
	Db *sql.DB
}

func NewItemLookupRepositoryImpl(db *sql.DB) *ItemLookupRepositoryImpl {
	return &ItemLookupRepositoryImpl{Db: db}
}

// SearchLINByPage searches for LIN (Line Item Number) by page.
// \param page - the page number to retrieve.
// \return a LINPageResponse containing the LIN data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchLINByPage(page int) (response.LINPageResponse, error) {
	if page < 1 {
		return response.LINPageResponse{}, errors.New("page number must be greater than 0")
	}

	var linData []model.LookupLinNiin
	offset := returnCount * int64(page-1)
	stmt := SELECT(
		view.LookupLinNiin.AllColumns,
	).FROM(view.LookupLinNiin).
		LIMIT(returnCount).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &linData)
	if err != nil {
		return response.LINPageResponse{}, fmt.Errorf("failed to query LIN data: %w", err)
	}

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(view.LookupLinNiin.Lin),
	).FROM(view.LookupLinNiin)

	err = countStmt.Query(repo.Db, &count)
	if err != nil {
		return response.LINPageResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	if len(linData) == 0 {
		return response.LINPageResponse{}, errors.New("no items found for the specified page")
	}

	totalPages := int(math.Ceil(float64(count.Count) / float64(returnCount)))
	return response.LINPageResponse{
		Lins:       linData,
		Count:      count.Count,
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

// SearchLINByNIIN searches for LIN (Line Item Number) by NIIN (National Item Identification Number).
// \param niin - the NIIN to search for.
// \return a slice of LookupLinNiin containing the LIN data.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchLINByNIIN(niin string) ([]model.LookupLinNiin, error) {
	if niin == "" {
		return nil, errors.New("niin cannot be empty")
	}

	var linData []model.LookupLinNiin
	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Niin.LIKE(String("%" + niin + "%")))

	err := stmt.Query(repo.Db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query LIN data by NIIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, errors.New("no LIN items found for the specified NIIN")
	}

	return linData, nil
}

// SearchNIINByLIN searches for NIIN (National Item Identification Number) by LIN (Line Item Number).
// \param lin - the LIN to search for.
// \return a slice of LookupLinNiin containing the NIIN data.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	if lin == "" {
		return nil, errors.New("lin cannot be empty")
	}

	var linData []model.LookupLinNiin
	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Lin.LIKE(String("%" + lin + "%")))

	err := stmt.Query(repo.Db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query NIIN data by LIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, errors.New("no NIIN items found for the specified LIN")
	}

	return linData, nil
}

// SearchUOCByPage searches for UOC (Unit of Consumption) by page.
// \param page - the page number to retrieve.
// \return a UOCPageResponse containing the UOC data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchUOCByPage(page int) (response.UOCPageResponse, error) {
	if page < 1 {
		return response.UOCPageResponse{}, errors.New("page number must be greater than 0")
	}

	var uocData []model.LookupUoc
	offset := returnCount * int64(page-1)
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		LIMIT(returnCount).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &uocData)
	if err != nil {
		return response.UOCPageResponse{}, fmt.Errorf("failed to query UOC data: %w", err)
	}

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(table.LookupUoc.Uoc),
	).FROM(table.LookupUoc)

	err = countStmt.Query(repo.Db, &count)
	if err != nil {
		return response.UOCPageResponse{}, fmt.Errorf("failed to get total count: %w", err)
	}

	if len(uocData) == 0 {
		return response.UOCPageResponse{}, errors.New("no items found for the specified page")
	}

	totalPages := int(math.Ceil(float64(count.Count) / float64(returnCount)))
	return response.UOCPageResponse{
		UOCs:       uocData,
		Count:      count.Count,
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

// SearchSpecificUOC searches for a specific UOC (Unit of Consumption).
// \param uoc - the UOC to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchSpecificUOC(uoc string) ([]model.LookupUoc, error) {
	if uoc == "" {
		return nil, errors.New("uoc cannot be empty")
	}

	var uocData []model.LookupUoc
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Uoc.EQ(String(uoc)))

	err := stmt.Query(repo.Db, &uocData)
	if err != nil {
		return nil, fmt.Errorf("failed to query specific UOC: %w", err)
	}

	if len(uocData) == 0 {
		return nil, errors.New("no UOC found for the specified code")
	}

	return uocData, nil
}

// SearchUOCByModel searches for UOC (Unit of Consumption) by vehicle model.
// \param vehModel - the vehicle model to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (repo *ItemLookupRepositoryImpl) SearchUOCByModel(vehModel string) ([]model.LookupUoc, error) {
	if vehModel == "" {
		return nil, errors.New("vehicle model cannot be empty")
	}

	var uocData []model.LookupUoc
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Model.LIKE(String("%" + vehModel + "%")))

	err := stmt.Query(repo.Db, &uocData)
	if err != nil {
		return nil, fmt.Errorf("failed to query UOC by model: %w", err)
	}

	if len(uocData) == 0 {
		return nil, errors.New("no UOC found for the specified vehicle model")
	}

	return uocData, nil
}
