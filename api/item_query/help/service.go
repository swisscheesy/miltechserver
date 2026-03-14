package help

import "miltechserver/.gen/miltech_ng/public/model"

type Service interface {
	FindByCode(code string) (model.Help, error)
}
