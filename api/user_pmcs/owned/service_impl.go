package owned

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/google/uuid"

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

func (service *ServiceImpl) Get(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
) (*shared.ChecklistAggregate, string, error) {
	ownerUID, apiError := authenticatedUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := parseUUID("checklist_id", checklistID)
	if apiError != nil {
		return nil, "", apiError
	}

	aggregate, err := service.repository.Get(
		ctx,
		ownerUID,
		parsedChecklistID,
	)
	if err != nil {
		return nil, "", err
	}
	return aggregate, shared.MakeChecklistETag(
		aggregate.ID,
		aggregate.SyncVersion,
	), nil
}

func (service *ServiceImpl) Create(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	draft shared.RevisionInput,
	ifNoneMatch string,
) (*MutationResult, string, error) {
	ownerUID, apiError := authenticatedUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := parseUUID("checklist_id", checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	precondition, err := shared.ParseCreatePrecondition(ifNoneMatch)
	if err != nil {
		return nil, "", err
	}
	prepared, err := service.prepareDraft(draft)
	if err != nil {
		return nil, "", err
	}

	result, err := service.repository.Create(
		ctx,
		ownerUID,
		parsedChecklistID,
		prepared,
		precondition,
	)
	return mutationResponse(result, err)
}

func (service *ServiceImpl) PutDraft(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	revisionID string,
	draft shared.RevisionInput,
	ifMatch string,
) (*MutationResult, string, error) {
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
	if draft.ID != parsedRevisionID {
		return nil, "", shared.NewInvalidRequest(
			"body revision ID must match the route revision ID",
			map[string]any{"revision_id": parsedRevisionID},
		)
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	prepared, err := service.prepareDraft(draft)
	if err != nil {
		return nil, "", err
	}

	result, err := service.repository.PutDraft(
		ctx,
		ownerUID,
		parsedChecklistID,
		prepared,
		precondition,
	)
	return mutationResponse(result, err)
}

func (service *ServiceImpl) DeleteDraft(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	revisionID string,
	ifMatch string,
) (*MutationResult, string, error) {
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

	result, err := service.repository.DeleteDraft(
		ctx,
		ownerUID,
		parsedChecklistID,
		parsedRevisionID,
		precondition,
	)
	return mutationResponse(result, err)
}

func (service *ServiceImpl) Publish(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	revisionID string,
	revision shared.RevisionInput,
	ifMatch string,
) (*MutationResult, string, error) {
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
	if revision.ID != parsedRevisionID {
		return nil, "", shared.NewInvalidRequest(
			"body revision ID must match the route revision ID",
			map[string]any{"revision_id": parsedRevisionID},
		)
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	if revision.RevisionNumber == nil || *revision.RevisionNumber <= 0 {
		return nil, "", shared.NewValidationFailed(
			"publication revision_number must be positive",
			map[string]any{"field": "revision.revision_number"},
		)
	}
	prepared, err := shared.PreparePublication(revision, service.config)
	if err != nil {
		return nil, "", err
	}

	result, err := service.repository.Publish(
		ctx,
		ownerUID,
		parsedChecklistID,
		prepared,
		precondition,
	)
	return mutationResponse(result, err)
}

func (service *ServiceImpl) GetRevision(
	ctx context.Context,
	user *bootstrap.User,
	checklistID string,
	revisionID string,
) (*HistoricalRevision, string, error) {
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
	result, err := service.repository.GetRevision(
		ctx,
		ownerUID,
		parsedChecklistID,
		parsedRevisionID,
	)
	if err != nil {
		return nil, "", err
	}
	return &result.Revision, immutableRevisionETag(
		parsedChecklistID,
		parsedRevisionID,
		result.ContentHash,
	), nil
}

func immutableRevisionETag(
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	contentHash [32]byte,
) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("owned-revision\x00"))
	_, _ = digest.Write(checklistID[:])
	_, _ = digest.Write(revisionID[:])
	_, _ = digest.Write(contentHash[:])
	return `"` + base64.RawURLEncoding.EncodeToString(digest.Sum(nil)) + `"`
}

func (service *ServiceImpl) prepareDraft(
	draft shared.RevisionInput,
) (shared.PreparedRevision, error) {
	if draft.RevisionNumber != nil {
		return shared.PreparedRevision{}, shared.NewValidationFailed(
			"draft revision_number must be null",
			map[string]any{"field": "revision.revision_number"},
		)
	}
	return shared.PrepareDraft(draft, service.config)
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
	result *MutationResult,
	err error,
) (*MutationResult, string, error) {
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
