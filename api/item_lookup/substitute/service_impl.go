package substitute

import "miltechserver/.gen/miltech_ng/public/model"

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) LookupAll() ([]model.ArmySubstituteLin, error) {
	return service.repo.SearchAll()
}
