package shared

import (
	"time"

	"github.com/go-jet/jet/v2/postgres"
	. "github.com/go-jet/jet/v2/postgres"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/request"
)

type ServiceFilters struct {
	Status, ServiceType string
	IsCompleted         *bool
	EquipmentID         *string
	From, To            *time.Time
}

// ServiceFiltersFromRequest preserves RFC3339 timestamp interpretation. The
// separate calendar-date parser remains behind its existing activation gate.
func ServiceFiltersFromRequest(req request.GetEquipmentServicesRequest) (ServiceFilters, error) {
	filters := ServiceFilters{IsCompleted: req.IsCompleted}
	if req.Status != nil {
		filters.Status = *req.Status
	}
	if req.ServiceType != nil {
		filters.ServiceType = *req.ServiceType
	}
	if req.EquipmentID != nil {
		filters.EquipmentID = req.EquipmentID
	}
	if req.StartDate != nil {
		date, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			return filters, ErrInvalidStartDate
		}
		filters.From = &date
	}
	if req.EndDate != nil {
		date, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			return filters, ErrInvalidEndDate
		}
		filters.To = &date
	}
	return filters, nil
}

// ServiceFilterPredicate is shared by count, page, calendar and status reads so
// supplied filters narrow each other rather than overriding status semantics.
func ServiceFilterPredicate(filters ServiceFilters, evaluationTime time.Time) (postgres.BoolExpression, error) {
	conditions := []postgres.BoolExpression{Bool(true)}
	if filters.EquipmentID != nil {
		conditions = append(conditions, EquipmentServices.EquipmentID.EQ(String(*filters.EquipmentID)))
	}
	if filters.ServiceType != "" {
		conditions = append(conditions, EquipmentServices.ServiceType.LIKE(String("%"+filters.ServiceType+"%")))
	}
	if filters.IsCompleted != nil {
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(*filters.IsCompleted)))
	}
	if filters.From != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.GT_EQ(TimestampzT(*filters.From)))
	}
	if filters.To != nil {
		conditions = append(conditions, EquipmentServices.ServiceDate.LT_EQ(TimestampzT(*filters.To)))
	}
	switch filters.Status {
	case "":
	case "completed":
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(true)))
	case "overdue":
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(false)), EquipmentServices.ServiceDate.IS_NOT_NULL(), EquipmentServices.ServiceDate.LT(TimestampzT(evaluationTime)))
	case "due_soon":
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(false)), EquipmentServices.ServiceDate.GT(TimestampzT(evaluationTime)), EquipmentServices.ServiceDate.LT_EQ(TimestampzT(evaluationTime.Add(7*24*time.Hour))))
	case "scheduled":
		conditions = append(conditions, EquipmentServices.IsCompleted.EQ(Bool(false)), EquipmentServices.ServiceDate.GT(TimestampzT(evaluationTime)))
	default:
		return nil, ErrInvalidServiceStatus
	}
	return postgres.AND(conditions...), nil
}
