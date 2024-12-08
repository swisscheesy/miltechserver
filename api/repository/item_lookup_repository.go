package repository

type ItemLookupRepository interface {
	SearchLINByPage(page int) ([]string, error)
	//SearchSpecificLIN(lin string) ([]string, error)
	//
	//SearchUOCByPage(page int) ([]string, error)
	//SearchSpecificUOC(uoc string) ([]string, error)
}
