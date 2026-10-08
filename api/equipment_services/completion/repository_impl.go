package completion

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"miltechserver/api/equipment_services/shared"
	sharedb "miltechserver/api/shared/db"
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

func (repo *RepositoryImpl) Complete(ctx context.Context, user *bootstrap.User, shopID, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error) {
	var completedService model.EquipmentServices
	err := sharedb.WithTxContext(ctx, repo.db, func(tx *sql.Tx) error {
		if err := shared.LockServiceMutation(ctx, tx, user, shopID, serviceID); err != nil {
			return err
		}
		var currentService model.EquipmentServices
		if err := SELECT(EquipmentServices.IsCompleted, EquipmentServices.CompletionDate).
			FROM(EquipmentServices).WHERE(EquipmentServices.ID.EQ(String(serviceID))).
			QueryContext(ctx, tx, &currentService); err != nil {
			return fmt.Errorf("failed to read equipment service completion state: %w", err)
		}
		now := time.Now()
		if completionDate == nil {
			// A retry keeps the persisted date, including legacy completed NULLs.
			if currentService.IsCompleted {
				completionDate = currentService.CompletionDate
			} else {
				completionDate = &now
			}
		}

		stmt := EquipmentServices.UPDATE(
			EquipmentServices.IsCompleted,
			EquipmentServices.CompletionDate,
			EquipmentServices.UpdatedAt,
		).MODEL(model.EquipmentServices{
			IsCompleted:    true,
			CompletionDate: completionDate,
			UpdatedAt:      now,
		}).WHERE(
			EquipmentServices.ID.EQ(String(serviceID)).
				AND(EquipmentServices.ShopID.IN(
					SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
				)),
		).RETURNING(EquipmentServices.AllColumns)

		err := stmt.QueryContext(ctx, tx, &completedService)
		if err != nil {
			return fmt.Errorf("failed to complete equipment service: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	slog.Info("Equipment service completed", "service_id", serviceID, "completed_by", user.UserID)
	return &completedService, nil
}
