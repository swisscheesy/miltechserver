package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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

func (repo *RepositoryImpl) Create(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	tx, err := repo.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, _, err := shopshared.LockShopMutation(context.Background(), tx, service.ShopID, user.UserID); err != nil {
		return nil, err
	}
	if err := shopshared.LockReferencedLists(context.Background(), tx, service.ShopID, service.ListID); err != nil {
		return nil, err
	}
	var vehicleShop string
	if err := tx.QueryRow(`SELECT shop_id FROM shop_vehicle WHERE id=$1`, service.EquipmentID).Scan(&vehicleShop); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shopshared.ErrVehicleNotFound
		}
		return nil, &shopshared.Failure{Code: "internal_error", PublicMessage: "Unable to verify vehicle", Status: 500, Cause: err}
	}
	if vehicleShop != service.ShopID {
		return nil, shopshared.ErrShopAccessDenied
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

	var createdService model.EquipmentServices
	err = stmt.Query(tx, &createdService)
	if err != nil {
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	slog.Info("Equipment service created", "service_id", service.ID, "created_by", user.UserID)
	return &createdService, nil
}

func (repo *RepositoryImpl) GetByID(user *bootstrap.User, serviceID string) (*model.EquipmentServices, error) {
	stmt := SELECT(EquipmentServices.AllColumns).FROM(EquipmentServices).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)),
	)

	var service model.EquipmentServices
	err := stmt.Query(repo.db, &service)
	if err != nil {
		return nil, fmt.Errorf("failed to get equipment service: %w", err)
	}

	return &service, nil
}

func (repo *RepositoryImpl) Update(user *bootstrap.User, service model.EquipmentServices) (*model.EquipmentServices, error) {
	tx, err := repo.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var shopID string
	if err := tx.QueryRow(`SELECT shop_id FROM equipment_services WHERE id=$1`, service.ID).Scan(&shopID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &shopshared.Failure{Code: "service_not_found", PublicMessage: "service not found", Status: 404, Cause: err}
		}
		return nil, &shopshared.Failure{Code: "internal_error", PublicMessage: "Unable to verify service", Status: 500, Cause: err}
	}
	admin, _, err := shopshared.LockShopMutation(context.Background(), tx, shopID, user.UserID)
	if err != nil {
		return nil, err
	}
	var oldList, creator string
	if err := tx.QueryRow(`SELECT list_id,created_by FROM equipment_services WHERE id=$1 AND shop_id=$2`, service.ID, shopID).Scan(&oldList, &creator); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &shopshared.Failure{Code: "service_not_found", PublicMessage: "service not found", Status: 404, Cause: err}
		}
		return nil, &shopshared.Failure{Code: "internal_error", PublicMessage: "Unable to verify service", Status: 500, Cause: err}
	}
	if creator != user.UserID && !admin {
		return nil, shopshared.ErrShopAccessDenied
	}
	if err := shopshared.LockReferencedLists(context.Background(), tx, shopID, oldList, service.ListID); err != nil {
		return nil, err
	}

	now := time.Now()
	service.UpdatedAt = now

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

	var updatedService model.EquipmentServices
	err = stmt.Query(tx, &updatedService)
	if err != nil {
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	slog.Info("Equipment service updated", "service_id", service.ID, "updated_by", user.UserID)
	return &updatedService, nil
}

func (repo *RepositoryImpl) Delete(user *bootstrap.User, serviceID string) error {
	stmt := EquipmentServices.DELETE().WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.ShopID.IN(
				SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
			)),
	)

	result, err := stmt.Exec(repo.db)
	if err != nil {
		return fmt.Errorf("failed to delete equipment service: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("equipment service not found or access denied")
	}

	slog.Info("Equipment service deleted", "service_id", serviceID, "deleted_by", user.UserID)
	return nil
}
