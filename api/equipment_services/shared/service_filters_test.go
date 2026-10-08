package shared

import (
	"testing"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
	. "miltechserver/.gen/miltech_ng/public/table"
	"miltechserver/api/request"
	shopshared "miltechserver/api/shops/shared"
)

func TestServiceFilterPredicateEvaluationTime(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.FixedZone("offset", -7*3600))
	predicate, err := ServiceFilterPredicate(ServiceFilters{Status: "due_soon"}, now)
	require.NoError(t, err)
	query, args := SELECT(EquipmentServices.ID).FROM(EquipmentServices).WHERE(predicate).Sql()
	require.Contains(t, query, "service_date >")
	require.Contains(t, query, "service_date <=")
	require.Contains(t, args, now)
	require.Contains(t, args, now.Add(7*24*time.Hour))
}

func TestServiceFilterInputValidation(t *testing.T) {
	for _, status := range []string{"", "completed", "overdue", "due_soon", "scheduled"} {
		_, err := ServiceFilterPredicate(ServiceFilters{Status: status}, time.Now())
		require.NoError(t, err)
	}
	_, err := ServiceFilterPredicate(ServiceFilters{Status: "unknown"}, time.Now())
	require.ErrorIs(t, err, ErrInvalidServiceStatus)
	require.Equal(t, "invalid", shopshared.ClassifyFailure(err).Code)
	good := "2026-10-04T12:00:00-07:00"
	bad := "2026-10-04"
	filters, err := ServiceFiltersFromRequest(request.GetEquipmentServicesRequest{StartDate: &good, EndDate: &good})
	require.NoError(t, err)
	require.Equal(t, "2026-10-04T12:00:00-07:00", filters.From.Format(time.RFC3339))
	require.True(t, filters.From.Equal(*filters.To))
	for _, req := range []request.GetEquipmentServicesRequest{{StartDate: &bad}, {EndDate: &bad}} {
		_, err := ServiceFiltersFromRequest(req)
		require.Error(t, err)
		require.Equal(t, "invalid", shopshared.ClassifyFailure(err).Code)
		require.NotContains(t, err.Error(), "parsing time")
	}
}
