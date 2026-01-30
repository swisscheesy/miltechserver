package queries

import (
	"database/sql"
	"fmt"
	"time"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/request"
	"miltechserver/bootstrap"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
)

type RepositoryImpl struct {
	db *sql.DB
}

type countResult struct {
	Count int64 `alias:"count"`
}

func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (repo *RepositoryImpl) GetByShop(user *bootstrap.User, shopID string, filters request.GetEquipmentServicesRequest) ([]model.EquipmentServices, int64, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.ShopID.EQ(String(shopID)),
	}

	if filters.EquipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*filters.EquipmentID)))
	}

	if filters.ServiceType != nil {
		conditions = append(conditions, EquipmentServices.ServiceType.LIKE(String("%"+*filters.ServiceType+"%")))
	}

	if filters.IsCompleted != nil {
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(*filters.IsCompleted)))
	}

	if filters.StartDate != nil {
		startTime, err := time.Parse(time.RFC3339, *filters.StartDate)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid start_date format: %w", err)
		}
		conditions = append(conditions, EquipmentServices.ServiceDate.GT_EQ(TimestampzT(startTime)))
	}

	if filters.EndDate != nil {
		endTime, err := time.Parse(time.RFC3339, *filters.EndDate)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid end_date format: %w", err)
		}
		conditions = append(conditions, EquipmentServices.ServiceDate.LT_EQ(TimestampzT(endTime)))
	}

	countStmt := SELECT(COUNT(STAR)).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...))

	var count countResult
	err := countStmt.Query(repo.db, &count)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}
	totalCount := count.Count

	dataStmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.CreatedAt.DESC()).
		LIMIT(int64(filters.Limit)).
		OFFSET(int64(filters.Offset))

	var services []model.EquipmentServices
	err = dataStmt.Query(repo.db, &services)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get services: %w", err)
	}

	return services, totalCount, nil
}

func (repo *RepositoryImpl) GetByEquipment(user *bootstrap.User, equipmentID string, limit, offset int, startDate, endDate *time.Time) ([]model.EquipmentServices, int64, error) {
	conditions := []postgres.BoolExpression{
		EquipmentServices.EquipmentID.EQ(String(equipmentID)),
	}

	if startDate != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.GT_EQ(TimestampzT(*startDate)))
	}
	if endDate != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.LT_EQ(TimestampzT(*endDate)))
	}

	countStmt := SELECT(COUNT(STAR)).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...))

	var count countResult
	err := countStmt.Query(repo.db, &count)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count services: %w", err)
	}
	totalCount := count.Count

	dataStmt := SELECT(EquipmentServices.AllColumns).FROM(
		EquipmentServices.
			INNER_JOIN(ShopMembers, ShopMembers.ShopID.EQ(EquipmentServices.ShopID)),
	).WHERE(postgres.AND(conditions...)).
		ORDER_BY(EquipmentServices.ServiceDate.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(offset))

	var services []model.EquipmentServices
	err = dataStmt.Query(repo.db, &services)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get services: %w", err)
	}

	return services, totalCount, nil
}
