package cage

import (
	"miltechserver/.gen/miltech_ng/public/model"
	"strings"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) LookupByCode(cage string) ([]model.CageAddress, error) {
	return service.repo.SearchByCode(strings.ToUpper(cage))
}
