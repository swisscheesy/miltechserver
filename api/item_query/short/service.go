package short

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	FindShortByNiin(niin string) (model.NiinLookup, error)
	FindShortByPart(part string) ([]model.NiinLookup, error)
}
