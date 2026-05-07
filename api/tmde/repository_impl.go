package tmde

import (
	"database/sql"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

const pageSize = int64(100)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByNIIN(niin string) (model.TmdeRequirements, error) {
	if strings.TrimSpace(niin) == "" {
		return model.TmdeRequirements{}, ErrEmptyParam
	}

	var results []model.TmdeRequirements
	stmt := SELECT(table.TmdeRequirements.AllColumns).
		FROM(table.TmdeRequirements).
		WHERE(table.TmdeRequirements.Niin.EQ(String(niin)))

	if err := stmt.Query(r.db, &results); err != nil {
		return model.TmdeRequirements{}, err
	}

	if len(results) == 0 {
		return model.TmdeRequirements{}, ErrNotFound
	}

	return results[0], nil
}

func (r *repository) GetAllPaginated(page int) (TmdePageResponse, error) {
	if page < 1 {
		return TmdePageResponse{}, ErrInvalidPage
	}

	offset := pageSize * int64(page-1)

	var items []model.TmdeRequirements
	stmt := SELECT(table.TmdeRequirements.AllColumns).
		FROM(table.TmdeRequirements).
		ORDER_BY(table.TmdeRequirements.Niin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)

	if err := stmt.Query(r.db, &items); err != nil {
		return TmdePageResponse{}, err
	}

	if len(items) == 0 {
		return TmdePageResponse{}, ErrNotFound
	}

	var dest struct {
		Count int64
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.TmdeRequirements)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return TmdePageResponse{}, err
	}
	totalCount := int(dest.Count)

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return TmdePageResponse{
		Items:      items,
		Count:      len(items),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}
