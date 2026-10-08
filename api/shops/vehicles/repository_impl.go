package vehicles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	sharedb "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) CreateShopVehicle(ctx context.Context, user *bootstrap.User, vehicle model.ShopVehicle) (*model.ShopVehicle, error) {
	var createdVehicle model.ShopVehicle
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if user == nil {
			return shared.ErrShopAccessDenied
		}
		if _, _, err := shared.LockShopMutation(ctx, tx, vehicle.ShopID, user.UserID); err != nil {
			return err
		}
		vehicle.CreatorID = user.UserID
		stmt := ShopVehicle.INSERT(
			ShopVehicle.ID,
			ShopVehicle.CreatorID,
			ShopVehicle.Niin,
			ShopVehicle.Admin,
			ShopVehicle.Model,
			ShopVehicle.Serial,
			ShopVehicle.Uoc,
			ShopVehicle.Mileage,
			ShopVehicle.Hours,
			ShopVehicle.Comment,
			ShopVehicle.SaveTime,
			ShopVehicle.LastUpdated,
			ShopVehicle.ShopID,
		).MODEL(vehicle).RETURNING(ShopVehicle.AllColumns)

		err := stmt.QueryContext(ctx, tx, &createdVehicle)
		if err != nil {
			return fmt.Errorf("failed to create shop vehicle: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &createdVehicle, nil
}

func (repo *RepositoryImpl) GetShopVehicles(ctx context.Context, user *bootstrap.User, shopID string) ([]model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ShopID.EQ(String(shopID))).
		ORDER_BY(ShopVehicle.SaveTime.DESC())

	var vehicles []model.ShopVehicle
	err := stmt.QueryContext(ctx, repo.db, &vehicles)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop vehicles: %w", err)
	}

	return vehicles, nil
}

func (repo *RepositoryImpl) GetShopVehicleByID(ctx context.Context, user *bootstrap.User, vehicleID string) (*model.ShopVehicle, error) {
	stmt := SELECT(ShopVehicle.AllColumns).
		FROM(ShopVehicle).
		WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

	var vehicle model.ShopVehicle
	err := stmt.QueryContext(ctx, repo.db, &vehicle)
	if err != nil {
		return nil, fmt.Errorf("shop vehicle not found: %w", err)
	}

	return &vehicle, nil
}

func (repo *RepositoryImpl) UpdateShopVehicleMetadata(ctx context.Context, user *bootstrap.User, input VehicleUpdateInput) error {
	return sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if user == nil {
			return shared.ErrShopAccessDenied
		}
		vehicle := input.Metadata
		if err := shared.AuthorizeOwnedMutation(ctx, tx, user.UserID, "vehicle", vehicle.VehicleID, true); err != nil {
			return err
		}
		if err := validateBaseUsage(vehicle.Mileage, vehicle.Hours); err != nil {
			return err
		}
		if err := validateAbsoluteTrackedUsage(input); err != nil {
			return err
		}
		if vehicle.Admin != nil && *vehicle.Admin == "" {
			return fmt.Errorf("%w: admin cannot be empty", ErrInvalidUsageAdjustment)
		}
		if vehicle.Uoc != nil && *vehicle.Uoc == "" {
			value := "UNK"
			vehicle.Uoc = &value
		}
		setClauses := []postgres.ColumnAssigment{ShopVehicle.LastUpdated.SET(TimestampzT(time.Now().UTC()))}
		for _, field := range []struct {
			column postgres.ColumnString
			value  *string
		}{
			{ShopVehicle.Admin, vehicle.Admin}, {ShopVehicle.Niin, vehicle.Niin}, {ShopVehicle.Model, vehicle.Model},
			{ShopVehicle.Serial, vehicle.Serial}, {ShopVehicle.Uoc, vehicle.Uoc}, {ShopVehicle.Comment, vehicle.Comment},
		} {
			if field.value != nil {
				value := *field.value
				setClauses = append(setClauses, field.column.SET(String(value)))
			}
		}
		for _, field := range []struct {
			column postgres.ColumnInteger
			value  *int32
		}{
			{ShopVehicle.Mileage, vehicle.Mileage}, {ShopVehicle.Hours, vehicle.Hours},
			{ShopVehicle.TrackedMileage, input.TrackedMileage}, {ShopVehicle.TrackedHours, input.TrackedHours},
		} {
			if field.value != nil {
				setClauses = append(setClauses, field.column.SET(Int32(*field.value)))
			}
		}
		setArgs := make([]interface{}, len(setClauses))
		for i, clause := range setClauses {
			setArgs[i] = clause
		}
		stmt := ShopVehicle.UPDATE().SET(setArgs[0], setArgs[1:]...).WHERE(ShopVehicle.ID.EQ(String(vehicle.VehicleID)))
		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to update shop vehicle: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return errors.New("vehicle not found")
		}
		return nil
	})
}

func (repo *RepositoryImpl) UpdateShopVehicleUsage(ctx context.Context, user *bootstrap.User, update ShopVehicleUsageUpdate) error {
	return sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if user == nil {
			return shared.ErrShopAccessDenied
		}
		if err := lockVehicleUsage(ctx, tx, user, update.VehicleID); err != nil {
			return err
		}
		setClauses := []postgres.ColumnAssigment{
			ShopVehicle.LastUpdated.SET(TimestampzT(update.LastUpdated)),
		}

		if update.TrackedMileage != nil {
			setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(*update.TrackedMileage)))
		}

		if update.TrackedHours != nil {
			setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(*update.TrackedHours)))
		}

		setArgs := make([]interface{}, len(setClauses))
		for i, clause := range setClauses {
			setArgs[i] = clause
		}

		stmt := ShopVehicle.UPDATE().SET(setArgs[0], setArgs[1:]...).WHERE(ShopVehicle.ID.EQ(String(update.VehicleID)))

		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to update shop vehicle usage: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("vehicle not found")
		}

		return nil
	})
}

func (repo *RepositoryImpl) AdjustShopVehicleUsage(ctx context.Context, user *bootstrap.User, adjustment UsageAdjustment) (*model.ShopVehicle, error) {
	var updatedVehicle model.ShopVehicle
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := lockVehicleUsage(ctx, tx, user, adjustment.VehicleID); err != nil {
			return err
		}
		lockedVehicleStmt := SELECT(ShopVehicle.AllColumns).
			FROM(ShopVehicle).
			WHERE(ShopVehicle.ID.EQ(String(adjustment.VehicleID))).
			FOR(UPDATE())

		var currentVehicle model.ShopVehicle
		if err := lockedVehicleStmt.QueryContext(ctx, tx, &currentVehicle); err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
				return shared.ErrVehicleNotFound
			}
			return fmt.Errorf("lock shop vehicle for usage adjustment: %w", err)
		}

		setClauses := []postgres.ColumnAssigment{
			ShopVehicle.LastUpdated.SET(TimestampzT(adjustment.LastUpdated)),
		}
		if adjustment.MileageAdjustment != nil {
			currentMileage := currentVehicle.Mileage
			if currentVehicle.TrackedMileage != nil {
				currentMileage = *currentVehicle.TrackedMileage
			}

			adjustedMileage, err := applyUsageAdjustment(currentMileage, *adjustment.MileageAdjustment, adjustment.Operation)
			if err != nil {
				return err
			}
			setClauses = append(setClauses, ShopVehicle.TrackedMileage.SET(Int32(adjustedMileage)))
		}
		if adjustment.HoursAdjustment != nil {
			currentHours := currentVehicle.Hours
			if currentVehicle.TrackedHours != nil {
				currentHours = *currentVehicle.TrackedHours
			}

			adjustedHours, err := applyUsageAdjustment(currentHours, *adjustment.HoursAdjustment, adjustment.Operation)
			if err != nil {
				return err
			}
			setClauses = append(setClauses, ShopVehicle.TrackedHours.SET(Int32(adjustedHours)))
		}

		setArgs := make([]interface{}, len(setClauses))
		for index, clause := range setClauses {
			setArgs[index] = clause
		}

		stmt := ShopVehicle.UPDATE().
			SET(setArgs[0], setArgs[1:]...).
			WHERE(ShopVehicle.ID.EQ(String(adjustment.VehicleID))).
			RETURNING(ShopVehicle.AllColumns)

		if err := stmt.QueryContext(ctx, tx, &updatedVehicle); err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
				return shared.ErrVehicleNotFound
			}
			return fmt.Errorf("update shop vehicle usage: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &updatedVehicle, nil
}

func applyUsageAdjustment(current int32, magnitude int32, operation UsageAdjustmentOperation) (int32, error) {
	result := int64(current)
	if operation == UsageOperationSubtract {
		result -= int64(magnitude)
	} else {
		result += int64(magnitude)
	}
	if result < 0 || result > math.MaxInt32 {
		return 0, ErrUsageOutOfRange
	}
	return int32(result), nil
}

func (repo *RepositoryImpl) DeleteShopVehicle(ctx context.Context, user *bootstrap.User, vehicleID string) error {
	ctx = shared.WithAuditCorrelation(ctx)
	return sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if user == nil {
			return shared.ErrShopAccessDenied
		}
		if err := shared.AuthorizeOwnedMutation(ctx, tx, user.UserID, "vehicle", vehicleID, true); err != nil {
			return err
		}

		// Capture and write the audit only after authorization, on the same connection.
		var vehicle model.ShopVehicle
		if err := SELECT(ShopVehicle.AllColumns).FROM(ShopVehicle).WHERE(ShopVehicle.ID.EQ(String(vehicleID))).QueryContext(ctx, tx, &vehicle); err != nil {
			return err
		}
		change := model.ShopVehicleNotificationChanges{
			ShopID: vehicle.ShopID, VehicleID: &vehicleID, ChangedBy: &user.UserID,
			ChangeType: "vehicle_deleted", FieldChanges: buildVehicleDeletionFieldChanges(&vehicle), VehicleAdmin: &vehicle.Admin,
		}
		// Auditing remains best effort, but a failed statement must not abort deletion.
		if _, err := tx.ExecContext(ctx, "SAVEPOINT vehicle_deletion_audit"); err != nil {
			return err
		}
		if err := createNotificationChange(func(query string, args ...any) (sql.Result, error) { return tx.ExecContext(ctx, query, args...) }, change); err != nil {
			if _, rollbackErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT vehicle_deletion_audit"); rollbackErr != nil {
				return rollbackErr
			}
			shared.WarnLegacyAuditFailure(ctx, "vehicle_deleted", vehicle.ShopID, vehicleID, "", user.UserID, err)
		}
		if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT vehicle_deletion_audit"); err != nil {
			return err
		}

		stmt := ShopVehicle.DELETE().
			WHERE(ShopVehicle.ID.EQ(String(vehicleID)))

		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to delete shop vehicle: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("vehicle not found")
		}

		return nil
	})
}

func (repo *RepositoryImpl) CreateNotificationChange(ctx context.Context, user *bootstrap.User, change model.ShopVehicleNotificationChanges) error {
	return createNotificationChange(func(query string, args ...any) (sql.Result, error) { return repo.db.ExecContext(ctx, query, args...) }, change)
}

func createNotificationChange(exec func(string, ...any) (sql.Result, error), change model.ShopVehicleNotificationChanges) error {
	rawSQL := `
		INSERT INTO shop_vehicle_notification_changes (
			notification_id,
			shop_id,
			vehicle_id,
			changed_by,
			change_type,
			field_changes,
			notification_title,
			notification_type,
			vehicle_admin
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := exec(
		rawSQL,
		change.NotificationID,
		change.ShopID,
		change.VehicleID,
		change.ChangedBy,
		change.ChangeType,
		change.FieldChanges,
		change.NotificationTitle,
		change.NotificationType,
		change.VehicleAdmin,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification change record: %w", err)
	}

	return nil
}

// Usage changes belong to any current member. Lock the Shop before its vehicle,
// then recheck the persisted parent so preflight cannot authorize the commit.
func lockVehicleUsage(ctx context.Context, tx *sql.Tx, user *bootstrap.User, vehicleID string) error {
	if user == nil {
		return shared.ErrShopAccessDenied
	}
	var vehicle model.ShopVehicle
	stmt := SELECT(ShopVehicle.ShopID).FROM(ShopVehicle).WHERE(ShopVehicle.ID.EQ(String(vehicleID)))
	if err := stmt.QueryContext(ctx, tx, &vehicle); err != nil {
		if shared.ErrorsIsNoRows(err) {
			return shared.ErrVehicleNotFound
		}
		return fmt.Errorf("resolve vehicle: %w", err)
	}
	shopID := vehicle.ShopID
	if _, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID); err != nil {
		return err
	}
	if err := stmt.FOR(UPDATE()).QueryContext(ctx, tx, &vehicle); err != nil {
		if shared.ErrorsIsNoRows(err) {
			return shared.ErrVehicleNotFound
		}
		return fmt.Errorf("lock vehicle: %w", err)
	}
	if vehicle.ShopID != shopID {
		return shared.ErrShopAccessDenied
	}
	return nil
}
