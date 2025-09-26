package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"miltechserver/api/response"
	"strings"

	"github.com/lib/pq"
)

var eicReturnCount = int64(40)

type EICRepositoryImpl struct {
	Db *sql.DB
}


func NewEICRepositoryImpl(db *sql.DB) *EICRepositoryImpl {
	return &EICRepositoryImpl{Db: db}
}

// GetByNIIN retrieves consolidated EIC records by National Item Identification Number.
// Duplicate records with the same NIIN are consolidated with UOEIC and MRC values aggregated into arrays.
// \param niin - the NIIN to search for.
// \return a slice of EICConsolidatedItem containing the consolidated EIC data.
// \return an error if the operation fails.
func (repo *EICRepositoryImpl) GetByNIIN(niin string) ([]response.EICConsolidatedItem, error) {
	if strings.TrimSpace(niin) == "" {
		return nil, errors.New("niin cannot be empty")
	}

	var consolidatedData []response.EICConsolidatedItem

	// Use direct SQL for aggregation query since go-jet has issues with array_agg
	query := `
	SELECT
		inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
		array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
		array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
		COUNT(*) as variant_count
	FROM eic
	WHERE niin = $1
	GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	`

	rows, err := repo.Db.Query(query, strings.TrimSpace(niin))
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by NIIN: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by NIIN: %w", err)
	}

	if len(consolidatedData) == 0 {
		return nil, errors.New("no EIC items found for the specified NIIN")
	}

	return consolidatedData, nil
}

// GetByLIN retrieves consolidated EIC records by Line Item Number.
// Duplicate records with the same LIN are consolidated with UOEIC and MRC values aggregated into arrays.
// \param lin - the LIN to search for.
// \return a slice of EICConsolidatedItem containing the consolidated EIC data.
// \return an error if the operation fails.
func (repo *EICRepositoryImpl) GetByLIN(lin string) ([]response.EICConsolidatedItem, error) {
	if strings.TrimSpace(lin) == "" {
		return nil, errors.New("lin cannot be empty")
	}

	var consolidatedData []response.EICConsolidatedItem

	// Use direct SQL for aggregation query
	query := `
	SELECT
		inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
		array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
		array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
		COUNT(*) as variant_count
	FROM eic
	WHERE lin = $1
	GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	`

	rows, err := repo.Db.Query(query, strings.TrimSpace(lin))
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by LIN: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("failed to query consolidated EIC data by LIN: %w", err)
	}

	if len(consolidatedData) == 0 {
		return nil, errors.New("no EIC items found for the specified LIN")
	}

	return consolidatedData, nil
}

// GetByFSCPaginated retrieves consolidated EIC records by Federal Supply Class with pagination.
// Duplicate records are consolidated with UOEIC and MRC values aggregated into arrays.
// \param fsc - the FSC to search for.
// \param page - the page number to retrieve.
// \return an EICPageResponse containing the consolidated EIC data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *EICRepositoryImpl) GetByFSCPaginated(fsc string, page int) (response.EICPageResponse, error) {
	if strings.TrimSpace(fsc) == "" {
		return response.EICPageResponse{}, errors.New("fsc cannot be empty")
	}

	if page < 1 {
		return response.EICPageResponse{}, errors.New("page number must be greater than 0")
	}

	fscTrimmed := strings.TrimSpace(fsc)
	var consolidatedData []response.EICConsolidatedItem
	offset := eicReturnCount * int64(page-1)

	// Use direct SQL for consolidation with pagination
	query := `
	SELECT
		inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
		array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
		array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
		COUNT(*) as variant_count
	FROM eic
	WHERE fsc = $1
	GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	LIMIT $2 OFFSET $3
	`

	rows, err := repo.Db.Query(query, fscTrimmed, eicReturnCount, offset)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data by FSC: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
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
		if err != nil {
			return response.EICPageResponse{}, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	err = rows.Err()
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data by FSC: %w", err)
	}

	// Get total count of consolidated records (not individual records)
	countQuery := `
	SELECT COUNT(*) FROM (
		SELECT 1
		FROM eic
		WHERE fsc = $1
		GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
			publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
			publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
			publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
			publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
			publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
			publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
			publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
			pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
			wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
			warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	) AS consolidated_count
	`

	var totalCount int
	err = repo.Db.QueryRow(countQuery, fscTrimmed).Scan(&totalCount)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to get total consolidated count for FSC: %w", err)
	}

	if len(consolidatedData) == 0 {
		return response.EICPageResponse{}, errors.New("no EIC items found for the specified FSC")
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(eicReturnCount)))
	return response.EICPageResponse{
		Items:      consolidatedData,
		Count:      len(consolidatedData),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}

// GetAllPaginated retrieves all consolidated EIC records with optional search and pagination.
// Duplicate records are consolidated with UOEIC and MRC values aggregated into arrays.
// \param page - the page number to retrieve.
// \param search - optional search term to filter across all text fields.
// \return an EICPageResponse containing the consolidated EIC data, count, page, total pages, and whether it is the last page.
// \return an error if the operation fails.
func (repo *EICRepositoryImpl) GetAllPaginated(page int, search string) (response.EICPageResponse, error) {
	if page < 1 {
		return response.EICPageResponse{}, errors.New("page number must be greater than 0")
	}

	var consolidatedData []response.EICConsolidatedItem
	offset := eicReturnCount * int64(page-1)
	searchTerm := strings.TrimSpace(search)

	// Build consolidated query with optional search
	var whereClause string
	var args []interface{}
	argIndex := 1

	if searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		whereClause = `WHERE niin ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR lin ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR fsc ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR nomen ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR model ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR eic ILIKE $` + fmt.Sprintf("%d", argIndex) + `
			OR EXISTS (SELECT 1 FROM unnest(array_agg(DISTINCT uoeic)) AS u(val) WHERE u.val ILIKE $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	query := `
	SELECT
		inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
		array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
		array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
		COUNT(*) as variant_count
	FROM eic
	` + whereClause + `
	GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
		publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
		publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
		publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
		publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
		publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
		publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
		publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
		pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
		wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
		warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, eicReturnCount, offset)

	rows, err := repo.Db.Query(query, args...)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
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
		if err != nil {
			return response.EICPageResponse{}, fmt.Errorf("failed to scan consolidated EIC data: %w", err)
		}
		consolidatedData = append(consolidatedData, item)
	}

	err = rows.Err()
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to query consolidated EIC data: %w", err)
	}

	// Get total count of consolidated records
	countQuery := `
	SELECT COUNT(*) FROM (
		SELECT 1
		FROM eic
		` + whereClause + `
		GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
			publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
			publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
			publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
			publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
			publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
			publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
			publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
			pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf, eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
			wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2, uom1, uom2, uom3, mau1, uom4, mau2,
			warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb, curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
	) AS consolidated_count
	`

	var countArgs []interface{}
	if searchTerm != "" {
		countArgs = []interface{}{searchTerm}
	}

	var totalCount int
	err = repo.Db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return response.EICPageResponse{}, fmt.Errorf("failed to get total consolidated count: %w", err)
	}

	if len(consolidatedData) == 0 {
		return response.EICPageResponse{}, errors.New("no EIC items found for the specified criteria")
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(eicReturnCount)))
	return response.EICPageResponse{
		Items:      consolidatedData,
		Count:      len(consolidatedData),
		Page:       page,
		TotalPages: totalPages,
		IsLastPage: page >= totalPages,
	}, nil
}
