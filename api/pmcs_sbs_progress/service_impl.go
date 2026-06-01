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

const maxBatchCompletionChanges = 100

func NewService(repository Repository) *ServiceImpl {
	return &ServiceImpl{repository: repository}
}

type validatedEquipment struct {
	ID              uuid.UUID
	EquipmentManual string
	Admin           string
	Serial          string
	Uic             string
}

func (service *ServiceImpl) ListEquipment(user *bootstrap.User) (*EquipmentListResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	rows, err := service.repository.ListEquipment(user)
	if err != nil {
		return nil, err
	}
	equipment := make([]EquipmentResponse, 0, len(rows))
	for _, row := range rows {
		equipment = append(equipment, mapEquipment(row))
	}
	return &EquipmentListResponse{Equipment: equipment, Count: len(equipment)}, nil
}

func (service *ServiceImpl) GetEquipment(user *bootstrap.User, equipmentID string) (*EquipmentAggregateResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	if _, err := uuid.Parse(strings.TrimSpace(equipmentID)); err != nil {
		return nil, ErrInvalidID
	}
	aggregate, err := service.repository.GetEquipmentAggregate(user, strings.TrimSpace(equipmentID))
	if err != nil {
		return nil, err
	}
	return mapAggregate(*aggregate), nil
}

func (service *ServiceImpl) UpsertEquipment(user *bootstrap.User, equipmentID string, req EquipmentRequest) (*EquipmentResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	validated, err := service.validateEquipmentRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := model.PmcsSbsEquipment{
		ID:              validated.ID,
		UserUID:         user.UserID,
		EquipmentManual: validated.EquipmentManual,
		Admin:           validated.Admin,
		Serial:          validated.Serial,
		Uic:             validated.Uic,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	saved, err := service.repository.UpsertEquipment(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapEquipment(*saved)
	return &resp, nil
}

func (service *ServiceImpl) DeleteEquipment(user *bootstrap.User, equipmentID string) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	trimmedID := strings.TrimSpace(equipmentID)
	if _, err := uuid.Parse(trimmedID); err != nil {
		return ErrInvalidID
	}
	return service.repository.DeleteEquipment(user, trimmedID)
}

func (service *ServiceImpl) UpsertCompletion(user *bootstrap.User, equipmentID string, req CompletionRequest) (*CompletionResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	row, err := service.validateCompletionRequest(equipmentID, req)
	if err != nil {
		return nil, err
	}
	saved, err := service.repository.UpsertCompletion(user, row)
	if err != nil {
		return nil, err
	}
	resp := mapCompletion(*saved)
	return &resp, nil
}

func (service *ServiceImpl) BatchCompletions(user *bootstrap.User, equipmentID string, req BatchCompletionsRequest) (*BatchCompletionsResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	upserts, deletes, err := service.buildBatchCompletionsChangeSet(equipmentID, req)
	if err != nil {
		return nil, err
	}
	result, err := service.repository.BatchCompletions(user, strings.TrimSpace(equipmentID), upserts, deletes)
	if err != nil {
		return nil, err
	}
	return &BatchCompletionsResponse{
		UpsertedCount: result.UpsertedCount,
		DeletedCount:  result.DeletedCount,
	}, nil
}

func (service *ServiceImpl) DeleteCompletion(user *bootstrap.User, equipmentID string, req DeleteCompletionRequest) error {
	if !hasAuthenticatedUser(user) {
		return ErrUnauthorized
	}
	key, err := service.validateDeleteCompletionRequest(equipmentID, req)
	if err != nil {
		return err
	}
	return service.repository.DeleteCompletion(user, key.EquipmentID, key.SectionID, key.ItemIndex, key.StepID)
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
	return service.repository.DeleteFault(user, key.EquipmentID, key.SectionID, key.ItemIndex)
}

func (service *ServiceImpl) Sync(user *bootstrap.User, req SyncRequest) (*SyncResponse, error) {
	if !hasAuthenticatedUser(user) {
		return nil, ErrUnauthorized
	}
	if err := service.validateSyncRequest(req); err != nil {
		return nil, err
	}
	changeSet, err := service.buildSyncChangeSet(user, req)
	if err != nil {
		return nil, err
	}
	result, err := service.repository.Sync(user, changeSet)
	if err != nil {
		return nil, err
	}
	resp := &SyncResponse{DeletedEquipmentIDs: result.DeletedEquipmentIDs}
	for _, aggregate := range result.Equipment {
		resp.Equipment = append(resp.Equipment, *mapAggregate(aggregate))
	}
	return resp, nil
}

func (service *ServiceImpl) validateEquipmentRequest(equipmentID string, req EquipmentRequest) (validatedEquipment, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return validatedEquipment{}, ErrInvalidID
	}
	equipmentManual := strings.TrimSpace(req.EquipmentManual)
	if !isValidEquipmentManual(equipmentManual) {
		return validatedEquipment{}, ErrInvalidBlobPath
	}
	admin := strings.TrimSpace(req.Admin)
	if admin == "" {
		return validatedEquipment{}, ErrInvalidRequest
	}
	return validatedEquipment{
		ID:              id,
		EquipmentManual: equipmentManual,
		Admin:           admin,
		Serial:          strings.TrimSpace(req.Serial),
		Uic:             strings.TrimSpace(req.Uic),
	}, nil
}

func (service *ServiceImpl) validateCompletionRequest(equipmentID string, req CompletionRequest) (model.PmcsSbsCompletions, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return model.PmcsSbsCompletions{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	itemNo := strings.TrimSpace(req.ItemNo)
	stepID := strings.TrimSpace(req.StepID)
	if sectionID == "" || itemNo == "" || stepID == "" || req.ItemIndex < 0 {
		return model.PmcsSbsCompletions{}, ErrInvalidRequest
	}
	return model.PmcsSbsCompletions{
		EquipmentID: id,
		SectionID:   sectionID,
		ItemIndex:   req.ItemIndex,
		ItemNo:      itemNo,
		StepID:      stepID,
		IsComplete:  true,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (service *ServiceImpl) validateDeleteCompletionRequest(equipmentID string, req DeleteCompletionRequest) (CompletionKey, error) {
	trimmedID := strings.TrimSpace(equipmentID)
	if _, err := uuid.Parse(trimmedID); err != nil {
		return CompletionKey{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	stepID := strings.TrimSpace(req.StepID)
	if sectionID == "" || stepID == "" || req.ItemIndex < 0 {
		return CompletionKey{}, ErrInvalidRequest
	}
	return CompletionKey{
		EquipmentID: trimmedID,
		SectionID:   sectionID,
		ItemIndex:   req.ItemIndex,
		StepID:      stepID,
	}, nil
}

func (service *ServiceImpl) buildBatchCompletionsChangeSet(equipmentID string, req BatchCompletionsRequest) ([]model.PmcsSbsCompletions, []CompletionKey, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return nil, nil, ErrInvalidID
	}
	canonicalEquipmentID := id.String()

	totalChanges := len(req.UpsertCompletions) + len(req.DeleteCompletions)
	if totalChanges > maxBatchCompletionChanges {
		return nil, nil, ErrInvalidRequest
	}

	upserts := make([]model.PmcsSbsCompletions, 0, len(req.UpsertCompletions))
	upsertKeys := map[string]struct{}{}
	for _, completion := range req.UpsertCompletions {
		row, err := service.validateCompletionRequest(canonicalEquipmentID, completion)
		if err != nil {
			return nil, nil, err
		}
		key := completionKey(row.EquipmentID.String(), row.SectionID, row.ItemIndex, row.StepID)
		if _, duplicate := upsertKeys[key]; duplicate {
			return nil, nil, ErrInvalidSyncRequest
		}
		upserts = append(upserts, row)
		upsertKeys[key] = struct{}{}
	}

	deletes := make([]CompletionKey, 0, len(req.DeleteCompletions))
	for _, completion := range req.DeleteCompletions {
		key, err := service.validateDeleteCompletionRequest(canonicalEquipmentID, completion)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := upsertKeys[completionKey(key.EquipmentID, key.SectionID, key.ItemIndex, key.StepID)]; duplicate {
			return nil, nil, ErrInvalidSyncRequest
		}
		deletes = append(deletes, key)
	}

	return upserts, deletes, nil
}

func (service *ServiceImpl) validateFaultRequest(equipmentID string, req FaultRequest) (model.PmcsSbsFaults, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return model.PmcsSbsFaults{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
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
		EquipmentID:      id,
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
	trimmedID := strings.TrimSpace(equipmentID)
	if _, err := uuid.Parse(trimmedID); err != nil {
		return FaultKey{}, ErrInvalidID
	}
	sectionID := strings.TrimSpace(req.SectionID)
	if sectionID == "" || req.ItemIndex < 0 {
		return FaultKey{}, ErrInvalidRequest
	}
	return FaultKey{EquipmentID: trimmedID, SectionID: sectionID, ItemIndex: req.ItemIndex}, nil
}

func (service *ServiceImpl) validateSyncRequest(req SyncRequest) error {
	deletedEquipment := map[string]struct{}{}
	for _, equipmentID := range req.DeleteEquipmentIDs {
		canonicalID, err := canonicalSyncID(equipmentID)
		if err != nil {
			return err
		}
		deletedEquipment[canonicalID] = struct{}{}
	}
	for _, equipment := range req.UpsertEquipment {
		canonicalID, err := canonicalSyncID(equipment.ID)
		if err != nil {
			return err
		}
		if _, deleted := deletedEquipment[canonicalID]; deleted {
			return ErrInvalidSyncRequest
		}
	}

	completionUpserts := map[string]struct{}{}
	for _, completion := range req.UpsertCompletions {
		canonicalID, err := canonicalSyncID(completion.EquipmentID)
		if err != nil {
			return err
		}
		if _, deleted := deletedEquipment[canonicalID]; deleted {
			return ErrInvalidSyncRequest
		}
		completionUpserts[completionKey(canonicalID, completion.SectionID, completion.ItemIndex, completion.StepID)] = struct{}{}
	}
	for _, completion := range req.DeleteCompletions {
		canonicalID, err := canonicalSyncID(completion.EquipmentID)
		if err != nil {
			return err
		}
		key := completionKey(canonicalID, completion.SectionID, completion.ItemIndex, completion.StepID)
		if _, duplicate := completionUpserts[key]; duplicate {
			return ErrInvalidSyncRequest
		}
	}

	faultUpserts := map[string]struct{}{}
	for _, fault := range req.UpsertFaults {
		canonicalID, err := canonicalSyncID(fault.EquipmentID)
		if err != nil {
			return err
		}
		if _, deleted := deletedEquipment[canonicalID]; deleted {
			return ErrInvalidSyncRequest
		}
		faultUpserts[faultKey(canonicalID, fault.SectionID, fault.ItemIndex)] = struct{}{}
	}
	for _, fault := range req.DeleteFaults {
		canonicalID, err := canonicalSyncID(fault.EquipmentID)
		if err != nil {
			return err
		}
		key := faultKey(canonicalID, fault.SectionID, fault.ItemIndex)
		if _, duplicate := faultUpserts[key]; duplicate {
			return ErrInvalidSyncRequest
		}
	}
	return nil
}

func (service *ServiceImpl) buildSyncChangeSet(user *bootstrap.User, req SyncRequest) (SyncChangeSet, error) {
	changeSet := SyncChangeSet{
		DeleteEquipmentIDs: make([]string, 0, len(req.DeleteEquipmentIDs)),
	}
	for _, equipmentID := range req.DeleteEquipmentIDs {
		changeSet.DeleteEquipmentIDs = append(changeSet.DeleteEquipmentIDs, strings.TrimSpace(equipmentID))
	}
	for _, equipment := range req.UpsertEquipment {
		validated, err := service.validateEquipmentRequest(equipment.ID, EquipmentRequest{
			EquipmentManual: equipment.EquipmentManual,
			Admin:           equipment.Admin,
			Serial:          equipment.Serial,
			Uic:             equipment.Uic,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		now := time.Now().UTC()
		changeSet.UpsertEquipment = append(changeSet.UpsertEquipment, model.PmcsSbsEquipment{
			ID:              validated.ID,
			UserUID:         user.UserID,
			EquipmentManual: validated.EquipmentManual,
			Admin:           validated.Admin,
			Serial:          validated.Serial,
			Uic:             validated.Uic,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}
	for _, completion := range req.UpsertCompletions {
		row, err := service.validateCompletionRequest(completion.EquipmentID, CompletionRequest{
			SectionID: completion.SectionID,
			ItemIndex: completion.ItemIndex,
			ItemNo:    completion.ItemNo,
			StepID:    completion.StepID,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.UpsertCompletions = append(changeSet.UpsertCompletions, row)
	}
	for _, completion := range req.DeleteCompletions {
		key, err := service.validateDeleteCompletionRequest(completion.EquipmentID, DeleteCompletionRequest{
			SectionID: completion.SectionID,
			ItemIndex: completion.ItemIndex,
			StepID:    completion.StepID,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.DeleteCompletions = append(changeSet.DeleteCompletions, key)
	}
	for _, fault := range req.UpsertFaults {
		row, err := service.validateFaultRequest(fault.EquipmentID, FaultRequest{
			SectionID:        fault.SectionID,
			ItemIndex:        fault.ItemIndex,
			ItemNo:           fault.ItemNo,
			Status:           fault.Status,
			FaultText:        fault.FaultText,
			CorrectiveAction: fault.CorrectiveAction,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.UpsertFaults = append(changeSet.UpsertFaults, row)
	}
	for _, fault := range req.DeleteFaults {
		key, err := service.validateDeleteFaultRequest(fault.EquipmentID, DeleteFaultRequest{
			SectionID: fault.SectionID,
			ItemIndex: fault.ItemIndex,
		})
		if err != nil {
			return SyncChangeSet{}, err
		}
		changeSet.DeleteFaults = append(changeSet.DeleteFaults, key)
	}
	return changeSet, nil
}

func hasAuthenticatedUser(user *bootstrap.User) bool {
	return user != nil && strings.TrimSpace(user.UserID) != ""
}

func isValidEquipmentManual(blobPath string) bool {
	cleaned := path.Clean(blobPath)
	return cleaned == blobPath &&
		strings.HasPrefix(cleaned, "pmcs_sbs/") &&
		strings.HasSuffix(strings.ToLower(cleaned), ".json")
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

func canonicalSyncID(equipmentID string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(equipmentID))
	if err != nil {
		return "", ErrInvalidID
	}
	return id.String(), nil
}

func completionKey(equipmentID string, sectionID string, itemIndex int32, stepID string) string {
	return fmt.Sprintf("%s|%s|%d|%s", strings.TrimSpace(equipmentID), strings.TrimSpace(sectionID), itemIndex, strings.TrimSpace(stepID))
}

func faultKey(equipmentID string, sectionID string, itemIndex int32) string {
	return fmt.Sprintf("%s|%s|%d", strings.TrimSpace(equipmentID), strings.TrimSpace(sectionID), itemIndex)
}

func mapEquipment(row model.PmcsSbsEquipment) EquipmentResponse {
	return EquipmentResponse{
		ID:              row.ID.String(),
		EquipmentManual: row.EquipmentManual,
		Admin:           row.Admin,
		Serial:          row.Serial,
		Uic:             row.Uic,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func mapCompletion(row model.PmcsSbsCompletions) CompletionResponse {
	return CompletionResponse{
		EquipmentID: row.EquipmentID.String(),
		SectionID:   row.SectionID,
		ItemIndex:   row.ItemIndex,
		ItemNo:      row.ItemNo,
		StepID:      row.StepID,
		IsComplete:  row.IsComplete,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapFault(row model.PmcsSbsFaults) FaultResponse {
	return FaultResponse{
		EquipmentID:      row.EquipmentID.String(),
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

func mapAggregate(row EquipmentAggregate) *EquipmentAggregateResponse {
	resp := &EquipmentAggregateResponse{Equipment: mapEquipment(row.Equipment)}
	for _, completion := range row.Completions {
		resp.Completions = append(resp.Completions, mapCompletion(completion))
	}
	for _, fault := range row.Faults {
		resp.Faults = append(resp.Faults, mapFault(fault))
	}
	return resp
}
