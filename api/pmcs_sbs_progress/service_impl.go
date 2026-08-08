package pmcs_sbs_progress

import (
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"

	"github.com/clipperhouse/uax29/v2/graphemes"
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
	resp := mapInspection(detail.PmcsSbsInspections, detail.PerformedByUsername, faults, comments)
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

func (service *ServiceImpl) validateInspectionRequest(equipmentID string, pmcsID string, userID string, req InspectionRequest) (model.PmcsSbsInspections, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	source, err := normalizeInspectionSource(req.InspectionSourceRequest)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	if req.PerformedDate.IsZero() {
		return model.PmcsSbsInspections{}, ErrInvalidRequest
	}

	notes, err := validateNotes(req.Notes)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}

	performedBy := strings.TrimSpace(userID)
	return model.PmcsSbsInspections{
		ID:                   parsedPmcsID,
		EquipmentID:          trimmedEquipmentID,
		SourceType:           source.SourceType,
		GuideManual:          source.GuideManual,
		CustomChecklistID:    source.CustomChecklistID,
		CustomRevisionID:     source.CustomRevisionID,
		CustomRevisionNumber: source.CustomRevisionNumber,
		CustomChecklistName:  source.CustomChecklistName,
		PerformedDate:        req.PerformedDate.UTC(),
		PerformedBy:          &performedBy,
		Notes:                notes,
	}, nil
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, pmcsID string, userID string, req FaultRequest) (model.PmcsSbsInspections, model.PmcsSbsFaults, error) {
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, userID, InspectionRequest{
		InspectionSourceRequest: req.InspectionSourceRequest,
		PerformedDate:           req.PerformedDate,
	})
	if err != nil {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	status, validStatus := normalizeFaultStatus(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, ErrInvalidRequest
	}
	if !validStatus {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, ErrInvalidStatus
	}
	sectionTitle, err := validateOptionalShortField(req.SectionTitle)
	if err != nil {
		return model.PmcsSbsInspections{}, model.PmcsSbsFaults{}, err
	}

	now := time.Now().UTC()
	fault := model.PmcsSbsFaults{
		PmcsID:           inspection.ID,
		SectionID:        sectionID,
		SectionTitle:     sectionTitle,
		ItemIndex:        req.ItemIndex,
		ItemNo:           itemNo,
		Status:           status,
		FaultText:        faultText,
		CorrectiveAction: strings.TrimSpace(req.CorrectiveAction),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return inspection, fault, nil
}

func (service *ServiceImpl) validateDeleteFaultRequest(pmcsID string, req DeleteFaultRequest) (FaultKey, error) {
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return FaultKey{}, err
	}
	sectionID := strings.TrimSpace(req.SectionID)
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{PmcsID: parsedPmcsID, SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func (service *ServiceImpl) validateBulkDeleteFaultRequest(equipmentID string, pmcsID string, req BulkDeleteFaultRequest) (string, uuid.UUID, []FaultKey, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return "", uuid.UUID{}, nil, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return "", uuid.UUID{}, nil, err
	}
	if len(req.Faults) == 0 || len(req.Faults) > maxBulkDeleteFaults {
		return "", uuid.UUID{}, nil, ErrInvalidRequest
	}
	keys := make([]FaultKey, 0, len(req.Faults))
	seen := make(map[string]struct{}, len(req.Faults))
	for _, fault := range req.Faults {
		sectionID := strings.TrimSpace(fault.SectionID)
		if sectionID == "" || fault.ItemIndex < 0 {
			return "", uuid.UUID{}, nil, ErrInvalidRequest
		}
		duplicateKey := fmt.Sprintf("%s\x00%d", sectionID, fault.ItemIndex)
		if _, exists := seen[duplicateKey]; exists {
			return "", uuid.UUID{}, nil, ErrInvalidRequest
		}
		seen[duplicateKey] = struct{}{}
		keys = append(keys, FaultKey{PmcsID: parsedPmcsID, SectionID: sectionID, ItemIndex: fault.ItemIndex})
	}
	return trimmedEquipmentID, parsedPmcsID, keys, nil
}

func validateEquipmentID(equipmentID string) (string, error) {
	trimmedEquipmentID := strings.TrimSpace(equipmentID)
	if trimmedEquipmentID == "" {
		return "", ErrInvalidID
	}
	return trimmedEquipmentID, nil
}

func validatePmcsID(pmcsID string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(pmcsID)
	if trimmed == "" {
		return uuid.UUID{}, ErrInvalidPmcsID
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, ErrInvalidPmcsID
	}
	return parsed, nil
}

func validateGuideManual(guideManual string) (string, error) {
	trimmedGuideManual := strings.TrimSpace(guideManual)
	if trimmedGuideManual == "" ||
		strings.Contains(trimmedGuideManual, "\\") ||
		!strings.HasPrefix(trimmedGuideManual, "pmcs_sbs/") ||
		!strings.HasSuffix(trimmedGuideManual, ".json") ||
		path.Clean(trimmedGuideManual) != trimmedGuideManual {
		return "", ErrInvalidGuideManual
	}
	return trimmedGuideManual, nil
}

func normalizeInspectionSource(req InspectionSourceRequest) (ValidatedInspectionSource, error) {
	if !utf8.ValidString(req.SourceType) ||
		!utf8.ValidString(req.GuideManual) ||
		!utf8.ValidString(req.CustomChecklistID) ||
		!utf8.ValidString(req.CustomRevisionID) ||
		!utf8.ValidString(req.CustomChecklistName) {
		return ValidatedInspectionSource{}, ErrInvalidRequest
	}

	sourceType := req.SourceType
	hasCustomFields := req.CustomChecklistID != "" ||
		req.CustomRevisionID != "" ||
		req.CustomRevisionNumber != nil ||
		req.CustomChecklistName != ""

	switch sourceType {
	case "":
		if strings.TrimSpace(req.GuideManual) == "" || hasCustomFields {
			return ValidatedInspectionSource{}, ErrInvalidRequest
		}
		return normalizeGuideInspectionSource(req.GuideManual)
	case "guide":
		if hasCustomFields {
			return ValidatedInspectionSource{}, ErrInvalidRequest
		}
		return normalizeGuideInspectionSource(req.GuideManual)
	case "custom":
		if req.GuideManual != "" {
			return ValidatedInspectionSource{}, ErrInvalidRequest
		}
		return normalizeCustomInspectionSource(req)
	default:
		return ValidatedInspectionSource{}, ErrInvalidRequest
	}
}

func normalizeGuideInspectionSource(guideManual string) (ValidatedInspectionSource, error) {
	validatedGuideManual, err := validateGuideManual(guideManual)
	if err != nil {
		return ValidatedInspectionSource{}, err
	}
	return ValidatedInspectionSource{
		SourceType:  "guide",
		GuideManual: &validatedGuideManual,
	}, nil
}

func normalizeCustomInspectionSource(req InspectionSourceRequest) (ValidatedInspectionSource, error) {
	checklistID, err := uuid.Parse(strings.TrimSpace(req.CustomChecklistID))
	if err != nil || checklistID == uuid.Nil {
		return ValidatedInspectionSource{}, ErrInvalidRequest
	}
	revisionID, err := uuid.Parse(strings.TrimSpace(req.CustomRevisionID))
	if err != nil || revisionID == uuid.Nil {
		return ValidatedInspectionSource{}, ErrInvalidRequest
	}
	if req.CustomRevisionNumber == nil || *req.CustomRevisionNumber < 0 {
		return ValidatedInspectionSource{}, ErrInvalidRequest
	}
	checklistName, err := validateRequiredShortField(req.CustomChecklistName)
	if err != nil {
		return ValidatedInspectionSource{}, err
	}

	return ValidatedInspectionSource{
		SourceType:           "custom",
		CustomChecklistID:    &checklistID,
		CustomRevisionID:     &revisionID,
		CustomRevisionNumber: req.CustomRevisionNumber,
		CustomChecklistName:  &checklistName,
	}, nil
}

func validateRequiredShortField(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrInvalidRequest
	}
	if err := validateShortField(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func validateOptionalShortField(value string) (*string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	if err := validateShortField(trimmed); err != nil {
		return nil, err
	}
	return &trimmed, nil
}

func validateShortField(value string) error {
	if !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 || len(value) > maxShortFieldBytes {
		return ErrInvalidRequest
	}
	graphemeCount := 0
	iterator := graphemes.FromString(value)
	for iterator.Next() {
		graphemeCount++
		if graphemeCount > maxShortFieldGraphemes {
			return ErrInvalidRequest
		}
	}
	return nil
}

func validateNotes(notes *string) (*string, error) {
	if notes == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*notes)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxNotesLength {
		return nil, ErrInvalidRequest
	}
	return &trimmed, nil
}

func validateCommentText(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || len(trimmed) > maxCommentTextLength {
		return "", ErrInvalidCommentText
	}
	return trimmed, nil
}

func hasAuthenticatedUser(user *bootstrap.User) bool {
	return user != nil && strings.TrimSpace(user.UserID) != ""
}

func normalizeFaultStatus(status string) (string, bool) {
	switch strings.TrimSpace(status) {
	case "X", "x":
		return "x", true
	case "/", "slash":
		return "slash", true
	case "-", "dash":
		return "dash", true
	default:
		return "", false
	}
}

func mapFault(row model.PmcsSbsFaults) FaultResponse {
	return FaultResponse{
		PmcsID:           row.PmcsID,
		SectionID:        row.SectionID,
		SectionTitle:     row.SectionTitle,
		ItemIndex:        row.ItemIndex,
		ItemNo:           row.ItemNo,
		Status:           row.Status,
		FaultText:        row.FaultText,
		CorrectiveAction: row.CorrectiveAction,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapInspection(row model.PmcsSbsInspections, performedByUsername *string, faultRows []model.PmcsSbsFaults, commentRows []CommentWithAuthor) InspectionResponse {
	faults := make([]FaultResponse, 0, len(faultRows))
	for _, faultRow := range faultRows {
		faults = append(faults, mapFault(faultRow))
	}
	comments := make([]CommentResponse, 0, len(commentRows))
	for _, commentRow := range commentRows {
		comments = append(comments, mapComment(commentRow))
	}
	return InspectionResponse{
		ID:                   row.ID,
		EquipmentID:          row.EquipmentID,
		SourceType:           row.SourceType,
		GuideManual:          row.GuideManual,
		CustomChecklistID:    row.CustomChecklistID,
		CustomRevisionID:     row.CustomRevisionID,
		CustomRevisionNumber: row.CustomRevisionNumber,
		CustomChecklistName:  row.CustomChecklistName,
		PerformedDate:        row.PerformedDate,
		PerformedBy:          row.PerformedBy,
		PerformedByUsername:  performedByUsername,
		Notes:                row.Notes,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		Faults:               faults,
		Comments:             comments,
	}
}

func mapComment(row CommentWithAuthor) CommentResponse {
	return CommentResponse{
		ID:             row.ID,
		PmcsID:         row.PmcsID,
		AuthorID:       row.AuthorID,
		AuthorUsername: row.AuthorUsername,
		Text:           row.Text,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
