package shops_test

import (
	"context"
	"errors"
	"testing"
	"time"

	shopcore "miltechserver/api/shops/core"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

func TestShopEquipmentOverviewRepository(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ensureUser(t, testDB, "other-user")
	router := newTestRouter(t)

	newerShopID := createShop(t, router, "overview-user", "Newer Shop")
	olderShopID := createShop(t, router, "overview-user", "Older Shop")
	hiddenShopID := createShop(t, router, "other-user", "Hidden Shop")
	newerEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	_, err := testDB.Exec(`UPDATE shop_vehicle SET admin=$1, serial=$2 WHERE id=$3`,
		"TEMP", "SER-TEMP", newerEquipmentID)
	require.NoError(t, err)
	olderEquipmentID := createVehicle(t, router, "overview-user", newerShopID)
	_ = createVehicle(t, router, "other-user", hiddenShopID)

	newerTime := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	olderTime := newerTime.Add(-time.Hour)
	_, err = testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, newerTime, newerShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shops SET created_at = $1 WHERE id = $2`, olderTime, olderShopID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"NEW", "M1097", "SER-NEW", "000000001", newerTime, newerEquipmentID)
	require.NoError(t, err)
	_, err = testDB.Exec(`UPDATE shop_vehicle SET admin=$1, model=$2, serial=$3, niin=$4, save_time=$5 WHERE id=$6`,
		"OLD", "M998", "SER-OLD", "000000002", olderTime, olderEquipmentID)
	require.NoError(t, err)

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	shops, err := repository.GetShopEquipmentOverview(context.Background(), &bootstrap.User{UserID: "overview-user"})
	require.NoError(t, err)
	require.Len(t, shops, 2)
	require.Equal(t, newerShopID, shops[0].ID)
	require.Equal(t, olderShopID, shops[1].ID)
	require.Len(t, shops[0].Equipment, 2)
	require.Equal(t, newerEquipmentID, shops[0].Equipment[0].ID)
	require.Equal(t, olderEquipmentID, shops[0].Equipment[1].ID)
	require.Equal(t, "NEW", shops[0].Equipment[0].Admin)
	require.Equal(t, "M1097", shops[0].Equipment[0].Model)
	require.Equal(t, "SER-NEW", shops[0].Equipment[0].Serial)
	require.Equal(t, "000000001", shops[0].Equipment[0].Niin)
	require.Empty(t, shops[1].Equipment)
	for _, shop := range shops {
		require.NotEqual(t, hiddenShopID, shop.ID)
	}
}

func TestShopEquipmentOverviewRepositoryHonorsCanceledContext(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "overview-user")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := shopcore.NewRepository(testDB, nil, &bootstrap.Env{})
	_, err := repository.GetShopEquipmentOverview(ctx, &bootstrap.User{UserID: "overview-user"})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), err.Error())
}
