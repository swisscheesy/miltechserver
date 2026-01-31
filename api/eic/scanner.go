package eic

import (
	"database/sql"

	"miltechserver/api/response"

	"github.com/lib/pq"
)

func scanConsolidatedItem(rows *sql.Rows) (response.EICConsolidatedItem, error) {
	var item response.EICConsolidatedItem
	err := rows.Scan(
		&item.Inc, &item.Fsc, &item.Niin, &item.Eic, &item.Lin, &item.Nomen, &item.Model,
		&item.Eicc, &item.Ecc, &item.Cmdtycd, &item.Reported, &item.Dahr,
		&item.Publvl1, &item.Pubno1, &item.Pubdate1, &item.Pubchg1, &item.Pubcgdt1,
		&item.Publcl2, &item.Pubno2, &item.Pubdate2, &item.Pubchg2, &item.Pubcgdt2,
		&item.Publvl3, &item.Pubno3, &item.Pubdate3, &item.Pubchg3, &item.Pubcgdt3,
		&item.Publvl4, &item.Pubno4, &item.Pubdate4, &item.Pubchg4, &item.Pubcgdt4,
		&item.Publvl5, &item.Pubno5, &item.Pubdate5, &item.Pubchg5, &item.Pubcgdt5,
		&item.Publvl6, &item.Pubno6, &item.Pubdate6, &item.Pubchg6, &item.Pubcgdt6,
		&item.Publvl7, &item.Pubno7, &item.Pubdate7, &item.Pubchg7, &item.Pubcgdt7,
		&item.Pubremks, &item.Eqpmcsa, &item.Eqpmcsb, &item.Eqpmcsc, &item.Eqpmcsd,
		&item.Eqpmcse, &item.Eqpmcsf, &item.Eqpmcsg, &item.Eqpmcsh, &item.Eqpmcsi,
		&item.Eqpmcsj, &item.Eqpmcsk, &item.Eqpmcsl, &item.Wpnrec, &item.Sernotrk,
		&item.Orf, &item.Aoap, &item.Gainloss, &item.Usage, &item.Urm1, &item.Urm2,
		&item.Uom1, &item.Uom2, &item.Uom3, &item.Mau1, &item.Uom4, &item.Mau2,
		&item.Warranty, &item.Rbm, &item.Sos, &item.Erc, &item.Eslvl, &item.Oslin,
		&item.Lcc, &item.Nounabb, &item.Curfmc, &item.Prevfmc, &item.Bstat1, &item.Bstat2,
		&item.Matcat, &item.Itemmgr, &item.Eos, &item.Sorts, &item.Status, &item.LstUpdt,
		pq.Array(&item.UoeicArray), pq.Array(&item.MrcArray), &item.VariantCount,
	)
	return item, err
}
