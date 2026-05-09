package tmde

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

type service struct {
	repository Repository
}

func NewService(repo Repository) Service {
	return &service{repository: repo}
}

func (s *service) LookupByNIIN(niin string) (model.TmdeIntervalMat, error) {
	normalized := strings.TrimSpace(strings.ToUpper(niin))
	return s.repository.GetByNIIN(normalized)
}

func (s *service) GetAllPaginated(page int) (TmdePageResponse, error) {
	return s.repository.GetAllPaginated(page)
}
