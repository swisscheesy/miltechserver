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

// GetAmdfData fetches AMDF (Army Master Data File) data using parallel queries.
// Executes 7 independent queries concurrently, with 1 conditional query if LIN exists.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetAmdfData(ctx context.Context, db *sql.DB, niin string) (details.Amdf, error) {
	amdf := details.Amdf{}
	var g errgroup.Group

	g.Go(func() error {
		stmt := SELECT(table.ArmyMasterDataFile.AllColumns).
			FROM(table.ArmyMasterDataFile).
			WHERE(table.ArmyMasterDataFile.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.ArmyMasterDataFile)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfManagement.AllColumns).
			FROM(table.AmdfManagement).
			WHERE(table.AmdfManagement.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfManagement)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfCredit.AllColumns).
			FROM(table.AmdfCredit).
			WHERE(table.AmdfCredit.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfCredit)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfBilling.AllColumns).
			FROM(table.AmdfBilling).
			WHERE(table.AmdfBilling.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfBilling)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfMatcat.AllColumns).
			FROM(table.AmdfMatcat).
			WHERE(table.AmdfMatcat.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfMatcat)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfPhrase.AllColumns).
			FROM(table.AmdfPhrase).
			WHERE(table.AmdfPhrase.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfPhrases)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.AmdfIAndS.AllColumns).
			FROM(table.AmdfIAndS).
			WHERE(table.AmdfIAndS.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &amdf.AmdfIandS)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Amdf{}, err
	}

	// Conditional query - depends on AmdfManagement.Lin being populated
	if amdf.AmdfManagement.Lin != nil {
		armyLinStmt := SELECT(table.ArmyLineItemNumber.AllColumns).
			FROM(table.ArmyLineItemNumber).
			WHERE(table.ArmyLineItemNumber.Lin.EQ(String(*amdf.AmdfManagement.Lin)))

		err := armyLinStmt.QueryContext(ctx, db, &amdf.ArmyLin)
		if err != nil && err != qrm.ErrNoRows {
			return details.Amdf{}, err
		}
	}

	return amdf, nil
}
