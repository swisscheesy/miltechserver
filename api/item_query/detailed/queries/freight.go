package queries

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/qrm"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

// GetFreight fetches freight data.
// Single query - no parallelization needed.
// Treats "no rows" as a valid empty result.
func GetFreight(ctx context.Context, db *sql.DB, niin string) (details.Freight, error) {
	freight := details.Freight{}

	stmt := SELECT(table.FlisFreight.AllColumns).
		FROM(table.FlisFreight).
		WHERE(table.FlisFreight.Niin.EQ(String(niin)))

	err := stmt.QueryContext(ctx, db, &freight.FlisFreight)
	if err != nil && err != qrm.ErrNoRows {
		return details.Freight{}, err
	}

	return freight, nil
}
