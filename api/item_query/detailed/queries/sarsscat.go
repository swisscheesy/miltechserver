package queries

import (
	"database/sql"

	"miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/details"

	. "github.com/go-jet/jet/v2/postgres"
)

func GetSarsscat(db *sql.DB, niin string) (details.Sarsscat, error) {
	sarsscat := details.Sarsscat{}

	sarsscatStmt := SELECT(
		table.ArmySarsscat.AllColumns,
	).FROM(table.ArmySarsscat).
		WHERE(table.ArmySarsscat.Niin.EQ(String(niin)))

	err := sarsscatStmt.Query(db, &sarsscat.ArmySarsscat)
	if err != nil {
		return details.Sarsscat{}, err
	}

	moeRuleStmt := SELECT(
		table.MoeRule.AllColumns,
	).FROM(table.MoeRule).
		WHERE(table.MoeRule.Niin.EQ(String(niin)))

	err = moeRuleStmt.Query(db, &sarsscat.MoeRule)
	if err != nil {
		return details.Sarsscat{}, err
	}

	amdfFreightStmt := SELECT(
		table.AmdfFreight.AllColumns,
	).FROM(table.AmdfFreight).
		WHERE(table.AmdfFreight.Niin.EQ(String(niin)))

	err = amdfFreightStmt.Query(db, &sarsscat.AmdfFreight)
	if err != nil {
		return details.Sarsscat{}, err
	}

	return sarsscat, nil
}
