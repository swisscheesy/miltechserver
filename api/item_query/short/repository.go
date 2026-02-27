package short

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	ShortItemSearchNiin(niin string) (model.NiinLookup, error)
	ShortItemSearchPart(part string) ([]model.NiinLookup, error)
	// ShortItemSearchCancelledNiin returns NSN rows whose cancelled_niin column
	// contains the given niin substring. Used as a fallback lookup path.
	ShortItemSearchCancelledNiin(niin string) ([]model.Nsn, error)
}
