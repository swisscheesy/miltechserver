package cage

import (
	"database/sql"
	"fmt"
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/item_lookup/shared"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) SearchByCode(cage string) ([]model.CageAddress, error) {
	if strings.TrimSpace(cage) == "" {
		return nil, shared.ErrEmptyParam
	}

	var cageData []model.CageAddress
	stmt := SELECT(
		table.CageAddress.AllColumns,
	).FROM(table.CageAddress).
		WHERE(table.CageAddress.CageCode.EQ(String(cage)))

	err := stmt.Query(repo.db, &cageData)
	if err != nil {
		return nil, fmt.Errorf("failed to query CAGE address data: %w", err)
	}

	if len(cageData) == 0 {
		return nil, shared.ErrNotFound
	}

	return cageData, nil
}
