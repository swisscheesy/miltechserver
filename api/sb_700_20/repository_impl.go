package sb_700_20

import "database/sql"

const pageSize = int64(100)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}
