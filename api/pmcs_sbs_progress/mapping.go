package pmcs_sbs_progress

import (
	"strings"

	"miltechserver/.gen/miltech_ng/public/model"
)

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

func mapFault(row model.UserPmcsFaults) FaultResponse {
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

func mapInspection(row model.UserPmcsInspections, performedByUsername *string, faultRows []model.UserPmcsFaults, commentRows []CommentWithAuthor) InspectionResponse {
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
