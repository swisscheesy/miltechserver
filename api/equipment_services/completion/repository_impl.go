package completion

import (
	"database/sql"
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

func (repo *RepositoryImpl) Complete(user *bootstrap.User, serviceID string, completionDate *time.Time) (*model.EquipmentServices, error) {
	now := time.Now()
	if completionDate == nil {
		completionDate = &now
	}

	stmt := EquipmentServices.UPDATE(
		EquipmentServices.IsCompleted,
		EquipmentServices.CompletionDate,
		EquipmentServices.UpdatedAt,
	).SET(
		EquipmentServices.IsCompleted.SET(Bool(true)),
		EquipmentServices.CompletionDate.SET(TimestampzT(*completionDate)),
		EquipmentServices.UpdatedAt.SET(TimestampzT(now)),
	).WHERE(
		EquipmentServices.ID.EQ(String(serviceID)).
			AND(EquipmentServices.ShopID.IN(
				SELECT(ShopMembers.ShopID).FROM(ShopMembers).WHERE(ShopMembers.UserID.EQ(String(user.UserID))),
			)),
	).RETURNING(EquipmentServices.AllColumns)

	var completedService model.EquipmentServices
	err := stmt.Query(repo.db, &completedService)
	if err != nil {
		return nil, fmt.Errorf("failed to complete equipment service: %w", err)
	}

	slog.Info("Equipment service completed", "service_id", serviceID, "completed_by", user.UserID)
	return &completedService, nil
}
