package lin

import (
	"database/sql"
	"fmt"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/view"
	"miltechserver/api/item_lookup/shared"
	"miltechserver/api/response"
	"strings"
	"sync"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
)

const countCacheTTL = 15 * 24 * time.Hour

type RepositoryImpl struct {
	db         *sql.DB
	countMu    sync.RWMutex
	countCache int
	countSetAt time.Time
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) getCachedCount() (int, bool) {
	repo.countMu.RLock()
	defer repo.countMu.RUnlock()

	if repo.countCache > 0 && time.Since(repo.countSetAt) < countCacheTTL {
		return repo.countCache, true
	}
	return 0, false
}

func (repo *RepositoryImpl) setCachedCount(count int) {
	repo.countMu.Lock()
	defer repo.countMu.Unlock()

	repo.countCache = count
	repo.countSetAt = time.Now()
}

func (repo *RepositoryImpl) SearchByPage(page int) (response.LINPageResponse, error) {
	if page < 1 {
		return response.LINPageResponse{}, shared.ErrInvalidPage
	}

	var linData []model.LookupLinNiinMat
	offset := shared.CalculateOffset(page, shared.DefaultPageSize)
	stmt := SELECT(
		view.LookupLinNiinMat.AllColumns,
	).FROM(view.LookupLinNiinMat).
		ORDER_BY(view.LookupLinNiinMat.Lin.ASC(), view.LookupLinNiinMat.Niin.ASC()).
		LIMIT(shared.DefaultPageSize).
		OFFSET(offset)

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return response.LINPageResponse{}, fmt.Errorf("failed to query LIN data: %w", err)
	}

	totalCount, ok := repo.getCachedCount()
	if !ok {
		var count struct {
			Count int
		}

		countStmt := SELECT(
			COUNT(view.LookupLinNiinMat.Lin),
		).FROM(view.LookupLinNiinMat)

		err = countStmt.Query(repo.db, &count)
		if err != nil {
			return response.LINPageResponse{}, fmt.Errorf("failed to get total LIN count: %w", err)
		}

		totalCount = count.Count
		repo.setCachedCount(totalCount)
	}

	if len(linData) == 0 {
		return response.LINPageResponse{}, shared.ErrNotFound
	}

	totalPages := shared.CalculateTotalPages(totalCount, shared.DefaultPageSize)
	return response.LINPageResponse{
		Lins:       linData,
		Count:      totalCount,
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

func (repo *RepositoryImpl) SearchByNIIN(niin string) ([]model.LookupLinNiinMat, error) {
	if strings.TrimSpace(niin) == "" {
		return nil, shared.ErrEmptyParam
	}

	var linData []model.LookupLinNiinMat
	stmt := SELECT(
		view.LookupLinNiinMat.AllColumns).
		FROM(view.LookupLinNiinMat).
		WHERE(view.LookupLinNiinMat.Niin.LIKE(String("%" + niin + "%")))

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query LIN data by NIIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, shared.ErrNotFound
	}

	return linData, nil
}

func (repo *RepositoryImpl) SearchNIINByLIN(lin string) ([]model.LookupLinNiinMat, error) {
	if strings.TrimSpace(lin) == "" {
		return nil, shared.ErrEmptyParam
	}

	var linData []model.LookupLinNiinMat
	stmt := SELECT(
		view.LookupLinNiinMat.AllColumns).
		FROM(view.LookupLinNiinMat).
		WHERE(view.LookupLinNiinMat.Lin.LIKE(String("%" + lin + "%")))

	err := stmt.Query(repo.db, &linData)
	if err != nil {
		return nil, fmt.Errorf("failed to query NIIN data by LIN: %w", err)
	}

	if len(linData) == 0 {
		return nil, shared.ErrNotFound
	}

	return linData, nil
}
