package pmcs_sbs_progress

import (
	"strings"

	"miltechserver/bootstrap"

	"github.com/google/uuid"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

const maxBulkDeleteFaults = 100
const defaultListInspectionsLimit = 1000
const maxNotesLength = 4000
const maxCommentTextLength = 2000
const maxShortFieldGraphemes = 200
const maxShortFieldBytes = 8 * 1024
const deletedCommentText = "Deleted by user"

func (service *ServiceImpl) EnsureInspection(user *bootstrap.User, equipmentID string, pmcsID string, req InspectionRequest) (*InspectionResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, user.UserID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.EnsureInspection(user, inspection)
	if err != nil {
		return nil, err
	}
	performedByUsername, err := service.resolvePerformedByUsername(user, saved.PerformedBy)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(*saved, performedByUsername, nil, nil)
	return &resp, nil
}

// resolvePerformedByUsername avoids a DB round trip in the common case: when
// the sticky performed_by owner is the caller themselves, their username is
// already on the auth token (bootstrap.User.Username). Only when a save
// touches an inspection whose sticky owner is a *different* user does this
// fall back to a single-row lookup.
func (service *ServiceImpl) resolvePerformedByUsername(user *bootstrap.User, performedBy *string) (*string, error) {
	if performedBy == nil {
		return nil, nil
	}
	if *performedBy == user.UserID {
		username := user.Username
		return &username, nil
	}
	return service.repository.LookupUsername(*performedBy)
}

func (service *ServiceImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID string) (*InspectionResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return nil, err
	}

	detail, faults, comments, err := service.repository.GetInspection(user, trimmedEquipmentID, parsedPmcsID)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(detail.UserPmcsInspections, detail.PerformedByUsername, faults, comments)
	return &resp, nil
}

func (service *ServiceImpl) ListInspections(user *bootstrap.User, equipmentID string, req ListInspectionsRequest) (*InspectionListResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}

	guideManual := strings.TrimSpace(req.GuideManual)
	if guideManual != "" {
		guideManual, err = validateGuideManual(guideManual)
		if err != nil {
			return nil, err
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultListInspectionsLimit
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	summaries, err := service.repository.ListInspections(user, trimmedEquipmentID, guideManual, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]InspectionSummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		responses = append(responses, InspectionSummaryResponse{
			ID:                   summary.ID,
			SourceType:           summary.SourceType,
			GuideManual:          summary.GuideManual,
			CustomChecklistID:    summary.CustomChecklistID,
			CustomRevisionID:     summary.CustomRevisionID,
			CustomRevisionNumber: summary.CustomRevisionNumber,
			CustomChecklistName:  summary.CustomChecklistName,
			PerformedDate:        summary.PerformedDate,
			FaultCount:           summary.FaultCount,
			CommentCount:         summary.CommentCount,
			CreatedAt:            summary.CreatedAt,
			PerformedBy:          summary.PerformedBy,
			PerformedByUsername:  summary.PerformedByUsername,
		})
	}
	return &InspectionListResponse{Inspections: responses, Count: len(responses)}, nil
}

func (service *ServiceImpl) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID string) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return err
	}
	return service.repository.DeleteInspection(user, trimmedEquipmentID, parsedPmcsID)
}

func (service *ServiceImpl) UpsertFault(user *bootstrap.User, equipmentID string, pmcsID string, req FaultRequest) (*FaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	inspection, fault, err := service.validateFaultRequest(equipmentID, pmcsID, user.UserID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertFault(user, inspection, fault)
	if err != nil {
		return nil, err
	}
	resp := mapFault(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteFault(user *bootstrap.User, equipmentID string, pmcsID string, req DeleteFaultRequest) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return err
	}
	key, err := service.validateDeleteFaultRequest(pmcsID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteFault(user, trimmedEquipmentID, key)
}

func (service *ServiceImpl) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (*BulkDeleteFaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, parsedPmcsID, keys, err := service.validateBulkDeleteFaultRequest(equipmentID, pmcsID, req)
	if err != nil {
		return nil, err
	}
	deletedCount, err := service.repository.DeleteFaults(user, trimmedEquipmentID, parsedPmcsID, keys)
	if err != nil {
		return nil, err
	}
	return &BulkDeleteFaultResponse{RequestedCount: len(keys), DeletedCount: int(deletedCount)}, nil
}

func (service *ServiceImpl) CreateComment(user *bootstrap.User, equipmentID string, pmcsID string, req CreateCommentRequest) (*CommentResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return nil, err
	}
	text, err := validateCommentText(req.Text)
	if err != nil {
		return nil, err
	}

	created, err := service.repository.CreateComment(user, trimmedEquipmentID, parsedPmcsID, text)
	if err != nil {
		return nil, err
	}
	resp := mapComment(*created)
	return &resp, nil
}

func (service *ServiceImpl) UpdateComment(user *bootstrap.User, equipmentID string, pmcsID string, commentID string, req UpdateCommentRequest) (*CommentResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return nil, err
	}
	parsedCommentID, err := uuid.Parse(strings.TrimSpace(commentID))
	if err != nil {
		return nil, ErrCommentNotFound
	}
	text, err := validateCommentText(req.Text)
	if err != nil {
		return nil, err
	}

	existing, err := service.repository.GetComment(user, trimmedEquipmentID, parsedPmcsID, parsedCommentID)
	if err != nil {
		return nil, err
	}
	if existing.AuthorID != user.UserID {
		return nil, ErrForbidden
	}

	updated, err := service.repository.UpdateComment(parsedCommentID, text)
	if err != nil {
		return nil, err
	}
	resp := mapComment(*updated)
	return &resp, nil
}

func (service *ServiceImpl) DeleteComment(user *bootstrap.User, equipmentID string, pmcsID string, commentID string) (*CommentResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return nil, err
	}
	parsedCommentID, err := uuid.Parse(strings.TrimSpace(commentID))
	if err != nil {
		return nil, ErrCommentNotFound
	}

	existing, err := service.repository.GetComment(user, trimmedEquipmentID, parsedPmcsID, parsedCommentID)
	if err != nil {
		return nil, err
	}
	if existing.AuthorID != user.UserID {
		return nil, ErrForbidden
	}

	updated, err := service.repository.UpdateComment(parsedCommentID, deletedCommentText)
	if err != nil {
		return nil, err
	}
	resp := mapComment(*updated)
	return &resp, nil
}

func hasAuthenticatedUser(user *bootstrap.User) bool {
	return user != nil && strings.TrimSpace(user.UserID) != ""
}
