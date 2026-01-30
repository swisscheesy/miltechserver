package calendar

import (
	"database/sql"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/bootstrap"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetInDateRange(user *bootstrap.User, shopID string, startDate, endDate time.Time, equipmentID *string) ([]model.EquipmentServices, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		EquipmentServices.ServiceDate.GT_EQ(TimestampzT(startDate)),
		EquipmentServices.ServiceDate.LT_EQ(TimestampzT(endDate)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC())

	var services []model.EquipmentServices
	err := stmt.Query(repo.db, &services)
	if err != nil {
		return nil, fmt.Errorf("failed to get services in date range: %w", err)
	}

	return services, nil
}
