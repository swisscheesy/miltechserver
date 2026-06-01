package pmcs_sbs_progress

import "time"

type EquipmentRequest struct {
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type CompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	ItemNo    string `json:"item_no"`
	StepID    string `json:"step_id"`
}

type DeleteCompletionRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
	StepID    string `json:"step_id"`
}

type BatchCompletionsRequest struct {
	UpsertCompletions []CompletionRequest       `json:"upsert_completions"`
	DeleteCompletions []DeleteCompletionRequest `json:"delete_completions"`
}

type FaultRequest struct {
	SectionID        string `json:"section_id"`
	ItemIndex        int32  `json:"item_index"`
	ItemNo           string `json:"item_no"`
	Status           string `json:"status"`
	FaultText        string `json:"fault_text"`
	CorrectiveAction string `json:"corrective_action"`
}

type DeleteFaultRequest struct {
	SectionID string `json:"section_id"`
	ItemIndex int32  `json:"item_index"`
}

type SyncRequest struct {
	UpsertEquipment    []SyncEquipmentRequest        `json:"upsert_equipment"`
	DeleteEquipmentIDs []string                      `json:"delete_equipment_ids"`
	UpsertCompletions  []SyncCompletionRequest       `json:"upsert_completions"`
	DeleteCompletions  []SyncDeleteCompletionRequest `json:"delete_completions"`
	UpsertFaults       []SyncFaultRequest            `json:"upsert_faults"`
	DeleteFaults       []SyncDeleteFaultRequest      `json:"delete_faults"`
}

type SyncEquipmentRequest struct {
	ID              string `json:"id"`
	EquipmentManual string `json:"equipment_manual"`
	Admin           string `json:"admin"`
	Serial          string `json:"serial"`
	Uic             string `json:"uic"`
}

type SyncCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	ItemNo      string `json:"item_no"`
	StepID      string `json:"step_id"`
}

type SyncDeleteCompletionRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
	StepID      string `json:"step_id"`
}

type SyncFaultRequest struct {
	EquipmentID      string `json:"equipment_id"`
	SectionID        string `json:"section_id"`
	ItemIndex        int32  `json:"item_index"`
	ItemNo           string `json:"item_no"`
	Status           string `json:"status"`
	FaultText        string `json:"fault_text"`
	CorrectiveAction string `json:"corrective_action"`
}

type SyncDeleteFaultRequest struct {
	EquipmentID string `json:"equipment_id"`
	SectionID   string `json:"section_id"`
	ItemIndex   int32  `json:"item_index"`
}

type EquipmentResponse struct {
	ID              string    `json:"id"`
	EquipmentManual string    `json:"equipment_manual"`
	Admin           string    `json:"admin"`
	Serial          string    `json:"serial"`
	Uic             string    `json:"uic"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CompletionResponse struct {
	EquipmentID string    `json:"equipment_id"`
	SectionID   string    `json:"section_id"`
	ItemIndex   int32     `json:"item_index"`
	ItemNo      string    `json:"item_no"`
	StepID      string    `json:"step_id"`
	IsComplete  bool      `json:"is_complete"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BatchCompletionsResponse struct {
	UpsertedCount int64 `json:"upserted_count"`
	DeletedCount  int64 `json:"deleted_count"`
}

type FaultResponse struct {
	EquipmentID      string    `json:"equipment_id"`
	SectionID        string    `json:"section_id"`
	ItemIndex        int32     `json:"item_index"`
	ItemNo           string    `json:"item_no"`
	Status           string    `json:"status"`
	FaultText        string    `json:"fault_text"`
	CorrectiveAction string    `json:"corrective_action"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type EquipmentListResponse struct {
	Equipment []EquipmentResponse `json:"equipment"`
	Count     int                 `json:"count"`
}

type EquipmentAggregateResponse struct {
	Equipment   EquipmentResponse    `json:"equipment"`
	Completions []CompletionResponse `json:"completions"`
	Faults      []FaultResponse      `json:"faults"`
}

type SyncResponse struct {
	Equipment           []EquipmentAggregateResponse `json:"equipment"`
	DeletedEquipmentIDs []string                     `json:"deleted_equipment_ids"`
}
