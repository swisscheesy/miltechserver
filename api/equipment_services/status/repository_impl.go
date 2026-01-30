package status

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

func (repo *RepositoryImpl) GetOverdue(user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]ServiceWithDays, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		postgres.RawBool("service_date < NOW()"),
		EquipmentServices.IsCompleted.EQ(Bool(false)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(
		EquipmentServices.AllColumns,
		Raw("EXTRACT(DAY FROM NOW() - service_date)").AS("days_overdue"),
	).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC()).
		LIMIT(int64(limit))

	var results []struct {
		model.EquipmentServices
		DaysOverdue int `sql:"days_overdue"`
	}

	err := stmt.Query(repo.db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue services: %w", err)
	}

	services := make([]ServiceWithDays, len(results))
	for i, result := range results {
		services[i] = ServiceWithDays{
			EquipmentServices: result.EquipmentServices,
			DaysCount:         result.DaysOverdue,
		}
	}

	return services, nil
}

func (repo *RepositoryImpl) GetDueSoon(user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]ServiceWithDays, error) {
	futureDate := time.Now().AddDate(0, 0, daysAhead)

	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
		EquipmentServices.ServiceDate.IS_NOT_NULL(),
		postgres.RawBool("service_date > NOW()"),
		EquipmentServices.ServiceDate.LT_EQ(TimestampzT(futureDate)),
		EquipmentServices.IsCompleted.EQ(Bool(false)),
	}

	if equipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*equipmentID)))
	}

	stmt := SELECT(
		EquipmentServices.AllColumns,
		Raw("EXTRACT(DAY FROM service_date - NOW())").AS("days_until_due"),
	).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC()).
		LIMIT(int64(limit))

	var results []struct {
		model.EquipmentServices
		DaysUntilDue int `sql:"days_until_due"`
	}

	err := stmt.Query(repo.db, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to get due soon services: %w", err)
	}

	services := make([]ServiceWithDays, len(results))
	for i, result := range results {
		services[i] = ServiceWithDays{
			EquipmentServices: result.EquipmentServices,
			DaysCount:         result.DaysUntilDue,
		}
	}

	return services, nil
}
