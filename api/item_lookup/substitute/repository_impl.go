package substitute

import (
	"database/sql"
	"fmt"

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

func (repo *RepositoryImpl) SearchAll() ([]model.ArmySubstituteLin, error) {
	var substituteData []model.ArmySubstituteLin
	stmt := SELECT(
		table.ArmySubstituteLin.AllColumns,
	).FROM(table.ArmySubstituteLin)

	err := stmt.Query(repo.db, &substituteData)
	if err != nil {
		return nil, fmt.Errorf("failed to query substitute LIN data: %w", err)
	}

	if len(substituteData) == 0 {
		return nil, shared.ErrNotFound
	}

	return substituteData, nil
}
