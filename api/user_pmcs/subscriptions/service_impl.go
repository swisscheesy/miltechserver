package subscriptions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

func NewService(repository Repository, config shared.Config) Service {
	return &ServiceImpl{repository: repository, config: config}
}

func (service *ServiceImpl) Install(ctx context.Context, user *bootstrap.User, checklistID, ifNoneMatch, ifMatch string) (*MutationResult, string, error) {
	uid, apiError := subscriptionUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedID, apiError := subscriptionUUID(checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	if strings.TrimSpace(ifNoneMatch) != "" && strings.TrimSpace(ifMatch) != "" {
		return nil, "", shared.NewInvalidPrecondition("only one conditional header may be supplied", nil)
	}
	var precondition shared.Precondition
	var err error
	if strings.TrimSpace(ifNoneMatch) != "" {
		precondition, err = shared.ParseCreatePrecondition(ifNoneMatch)
	} else {
		precondition, err = shared.ParseExistingPrecondition(ifMatch)
	}
	if err != nil {
		return nil, "", err
	}
	result, err := service.repository.Install(ctx, uid, parsedID, precondition)
	if err != nil {
		return nil, "", err
	}
	if result == nil {
		return nil, "", shared.NewInternalError("repository returned an empty mutation result", nil)
	}
	return result, shared.MakeSubscriptionETag(parsedID, result.Subscription.SyncVersion), nil
}

func (service *ServiceImpl) Unsubscribe(ctx context.Context, user *bootstrap.User, checklistID, ifMatch string) (*MutationResult, string, error) {
	uid, apiError := subscriptionUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedID, apiError := subscriptionUUID(checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	result, err := service.repository.Unsubscribe(ctx, uid, parsedID, precondition)
	if err != nil {
		return nil, "", err
	}
	if result == nil {
		return nil, "", shared.NewInternalError("repository returned an empty mutation result", nil)
	}
	return result, shared.MakeSubscriptionETag(parsedID, result.Subscription.SyncVersion), nil
}

func (service *ServiceImpl) GetInstalledRelease(ctx context.Context, user *bootstrap.User, checklistID, revisionID string) (*shared.InstalledChecklistRelease, string, error) {
	uid, apiError := subscriptionUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := subscriptionUUID(checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedRevisionID, apiError := subscriptionUUID(revisionID)
	if apiError != nil {
		return nil, "", apiError
	}
	release, err := service.repository.GetInstalledRelease(ctx, uid, parsedChecklistID, parsedRevisionID)
	if err != nil {
		return nil, "", err
	}
	if release == nil {
		return nil, "", shared.NewInternalError("repository returned an empty installed release", nil)
	}
	etag, err := installedReleaseETag(release)
	if err != nil {
		return nil, "", err
	}
	return release, etag, nil
}

func (service *ServiceImpl) ListUpdates(ctx context.Context, user *bootstrap.User, afterValue, limitValue string) (*shared.SubscriptionUpdatePage, error) {
	uid, apiError := subscriptionUID(user)
	if apiError != nil {
		return nil, apiError
	}
	limit := service.config.UpdatesDefaultLimit
	if strings.TrimSpace(limitValue) != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil || parsed <= 0 || parsed > service.config.UpdatesMaxLimit {
			return nil, shared.NewInvalidRequest(fmt.Sprintf("limit must be between 1 and %d", service.config.UpdatesMaxLimit), map[string]any{"limit": service.config.UpdatesMaxLimit})
		}
		limit = parsed
	}
	var after *uuid.UUID
	if strings.TrimSpace(afterValue) != "" {
		cursor, err := shared.DecodeSubscriptionUpdateCursor(afterValue)
		if err != nil {
			return nil, shared.NewInvalidRequest("invalid subscription update cursor", nil)
		}
		after = &cursor.Checklist
	}
	return service.repository.ListUpdates(ctx, uid, after, limit)
}

func (service *ServiceImpl) AcceptUpdate(ctx context.Context, user *bootstrap.User, checklistID, revisionID, ifMatch string) (*MutationResult, string, error) {
	uid, apiError := subscriptionUID(user)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedChecklistID, apiError := subscriptionUUID(checklistID)
	if apiError != nil {
		return nil, "", apiError
	}
	parsedRevisionID, apiError := subscriptionUUID(revisionID)
	if apiError != nil {
		return nil, "", apiError
	}
	precondition, err := shared.ParseExistingPrecondition(ifMatch)
	if err != nil {
		return nil, "", err
	}
	result, err := service.repository.AcceptUpdate(ctx, uid, parsedChecklistID, parsedRevisionID, precondition)
	if err != nil {
		return nil, "", err
	}
	if result == nil {
		return nil, "", shared.NewInternalError("repository returned an empty mutation result", nil)
	}
	return result, shared.MakeSubscriptionETag(parsedChecklistID, result.Subscription.SyncVersion), nil
}

func subscriptionUID(user *bootstrap.User) (string, *shared.APIError) {
	if user == nil || strings.TrimSpace(user.UserID) == "" {
		return "", shared.NewAuthenticationRequired("authentication is required", nil)
	}
	return user.UserID, nil
}
func subscriptionUUID(value string) (uuid.UUID, *shared.APIError) {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, shared.NewInvalidRequest("checklist_id must be a non-zero UUID", map[string]any{"field": "checklist_id"})
	}
	return parsed, nil
}
func installedReleaseETag(release *shared.InstalledChecklistRelease) (string, error) {
	representation, err := json.Marshal(release)
	if err != nil {
		return "", shared.NewInternalError("encode installed release ETag", nil)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("subscription-release-representation\x00"))
	_, _ = digest.Write(representation)
	return `"` + base64.RawURLEncoding.EncodeToString(digest.Sum(nil)) + `"`, nil
}
