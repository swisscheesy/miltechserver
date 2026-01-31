package substitute

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	SearchAll() ([]model.ArmySubstituteLin, error)
}
