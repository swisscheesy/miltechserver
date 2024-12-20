package repository

import (
	"database/sql"
	. "github.com/go-jet/jet/v2/postgres"
	"math"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/response"
)

var returnCount = 20

type ItemLokupRepositoryImpl struct {
	Db *sql.DB
}

func NewItemLookupRepositoryImpl(db *sql.DB) *ItemLokupRepositoryImpl {
	return &ItemLokupRepositoryImpl{Db: db}
}

// SearchLINByPage searches for LIN (Line Item Number) by page.
// \param ctx - the context for the request.
// \param page - the page number to retrieve.
// \return a LINPageResponse containing the LIN data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *ItemLokupRepositoryImpl) SearchLINByPage(page int) (response.LINPageResponse, error) {

	var linData []model.ArmyLineItemNumber
	offset := int64(20 * (page - 1))
	stmt := SELECT(
		table.ArmyLineItemNumber.AllColumns,
	).FROM(table.ArmyLineItemNumber).
		LIMIT(20).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &linData)

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(table.ArmyLineItemNumber.Lin),
	).FROM(table.ArmyLineItemNumber)

	err = countStmt.Query(repo.Db, &count)

	if err != nil {
		return response.LINPageResponse{}, err
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

func (repo *ItemLokupRepositoryImpl) SearchLINByNIIN(niin string) ([]model.LookupLinNiin, error) {
	var linData []model.LookupLinNiin

	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Niin.LIKE(String("%" + niin + "%")))

	err := stmt.Query(repo.Db, &linData)

	if err != nil {
		return []model.LookupLinNiin{}, err
	} else {
		return linData, nil
	}
}

func (repo *ItemLokupRepositoryImpl) SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	var linData []model.LookupLinNiin

	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Lin.LIKE(String("%" + lin + "%")))

	err := stmt.Query(repo.Db, &linData)

	if err != nil {
		return []model.LookupLinNiin{}, err
	} else {
		return linData, nil
	}
}

func (repo *ItemLokupRepositoryImpl) SearchUOCByPage(page int) (response.UOCPageResponse, error) {

	var uocData []model.LookupUoc
	offset := int64(20 * (page - 1))
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		LIMIT(20).
		OFFSET(offset)

	err := stmt.Query(repo.Db, &uocData)

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(table.LookupUoc.Uoc),
	).FROM(table.LookupUoc)

	err = countStmt.Query(repo.Db, &count)

	if err != nil {
		return response.UOCPageResponse{}, err
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

func (repo *ItemLokupRepositoryImpl) SearchSpecificUOC(uoc string) ([]model.LookupUoc, error) {
	var uocData []model.LookupUoc

	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Uoc.EQ(String(uoc)))

	err := stmt.Query(repo.Db, &uocData)

	if err != nil {
		return nil, err
	} else {
		return uocData, nil
	}

}

//func (repo *ItemLokupRepositoryImpl) SearchUOCByModel(ctx *gin.Context, model string) ([]db.LookupUocModel, error) {
//	uocData, _ := repo.Db.LookupUoc.FindMany(db.LookupUoc.Model.Contains(strings.ToUpper(model))).Exec(ctx)
//
//	return uocData, nil
//}
