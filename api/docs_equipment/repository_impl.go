package docs_equipment

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

const pageSize = 40

type repositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repositoryImpl{db: db}
}

const selectAll = `SELECT id, model, lin, mode, description, length, width, height, weight, kw, hz, family FROM docs_equipment_details`

func scanEquipmentItem(rows *sql.Rows) (model.DocsEquipmentDetails, error) {
	var item model.DocsEquipmentDetails
	err := rows.Scan(
		&item.ID, &item.Model, &item.Lin, &item.Mode,
		&item.Description, &item.Length, &item.Width,
		&item.Height, &item.Weight, &item.Kw, &item.Hz, &item.Family,
	)
	return item, err
}

func collectItems(rows *sql.Rows) ([]model.DocsEquipmentDetails, error) {
	var items []model.DocsEquipmentDetails
	for rows.Next() {
		item, err := scanEquipmentItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan equipment item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}
	return items, nil
}

func buildPageResponse(items []model.DocsEquipmentDetails, page, totalCount int) EquipmentDetailsPageResponse {
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	return EquipmentDetailsPageResponse{
		Items:      items,
		Count:      len(items),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}
}

func (r *repositoryImpl) GetAllPaginated(page int) (EquipmentDetailsPageResponse, error) {
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)

	query := selectAll + ` ORDER BY id LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to query equipment details: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details`).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count equipment details: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}

func (r *repositoryImpl) GetFamilies() (FamiliesResponse, error) {
	rows, err := r.db.Query(`SELECT DISTINCT family FROM docs_equipment_details WHERE family IS NOT NULL ORDER BY family`)
	if err != nil {
		return FamiliesResponse{}, fmt.Errorf("failed to query families: %w", err)
	}
	defer rows.Close()

	var families []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return FamiliesResponse{}, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, f)
	}
	if err := rows.Err(); err != nil {
		return FamiliesResponse{}, fmt.Errorf("failed to iterate families: %w", err)
	}

	return FamiliesResponse{Families: families, Count: len(families)}, nil
}

func (r *repositoryImpl) GetByFamilyPaginated(family string, page int) (EquipmentDetailsPageResponse, error) {
	if strings.TrimSpace(family) == "" {
		return EquipmentDetailsPageResponse{}, ErrEmptyParam
	}
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)

	query := selectAll + ` WHERE LOWER(family) = LOWER($1) ORDER BY id LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(query, strings.TrimSpace(family), pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to query by family: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details WHERE LOWER(family) = LOWER($1)`, strings.TrimSpace(family)).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count by family: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found for family: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}

func (r *repositoryImpl) SearchPaginated(query string, page int) (EquipmentDetailsPageResponse, error) {
	if strings.TrimSpace(query) == "" {
		return EquipmentDetailsPageResponse{}, ErrEmptyParam
	}
	if page < 1 {
		return EquipmentDetailsPageResponse{}, ErrInvalidPage
	}
	offset := int64(pageSize) * int64(page-1)
	searchPattern := "%" + strings.TrimSpace(query) + "%"

	stmt := selectAll + ` WHERE model ILIKE $1 OR lin ILIKE $1 ORDER BY id LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(stmt, searchPattern, pageSize, offset)
	if err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	items, err := collectItems(rows)
	if err != nil {
		return EquipmentDetailsPageResponse{}, err
	}

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM docs_equipment_details WHERE model ILIKE $1 OR lin ILIKE $1`, searchPattern).Scan(&totalCount); err != nil {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("failed to count search results: %w", err)
	}

	if len(items) == 0 {
		return EquipmentDetailsPageResponse{}, fmt.Errorf("no equipment found matching query: %w", ErrNotFound)
	}

	return buildPageResponse(items, page, totalCount), nil
}
