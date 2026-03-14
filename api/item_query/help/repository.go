package help

import "miltechserver/.gen/miltech_ng/public/model"

type Repository interface {
	FindByCode(code string) ([]model.Help, error)
}
