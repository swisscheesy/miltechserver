package short

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	ShortItemSearchNiin(niin string) (model.NiinLookup, error)
	ShortItemSearchPart(part string) ([]model.NiinLookup, error)
}
