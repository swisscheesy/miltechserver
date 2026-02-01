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

// GetManagement fetches management data from 8 different tables using parallel queries.
// All queries are independent and execute concurrently.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetManagement(ctx context.Context, db *sql.DB, niin string) (details.Management, error) {
	management := details.Management{}
	var g errgroup.Group

	g.Go(func() error {
		stmt := SELECT(table.FlisManagement.AllColumns).
			FROM(table.FlisManagement).
			WHERE(table.FlisManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.FLisManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FlisPhrase.AllColumns).
			FROM(table.FlisPhrase).
			WHERE(table.FlisPhrase.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.FlisPhrase)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ComponentEndItem.AllColumns).
			FROM(table.ComponentEndItem).
			WHERE(table.ComponentEndItem.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.ComponentEndItem)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyManagement.AllColumns).
			FROM(table.ArmyManagement).
			WHERE(table.ArmyManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.ArmyManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AirForceManagement.AllColumns).
			FROM(table.AirForceManagement).
			WHERE(table.AirForceManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.AirForceManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.MarineCorpsManagement.AllColumns).
			FROM(table.MarineCorpsManagement).
			WHERE(table.MarineCorpsManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.MarineCorpsManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.NavyManagement.AllColumns).
			FROM(table.NavyManagement).
			WHERE(table.NavyManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.NavyManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FaaManagement.AllColumns).
			FROM(table.FaaManagement).
			WHERE(table.FaaManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &management.FaaManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Management{}, err
	}

	return management, nil
}
