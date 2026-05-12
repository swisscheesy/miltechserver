package sb_700_20

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

func (r *repository) GetChp4ByLIN(lin string) (model.Sb70020Chp4, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020Chp4{}, ErrEmptyParam
	}
	var results []model.Sb70020Chp4
	stmt := SELECT(table.Sb70020Chp4.AllColumns).
		FROM(table.Sb70020Chp4).
		WHERE(table.Sb70020Chp4.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020Chp4{}, err
	}
	if len(results) == 0 {
		return model.Sb70020Chp4{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetChp4Paginated(page int) (PageResponse[model.Sb70020Chp4], error) {
	if page < 1 {
		return PageResponse[model.Sb70020Chp4]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020Chp4
	stmt := SELECT(table.Sb70020Chp4.AllColumns).
		FROM(table.Sb70020Chp4).
		ORDER_BY(table.Sb70020Chp4.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020Chp4]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020Chp4]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020Chp4)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020Chp4]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020Chp4]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetChp6ByLIN(lin string) ([]model.Sb70020Chp6, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020Chp6
	stmt := SELECT(table.Sb70020Chp6.AllColumns).
		FROM(table.Sb70020Chp6).
		WHERE(table.Sb70020Chp6.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetChp6Paginated(page int) (PageResponse[model.Sb70020Chp6], error) {
	if page < 1 {
		return PageResponse[model.Sb70020Chp6]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020Chp6
	stmt := SELECT(table.Sb70020Chp6.AllColumns).
		FROM(table.Sb70020Chp6).
		ORDER_BY(table.Sb70020Chp6.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020Chp6]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020Chp6]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020Chp6)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020Chp6]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020Chp6]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetChp8ByLIN(lin string) ([]model.Sb70020Chp8, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020Chp8
	stmt := SELECT(table.Sb70020Chp8.AllColumns).
		FROM(table.Sb70020Chp8).
		WHERE(table.Sb70020Chp8.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetChp8Paginated(page int) (PageResponse[model.Sb70020Chp8], error) {
	if page < 1 {
		return PageResponse[model.Sb70020Chp8]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020Chp8
	stmt := SELECT(table.Sb70020Chp8.AllColumns).
		FROM(table.Sb70020Chp8).
		ORDER_BY(table.Sb70020Chp8.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020Chp8]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020Chp8]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020Chp8)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020Chp8]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020Chp8]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}
