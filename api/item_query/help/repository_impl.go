package help

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) FindByCode(code string) ([]model.Help, error) {
	var rows []model.Help

	stmt := SELECT(
		Help.AllColumns,
	).FROM(Help).
		WHERE(Help.Code.EQ(String(code))).
		ORDER_BY(Help.Description.ASC())

	err := stmt.Query(repo.db, &rows)
	if err != nil || len(rows) == 0 {
		return []model.Help{}, ErrHelpNotFound
	}

	return rows, nil
}
