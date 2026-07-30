package community

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
	config     shared.Config
}

func NewService(repository Repository, configs ...shared.Config) Service {
	config := shared.DefaultConfig()
	if len(configs) > 0 {
		config = configs[0]
	}
	return &ServiceImpl{repository: repository, config: config}
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

func (service *ServiceImpl) Browse(
	ctx context.Context,
	after string,
	limit string,
	model string,
) (*shared.CommunityPage, error) {
	filter := shared.CommunityBrowseFilter{
		Limit: service.config.CommunityDefaultLimit,
	}
	if strings.TrimSpace(limit) != "" {
		parsedLimit, err := strconv.Atoi(limit)
		if err != nil || parsedLimit <= 0 ||
			parsedLimit > service.config.CommunityMaxLimit {
			return nil, shared.NewInvalidRequest(
				fmt.Sprintf(
					"limit must be between 1 and %d",
					service.config.CommunityMaxLimit,
				),
				map[string]any{"limit": service.config.CommunityMaxLimit},
			)
		}
		filter.Limit = parsedLimit
	}
	if strings.TrimSpace(after) != "" {
		cursor, err := shared.DecodeCommunityCursor(after)
		if err != nil {
			return nil, shared.NewInvalidRequest(
				"invalid community cursor",
				nil,
			)
		}
		filter.After = &cursor
	}
	if strings.TrimSpace(model) != "" {
		normalized, err := shared.NormalizeModel(model)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			return nil, shared.NewInvalidRequest(
				"model filter must contain text",
				nil,
			)
		}
		filter.NormalizedModel = normalized
	}
	return service.repository.Browse(ctx, filter)
}

func (service *ServiceImpl) GetCurrentRelease(
	ctx context.Context,
	checklistID string,
) (*shared.PublicChecklistRelease, string, error) {
	parsedChecklistID, apiError := parseUUID("checklist_id", checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	release, err := service.repository.GetCurrentRelease(
		ctx,
		parsedChecklistID,
	)
	if err != nil {
		return nil, "", err
	}
	if release == nil {
		return nil, "", shared.NewInternalError(
			"repository returned an empty public release",
			nil,
		)
	}
	contentHash, err := shared.CanonicalRevisionHash(
		revisionInput(release.Revision),
	)
	if err != nil {
		return nil, "", err
	}
	return release, makePublicReleaseETag(
		release.ChecklistID,
		release.Revision.ID,
		contentHash,
	), nil
}

func makePublicReleaseETag(
	checklistID uuid.UUID,
	revisionID uuid.UUID,
	contentHash [sha256.Size]byte,
) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte("community-release"))
	_, _ = digest.Write(checklistID[:])
	_, _ = digest.Write(revisionID[:])
	_, _ = digest.Write(contentHash[:])
	return `"` + base64.RawURLEncoding.EncodeToString(digest.Sum(nil)) + `"`
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
