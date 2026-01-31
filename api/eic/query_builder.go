package eic

func selectColumns() string {
	return `
SELECT
	inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
	publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
	publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
	publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
	publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
	publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
	publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
	publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
	pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf,
	eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
	wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2,
	uom1, uom2, uom3, mau1, uom4, mau2,
	warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb,
	curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt,
	array_agg(DISTINCT uoeic ORDER BY uoeic) as uoeic_array,
	array_agg(DISTINCT mrc ORDER BY mrc) as mrc_array,
	COUNT(*) as variant_count
FROM eic
`
}

func groupByColumns() string {
	return `
GROUP BY inc, fsc, niin, eic, lin, nomen, model, eicc, ecc, cmdtycd, reported, dahr,
	publvl1, pubno1, pubdate1, pubchg1, pubcgdt1,
	publcl2, pubno2, pubdate2, pubchg2, pubcgdt2,
	publvl3, pubno3, pubdate3, pubchg3, pubcgdt3,
	publvl4, pubno4, pubdate4, pubchg4, pubcgdt4,
	publvl5, pubno5, pubdate5, pubchg5, pubcgdt5,
	publvl6, pubno6, pubdate6, pubchg6, pubcgdt6,
	publvl7, pubno7, pubdate7, pubchg7, pubcgdt7,
	pubremks, eqpmcsa, eqpmcsb, eqpmcsc, eqpmcsd, eqpmcse, eqpmcsf,
	eqpmcsg, eqpmcsh, eqpmcsi, eqpmcsj, eqpmcsk, eqpmcsl,
	wpnrec, sernotrk, orf, aoap, gainloss, usage, urm1, urm2,
	uom1, uom2, uom3, mau1, uom4, mau2,
	warranty, rbm, sos, erc, eslvl, oslin, lcc, nounabb,
	curfmc, prevfmc, bstat1, bstat2, matcat, itemmgr, eos, sorts, status, lst_updt
`
}
