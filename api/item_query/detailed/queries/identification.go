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

// GetIdentification fetches identification data with parallel queries where possible.
// Phase 1: FlisManagementId, FlisStandardization, FlisCancelledNiin run in parallel.
// Phase 2 (conditional): ColloquialName query if Inc exists.
// Uses plain errgroup to prevent context cancellation on "no rows" errors.
func GetIdentification(ctx context.Context, db *sql.DB, niin string) (details.Identification, error) {
	identification := details.Identification{}
	var g errgroup.Group

	// FlisManagementId is needed for conditional query, but others can run in parallel
	g.Go(func() error {
		stmt := SELECT(table.FlisManagementID.AllColumns).
			FROM(table.FlisManagementID).
			WHERE(table.FlisManagementID.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &identification.FlisManagementId)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FlisStandardization.AllColumns).
			FROM(table.FlisStandardization).
			WHERE(table.FlisStandardization.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &identification.FlisStandardization)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	g.Go(func() error {
		stmt := SELECT(table.FlisCancelledNiin.AllColumns).
			FROM(table.FlisCancelledNiin).
			WHERE(table.FlisCancelledNiin.Niin.EQ(String(niin)))
		err := stmt.QueryContext(ctx, db, &identification.FlisCancelledNiin)
		if err != nil && err != qrm.ErrNoRows {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return details.Identification{}, err
	}

	// Conditional query - depends on FlisManagementId.Inc being populated
	if identification.FlisManagementId.Inc != nil {
		colloquialNamesStmt := SELECT(table.ColloquialName.AllColumns).
			FROM(table.ColloquialName).
			WHERE(table.ColloquialName.Inc.EQ(String(*identification.FlisManagementId.Inc)))

		err := colloquialNamesStmt.QueryContext(ctx, db, &identification.ColloquialName)
		if err != nil && err != qrm.ErrNoRows {
			return details.Identification{}, err
		}
	}

	return identification, nil
}
