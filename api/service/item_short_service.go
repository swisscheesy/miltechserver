package service

import (
	"miltechserver/.gen/miltech_ng/public/model"
)

type ItemShortService interface {
	FindShortByNiin(niin string) (model.NiinLookup, error)
	FindShortByPart(part string) ([]model.NiinLookup, error)
}
