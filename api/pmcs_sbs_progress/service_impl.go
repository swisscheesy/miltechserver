package pmcs_sbs_progress

import (
	"fmt"
	"path"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
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
	resp := mapInspection(*saved, nil)
	return &resp, nil
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

	inspection, faults, err := service.repository.GetInspection(user, trimmedEquipmentID, parsedPmcsID)
	if err != nil {
		return nil, err
	}
	resp := mapInspection(*inspection, faults)
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
			ID:            summary.ID,
			GuideManual:   summary.GuideManual,
			PerformedDate: summary.PerformedDate,
			FaultCount:    summary.FaultCount,
			CreatedAt:     summary.CreatedAt,
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

func (service *ServiceImpl) validateInspectionRequest(equipmentID string, pmcsID string, userID string, req InspectionRequest) (model.PmcsSbsInspections, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	parsedPmcsID, err := validatePmcsID(pmcsID)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	guideManual, err := validateGuideManual(req.GuideManual)
	if err != nil {
		return model.PmcsSbsInspections{}, err
	}
	if req.PerformedDate.IsZero() {
		return model.PmcsSbsInspections{}, ErrInvalidRequest
	}

	createdBy := strings.TrimSpace(userID)
	return model.PmcsSbsInspections{
		ID:            parsedPmcsID,
		EquipmentID:   trimmedEquipmentID,
		GuideManual:   guideManual,
		PerformedDate: req.PerformedDate.UTC(),
		CreatedBy:     &createdBy,
	}, nil
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, pmcsID string, userID string, req FaultRequest) (model.PmcsSbsInspections, model.PmcsSbsFaults, error) {
	inspection, err := service.validateInspectionRequest(equipmentID, pmcsID, userID, InspectionRequest{
		GuideManual:   req.GuideManual,
		PerformedDate: req.PerformedDate,
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

	now := time.Now().UTC()
	fault := model.PmcsSbsFaults{
		PmcsID:           inspection.ID,
		SectionID:        sectionID,
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
		ItemIndex:        row.ItemIndex,
		ItemNo:           row.ItemNo,
		Status:           row.Status,
		FaultText:        row.FaultText,
		CorrectiveAction: row.CorrectiveAction,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func mapInspection(row model.PmcsSbsInspections, faultRows []model.PmcsSbsFaults) InspectionResponse {
	faults := make([]FaultResponse, 0, len(faultRows))
	for _, faultRow := range faultRows {
		faults = append(faults, mapFault(faultRow))
	}
	return InspectionResponse{
		ID:            row.ID,
		EquipmentID:   row.EquipmentID,
		GuideManual:   row.GuideManual,
		PerformedDate: row.PerformedDate,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Faults:        faults,
	}
}
