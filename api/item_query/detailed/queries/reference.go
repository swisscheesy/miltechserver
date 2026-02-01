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

// GetReference fetches reference data with parallel queries where possible.
// Phase 1: FlisIdentification and FlisReference run in parallel.
// Phase 2 (conditional): CageAddresses and CageStatusAndType if cage codes exist.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetReference(ctx context.Context, db *sql.DB, niin string) (details.Reference, error) {
	reference := details.Reference{}
	var g errgroup.Group

	// Phase 1: Independent queries
	g.Go(func() error {
		stmt := SELECT(table.FlisIdentification.AllColumns).
			FROM(table.FlisIdentification).
			WHERE(table.FlisIdentification.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &reference.FlisReference)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FlisReference.AllColumns).
			FROM(table.FlisReference).
			WHERE(table.FlisReference.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &reference.ReferenceAndPartNumber)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Reference{}, err
	}

	// Build cage codes list from results
	var cageCodes string
	for _, ref := range reference.ReferenceAndPartNumber {
		if cageCodes != "" {
			cageCodes += ","
		}
		cageCodes += ref.CageCode
	}

	// Phase 2: Conditional queries run in parallel if cage codes exist
	if cageCodes != "" {
		var g2 errgroup.Group

		g2.Go(func() error {
			stmt := SELECT(table.CageAddress.AllColumns).
				FROM(table.CageAddress).
				WHERE(table.CageAddress.CageCode.IN(String(cageCodes)))
			err := stmt.QueryContext(ctx, db, &reference.CageAddresses)
			if err != nil && err != qrm.ErrNoRows {
				return err
			}
			return nil
		})

		g2.Go(func() error {
			stmt := SELECT(table.CageStatusAndType.AllColumns).
				FROM(table.CageStatusAndType).
				WHERE(table.CageStatusAndType.CageCode.IN(String(cageCodes)))
			err := stmt.QueryContext(ctx, db, &reference.CageStatusAndType)
			if err != nil && err != qrm.ErrNoRows {
				return err
			}
			return nil
		})

		if err := g2.Wait(); err != nil {
			return details.Reference{}, err
		}
	}

	return reference, nil
}
