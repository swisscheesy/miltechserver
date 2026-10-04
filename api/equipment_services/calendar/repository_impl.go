package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/equipment_services/shared"
	"miltechserver/bootstrap"

	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetInDateRange(ctx context.Context, user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}
	filters := shared.ServiceFilters{From: &startDate, To: &endDate}
	if equipmentID != nil {
		filters.EquipmentID = equipmentID
	}
	predicate, err := shared.ServiceFilterPredicate(filters, time.Now())
	if err != nil {
		return nil, err
	}
	predicate = EquipmentServices.ShopID.EQ(String(shopID)).AND(predicate)

	stmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers,
				ShopMembers.ShopID.EQ(EquipmentServices.ShopID).
					AND(ShopMembers.UserID.EQ(String(user.UserID))),
			),
	).WHERE(predicate).
		ORDER_BY(EquipmentServices.ServiceDate.ASC().NULLS_LAST(), EquipmentServices.ID.ASC())

	var services []model.EquipmentServices
	err = stmt.QueryContext(ctx, repo.db, &services)
	if err != nil {
		return nil, fmt.Errorf("failed to get services in date range: %w", err)
	}

	return services, nil
}
