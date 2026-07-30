package sync

import (
	"context"
	"strconv"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
	config     shared.Config
}

func NewService(repository Repository, config shared.Config) Service {
	return &ServiceImpl{repository: repository, config: config}
}

func (service *ServiceImpl) GetDelta(
	ctx context.Context,
	user *bootstrap.User,
	afterValue string,
	limitValue string,
) (*AccountDelta, error) {
	if user == nil || user.UserID == "" {
		return nil, shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}

	after, apiError := parseNonnegativeInt64("after", afterValue, 0)
	if apiError != nil {
		return nil, apiError
	}
	limit, apiError := parseLimit(limitValue, service.config)
	if apiError != nil {
		return nil, apiError
	}

	return service.repository.GetDelta(
		ctx,
		user.UserID,
		after,
		limit,
		service.config.MaxDeltaResponseBytes,
	)
}

func parseNonnegativeInt64(
	name string,
	value string,
	defaultValue int64,
) (int64, *shared.APIError) {
	if value == "" {
		return defaultValue, nil
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, invalidPagination(name)
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, invalidPagination(name)
	}
	return parsed, nil
}

func parseLimit(
	value string,
	config shared.Config,
) (int, *shared.APIError) {
	if value == "" {
		return config.DeltaDefaultLimit, nil
	}
	parsed, apiError := parseNonnegativeInt64("limit", value, 0)
	if apiError != nil || parsed == 0 || parsed > int64(config.DeltaMaxLimit) {
		return 0, invalidPagination("limit")
	}
	return int(parsed), nil
}

func invalidPagination(name string) *shared.APIError {
	return shared.NewInvalidRequest(
		"invalid account delta pagination",
		map[string]any{"field": name},
	)
}
