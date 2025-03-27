package repository

import (
	"database/sql"
	"errors"
	. "github.com/go-jet/jet/v2/postgres"
	"math"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/response"
)

var returnCount = int64(20)

type ItemLokupRepositoryImpl struct {
	Db *sql.DB
}

func NewItemLookupRepositoryImpl(db *sql.DB) *ItemLokupRepositoryImpl {
	return &ItemLokupRepositoryImpl{Db: db}
}

// SearchLINByPage searches for LIN (Line Item Number) by page.
// \param page - the page number to retrieve.
// \return a LINPageResponse containing the LIN data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchLINByPage(page int) (response.LINPageResponse, error) {

	var linData []model.LookupLinNiin
	offset := returnCount * int64(page-1)
	stmt := SELECT(
		view.LookupLinNiin.AllColumns,
	).FROM(view.LookupLinNiin).
		LIMIT(returnCount).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &linData)

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(view.LookupLinNiin.Lin),
	).FROM(view.LookupLinNiin)

	err = countStmt.Query(repo.Db, &count)

	if err != nil || len(linData) == 0 {
		return response.LINPageResponse{}, errors.New("no items found")
	} else {
		totalPages := math.Ceil(float64(count.Count / 20))
		return response.LINPageResponse{
			Lins:       linData,
			Count:      count.Count,
			Page:       page,
			TotalPages: int(totalPages),
			IsLastPage: float64(page) == totalPages,
		}, nil
	}

}

// SearchLINByNIIN searches for LIN (Line Item Number) by NIIN (National Item Identification Number).
// \param niin - the NIIN to search for.
// \return a slice of LookupLinNiin containing the LIN data.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchLINByNIIN(niin string) ([]model.LookupLinNiin, error) {
	var linData []model.LookupLinNiin

	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Niin.LIKE(String("%" + niin + "%")))

	err := stmt.Query(repo.Db, &linData)

	if err != nil || len(linData) == 0 {
		return []model.LookupLinNiin{}, errors.New("no items found")
	} else {
		return linData, nil
	}
}

// SearchNIINByLIN searches for NIIN (National Item Identification Number) by LIN (Line Item Number).
// \param lin - the LIN to search for.
// \return a slice of LookupLinNiin containing the NIIN data.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	var linData []model.LookupLinNiin

	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Lin.LIKE(String("%" + lin + "%")))

	err := stmt.Query(repo.Db, &linData)

	if err != nil || len(linData) == 0 {
		return []model.LookupLinNiin{}, errors.New("no items found")
	} else {
		return linData, nil
	}
}

// SearchUOCByPage searches for UOC (Unit of Consumption) by page.
// \param page - the page number to retrieve.
// \return a UOCPageResponse containing the UOC data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchUOCByPage(page int) (response.UOCPageResponse, error) {

	var uocData []model.LookupUoc
	offset := returnCount * int64(page-1)
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		LIMIT(returnCount).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &uocData)

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(table.LookupUoc.Uoc),
	).FROM(table.LookupUoc)

	err = countStmt.Query(repo.Db, &count)

	if err != nil || len(uocData) == 0 {
		return response.UOCPageResponse{}, errors.New("no items found")
	} else {
		totalPages := math.Ceil(float64(count.Count / 20))
		return response.UOCPageResponse{
			UOCs:       uocData,
			Count:      count.Count,
			Page:       page,
			TotalPages: int(totalPages),
			IsLastPage: float64(page) == totalPages,
		}, nil
	}
}

// SearchSpecificUOC searches for a specific UOC (Unit of Consumption).
// \param uoc - the UOC to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchSpecificUOC(uoc string) ([]model.LookupUoc, error) {
	var uocData []model.LookupUoc

	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Uoc.EQ(String(uoc)))

	err := stmt.Query(repo.Db, &uocData)

	if err != nil || len(uocData) == 0 {
		return nil, errors.New("no items found")
	} else {
		return uocData, nil
	}

}

// SearchUOCByModel searches for UOC (Unit of Consumption) by vehicle model.
// \param vehModel - the vehicle model to search for.
// \return a slice of LookupUoc containing the UOC data.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchUOCByModel(vehModel string) ([]model.LookupUoc, error) {

	var uocData []model.LookupUoc

	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Model.LIKE(String("%" + vehModel + "%")))

	err := stmt.Query(repo.Db, &uocData)

	if err != nil || len(uocData) == 0 {
		return nil, errors.New("no items found")
	} else {
		return uocData, nil
	}
}
