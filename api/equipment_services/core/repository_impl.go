package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	equipmentshared "miltechserver/api/equipment_services/shared"
	sharedb "miltechserver/api/shared/db"
	shopshared "miltechserver/api/shops/shared"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) Create(ctx context.Context, user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	var createdService model.EquipmentServices
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if user == nil {
			return equipmentshared.ErrUnauthorizedUser
		}
		service.CreatedBy = user.UserID
		if _, _, err := shopshared.LockShopMutation(ctx, tx, service.ShopID, user.UserID); err != nil {
			return err
		}
		if err := shopshared.LockReferencedLists(ctx, tx, service.ShopID, service.ListID); err != nil {
			return err
		}
		var vehicleShop string
		if err := tx.QueryRowContext(ctx, `SELECT shop_id FROM shop_vehicle WHERE id=$1 FOR UPDATE`, service.EquipmentID).Scan(&vehicleShop); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return shopshared.ErrVehicleNotFound
			}
			return &shopshared.Failure{Code: "internal_error", PublicMessage: "Unable to verify vehicle", Status: 500, Cause: err}
		}
		if vehicleShop != service.ShopID {
			return shopshared.ErrShopAccessDenied
		}

		stmt := EquipmentServices.INSERT(
			EquipmentServices.ID,
			EquipmentServices.ShopID,
			EquipmentServices.EquipmentID,
			EquipmentServices.ListID,
			EquipmentServices.Description,
			EquipmentServices.ServiceType,
			EquipmentServices.CreatedBy,
			EquipmentServices.IsCompleted,
			EquipmentServices.CreatedAt,
			EquipmentServices.UpdatedAt,
			EquipmentServices.ServiceDate,
			EquipmentServices.ServiceHours,
			EquipmentServices.CompletionDate,
		).MODEL(service).RETURNING(EquipmentServices.AllColumns)

		err := stmt.QueryContext(ctx, tx, &createdService)
		if err != nil {
			return fmt.Errorf("failed to create equipment service: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Equipment service created", "service_id", service.ID, "created_by", user.UserID)
	return &createdService, nil
}

func (repo *RepositoryImpl) GetByID(ctx context.Context, user *bootstrap.User, serviceID string) (*model.EquipmentServices, error) {
	stmt := SELECT(EquipmentServices.AllColumns).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)),
	)

	var service model.EquipmentServices
	err := stmt.QueryContext(ctx, repo.db, &service)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	return &service, nil
}

func (repo *RepositoryImpl) Update(ctx context.Context, user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	var updatedService model.EquipmentServices
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := equipmentshared.LockServiceMutation(ctx, tx, user, service.ShopID, service.ID, service.ListID); err != nil {
			return err
		}
		var currentService model.EquipmentServices
		if err := SELECT(EquipmentServices.IsCompleted, EquipmentServices.CompletionDate).
			FROM(EquipmentServices).WHERE(EquipmentServices.ID.EQ(String(service.ID))).
			QueryContext(ctx, tx, &currentService); err != nil {
			return fmt.Errorf("failed to read equipment service completion state: %w", err)
		}
		now := time.Now()
		service.UpdatedAt = now
		if !service.IsCompleted {
			service.CompletionDate = nil
		} else if service.CompletionDate == nil {
			// The locked state preserves history even when a queued request saw
			// an incomplete service before another writer committed.
			if currentService.IsCompleted {
				service.CompletionDate = currentService.CompletionDate
			} else {
				service.CompletionDate = &now
			}
		}

		stmt := EquipmentServices.UPDATE(
			EquipmentServices.Description,
			EquipmentServices.ServiceType,
			EquipmentServices.ListID,
			EquipmentServices.IsCompleted,
			EquipmentServices.ServiceDate,
			EquipmentServices.ServiceHours,
			EquipmentServices.CompletionDate,
			EquipmentServices.UpdatedAt,
		).MODEL(service).WHERE(
			EquipmentServices.ID.EQ(String(service.ID)).
				AND(EquipmentServices.ShopID.IN(
					SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
				)),
		).RETURNING(EquipmentServices.AllColumns)

		err := stmt.QueryContext(ctx, tx, &updatedService)
		if err != nil {
			return fmt.Errorf("failed to update equipment service: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Equipment service updated", "service_id", service.ID, "updated_by", user.UserID)
	return &updatedService, nil
}

func (repo *RepositoryImpl) Delete(ctx context.Context, user *bootstrap.User, shopID, serviceID string) error {
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := equipmentshared.LockServiceMutation(ctx, tx, user, shopID, serviceID); err != nil {
			return err
		}
		stmt := EquipmentServices.DELETE().WHERE(
			EquipmentServices.ID.EQ(String(serviceID)).
				AND(EquipmentServices.ShopID.IN(
					SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
				)),
		)

		result, err := stmt.ExecContext(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to delete equipment service: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count deleted services: %w", err)
		}
		if rowsAffected == 0 {
			return errors.New("equipment service not found or access denied")
		}

		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("Equipment service deleted", "service_id", serviceID, "deleted_by", user.UserID)
	return nil
}
