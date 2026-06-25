package pmcs_sbs_progress

import (
	"path"
	"strings"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/bootstrap"
)

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

func (service *ServiceImpl) ListFaults(user *bootstrap.User, equipmentID string, guideManual string) (*FaultListResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return nil, err
	}
	trimmedGuideManual, err := validateGuideManual(guideManual)
	if err != nil {
		return nil, err
	}

	rows, err := service.repository.ListFaults(user, trimmedEquipmentID, trimmedGuideManual)
	if err != nil {
		return nil, err
	}

	faults := make([]FaultResponse, 0, len(rows))
	for _, row := range rows {
		faults = append(faults, mapFault(row))
	}
	return &FaultListResponse{Faults: faults, Count: len(faults)}, nil
}

func (service *ServiceImpl) UpsertFault(user *bootstrap.User, equipmentID string, req FaultRequest) (*FaultResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	row, err := service.validateFaultRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertFault(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapFault(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteFault(user *bootstrap.User, equipmentID string, req DeleteFaultRequest) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	key, err := service.validateDeleteFaultRequest(equipmentID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteFault(user, key)
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, req FaultRequest) (model.PmcsSbsFaults, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return model.PmcsSbsFaults{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	guideManual, err := validateGuideManual(req.GuideManual)
	if err != nil {
		return model.PmcsSbsFaults{}, err
	}
	itemNo := strings.TrimSpace(req.ItemNo)
	status, validStatus := normalizeFaultStatus(req.Status)
	faultText := strings.TrimSpace(req.FaultText)
	if sectionID == "" || itemNo == "" || req.ItemIndex < 0 || faultText == "" {
		return model.PmcsSbsFaults{}, ErrInvalidRequest
	}
	if !validStatus {
		return model.PmcsSbsFaults{}, ErrInvalidStatus
	}

	now := time.Now().UTC()
	return model.PmcsSbsFaults{
		EquipmentID:      trimmedEquipmentID,
		GuideManual:      guideManual,
		SectionID:        sectionID,
		ItemIndex:        req.ItemIndex,
		ItemNo:           itemNo,
		Status:           status,
		FaultText:        faultText,
		CorrectiveAction: strings.TrimSpace(req.CorrectiveAction),
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (service *ServiceImpl) validateDeleteFaultRequest(equipmentID string, req DeleteFaultRequest) (FaultKey, error) {
	trimmedEquipmentID, err := validateEquipmentID(equipmentID)
	if err != nil {
		return FaultKey{}, err
	}

	sectionID := strings.TrimSpace(req.SectionID)
	guideManual, err := validateGuideManual(req.GuideManual)
	if err != nil {
		return FaultKey{}, err
	}
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{EquipmentID: trimmedEquipmentID, GuideManual: guideManual, SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func validateEquipmentID(equipmentID string) (string, error) {
	trimmedEquipmentID := strings.TrimSpace(equipmentID)
	if trimmedEquipmentID == "" {
		return "", ErrInvalidID
	}
	return trimmedEquipmentID, nil
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
		EquipmentID:      row.EquipmentID,
		GuideManual:      row.GuideManual,
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
