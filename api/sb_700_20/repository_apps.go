package sb_700_20

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

func (r *repository) GetAppBByLIN(lin string) ([]model.Sb70020AppB, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppB
	stmt := SELECT(table.Sb70020AppB.AllColumns).
		FROM(table.Sb70020AppB).
		WHERE(table.Sb70020AppB.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppBPaginated(page int) (PageResponse[model.Sb70020AppB], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppB]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppB
	stmt := SELECT(table.Sb70020AppB.AllColumns).
		FROM(table.Sb70020AppB).
		ORDER_BY(table.Sb70020AppB.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppB]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppB]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppB)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppB]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppB]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppCByLIN(lin string) (model.Sb70020AppC, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020AppC{}, ErrEmptyParam
	}
	var results []model.Sb70020AppC
	stmt := SELECT(table.Sb70020AppC.AllColumns).
		FROM(table.Sb70020AppC).
		WHERE(table.Sb70020AppC.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020AppC{}, err
	}
	if len(results) == 0 {
		return model.Sb70020AppC{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetAppCPaginated(page int) (PageResponse[model.Sb70020AppC], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppC]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppC
	stmt := SELECT(table.Sb70020AppC.AllColumns).
		FROM(table.Sb70020AppC).
		ORDER_BY(table.Sb70020AppC.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppC]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppC]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppC)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppC]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppC]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppDByLIN(lin string) ([]model.Sb70020AppD, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppD
	stmt := SELECT(table.Sb70020AppD.AllColumns).
		FROM(table.Sb70020AppD).
		WHERE(table.Sb70020AppD.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppDPaginated(page int) (PageResponse[model.Sb70020AppD], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppD]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppD
	stmt := SELECT(table.Sb70020AppD.AllColumns).
		FROM(table.Sb70020AppD).
		ORDER_BY(table.Sb70020AppD.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppD]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppD]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppD)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppD]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppD]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppEByLIN(lin string) ([]model.Sb70020AppE, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppE
	stmt := SELECT(table.Sb70020AppE.AllColumns).
		FROM(table.Sb70020AppE).
		WHERE(table.Sb70020AppE.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppEPaginated(page int) (PageResponse[model.Sb70020AppE], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppE]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppE
	stmt := SELECT(table.Sb70020AppE.AllColumns).
		FROM(table.Sb70020AppE).
		ORDER_BY(table.Sb70020AppE.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppE]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppE]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppE)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppE]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppE]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppFByLIN(lin string) (model.Sb70020AppF, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020AppF{}, ErrEmptyParam
	}
	var results []model.Sb70020AppF
	stmt := SELECT(table.Sb70020AppF.AllColumns).
		FROM(table.Sb70020AppF).
		WHERE(table.Sb70020AppF.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020AppF{}, err
	}
	if len(results) == 0 {
		return model.Sb70020AppF{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetAppFPaginated(page int) (PageResponse[model.Sb70020AppF], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppF]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppF
	stmt := SELECT(table.Sb70020AppF.AllColumns).
		FROM(table.Sb70020AppF).
		ORDER_BY(table.Sb70020AppF.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppF]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppF]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppF)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppF]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppF]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppGByLIN(lin string) (model.Sb70020AppG, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020AppG{}, ErrEmptyParam
	}
	var results []model.Sb70020AppG
	stmt := SELECT(table.Sb70020AppG.AllColumns).
		FROM(table.Sb70020AppG).
		WHERE(table.Sb70020AppG.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020AppG{}, err
	}
	if len(results) == 0 {
		return model.Sb70020AppG{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetAppGPaginated(page int) (PageResponse[model.Sb70020AppG], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppG]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppG
	stmt := SELECT(table.Sb70020AppG.AllColumns).
		FROM(table.Sb70020AppG).
		ORDER_BY(table.Sb70020AppG.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppG]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppG]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppG)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppG]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppG]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

// app_h1 and app_h2 search by lin_zmm_lin, not lin
func (r *repository) GetAppH1ByLIN(lin string) ([]model.Sb70020AppH1, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppH1
	stmt := SELECT(table.Sb70020AppH1.AllColumns).
		FROM(table.Sb70020AppH1).
		WHERE(table.Sb70020AppH1.LinZmmLin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppH1Paginated(page int) (PageResponse[model.Sb70020AppH1], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppH1]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppH1
	stmt := SELECT(table.Sb70020AppH1.AllColumns).
		FROM(table.Sb70020AppH1).
		ORDER_BY(table.Sb70020AppH1.LinZmmLin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppH1]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppH1]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppH1)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppH1]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppH1]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppH2ByLIN(lin string) ([]model.Sb70020AppH2, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return nil, ErrEmptyParam
	}
	var results []model.Sb70020AppH2
	stmt := SELECT(table.Sb70020AppH2.AllColumns).
		FROM(table.Sb70020AppH2).
		WHERE(table.Sb70020AppH2.LinZmmLin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (r *repository) GetAppH2Paginated(page int) (PageResponse[model.Sb70020AppH2], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppH2]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppH2
	stmt := SELECT(table.Sb70020AppH2.AllColumns).
		FROM(table.Sb70020AppH2).
		ORDER_BY(table.Sb70020AppH2.LinZmmLin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppH2]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppH2]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppH2)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppH2]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppH2]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppIByLIN(lin string) (model.Sb70020AppI, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020AppI{}, ErrEmptyParam
	}
	var results []model.Sb70020AppI
	stmt := SELECT(table.Sb70020AppI.AllColumns).
		FROM(table.Sb70020AppI).
		WHERE(table.Sb70020AppI.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020AppI{}, err
	}
	if len(results) == 0 {
		return model.Sb70020AppI{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetAppIPaginated(page int) (PageResponse[model.Sb70020AppI], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppI]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppI
	stmt := SELECT(table.Sb70020AppI.AllColumns).
		FROM(table.Sb70020AppI).
		ORDER_BY(table.Sb70020AppI.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppI]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppI]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppI)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppI]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppI]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}

func (r *repository) GetAppJByLIN(lin string) (model.Sb70020AppJ, error) {
	lin = strings.TrimSpace(lin)
	if lin == "" {
		return model.Sb70020AppJ{}, ErrEmptyParam
	}
	var results []model.Sb70020AppJ
	stmt := SELECT(table.Sb70020AppJ.AllColumns).
		FROM(table.Sb70020AppJ).
		WHERE(table.Sb70020AppJ.Lin.EQ(String(lin)))
	if err := stmt.Query(r.db, &results); err != nil {
		return model.Sb70020AppJ{}, err
	}
	if len(results) == 0 {
		return model.Sb70020AppJ{}, ErrNotFound
	}
	return results[0], nil
}

func (r *repository) GetAppJPaginated(page int) (PageResponse[model.Sb70020AppJ], error) {
	if page < 1 {
		return PageResponse[model.Sb70020AppJ]{}, ErrInvalidPage
	}
	offset := pageSize * int64(page-1)
	var items []model.Sb70020AppJ
	stmt := SELECT(table.Sb70020AppJ.AllColumns).
		FROM(table.Sb70020AppJ).
		ORDER_BY(table.Sb70020AppJ.Lin.ASC()).
		LIMIT(pageSize).
		OFFSET(offset)
	if err := stmt.Query(r.db, &items); err != nil {
		return PageResponse[model.Sb70020AppJ]{}, err
	}
	if len(items) == 0 {
		return PageResponse[model.Sb70020AppJ]{}, ErrNotFound
	}
	var dest struct {
		Count int64 `sql:"count"`
	}
	countStmt := SELECT(COUNT(Raw("*")).AS("count")).FROM(table.Sb70020AppJ)
	if err := countStmt.Query(r.db, &dest); err != nil {
		return PageResponse[model.Sb70020AppJ]{}, err
	}
	totalPages := int((dest.Count + pageSize - 1) / pageSize)
	return PageResponse[model.Sb70020AppJ]{
		Items: items, Count: len(items), Page: page,
		TotalPages: totalPages, IsLastPage: page >= totalPages,
	}, nil
}
