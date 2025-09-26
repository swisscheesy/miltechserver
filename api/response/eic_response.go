package response

// EICConsolidatedItem represents a consolidated EIC record where duplicate entries
// are merged with varying fields (uoeic, mrc) collected into arrays.
// This reduces response size and eliminates client-side duplicate processing.
type EICConsolidatedItem struct {
	Inc      *string `json:"inc"`
	Fsc      *string `json:"fsc"`
	Niin     string  `json:"niin"`
	Eic      *string `json:"eic"`
	Lin      *string `json:"lin"`
	Nomen    *string `json:"nomen"`
	Model    *string `json:"model"`
	Eicc     *string `json:"eicc"`
	Ecc      *string `json:"ecc"`
	Cmdtycd  *string `json:"cmdtycd"`
	Reported *string `json:"reported"`
	Dahr     *string `json:"dahr"`
	Publvl1  *string `json:"publvl1"`
	Pubno1   *string `json:"pubno1"`
	Pubdate1 *string `json:"pubdate1"`
	Pubchg1  *string `json:"pubchg1"`
	Pubcgdt1 *string `json:"pubcgdt1"`
	Publcl2  *string `json:"publcl2"`
	Pubno2   *string `json:"pubno2"`
	Pubdate2 *string `json:"pubdate2"`
	Pubchg2  *string `json:"pubchg2"`
	Pubcgdt2 *string `json:"pubcgdt2"`
	Publvl3  *string `json:"publvl3"`
	Pubno3   *string `json:"pubno3"`
	Pubdate3 *string `json:"pubdate3"`
	Pubchg3  *string `json:"pubchg3"`
	Pubcgdt3 *string `json:"pubcgdt3"`
	Publvl4  *string `json:"publvl4"`
	Pubno4   *string `json:"pubno4"`
	Pubdate4 *string `json:"pubdate4"`
	Pubchg4  *string `json:"pubchg4"`
	Pubcgdt4 *string `json:"pubcgdt4"`
	Publvl5  *string `json:"publvl5"`
	Pubno5   *string `json:"pubno5"`
	Pubdate5 *string `json:"pubdate5"`
	Pubchg5  *string `json:"pubchg5"`
	Pubcgdt5 *string `json:"pubcgdt5"`
	Publvl6  *string `json:"publvl6"`
	Pubno6   *string `json:"pubno6"`
	Pubdate6 *string `json:"pubdate6"`
	Pubchg6  *string `json:"pubchg6"`
	Pubcgdt6 *string `json:"pubcgdt6"`
	Publvl7  *string `json:"publvl7"`
	Pubno7   *string `json:"pubno7"`
	Pubdate7 *string `json:"pubdate7"`
	Pubchg7  *string `json:"pubchg7"`
	Pubcgdt7 *string `json:"pubcgdt7"`
	Pubremks *string `json:"pubremks"`
	Eqpmcsa  *string `json:"eqpmcsa"`
	Eqpmcsb  *string `json:"eqpmcsb"`
	Eqpmcsc  *string `json:"eqpmcsc"`
	Eqpmcsd  *string `json:"eqpmcsd"`
	Eqpmcse  *string `json:"eqpmcse"`
	Eqpmcsf  *string `json:"eqpmcsf"`
	Eqpmcsg  *string `json:"eqpmcsg"`
	Eqpmcsh  *string `json:"eqpmcsh"`
	Eqpmcsi  *string `json:"eqpmcsi"`
	Eqpmcsj  *string `json:"eqpmcsj"`
	Eqpmcsk  *string `json:"eqpmcsk"`
	Eqpmcsl  *string `json:"eqpmcsl"`
	Wpnrec   *string `json:"wpnrec"`
	Sernotrk *string `json:"sernotrk"`
	Orf      *string `json:"orf"`
	Aoap     *string `json:"aoap"`
	Gainloss *string `json:"gainloss"`
	Usage    *string `json:"usage"`
	Urm1     *string `json:"urm1"`
	Urm2     *string `json:"urm2"`
	Uom1     *string `json:"uom1"`
	Uom2     *string `json:"uom2"`
	Uom3     *string `json:"uom3"`
	Mau1     *string `json:"mau1"`
	Uom4     *string `json:"uom4"`
	Mau2     *string `json:"mau2"`
	Warranty *string `json:"warranty"`
	Rbm      *string `json:"rbm"`
	Sos      *string `json:"sos"`
	Erc      *string `json:"erc"`
	Eslvl    *string `json:"eslvl"`
	Oslin    *string `json:"oslin"`
	Lcc      *string `json:"lcc"`
	Nounabb  *string `json:"nounabb"`
	Curfmc   *string `json:"curfmc"`
	Prevfmc  *string `json:"prevfmc"`
	Bstat1   *string `json:"bstat1"`
	Bstat2   *string `json:"bstat2"`
	Matcat   *string `json:"matcat"`
	Itemmgr  *string `json:"itemmgr"`
	Eos      *string `json:"eos"`
	Sorts    *string `json:"sorts"`
	Status   *string `json:"status"`
	LstUpdt  *string `json:"lst_updt"`

	// Consolidated fields - arrays of values that vary across duplicates
	UoeicArray   []string `json:"uoeic_array"`
	MrcArray     []string `json:"mrc_array"`
	VariantCount int      `json:"variant_count"`
}

// EICPageResponse represents the response structure for paginated EIC queries.
// Used for GET /api/eic/items and GET /api/eic/fsc/{fsc} endpoints.
// \param Items - the consolidated EIC data retrieved from the database.
// \param Count - the total count of EIC items on this page.
// \param Page - the current page number.
// \param TotalPages - the total number of pages.
// \param IsLastPage - indicates if this is the last page.
type EICPageResponse struct {
	Items      []EICConsolidatedItem `json:"items"`
	Count      int                   `json:"count"`
	Page       int                   `json:"page"`
	TotalPages int                   `json:"total_pages"`
	IsLastPage bool                  `json:"is_last_page"`
}

// EICSearchResponse represents the response structure for non-paginated EIC queries.
// Used for GET /api/eic/niin/{niin} and GET /api/eic/lin/{lin} endpoints.
// \param Count - the total count of consolidated EIC items found.
// \param Items - the consolidated EIC data retrieved from the database.
type EICSearchResponse struct {
	Count int                   `json:"count"`
	Items []EICConsolidatedItem `json:"items"`
}