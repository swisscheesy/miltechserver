package tmde

import (
	"database/sql"
	"math"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/view"

	. "github.com/go-jet/jet/v2/postgres"
)

const pageSize = int64(100)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByNIIN(niin string) (model.TmdeIntervalMat, error) {
	if strings.TrimSpace(niin) == "" {
		return model.TmdeIntervalMat{}, ErrEmptyParam
	}

	var results []model.TmdeIntervalMat
	stmt := SELECT(view.TmdeIntervalMat.AllColumns).
		FROM(view.TmdeIntervalMat).
		WHERE(view.TmdeIntervalMat.Niin.EQ(String(niin)))

	if err := stmt.Query(r.db, &results); err != nil {
		return model.TmdeIntervalMat{}, err
	}

	if len(results) == 0 {
		return model.TmdeIntervalMat{}, ErrNotFound
	}

	return results[0], nil
}

func (r *repository) GetAllPaginated(page int) (TmdePageResponse, error) {
	if page < 1 {
		return TmdePageResponse{}, ErrInvalidPage
	}

	offset := pageSize * int64(page-1)

	var items []model.TmdeIntervalMat
	stmt := SELECT(view.TmdeIntervalMat.AllColumns).
		FROM(view.TmdeIntervalMat).
		ORDER_BY(view.TmdeIntervalMat.Niin.ASC()).
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
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(view.TmdeIntervalMat)
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
