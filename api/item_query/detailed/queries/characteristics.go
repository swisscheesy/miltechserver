package queries

import (
	"context"
	"database/sql"

	"github.com/go-jet/jet/v2/qrm"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

// GetCharacteristics fetches item characteristics data.
// Single query - no parallelization needed.
// Treats "no rows" as a valid empty result.
func GetCharacteristics(ctx context.Context, db *sql.DB, niin string) (details.Characteristics, error) {
	characteristics := details.Characteristics{}

	stmt := SELECT(table.FlisItemCharacteristics.AllColumns).
		FROM(table.FlisItemCharacteristics).
		WHERE(table.FlisItemCharacteristics.Niin.EQ(String(niin)))

	err := stmt.QueryContext(ctx, db, &characteristics.Characteristics)
	if err != nil && err != qrm.ErrNoRows {
		return details.Characteristics{}, err
	}

	return characteristics, nil
}
