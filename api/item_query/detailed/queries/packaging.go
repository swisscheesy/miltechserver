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

// GetArmyPackagingAndFreight fetches Army packaging and freight data using parallel queries.
// All 6 queries are independent and execute concurrently.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetArmyPackagingAndFreight(ctx context.Context, db *sql.DB, niin string) (details.ArmyPackagingAndFreight, error) {
	armyPackagingAndFreight := details.ArmyPackagingAndFreight{}
	var g errgroup.Group

	g.Go(func() error {
		stmt := SELECT(table.ArmyPackagingAndFreight.AllColumns).
			FROM(table.ArmyPackagingAndFreight).
			WHERE(table.ArmyPackagingAndFreight.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyPackagingAndFreight)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyPackaging1.AllColumns).
			FROM(table.ArmyPackaging1).
			WHERE(table.ArmyPackaging1.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyPackaging1)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyPackaging2.AllColumns).
			FROM(table.ArmyPackaging2).
			WHERE(table.ArmyPackaging2.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyPackaging2)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyPackagingSpecialInstruct.AllColumns).
			FROM(table.ArmyPackagingSpecialInstruct).
			WHERE(table.ArmyPackagingSpecialInstruct.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyPackSpecialInstruct)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyFreight.AllColumns).
			FROM(table.ArmyFreight).
			WHERE(table.ArmyFreight.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyFreight)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.ArmyPackSupplementalInstruct.AllColumns).
			FROM(table.ArmyPackSupplementalInstruct).
			WHERE(table.ArmyPackSupplementalInstruct.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &armyPackagingAndFreight.ArmyPackSupplementalInstruct)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.ArmyPackagingAndFreight{}, err
	}

	return armyPackagingAndFreight, nil
}

// GetPackaging fetches FLIS packaging data with parallel queries where possible.
// Phase 1: FlisPackaging1, FlisPackaging2, and DssWeightAndCube run in parallel.
// Phase 2 (conditional): CageAddress query if cage codes exist.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetPackaging(ctx context.Context, db *sql.DB, niin string) (details.Packaging, error) {
	packaging := details.Packaging{}
	var g errgroup.Group

	// Phase 1: Independent queries
	g.Go(func() error {
		stmt := SELECT(table.FlisPackaging1.AllColumns).
			FROM(table.FlisPackaging1).
			WHERE(table.FlisPackaging1.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &packaging.FlisPackaging1)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FlisPackaging2.AllColumns).
			FROM(table.FlisPackaging2).
			WHERE(table.FlisPackaging2.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &packaging.FlisPackaging2)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.DssWeightAndCube.AllColumns).
			FROM(table.DssWeightAndCube).
			WHERE(table.DssWeightAndCube.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &packaging.DssWeightAndCube)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Packaging{}, err
	}

	// Phase 2: Conditional query depends on FlisPackaging1 results
	var cageCodes string
	for _, ref := range packaging.FlisPackaging1 {
		if ref.PkgDesignActy != nil {
			if cageCodes != "" {
				cageCodes += ","
			}
			cageCodes += *ref.PkgDesignActy
		}
	}

	if cageCodes != "" {
		cageAddressStmt := SELECT(table.CageAddress.AllColumns).
			FROM(table.CageAddress).
			WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))

		err := cageAddressStmt.QueryContext(ctx, db, &packaging.CageAddress)
		if err != nil && err != qrm.ErrNoRows {
			return details.Packaging{}, err
		}
	}

	return packaging, nil
}
