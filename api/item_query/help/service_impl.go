package help

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

type ServiceImpl struct {
	repo Repository
}

func NewService(repo Repository) *ServiceImpl {
	return &ServiceImpl{repo: repo}
}

func (service *ServiceImpl) FindByCode(code string) (model.Help, error) {
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	if normalizedCode == "" {
		return model.Help{}, ErrInvalidCode
	}

	rows, err := service.repo.FindByCode(normalizedCode)
	if err != nil {
		return model.Help{}, err
	}

	// Multiple rows may exist for the same code. Return the first
	// deterministic row based on repository ordering.
	return rows[0], nil
}
