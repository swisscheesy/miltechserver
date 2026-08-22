package shops_test

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestShopVehicleTrackedUsageConstraints(t *testing.T) {
	for _, constraintName := range []string{
		"shop_vehicle_tracked_mileage_nonnegative",
		"shop_vehicle_tracked_hours_nonnegative",
	} {
		var validated bool
		err := testDB.QueryRow(`
			SELECT convalidated
			FROM pg_catalog.pg_constraint
			WHERE conrelid = 'public.shop_vehicle'::regclass
			  AND conname = $1`, constraintName).Scan(&validated)
		require.NoError(t, err)
		require.True(t, validated)
	}
}

func TestShopVehicleTrackedUsageConstraintsRejectNegativeValue(t *testing.T) {
	fixture := newVehicleUsageFixture(t, 100, 50)
	tx, err := testDB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE public.shop_vehicle SET tracked_mileage = -1 WHERE id = $1`,
		fixture.vehicleID,
	)
	require.Error(t, err)
	var postgresError *pq.Error
	require.ErrorAs(t, err, &postgresError)
	require.Equal(
		t,
		"shop_vehicle_tracked_mileage_nonnegative",
		postgresError.Constraint,
	)
}
