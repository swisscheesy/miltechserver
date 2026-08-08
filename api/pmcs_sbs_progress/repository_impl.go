package pmcs_sbs_progress

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/google/uuid"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) EnsureInspection(user *bootstrap.User, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}
	return ensureInspection(repo.db, inspection)
}

func (repo *RepositoryImpl) GetInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) (*InspectionDetail, []model.PmcsSbsFaults, []CommentWithAuthor, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, nil, nil, err
	}

	var row struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	stmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		)

	if err := stmt.Query(repo.db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, nil, nil, ErrInspectionNotFound
		}
		return nil, nil, nil, fmt.Errorf("get pmcs sbs inspection: %w", err)
	}

	var faults []model.PmcsSbsFaults
	faultsStmt := SELECT(PmcsSbsFaults.AllColumns).
		FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID))).
		ORDER_BY(PmcsSbsFaults.SectionID.ASC(), PmcsSbsFaults.ItemIndex.ASC())

	if err := faultsStmt.Query(repo.db, &faults); err != nil {
		return nil, nil, nil, fmt.Errorf("list pmcs sbs inspection faults: %w", err)
	}

	var comments []struct {
		model.PmcsSbsInspectionComments
		AuthorUsername *string `sql:"author_username"`
	}
	commentsStmt := SELECT(
		PmcsSbsInspectionComments.AllColumns,
		Users.Username.AS("author_username"),
	).
		FROM(PmcsSbsInspectionComments.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspectionComments.AuthorID))).
		WHERE(PmcsSbsInspectionComments.PmcsID.EQ(UUID(pmcsID))).
		ORDER_BY(PmcsSbsInspectionComments.CreatedAt.ASC())

	if err := commentsStmt.Query(repo.db, &comments); err != nil {
		return nil, nil, nil, fmt.Errorf("list pmcs sbs inspection comments: %w", err)
	}

	commentsWithAuthor := make([]CommentWithAuthor, 0, len(comments))
	for _, comment := range comments {
		commentsWithAuthor = append(commentsWithAuthor, CommentWithAuthor{
			PmcsSbsInspectionComments: comment.PmcsSbsInspectionComments,
			AuthorUsername:            comment.AuthorUsername,
		})
	}

	return &InspectionDetail{PmcsSbsInspections: row.PmcsSbsInspections, PerformedByUsername: row.PerformedByUsername}, faults, commentsWithAuthor, nil
}

func (repo *RepositoryImpl) ListInspections(user *bootstrap.User, equipmentID string, guideManual string, limit int, offset int) ([]InspectionSummary, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}

	condition := PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))
	if guideManual != "" {
		condition = condition.
			AND(PmcsSbsInspections.SourceType.EQ(String("guide"))).
			AND(PmcsSbsInspections.GuideManual.EQ(String(guideManual)))
	}

	var inspections []struct {
		model.PmcsSbsInspections
		PerformedByUsername *string `sql:"performed_by_username"`
	}
	stmt := SELECT(
		PmcsSbsInspections.AllColumns,
		Users.Username.AS("performed_by_username"),
	).
		FROM(PmcsSbsInspections.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspections.PerformedBy))).
		WHERE(condition).
		ORDER_BY(PmcsSbsInspections.PerformedDate.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	if err := stmt.Query(repo.db, &inspections); err != nil {
		return nil, fmt.Errorf("list pmcs sbs inspections: %w", err)
	}
	if len(inspections) == 0 {
		return []InspectionSummary{}, nil
	}

	ids := make([]Expression, 0, len(inspections))
	for _, inspection := range inspections {
		ids = append(ids, UUID(inspection.ID))
	}

	var counts []struct {
		PmcsID uuid.UUID `sql:"pmcs_id"`
		Total  int32     `sql:"total"`
	}
	countStmt := SELECT(
		PmcsSbsFaults.PmcsID.AS("pmcs_id"),
		COUNT(PmcsSbsFaults.PmcsID).AS("total"),
	).FROM(PmcsSbsFaults).
		WHERE(PmcsSbsFaults.PmcsID.IN(ids...)).
		GROUP_BY(PmcsSbsFaults.PmcsID)

	if err := countStmt.Query(repo.db, &counts); err != nil {
		return nil, fmt.Errorf("count pmcs sbs faults: %w", err)
	}

	countByID := make(map[uuid.UUID]int, len(counts))
	for _, c := range counts {
		countByID[c.PmcsID] = int(c.Total)
	}

	var commentCounts []struct {
		PmcsID uuid.UUID `sql:"pmcs_id"`
		Total  int32     `sql:"total"`
	}
	commentCountStmt := SELECT(
		PmcsSbsInspectionComments.PmcsID.AS("pmcs_id"),
		COUNT(PmcsSbsInspectionComments.PmcsID).AS("total"),
	).FROM(PmcsSbsInspectionComments).
		WHERE(PmcsSbsInspectionComments.PmcsID.IN(ids...)).
		GROUP_BY(PmcsSbsInspectionComments.PmcsID)

	if err := commentCountStmt.Query(repo.db, &commentCounts); err != nil {
		return nil, fmt.Errorf("count pmcs sbs inspection comments: %w", err)
	}

	commentCountByID := make(map[uuid.UUID]int, len(commentCounts))
	for _, c := range commentCounts {
		commentCountByID[c.PmcsID] = int(c.Total)
	}

	summaries := make([]InspectionSummary, 0, len(inspections))
	for _, inspection := range inspections {
		summaries = append(summaries, InspectionSummary{
			ID:                   inspection.ID,
			SourceType:           inspection.SourceType,
			GuideManual:          inspection.GuideManual,
			CustomChecklistID:    inspection.CustomChecklistID,
			CustomRevisionID:     inspection.CustomRevisionID,
			CustomRevisionNumber: inspection.CustomRevisionNumber,
			CustomChecklistName:  inspection.CustomChecklistName,
			PerformedDate:        inspection.PerformedDate,
			FaultCount:           countByID[inspection.ID],
			CommentCount:         commentCountByID[inspection.ID],
			CreatedAt:            inspection.CreatedAt,
			PerformedBy:          inspection.PerformedBy,
			PerformedByUsername:  inspection.PerformedByUsername,
		})
	}
	return summaries, nil
}

func (repo *RepositoryImpl) DeleteInspection(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}

	result, err := PmcsSbsInspections.DELETE().
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		Exec(repo.db)
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete pmcs sbs inspection rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

func (repo *RepositoryImpl) UpsertFault(user *bootstrap.User, inspection model.PmcsSbsInspections, fault model.PmcsSbsFaults) (*model.PmcsSbsFaults, error) {
	if err := repo.requireVehicleAccess(user, inspection.EquipmentID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	fault.PmcsID = inspection.ID
	if fault.CreatedAt.IsZero() {
		fault.CreatedAt = now
	}
	fault.UpdatedAt = now

	stmt := PmcsSbsFaults.INSERT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
		PmcsSbsFaults.ItemNo,
		PmcsSbsFaults.Status,
		PmcsSbsFaults.FaultText,
		PmcsSbsFaults.CorrectiveAction,
		PmcsSbsFaults.SectionTitle,
		PmcsSbsFaults.CreatedAt,
		PmcsSbsFaults.UpdatedAt,
	).VALUES(
		UUID(fault.PmcsID),
		String(fault.SectionID),
		Int32(fault.ItemIndex),
		String(fault.ItemNo),
		String(fault.Status),
		String(fault.FaultText),
		String(fault.CorrectiveAction),
		nullableStringExpression(fault.SectionTitle),
		TimestampzT(fault.CreatedAt),
		TimestampzT(now),
	).ON_CONFLICT(
		PmcsSbsFaults.PmcsID,
		PmcsSbsFaults.SectionID,
		PmcsSbsFaults.ItemIndex,
	).DO_UPDATE(SET(
		PmcsSbsFaults.ItemNo.SET(String(fault.ItemNo)),
		PmcsSbsFaults.Status.SET(String(fault.Status)),
		PmcsSbsFaults.FaultText.SET(String(fault.FaultText)),
		PmcsSbsFaults.CorrectiveAction.SET(String(fault.CorrectiveAction)),
		PmcsSbsFaults.SectionTitle.SET(nullableStringExpression(fault.SectionTitle)),
		PmcsSbsFaults.UpdatedAt.SET(TimestampzT(now)),
	)).RETURNING(PmcsSbsFaults.AllColumns)

	tx, err := repo.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin upsert pmcs sbs fault transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := ensureInspection(tx, inspection); err != nil {
		return nil, err
	}

	var saved model.PmcsSbsFaults
	if err := stmt.Query(tx, &saved); err != nil {
		return nil, fmt.Errorf("upsert pmcs sbs fault: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit upsert pmcs sbs fault transaction: %w", err)
	}
	return &saved, nil
}

func (repo *RepositoryImpl) DeleteFault(user *bootstrap.User, equipmentID string, key FaultKey) error {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, key.PmcsID); err != nil {
		return err
	}

	if _, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(key.PmcsID)).
				AND(PmcsSbsFaults.SectionID.EQ(String(key.SectionID))).
				AND(PmcsSbsFaults.ItemIndex.EQ(Int32(key.ItemIndex))),
		).
		Exec(repo.db); err != nil {
		return fmt.Errorf("delete pmcs sbs fault: %w", err)
	}
	return nil
}

func (repo *RepositoryImpl) DeleteFaults(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, keys []FaultKey) (int64, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return 0, err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, pmcsID); err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}

	keyRows := make([]Expression, 0, len(keys))
	for _, key := range keys {
		keyRows = append(keyRows, ROW(String(key.SectionID), Int32(key.ItemIndex)))
	}

	result, err := PmcsSbsFaults.DELETE().
		WHERE(
			PmcsSbsFaults.PmcsID.EQ(UUID(pmcsID)).
				AND(ROW(PmcsSbsFaults.SectionID, PmcsSbsFaults.ItemIndex).IN(keyRows...)),
		).
		Exec(repo.db)
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults: %w", err)
	}
	deletedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("bulk delete pmcs sbs faults rows affected: %w", err)
	}
	return deletedCount, nil
}

func (repo *RepositoryImpl) CreateComment(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, text string) (*CommentWithAuthor, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, pmcsID); err != nil {
		return nil, err
	}

	stmt := PmcsSbsInspectionComments.INSERT(
		PmcsSbsInspectionComments.PmcsID,
		PmcsSbsInspectionComments.AuthorID,
		PmcsSbsInspectionComments.Text,
	).VALUES(
		UUID(pmcsID),
		String(user.UserID),
		String(text),
	).RETURNING(PmcsSbsInspectionComments.AllColumns)

	var created model.PmcsSbsInspectionComments
	if err := stmt.Query(repo.db, &created); err != nil {
		return nil, fmt.Errorf("create pmcs sbs inspection comment: %w", err)
	}

	username := user.Username
	return &CommentWithAuthor{PmcsSbsInspectionComments: created, AuthorUsername: &username}, nil
}

func (repo *RepositoryImpl) GetComment(user *bootstrap.User, equipmentID string, pmcsID uuid.UUID, commentID uuid.UUID) (*CommentWithAuthor, error) {
	if err := repo.requireVehicleAccess(user, equipmentID); err != nil {
		return nil, err
	}
	if err := repo.requireInspectionOwnership(repo.db, equipmentID, pmcsID); err != nil {
		return nil, err
	}

	var row struct {
		model.PmcsSbsInspectionComments
		AuthorUsername *string `sql:"author_username"`
	}
	stmt := SELECT(
		PmcsSbsInspectionComments.AllColumns,
		Users.Username.AS("author_username"),
	).
		FROM(PmcsSbsInspectionComments.LEFT_JOIN(Users, Users.UID.EQ(PmcsSbsInspectionComments.AuthorID))).
		WHERE(
			PmcsSbsInspectionComments.ID.EQ(UUID(commentID)).
				AND(PmcsSbsInspectionComments.PmcsID.EQ(UUID(pmcsID))),
		)

	if err := stmt.Query(repo.db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get pmcs sbs inspection comment: %w", err)
	}
	return &CommentWithAuthor{PmcsSbsInspectionComments: row.PmcsSbsInspectionComments, AuthorUsername: row.AuthorUsername}, nil
}

func (repo *RepositoryImpl) UpdateComment(commentID uuid.UUID, text string) (*CommentWithAuthor, error) {
	now := time.Now().UTC()

	stmt := PmcsSbsInspectionComments.UPDATE().
		SET(
			PmcsSbsInspectionComments.Text.SET(String(text)),
			PmcsSbsInspectionComments.UpdatedAt.SET(TimestampzT(now)),
		).
		WHERE(PmcsSbsInspectionComments.ID.EQ(UUID(commentID))).
		RETURNING(PmcsSbsInspectionComments.AllColumns)

	var updated model.PmcsSbsInspectionComments
	if err := stmt.Query(repo.db, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("update pmcs sbs inspection comment: %w", err)
	}

	username, err := repo.LookupUsername(updated.AuthorID)
	if err != nil {
		return nil, err
	}
	return &CommentWithAuthor{PmcsSbsInspectionComments: updated, AuthorUsername: username}, nil
}

func (repo *RepositoryImpl) requireVehicleAccess(user *bootstrap.User, equipmentID string) error {
	if user == nil {
		return ErrUnauthorized
	}

	stmt := SELECT(Int(1).AS("exists")).
		FROM(
			ShopVehicle.
				INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(ShopVehicle.ShopID)),
		).
		WHERE(
			ShopVehicle.ID.EQ(String(equipmentID)).
				AND(ShopMembers.UserID.EQ(String(user.UserID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(repo.db, &rows); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("authorize pmcs sbs vehicle fault access: %w", err)
	}
	if len(rows) == 0 {
		return ErrNotFound
	}
	return nil
}

func (repo *RepositoryImpl) requireInspectionOwnership(queryable qrm.Queryable, equipmentID string, pmcsID uuid.UUID) error {
	stmt := SELECT(Int(1).AS("exists")).
		FROM(PmcsSbsInspections).
		WHERE(
			PmcsSbsInspections.ID.EQ(UUID(pmcsID)).
				AND(PmcsSbsInspections.EquipmentID.EQ(String(equipmentID))),
		).
		LIMIT(1)

	var rows []struct {
		Exists int `sql:"exists"`
	}
	if err := stmt.Query(queryable, &rows); err != nil {
		return fmt.Errorf("authorize pmcs sbs inspection access: %w", err)
	}
	if len(rows) == 0 {
		return ErrInspectionNotFound
	}
	return nil
}

func (repo *RepositoryImpl) LookupUsername(userID string) (*string, error) {
	var row struct {
		Username *string `sql:"username"`
	}
	stmt := SELECT(Users.Username.AS("username")).
		FROM(Users).
		WHERE(Users.UID.EQ(String(userID))).
		LIMIT(1)

	if err := stmt.Query(repo.db, &row); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup username: %w", err)
	}
	return row.Username, nil
}

// ensureInspection inserts the inspection if it doesn't exist yet. A retry
// updates mutable inspection metadata only when equipment and the complete
// immutable source tuple match the existing row. queryable is either *sql.DB
// for standalone calls or *sql.Tx for fault upserts.
func ensureInspection(queryable qrm.Queryable, inspection model.PmcsSbsInspections) (*model.PmcsSbsInspections, error) {
	now := time.Now().UTC()
	var performedByExpr Expression = NULL
	if inspection.PerformedBy != nil {
		performedByExpr = String(*inspection.PerformedBy)
	}
	var notesExpr StringExpression = StringExp(NULL)
	if inspection.Notes != nil {
		notesExpr = String(*inspection.Notes)
	}

	stmt := PmcsSbsInspections.INSERT(
		PmcsSbsInspections.ID,
		PmcsSbsInspections.EquipmentID,
		PmcsSbsInspections.SourceType,
		PmcsSbsInspections.GuideManual,
		PmcsSbsInspections.CustomChecklistID,
		PmcsSbsInspections.CustomRevisionID,
		PmcsSbsInspections.CustomRevisionNumber,
		PmcsSbsInspections.CustomChecklistName,
		PmcsSbsInspections.PerformedDate,
		PmcsSbsInspections.PerformedBy,
		PmcsSbsInspections.Notes,
		PmcsSbsInspections.CreatedAt,
		PmcsSbsInspections.UpdatedAt,
	).VALUES(
		UUID(inspection.ID),
		String(inspection.EquipmentID),
		String(inspection.SourceType),
		nullableStringExpression(inspection.GuideManual),
		nullableUUIDExpression(inspection.CustomChecklistID),
		nullableUUIDExpression(inspection.CustomRevisionID),
		nullableInt32Expression(inspection.CustomRevisionNumber),
		nullableStringExpression(inspection.CustomChecklistName),
		TimestampzT(inspection.PerformedDate),
		performedByExpr,
		notesExpr,
		TimestampzT(now),
		TimestampzT(now),
	).ON_CONFLICT(PmcsSbsInspections.ID).DO_UPDATE(
		SET(
			PmcsSbsInspections.PerformedDate.SET(TimestampzT(inspection.PerformedDate)),
			PmcsSbsInspections.Notes.SET(notesExpr),
			PmcsSbsInspections.UpdatedAt.SET(TimestampzT(now)),
		).WHERE(
			PmcsSbsInspections.EquipmentID.EQ(PmcsSbsInspections.EXCLUDED.EquipmentID).
				AND(PmcsSbsInspections.SourceType.EQ(PmcsSbsInspections.EXCLUDED.SourceType)).
				AND(PmcsSbsInspections.GuideManual.IS_NOT_DISTINCT_FROM(PmcsSbsInspections.EXCLUDED.GuideManual)).
				AND(PmcsSbsInspections.CustomChecklistID.IS_NOT_DISTINCT_FROM(PmcsSbsInspections.EXCLUDED.CustomChecklistID)).
				AND(PmcsSbsInspections.CustomRevisionID.IS_NOT_DISTINCT_FROM(PmcsSbsInspections.EXCLUDED.CustomRevisionID)).
				AND(PmcsSbsInspections.CustomRevisionNumber.IS_NOT_DISTINCT_FROM(PmcsSbsInspections.EXCLUDED.CustomRevisionNumber)).
				AND(PmcsSbsInspections.CustomChecklistName.IS_NOT_DISTINCT_FROM(PmcsSbsInspections.EXCLUDED.CustomChecklistName)),
		),
	).RETURNING(PmcsSbsInspections.AllColumns)

	var saved model.PmcsSbsInspections
	if err := stmt.Query(queryable, &saved); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrInspectionConflict
		}
		return nil, fmt.Errorf("ensure pmcs sbs inspection: %w", err)
	}
	return &saved, nil
}

func nullableStringExpression(value *string) StringExpression {
	if value == nil {
		return StringExp(NULL)
	}
	return String(*value)
}

func nullableUUIDExpression(value *uuid.UUID) StringExpression {
	if value == nil {
		return StringExp(NULL)
	}
	return UUID(*value)
}

func nullableInt32Expression(value *int32) IntegerExpression {
	if value == nil {
		return IntExp(NULL)
	}
	return Int32(*value)
}
