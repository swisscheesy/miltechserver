package pmcs_sbs_progress

import "time"

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

type FaultListResponse struct {
	Faults []FaultResponse `json:"faults"`
	Count  int             `json:"count"`
}
