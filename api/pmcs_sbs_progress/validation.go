package pmcs_sbs_progress

import (
	"fmt"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"miltechserver/.gen/miltech_ng/public/model"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/google/uuid"
)

func (service *ServiceImpl) validateInspectionRequest(equipmentID string, pmcsID string, userID string, req InspectionRequest) (model.UserPmcsInspections, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.UserPmcsInspections{}, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return model.UserPmcsInspections{}, err
	}
	source, err := normalizeInspectionSource(req.InspectionSourceRequest)
	if err != nil {
		return model.UserPmcsInspections{}, err
	}
	if req.PerformedDate.IsZero() {
		return model.UserPmcsInspections{}, ErrInvalidRequest
	}

	notes, err := validateNotes(req.Notes)
	if err != nil {
		return model.UserPmcsInspections{}, err
	}

	performedBy := strings.TrimSpace(userID)
	return model.UserPmcsInspections{
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

func (service *ServiceImpl) validateFaultRequest(equipmentID string, pmcsID string, userID string, req FaultRequest) (model.UserPmcsInspections, model.UserPmcsFaults, error) {
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, userID, InspectionRequest{
		InspectionSourceRequest: req.InspectionSourceRequest,
		PerformedDate:           req.PerformedDate,
	})
	if err != nil {
		return model.UserPmcsInspections{}, model.UserPmcsFaults{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	status, validStatus := normalizeFaultStatus(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.UserPmcsInspections{}, model.UserPmcsFaults{}, ErrInvalidRequest
	}
	if !validStatus {
		return model.UserPmcsInspections{}, model.UserPmcsFaults{}, ErrInvalidStatus
	}
	sectionTitle, err := validateOptionalShortField(req.SectionTitle)
	if err != nil {
		return model.UserPmcsInspections{}, model.UserPmcsFaults{}, err
	}

	now := time.Now().UTC()
	fault := model.UserPmcsFaults{
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
