package short

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	FindShortByNiin(niin string) (model.NiinLookup, error)
	FindShortByPart(part string) ([]model.NiinLookup, error)
	// FindShortByNiinCancelled first searches niin_lookup by the given niin.
	// If no results are found it falls back to searching nsn.cancelled_niin,
	// then re-queries niin_lookup for each canonical NIIN found there.
	// Always returns a slice so the response shape is consistent regardless of
	// whether the primary or fallback path produced the results.
	FindShortByNiinCancelled(niin string) ([]model.NiinLookup, error)
}
