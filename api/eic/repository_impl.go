package eic

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"miltechserver/api/response"
)

const eicReturnCount = int64(40)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (repo *repository) GetByNIIN(niin string) ([]response.EICConsolidatedItem, error) {
	if strings.TrimSpace(niin) == "" {
		return nil, ErrEmptyParam
	}

	var consolidatedData []response.EICConsolidatedItem
	query := selectColumns() + `
WHERE niin = $1
` + groupByColumns()

	rows, err := repo.db.Query(query, strings.TrimSpace(niin))
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by NIIN: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanConsolidatedItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by NIIN: %w", err)
	}

	if len(consolidatedData) == 0 {
		return nil, fmt.Errorf("no EIC items found for the specified NIIN: %w", ErrNotFound)
	}

	return consolidatedData, nil
}

func (repo *repository) GetByLIN(lin string) ([]response.EICConsolidatedItem, error) {
	if strings.TrimSpace(lin) == "" {
		return nil, ErrEmptyParam
	}

	var consolidatedData []response.EICConsolidatedItem
	query := selectColumns() + `
WHERE lin = $1
` + groupByColumns()

	rows, err := repo.db.Query(query, strings.TrimSpace(lin))
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by LIN: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanConsolidatedItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by LIN: %w", err)
	}

	if len(consolidatedData) == 0 {
		return nil, fmt.Errorf("no EIC items found for the specified LIN: %w", ErrNotFound)
	}

	return consolidatedData, nil
}

func (repo *repository) GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error) {
	if strings.TrimSpace(fsc) == "" {
		return response.EICPageResponse{}, ErrEmptyParam
	}

	if page < 1 {
		return response.EICPageResponse{}, ErrInvalidPage
	}

	var consolidatedData []response.EICConsolidatedItem
	offset := eicReturnCount * int64(page-1)

	query := selectColumns() + `
WHERE fsc = $1
` + groupByColumns() + `
LIMIT $2 OFFSET $3
`

	rows, err := repo.db.Query(query, strings.TrimSpace(fsc), eicReturnCount, offset)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data by FSC: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanConsolidatedItem(rows)
		if err != nil {
			return response.EICPageResponse{}, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	if err := rows.Err(); err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data by FSC: %w", err)
	}

	countQuery := `
SELECT COUNT(*) FROM (
	SELECT 1
	FROM eic
	WHERE fsc = $1
` + groupByColumns() + `
) AS consolidated_count
`

	var totalCount int
	if err := repo.db.QueryRow(countQuery, strings.TrimSpace(fsc)).Scan(&totalCount); err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to get total consolidated count for FSC: %w", err)
	}

	if len(consolidatedData) == 0 {
		return response.EICPageResponse{}, fmt.Errorf("no EIC items found for the specified FSC: %w", ErrNotFound)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(eicReturnCount)))
	return response.EICPageResponse{
		Items:      consolidatedData,
		Count:      len(consolidatedData),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

func (repo *repository) GetAllPaginated(page int, search string) (response.EICPageResponse, error) {
	if page < 1 {
		return response.EICPageResponse{}, ErrInvalidPage
	}

	var consolidatedData []response.EICConsolidatedItem
	offset := eicReturnCount * int64(page-1)
	searchTerm := strings.TrimSpace(search)

	var whereClause string
	var args []interface{}
	argIndex := 1

	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		whereClause = `WHERE niin ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR lin ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR fsc ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR nomen ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR model ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR eic ILIKE $` + fmt.Sprintf("%d", argIndex) + `
		OR EXISTS (SELECT 1 FROM unnest(array_agg(DISTINCT uoeic)) AS u(val) WHERE u.val ILIKE $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	query := selectColumns() + `
` + whereClause + `
` + groupByColumns() + `
LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, eicReturnCount, offset)

	rows, err := repo.db.Query(query, args...)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanConsolidatedItem(rows)
		if err != nil {
			return response.EICPageResponse{}, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	if err := rows.Err(); err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data: %w", err)
	}

	countQuery := `
SELECT COUNT(*) FROM (
	SELECT 1
	FROM eic
	` + whereClause + `
` + groupByColumns() + `
) AS consolidated_count
`

	var countArgs []interface{}
	if searchTerm != "" {
		countArgs = []interface{}{searchTerm}
	}

	var totalCount int
	if err := repo.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to get total consolidated count: %w", err)
	}

	if len(consolidatedData) == 0 {
		return response.EICPageResponse{}, fmt.Errorf("no EIC items found for the specified criteria: %w", ErrNotFound)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(eicReturnCount)))
	return response.EICPageResponse{
		Items:      consolidatedData,
		Count:      len(consolidatedData),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

func isNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}
