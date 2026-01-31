package substitute

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	LookupAll() ([]model.ArmySubstituteLin, error)
}
