package queries

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/qrm"
	"golang.org/x/sync/errgroup"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

// GetSarsscat fetches SARSSCAT data using parallel queries.
// All 3 queries are independent and execute concurrently.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetSarsscat(ctx context.Context, db *sql.DB, niin string) (details.Sarsscat, error) {
	sarsscat := details.Sarsscat{}
	var g errgroup.Group

	g.Go(func() error {
		stmt := SELECT(table.ArmySarsscat.AllColumns).
			FROM(table.ArmySarsscat).
			WHERE(table.ArmySarsscat.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &sarsscat.ArmySarsscat)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.MoeRule.AllColumns).
			FROM(table.MoeRule).
			WHERE(table.MoeRule.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &sarsscat.MoeRule)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfFreight.AllColumns).
			FROM(table.AmdfFreight).
			WHERE(table.AmdfFreight.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &sarsscat.AmdfFreight)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Sarsscat{}, err
	}

	return sarsscat, nil
}
