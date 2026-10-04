package aggregates

import (
	"context"
	"database/sql"
	"fmt"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/response"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
)

func (repo *RepositoryImpl) getEquipmentPmcsHistory(ctx context.Context, tx *sql.Tx, user *bootstrap.User) ([]response.EquipmentWithPmcsHistory, error) {
	equipmentStmt := SELECT(
		ShopVehicle.ID.AS("id"),
		ShopVehicle.ShopID.AS("shop_id"),
		ShopVehicle.Admin.AS("admin"),
		ShopVehicle.Model.AS("model"),
		ShopVehicle.Serial.AS("serial"),
		ShopVehicle.Niin.AS("niin"),
	).FROM(
		ShopVehicle.INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
	).WHERE(
		ShopMembers.UserID.EQ(String(user.UserID)),
	).ORDER_BY(
		ShopVehicle.SaveTime.DESC(),
		ShopVehicle.ID.DESC(),
	)

	// NOTE: the destination must be an anonymous struct, not a named local type.
	// go-jet's query result mapper derives its column-matching key from the destination
	// struct's reflect.Type.Name(): a named type causes jet to require dot-prefixed column
	// aliases (e.g. "equipmentrow.id", the convention used for embedded/nested table structs),
	// while an anonymous struct type (Name() == "") matches plain bare-field-name aliases like
	// "id" directly. Since this query aliases columns as plain names via .AS("id") etc., a named
	// struct here silently matches zero columns and returns zero rows with no error. Verified via
	// isolated repro against the live DB: identical query + fields, only named vs. anonymous type,
	// produced 0 vs 2 rows. This matches the working pattern already used a few lines below
	// (the `counts` query) and in pmcs_sbs_progress/repository_impl.go's requireVehicleAccess.
	var equipmentRows []struct {
		ID     string `sql:"id"`
		ShopID string `sql:"shop_id"`
		Admin  string `sql:"admin"`
		Model  string `sql:"model"`
		Serial string `sql:"serial"`
		Niin   string `sql:"niin"`
	}
	if err := equipmentStmt.QueryContext(ctx, tx, &equipmentRows); err != nil {
		return nil, fmt.Errorf("failed to query equipment for pmcs history: %w", err)
	}
	if len(equipmentRows) == 0 {
		return []response.EquipmentWithPmcsHistory{}, nil
	}

	var inspections []struct {
		model.UserPmcsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	inspectionsStmt := SELECT(
		UserPmcsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(
			UserPmcsInspections.
				INNER_JOIN(ShopVehicle, ShopVehicle.ID.EQ(UserPmcsInspections.EquipmentID)).
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)).
				LEFT_JOIN(Users, Users.UID.EQ(UserPmcsInspections.PerformedBy)),
		).
		WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
		ORDER_BY(UserPmcsInspections.EquipmentID.ASC(), UserPmcsInspections.PerformedDate.DESC())

	if err := inspectionsStmt.QueryContext(ctx, tx, &inspections); err != nil {
		return nil, fmt.Errorf("failed to query pmcs inspections for equipment history: %w", err)
	}

	faultCountByInspectionID := make(map[uuid.UUID]int)
	if len(inspections) > 0 {
		var counts []struct {
			PmcsID uuid.UUID `sql:"pmcs_id"`
			Total  int32     `sql:"total"`
		}
		countStmt := SELECT(
			UserPmcsFaults.PmcsID.AS("pmcs_id"),
			COUNT(UserPmcsFaults.PmcsID).AS("total"),
		).FROM(
			UserPmcsFaults.
				INNER_JOIN(UserPmcsInspections, UserPmcsInspections.ID.EQ(UserPmcsFaults.PmcsID)).
				INNER_JOIN(ShopVehicle, ShopVehicle.ID.EQ(UserPmcsInspections.EquipmentID)).
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
		).
			WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
			GROUP_BY(UserPmcsFaults.PmcsID)

		if err := countStmt.QueryContext(ctx, tx, &counts); err != nil {
			return nil, fmt.Errorf("failed to count pmcs faults for equipment history: %w", err)
		}
		for _, count := range counts {
			faultCountByInspectionID[count.PmcsID] = int(count.Total)
		}
	}

	commentCountByInspectionID := make(map[uuid.UUID]int)
	if len(inspections) > 0 {
		var commentCounts []struct {
			PmcsID uuid.UUID `sql:"pmcs_id"`
			Total  int32     `sql:"total"`
		}
		commentCountStmt := SELECT(
			UserPmcsInspectionComments.PmcsID.AS("pmcs_id"),
			COUNT(UserPmcsInspectionComments.PmcsID).AS("total"),
		).FROM(
			UserPmcsInspectionComments.
				INNER_JOIN(UserPmcsInspections, UserPmcsInspections.ID.EQ(UserPmcsInspectionComments.PmcsID)).
				INNER_JOIN(ShopVehicle, ShopVehicle.ID.EQ(UserPmcsInspections.EquipmentID)).
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
		).
			WHERE(ShopMembers.UserID.EQ(String(user.UserID))).
			GROUP_BY(UserPmcsInspectionComments.PmcsID)

		if err := commentCountStmt.QueryContext(ctx, tx, &commentCounts); err != nil {
			return nil, fmt.Errorf("failed to count pmcs inspection comments for equipment history: %w", err)
		}
		for _, count := range commentCounts {
			commentCountByInspectionID[count.PmcsID] = int(count.Total)
		}
	}

	historyByEquipmentID := make(map[string][]response.PmcsHistorySummary, len(equipmentRows))
	for _, inspection := range inspections {
		historyByEquipmentID[inspection.EquipmentID] = append(historyByEquipmentID[inspection.EquipmentID], response.PmcsHistorySummary{
			ID:                   inspection.ID,
			SourceType:           inspection.SourceType,
			GuideManual:          inspection.GuideManual,
			CustomChecklistID:    inspection.CustomChecklistID,
			CustomRevisionID:     inspection.CustomRevisionID,
			CustomRevisionNumber: inspection.CustomRevisionNumber,
			CustomChecklistName:  inspection.CustomChecklistName,
			PerformedDate:        inspection.PerformedDate,
			FaultCount:           faultCountByInspectionID[inspection.ID],
			CommentCount:         commentCountByInspectionID[inspection.ID],
			CreatedAt:            inspection.CreatedAt,
			PerformedBy:          inspection.PerformedBy,
			PerformedByUsername:  inspection.PerformedByUsername,
		})
	}

	equipment := make([]response.EquipmentWithPmcsHistory, 0, len(equipmentRows))
	for _, row := range equipmentRows {
		history := historyByEquipmentID[row.ID]
		if history == nil {
			history = []response.PmcsHistorySummary{}
		}
		equipment = append(equipment, response.EquipmentWithPmcsHistory{
			ShopEquipmentSummary: response.ShopEquipmentSummary{
				ID:     row.ID,
				Admin:  row.Admin,
				Model:  row.Model,
				Serial: row.Serial,
				Niin:   row.Niin,
			},
			ShopID:         row.ShopID,
			HistoricalPmcs: history,
		})
	}
	return equipment, nil
}
