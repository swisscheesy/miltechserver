package shops_test

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/lists"
	listitems "miltechserver/api/shops/lists/items"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"miltechserver/tests/testutil"
	"os"
	"testing"
	"time"
)

func TestListServicePhysicalPoolCancellation(t *testing.T) {
	router, shopID, _ := atomicFixture(t, "atomic-owner")
	listID := createList(t, router, "atomic-owner", shopID)
	poolDB, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	defer poolDB.Close()
	repo := lists.NewRepository(poolDB)
	service := lists.NewService(repo, nil, shared.NewShopAuthorization(poolDB))
	user := &bootstrap.User{UserID: "atomic-owner"}
	// Hold the sole physical connection so the first repository lookup cannot run.
	poolDB.SetMaxOpenConns(1)
	conn, err := poolDB.Conn(context.Background())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := service.GetShopListByID(ctx, user, listID); done <- err }()
	var result error
	canceledBeforeRelease := false
	select {
	case result = <-done:
		canceledBeforeRelease = true
	case <-time.After(400 * time.Millisecond):
	}
	require.NoError(t, conn.Close())
	if result == nil {
		select {
		case result = <-done:
		case <-time.After(time.Second):
			t.Fatal("pool call did not finish after release")
		}
	}
	require.True(t, canceledBeforeRelease, "request was blocked until the occupied connection was released")
	require.True(t, errors.Is(result, context.DeadlineExceeded), "request context must cancel the pool wait: %v", result)
}

func TestListRepositoryPhysicalCancellation(t *testing.T) {
	router, shopID, _ := atomicFixture(t, "atomic-owner")
	listID := createList(t, router, "atomic-owner", shopID)
	itemID := createListItem(t, router, "atomic-owner", listID, "seed", "Seed")
	poolDB, e := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, e)
	defer poolDB.Close()
	repo, items := lists.NewRepository(poolDB), listitems.NewRepository(poolDB)
	user := &bootstrap.User{UserID: "atomic-owner"}
	item := model.ShopListItems{ID: itemID, ListID: listID, Niin: "seed", Nomenclature: "Seed", Quantity: 1}
	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{"CreateShopList", func(ctx context.Context) error {
			_, e := repo.CreateShopList(ctx, user, model.ShopLists{ShopID: shopID})
			return e
		}},
		{"GetShopLists", func(ctx context.Context) error { _, e := repo.GetShopLists(ctx, user, shopID); return e }},
		{"GetShopListByID", func(ctx context.Context) error { _, e := repo.GetShopListByID(ctx, user, listID); return e }},
		{"UpdateShopList", func(ctx context.Context) error { return repo.UpdateShopList(ctx, user, model.ShopLists{ID: listID}) }},
		{"DeleteShopList", func(ctx context.Context) error { return repo.DeleteShopList(ctx, user, listID) }},
		{"AddListItem", func(ctx context.Context) error { _, e := items.AddListItem(ctx, user, item); return e }},
		{"GetListItems", func(ctx context.Context) error { _, e := items.GetListItems(ctx, user, listID); return e }},
		{"GetListItemByID", func(ctx context.Context) error { _, e := items.GetListItemByID(ctx, user, itemID); return e }},
		{"UpdateListItem", func(ctx context.Context) error { return items.UpdateListItem(ctx, user, item) }},
		{"RemoveListItem", func(ctx context.Context) error { return items.RemoveListItem(ctx, user, itemID) }},
		{"AddListItemBatch", func(ctx context.Context) error {
			_, e := items.AddListItemBatch(ctx, user, []model.ShopListItems{item})
			return e
		}},
		{"RemoveListItemBatch", func(ctx context.Context) error {
			_, err := items.RemoveListItemBatch(ctx, user, []string{itemID})
			return err
		}},
	}
	poolDB.SetMaxOpenConns(1)
	for _, tc := range cases {
		t.Run("pool/"+tc.name, func(t *testing.T) {
			before := writerSnapshot(t)
			conn, e := poolDB.Conn(context.Background())
			require.NoError(t, e)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			e = tc.call(ctx)
			require.NoError(t, conn.Close())
			require.ErrorIs(t, e, context.DeadlineExceeded)
			require.Equal(t, before, writerSnapshot(t))
		})
	}
	poolDB.SetMaxOpenConns(4)
	for _, tc := range cases {
		if tc.name == "GetShopLists" || tc.name == "GetShopListByID" || tc.name == "GetListItems" || tc.name == "GetListItemByID" {
			continue
		}
		t.Run("shop-lock/"+tc.name, func(t *testing.T) {
			before := writerSnapshot(t)
			lock, e := testDB.BeginTx(context.Background(), nil)
			require.NoError(t, e)
			_, e = lock.Exec(`SELECT id FROM shops WHERE id=$1 FOR UPDATE`, shopID)
			require.NoError(t, e)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			e = tc.call(ctx)
			require.NoError(t, lock.Rollback())
			require.Error(t, e)
			require.Error(t, ctx.Err())
			require.Less(t, time.Since(started), time.Second)
			require.Equal(t, before, writerSnapshot(t))
		})
	}
}
