package pol_products

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/.gen/miltech_ng/public/table"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetPolProducts() (PolProductsResponse, error) {
	var products []model.PolProducts
	stmt := SELECT(
		table.PolProducts.AllColumns,
	).FROM(table.PolProducts)

	if err := stmt.Query(repo.db, &products); err != nil {
		return PolProductsResponse{}, err
	}

	return PolProductsResponse{
		Products: products,
		Count:    len(products),
	}, nil
}
