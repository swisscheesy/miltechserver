package community

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{repository: repository}
}

func (service *ServiceImpl) Release(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	revisionID string,
	ifMatch string,
) (*ReleaseMutationResult, string, error) {
	ownerUID, apiError := authenticatedUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := parseUUID("checklist_id", checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedRevisionID, apiError := parseUUID("revision_id", revisionID)
	if apiError != nil {
		return nil, "", apiError
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	result, err := service.repository.Release(
		ctx,
		ownerUID,
		parsedChecklistID,
		parsedRevisionID,
		precondition,
	)
	return mutationResponse(result, err)
}

func (service *ServiceImpl) Retire(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	ifMatch string,
) (*ReleaseMutationResult, string, error) {
	ownerUID, apiError := authenticatedUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := parseUUID("checklist_id", checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	result, err := service.repository.Retire(
		ctx,
		ownerUID,
		parsedChecklistID,
		precondition,
	)
	return mutationResponse(result, err)
}

func authenticatedUID(user *bootstrap.User) (string, *shared.APIError) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return "", shared.NewAuthenticationRequired(
			"authentication is required",
			nil,
		)
	}
	return user.UserID, nil
}

func parseUUID(field string, value string) (uuid.UUID, *shared.APIError) {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, shared.NewInvalidRequest(
			field+" must be a non-zero UUID",
			map[string]any{"field": field},
		)
	}
	return parsed, nil
}

func mutationResponse(
	result *ReleaseMutationResult,
	err error,
) (*ReleaseMutationResult, string, error) {
	if err != nil {
		return nil, "", err
	}
	if result == nil {
		return nil, "", shared.NewInternalError(
			"repository returned an empty mutation result",
			nil,
		)
	}
	return result, shared.MakeChecklistETag(
		result.Aggregate.ID,
		result.Aggregate.SyncVersion,
	), nil
}
