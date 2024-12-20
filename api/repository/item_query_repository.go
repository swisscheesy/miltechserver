package repository

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type ItemQueryRepository interface {
	ShortItemSearchNiin(niin string) (model.NiinLookup, error)
	ShortItemSearchPart(part string) ([]model.NiinLookup, error)
}
