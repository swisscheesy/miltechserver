package queries

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/qrm"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

// GetDisposition fetches disposition data.
// Single query - no parallelization needed.
// Treats "no rows" as a valid empty result.
func GetDisposition(ctx context.Context, db *sql.DB, niin string) (details.Disposition, error) {
	disposition := details.Disposition{}

	stmt := SELECT(table.Disposition.AllColumns).
		FROM(table.Disposition).
		WHERE(table.Disposition.Niin.EQ(String(niin)))

	err := stmt.QueryContext(ctx, db, &disposition.Disposition)
	if err != nil && err != qrm.ErrNoRows {
		return details.Disposition{}, err
	}

	return disposition, nil
}
