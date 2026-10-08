package status

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

type RepositoryImpl struct{ db *sql.DB }

func NewRepository(db *sql.DB) *RepositoryImpl { return &RepositoryImpl{db: db} }

func (repo *RepositoryImpl) GetOverdue(ctx context.Context, user *bootstrap.User, shopID string, equipmentID *string, limit int) ([]ServiceWithDays, error) {
	return repo.getByStatus(ctx, user, shopID, "overdue", 0, equipmentID, limit)
}
func (repo *RepositoryImpl) GetDueSoon(ctx context.Context, user *bootstrap.User, shopID string, daysAhead int, equipmentID *string, limit int) ([]ServiceWithDays, error) {
	if daysAhead < 1 || daysAhead > 30 {
		return nil, shared.ErrInvalidDaysAhead
	}

	return repo.getByStatus(ctx, user, shopID, "scheduled", daysAhead, equipmentID, limit)
}
func (repo *RepositoryImpl) getByStatus(ctx context.Context, user *bootstrap.User, shopID, status string, daysAhead int, equipmentID *string, limit int) ([]ServiceWithDays, error) {
	if user == nil {
		return nil, shared.ErrUnauthorizedUser
	}
	evaluationTime := time.Now()
	filters := shared.ServiceFilters{Status: status}
	if equipmentID != nil {
		filters.EquipmentID = equipmentID
	}
	if daysAhead > 0 {
		futureDate := evaluationTime.AddDate(0, 0, daysAhead)
		filters.To = &futureDate
	}
	predicate, err := shared.ServiceFilterPredicate(filters, evaluationTime)
	if err != nil {
		return nil, err
	}
	// Keep PostgreSQL interval extraction: Go duration subtraction saturates
	// for accepted timestamps more than roughly 290 years in the past.
	dayDifference := "EXTRACT(DAY FROM CAST(:evaluation_time AS timestamptz) - equipment_services.service_date)"
	if status == "scheduled" {
		dayDifference = "EXTRACT(DAY FROM equipment_services.service_date - CAST(:evaluation_time AS timestamptz))"
	}
	stmt := SELECT(
		EquipmentServices.AllColumns,
		Raw(dayDifference, RawArgs{":evaluation_time": evaluationTime}).AS("days_count"),
	).FROM(
		EquipmentServices.INNER_JOIN(ShopMembers,
			ShopMembers.ShopID.EQ(EquipmentServices.ShopID).AND(ShopMembers.UserID.EQ(String(user.UserID))),
		),
	).WHERE(EquipmentServices.ShopID.EQ(String(shopID)).AND(predicate)).
		ORDER_BY(EquipmentServices.ServiceDate.ASC().NULLS_LAST(), EquipmentServices.ID.ASC()).LIMIT(int64(limit))
	var results []struct {
		model.EquipmentServices
		DaysCount int `sql:"days_count"`
	}
	if err := stmt.QueryContext(ctx, repo.db, &results); err != nil {
		return nil, fmt.Errorf("failed to get status services: %w", err)
	}
	services := make([]ServiceWithDays, len(results))
	for i, result := range results {
		services[i] = ServiceWithDays{EquipmentServices: result.EquipmentServices, DaysCount: result.DaysCount}
	}
	return services, nil
}

var _ Repository = (*RepositoryImpl)(nil)
