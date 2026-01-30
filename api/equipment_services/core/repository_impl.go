package core

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	err := stmt.Query(repo.db, &createdService)
	if err != nil {
		return nil, fmt.Errorf("failed to create equipment service: %w", err)
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
	err := stmt.Query(repo.db, &updatedService)
	if err != nil {
		return nil, fmt.Errorf("failed to update equipment service: %w", err)
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
