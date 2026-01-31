package lin

import (
	"database/sql"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/item_lookup/shared"
	"miltechserver/api/response"
	"strings"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) SearchByPage(page int) (response.LINPageResponse, error) {
	if page < 1 {
		return response.LINPageResponse{}, shared.ErrInvalidPage
	}

	var linData []model.LookupLinNiin
	offset := shared.CalculateOffset(page, shared.DefaultPageSize)
	stmt := SELECT(
		view.LookupLinNiin.AllColumns,
	).FROM(view.LookupLinNiin).
		LIMIT(shared.DefaultPageSize).
		OFFSET(offset)

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return response.LINPageResponse{}, fmt.Errorf("failed to query LIN data: %w", err)
	}

	var count struct {
		Count int
	}

	countStmt := SELECT(
		COUNT(view.LookupLinNiin.Lin),
	).FROM(view.LookupLinNiin)

	err = countStmt.Query(repo.db, &count)
	if err != nil {
		return response.LINPageResponse{}, fmt.Errorf("failed to get total LIN count: %w", err)
	}

	if len(linData) == 0 {
		return response.LINPageResponse{}, shared.ErrNotFound
	}

	totalPages := shared.CalculateTotalPages(count.Count, shared.DefaultPageSize)
	return response.LINPageResponse{
		Lins:       linData,
		Count:      count.Count,
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

func (repo *RepositoryImpl) SearchByNIIN(niin string) ([]model.LookupLinNiin, error) {
	if strings.TrimSpace(niin) == "" {
		return nil, shared.ErrEmptyParam
	}

	var linData []model.LookupLinNiin
	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Niin.LIKE(String("%" + niin + "%")))

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query LIN data by NIIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, shared.ErrNotFound
	}

	return linData, nil
}

func (repo *RepositoryImpl) SearchNIINByLIN(lin string) ([]model.LookupLinNiin, error) {
	if strings.TrimSpace(lin) == "" {
		return nil, shared.ErrEmptyParam
	}

	var linData []model.LookupLinNiin
	stmt := SELECT(
		view.LookupLinNiin.AllColumns).
		FROM(view.LookupLinNiin).
		WHERE(view.LookupLinNiin.Lin.LIKE(String("%" + lin + "%")))

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query NIIN data by LIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, shared.ErrNotFound
	}

	return linData, nil
}
