package equipment_services_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"miltechserver/api/equipment_services/calendar"
	"miltechserver/api/equipment_services/completion"
	"miltechserver/api/equipment_services/core"
	"miltechserver/api/equipment_services/queries"
	serviceshared "miltechserver/api/equipment_services/shared"
	"miltechserver/api/equipment_services/status"
	"miltechserver/api/request"
	shopshared "miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"

	"github.com/stretchr/testify/require"
)

type observedUsernameLookup struct {
	resolver serviceshared.UsernameResolver
	entered  chan struct{}
}

func (lookup observedUsernameLookup) GetUsernameByUserID(ctx context.Context, userID string) (string, error) {
	select {
	case lookup.entered <- struct{}{}:
	default:
	}
	return lookup.resolver.GetUsernameByUserID(ctx, userID)
}

// Returning fallback success after a canceled username query must fail each
// service boundary, even when the primary data read/write already succeeded.
func TestServiceUsernameLookupCancellation(t *testing.T) {
	for _, operation := range []string{"Create", "GetByID", "Update", "Complete", "GetByShop", "GetByEquipment", "GetCalendarServices", "GetOverdue", "GetDueSoon"} {
		t.Run(operation, func(t *testing.T) {
			clearEquipmentServicesTables(t, testDB)
			t.Cleanup(func() { clearEquipmentServicesTables(t, testDB) })
			const actor = "username-cancellation-user"
			ensureUser(t, testDB, actor)
			router := newTestRouter(t)
			shopID := createShop(t, router, actor, "Username cancellation")
			equipmentID := createVehicle(t, router, actor, shopID)
			now := time.Now().UTC()
			past, future := now.Add(-48*time.Hour), now.Add(48*time.Hour)
			serviceID := createEquipmentService(t, router, actor, shopID, equipmentID, "", "original", &past, false)
			otherID := createEquipmentService(t, router, actor, shopID, equipmentID, "", "unchanged", &future, false)
			before := usernameCancellationRows(t)

			lookupDB, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
			require.NoError(t, err)
			defer lookupDB.Close()
			lookupDB.SetMaxOpenConns(1)
			outer, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			reserved, err := lookupDB.Conn(outer)
			require.NoError(t, err)
			defer reserved.Close()
			ctx, cancel := context.WithCancel(outer)
			defer cancel()
			lookup := observedUsernameLookup{serviceshared.NewUsernameRepository(lookupDB), make(chan struct{}, 1)}
			authorization := serviceshared.NewAuthorization(testDB, shopshared.NewShopAuthorization(testDB))
			coreService := core.NewService(core.NewRepository(testDB), authorization, lookup)
			queryService := queries.NewService(queries.NewRepository(testDB), authorization, lookup)
			user := &bootstrap.User{UserID: actor}
			type outcome struct {
				value any
				err   error
			}
			done := make(chan outcome, 1)
			go func() {
				var value any
				var err error
				switch operation {
				case "Create":
					value, err = coreService.Create(ctx, user, shopID, request.CreateEquipmentServiceRequest{EquipmentID: equipmentID, Description: "created before cancellation", ServiceType: "inspection"})
				case "GetByID":
					value, err = coreService.GetByID(ctx, user, shopID, serviceID)
				case "Update":
					value, err = coreService.Update(ctx, user, shopID, request.UpdateEquipmentServiceRequest{ServiceID: serviceID, Description: "updated before cancellation", ServiceType: "inspection", ServiceDate: &past})
				case "Complete":
					value, err = completion.NewService(completion.NewRepository(testDB), authorization, lookup).Complete(ctx, user, shopID, serviceID, request.CompleteEquipmentServiceRequest{CompletionDate: &now})
				case "GetByShop":
					value, err = queryService.GetByShop(ctx, user, shopID, request.GetEquipmentServicesRequest{Limit: 50})
				case "GetByEquipment":
					value, err = queryService.GetByEquipment(ctx, user, equipmentID, request.GetEquipmentServicesRequest{Limit: 50})
				case "GetCalendarServices":
					value, err = calendar.NewService(calendar.NewRepository(testDB), authorization, lookup).GetCalendarServices(ctx, user, shopID, request.GetCalendarServicesRequest{StartDate: past.Add(-time.Hour).Format(time.RFC3339), EndDate: future.Add(time.Hour).Format(time.RFC3339)})
				case "GetOverdue":
					value, err = status.NewService(status.NewRepository(testDB), authorization, lookup).GetOverdue(ctx, user, shopID, request.GetOverdueServicesRequest{Limit: 50})
				case "GetDueSoon":
					value, err = status.NewService(status.NewRepository(testDB), authorization, lookup).GetDueSoon(ctx, user, shopID, request.GetDueSoonServicesRequest{DaysAhead: 7, Limit: 50})
				}
				done <- outcome{value, err}
			}()
			select {
			case <-lookup.entered:
			case <-outer.Done():
				t.Fatal("service never reached username lookup")
			}
			require.Eventually(t, func() bool { return lookupDB.Stats().WaitCount > 0 }, time.Second, time.Millisecond, "real username query must wait for the occupied pool")
			cancel()
			var result outcome
			select {
			case result = <-done:
			case <-outer.Done():
				t.Fatal("canceled username lookup did not release pool wait")
			}
			after := usernameCancellationRows(t)
			require.Equal(t, before[otherID], after[otherID], "unrelated service must not change")
			switch operation {
			case "Create":
				require.Len(t, after, 3)
				require.Equal(t, before[serviceID], after[serviceID])
				var count int
				require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM equipment_services WHERE description='created before cancellation'`).Scan(&count))
				require.Equal(t, 1, count, "already committed create persists exactly once")
			case "Update":
				require.Len(t, after, 2)
				var description string
				require.NoError(t, testDB.QueryRow(`SELECT description FROM equipment_services WHERE id=$1`, serviceID).Scan(&description))
				require.Equal(t, "updated before cancellation", description)
			case "Complete":
				require.Len(t, after, 2)
				var completed bool
				var date time.Time
				require.NoError(t, testDB.QueryRow(`SELECT is_completed,completion_date FROM equipment_services WHERE id=$1`, serviceID).Scan(&completed, &date))
				require.True(t, completed)
				require.WithinDuration(t, now, date, time.Microsecond)
			default:
				require.Equal(t, before, after, "read cancellation cannot mutate persisted services")
			}
			require.ErrorIs(t, result.err, context.Canceled, "canceled username lookup must reach the service caller")
			require.Nil(t, result.value, "failed lookup must not return a success response")
		})
	}
}

func usernameCancellationRows(t *testing.T) map[string]string {
	t.Helper()
	rows, err := testDB.Query(`SELECT id,row_to_json(s)::text FROM equipment_services s ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var id, body string
		require.NoError(t, rows.Scan(&id, &body))
		require.True(t, json.Valid([]byte(body)))
		result[id] = body
	}
	require.NoError(t, rows.Err())
	return result
}

var _ serviceshared.UsernameResolver = observedUsernameLookup{}
