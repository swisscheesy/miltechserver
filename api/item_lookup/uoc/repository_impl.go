package uoc

import (
	"database/sql"
	"fmt"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/item_lookup/shared"
	"miltechserver/api/response"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) SearchByPage(page int) (response.UOCPageResponse, error) {
	if page < 1 {
		return response.UOCPageResponse{}, shared.ErrInvalidPage
	}

	var uocData []model.LookupUoc
	offset := shared.CalculateOffset(page, shared.DefaultPageSize)
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		LIMIT(shared.DefaultPageSize).
		OFFSET(offset)

	err := stmt.Query(repo.db, &uocData)
	if err != nil {
		return response.UOCPageResponse{}, fmt.Errorf("failed to query UOC data: %w", err)
	}

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(table.LookupUoc.Uoc),
	).FROM(table.LookupUoc)

	err = countStmt.Query(repo.db, &count)
	if err != nil {
		return response.UOCPageResponse{}, fmt.Errorf("failed to get total UOC count: %w", err)
	}

	if len(uocData) == 0 {
		return response.UOCPageResponse{}, shared.ErrNotFound
	}

	totalPages := shared.CalculateTotalPages(count.Count, shared.DefaultPageSize)
	return response.UOCPageResponse{
		UOCs:       uocData,
		Count:      count.Count,
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

func (repo *RepositoryImpl) SearchSpecific(uoc string) ([]model.LookupUoc, error) {
	if strings.TrimSpace(uoc) == "" {
		return nil, shared.ErrEmptyParam
	}

	var uocData []model.LookupUoc
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Uoc.EQ(String(uoc)))

	err := stmt.Query(repo.db, &uocData)
	if err != nil {
		return nil, fmt.Errorf("failed to query specific UOC: %w", err)
	}

	if len(uocData) == 0 {
		return nil, shared.ErrNotFound
	}

	return uocData, nil
}

func (repo *RepositoryImpl) SearchByModel(vehicleModel string) ([]model.LookupUoc, error) {
	if strings.TrimSpace(vehicleModel) == "" {
		return nil, shared.ErrEmptyParam
	}

	var uocData []model.LookupUoc
	stmt := SELECT(
		table.LookupUoc.AllColumns,
	).FROM(table.LookupUoc).
		WHERE(table.LookupUoc.Model.LIKE(String("%" + vehicleModel + "%")))

	err := stmt.Query(repo.db, &uocData)
	if err != nil {
		return nil, fmt.Errorf("failed to query UOC by model: %w", err)
	}

	if len(uocData) == 0 {
		return nil, shared.ErrNotFound
	}

	return uocData, nil
}
