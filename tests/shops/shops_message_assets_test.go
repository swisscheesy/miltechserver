package shops_test

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"strings"
	"testing"
	"time"
)

func assetTransaction(fn func(*sql.Tx) error) error {
	tx, err := testDB.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func assetCount(t *testing.T, table, column, id string) int {
	t.Helper()
	var count int
	// All identifiers are literal test code, never request inputs.
	require.NoError(t, testDB.QueryRow("SELECT count(*) FROM "+table+" WHERE "+column+"=$1", id).Scan(&count))
	return count
}
func TestMessageAssetDeletionAuthority(t *testing.T) {
	ctx := context.Background()
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	for _, id := range []string{"asset-owner", "asset-admin", "asset-member", "asset-foreign"} {
		ensureUser(t, testDB, id)
	}
	shopID := createShop(t, router, "asset-admin", "Assets")
	foreignShop := createShop(t, router, "asset-foreign", "Foreign")
	for _, id := range []string{"asset-owner", "asset-member"} {
		_, err := testDB.Exec("INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,$3,'member')", uuid.NewString(), shopID, id)
		require.NoError(t, err)
	}
	owner := &bootstrap.User{UserID: "asset-owner"}
	admin := &bootstrap.User{UserID: "asset-admin"}
	other := &bootstrap.User{UserID: "asset-member"}
	storage := messages.AssetStorage{Account: "owned", Container: "shop-message-images"}
	assets := messages.NewAssetRepository(testDB, storage)
	repo := messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
	reserve := func(actor *bootstrap.User, shop string) messages.Asset {
		var a messages.Asset
		require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
			var err error
			a, err = assets.Reserve(ctx, tx, actor, shop, ".jpg")
			return err
		}))
		require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.Finalize(ctx, tx, actor, a.ID) }))
		return a
	}
	create := func(text string) string {
		now := time.Now()
		row, err := repo.CreateShopMessage(ctx, owner, model.ShopMessages{ID: uuid.NewString(), ShopID: shopID, UserID: owner.UserID, Message: text, CreatedAt: &now, UpdatedAt: &now})
		require.NoError(t, err)
		require.Equal(t, text, row.Message)
		return row.ID
	}
	discard := func(actor *bootstrap.User, a messages.Asset) error {
		return assetTransaction(func(tx *sql.Tx) error { return assets.Discard(ctx, tx, actor, shopID, a.ID) })
	}
	a := reserve(owner, shopID)
	b := reserve(owner, shopID)
	foreign := reserve(&bootstrap.User{UserID: "asset-foreign"}, foreignShop)
	require.Error(t, discard(other, a))
	require.Error(t, discard(owner, foreign))
	escaped := strings.Replace(a.URL, "/"+a.ID, "/%"+fmt.Sprintf("%02x", a.ID[0])+a.ID[1:], 1)
	text := "first [IMAGE:" + a.URL + "?sig=ignored] [IMAGE:" + escaped + "] [IMAGE:" + b.URL + "] [IMAGE:" + foreign.URL + "] [IMAGE:https://external.example/x.jpg]"
	first := create(text)
	second := create("[IMAGE:" + a.URL + "]")
	// Configuration changes cannot erase registry-proven published references.
	rotated := messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "replacement"})
	editTime := time.Now()
	isEdited := true
	require.NoError(t, rotated.UpdateShopMessage(ctx, owner, model.ShopMessages{ID: first, Message: text, UpdatedAt: &editTime, IsEdited: &isEdited}))
	require.Equal(t, 2, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.Equal(t, 1, assetCount(t, "shop_message_asset_references", "upload_id", b.ID))
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", foreign.ID))
	require.Error(t, discard(owner, a))
	require.Error(t, discard(admin, a))
	// An edit retains shared refs, removes all obsolete markers and preserves display text.
	edited := time.Now()
	yes := true
	require.NoError(t, repo.UpdateShopMessage(ctx, owner, model.ShopMessages{ID: first, Message: "[IMAGE:" + a.URL + "] unchanged external [IMAGE:https://external.example/x.jpg]", UpdatedAt: &edited, IsEdited: &yes}))
	require.Equal(t, 2, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", b.ID))
	var retired string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", b.ID).Scan(&retired))
	require.Equal(t, "cleanup_pending", retired)
	require.Error(t, repo.UpdateShopMessage(ctx, owner, model.ShopMessages{ID: first, Message: "[IMAGE:" + b.URL + "]", UpdatedAt: &edited, IsEdited: &yes}))
	require.Equal(t, 2, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))

	require.NoError(t, repo.DeleteShopMessage(ctx, owner, first))
	require.Equal(t, 1, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.Error(t, discard(owner, a))
	require.Greater(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", b.ID), 0)
	require.NoError(t, repo.DeleteShopMessage(ctx, owner, second))
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.NoError(t, discard(owner, a))
	now := time.Now()
	_, err := repo.CreateShopMessage(ctx, owner, model.ShopMessages{ID: uuid.NewString(), ShopID: shopID, UserID: owner.UserID, Message: "[IMAGE:" + a.URL + "]", CreatedAt: &now})
	require.Error(t, err)
	unattached := reserve(owner, shopID)
	require.NoError(t, discard(admin, unattached))
	uploaderDraft := reserve(owner, shopID)
	require.NoError(t, discard(owner, uploaderDraft))
	current := reserve(owner, shopID)
	_, err = testDB.Exec("DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2", shopID, owner.UserID)
	require.NoError(t, err)
	require.Error(t, discard(owner, current))
	_, err = testDB.Exec("UPDATE shop_members SET role='member' WHERE shop_id=$1 AND user_id=$2", shopID, admin.UserID)
	require.NoError(t, err)
	require.Error(t, discard(admin, current))
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", foreign.ID))
}

func TestMessageAssetCascadeDurability(t *testing.T) {
	ctx := context.Background()
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	ensureUser(t, testDB, "asset-cascade")
	shopID := createShop(t, router, "asset-cascade", "Cascade")
	user := &bootstrap.User{UserID: "asset-cascade"}
	assets := messages.NewAssetRepository(testDB, messages.AssetStorage{Account: "owned", Container: "shop-message-images"})
	repo := messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
	var a messages.Asset
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		var err error
		a, err = assets.Reserve(ctx, tx, user, shopID, ".png")
		return err
	}))
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.Finalize(ctx, tx, user, a.ID) }))
	now := time.Now()
	parent := uuid.NewString()
	child := uuid.NewString()
	_, err := repo.CreateShopMessage(ctx, user, model.ShopMessages{ID: parent, ShopID: shopID, UserID: user.UserID, Message: "parent", CreatedAt: &now})
	require.NoError(t, err)
	_, err = repo.CreateShopMessage(ctx, user, model.ShopMessages{ID: child, ShopID: shopID, UserID: user.UserID, ParentID: &parent, Message: "[IMAGE:" + a.URL + "]", CreatedAt: &now})
	require.NoError(t, err)
	_, err = testDB.Exec("UPDATE shop_message_uploads SET blob_key='another' WHERE id=$1", a.ID)
	require.ErrorContains(t, err, "immutable")
	_, err = testDB.Exec("DELETE FROM shop_message_uploads WHERE id=$1", a.ID)
	require.Error(t, err)
	// The verified baseline cascades same-Shop reply deletion.
	require.NoError(t, repo.DeleteShopMessage(ctx, user, parent))
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.Greater(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID), 0)
	ensureUser(t, testDB, "asset-account-author")
	_, err = testDB.Exec("INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,$3,'member')", uuid.NewString(), shopID, "asset-account-author")
	require.NoError(t, err)
	author := &bootstrap.User{UserID: "asset-account-author"}
	_, err = repo.CreateShopMessage(ctx, author, model.ShopMessages{ID: uuid.NewString(), ShopID: shopID, UserID: author.UserID, Message: "[IMAGE:" + a.URL + "]", CreatedAt: &now})
	require.NoError(t, err)
	// Account cascade removes messages; creator-account deletion remains restricted by the baseline.
	_, err = testDB.Exec("DELETE FROM users WHERE uid=$1", author.UserID)
	require.NoError(t, err)
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
	require.Equal(t, 1, assetCount(t, "shop_message_uploads", "id", a.ID))
	require.Greater(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID), 0)
}

func TestMessageAssetReservationCompensation(t *testing.T) {
	ctx := context.Background()
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	ensureUser(t, testDB, "asset-reserve")
	shopID := createShop(t, router, "asset-reserve", "Reserve")
	user := &bootstrap.User{UserID: "asset-reserve"}
	assets := messages.NewAssetRepository(testDB, messages.AssetStorage{Account: "owned", Container: "shop-message-images"})
	var a messages.Asset
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		var err error
		a, err = assets.Reserve(ctx, tx, user, shopID, ".gif")
		return err
	}))
	wrong := a
	wrong.OperationID = uuid.NewString()
	require.Error(t, assetTransaction(func(tx *sql.Tx) error { return assets.FailReserved(ctx, tx, wrong, "secret") }))
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.Finalize(ctx, tx, user, a.ID) }))
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.FailReserved(ctx, tx, a, "secret") }))
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", a.ID).Scan(&state))
	require.Equal(t, "ready", state)
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID))
	// Ready draft lifetime is unbounded: a past upload lease does not expire it.
	_, err := testDB.Exec("UPDATE shop_message_uploads SET lease_until=now()-interval '1 day' WHERE id=$1", a.ID)
	require.NoError(t, err)
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.FailReserved(ctx, tx, a, "secret") }))
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID))
	var interrupted messages.Asset
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		var err error
		interrupted, err = assets.Reserve(ctx, tx, user, shopID, ".png")
		return err
	}))
	_, err = testDB.Exec("DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2", shopID, user.UserID)
	require.NoError(t, err)
	require.Error(t, assetTransaction(func(tx *sql.Tx) error { return assets.Finalize(ctx, tx, user, interrupted.ID) }))
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error { return assets.FailReserved(ctx, tx, interrupted, "sensitive remote error") }))
	require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", interrupted.ID))
	require.Error(t, assetTransaction(func(tx *sql.Tx) error { return assets.Finalize(ctx, tx, user, interrupted.ID) }))
}

func TestMessageAssetShopCleanupIsTransactional(t *testing.T) {
	ctx := context.Background()
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	ensureUser(t, testDB, "asset-shop-cleanup")
	shopID := createShop(t, router, "asset-shop-cleanup", "Cleanup")
	user := &bootstrap.User{UserID: "asset-shop-cleanup"}
	assets := messages.NewAssetRepository(testDB, messages.AssetStorage{Account: "owned", Container: "shop-message-images"})
	var a messages.Asset
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		var err error
		a, err = assets.Reserve(ctx, tx, user, shopID, ".png")
		return err
	}))
	tx, err := testDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, assets.EnqueueShopCleanup(ctx, tx, shopID))
	require.NoError(t, tx.Rollback())
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID))
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", a.ID).Scan(&state))
	require.Equal(t, "uploading", state)
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		if err := assets.EnqueueShopCleanup(ctx, tx, shopID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM shops WHERE id=$1", shopID)
		return err
	}))
	require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID))
	require.Equal(t, 1, assetCount(t, "shop_message_uploads", "id", a.ID))
	var lease time.Time
	require.NoError(t, testDB.QueryRow("SELECT state,lease_until FROM shop_message_uploads WHERE id=$1", a.ID).Scan(&state, &lease))
	require.Equal(t, "cleanup_pending", state)
	require.WithinDuration(t, a.LeaseUntil, lease, time.Microsecond)
}

func TestMessageMutationCancellationReleasesLocks(t *testing.T) {
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	ensureUser(t, testDB, "asset-cancel")
	shopID := createShop(t, router, "asset-cancel", "Cancel")
	user := &bootstrap.User{UserID: "asset-cancel"}
	repo := messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
	locked, err := testDB.Begin()
	require.NoError(t, err)
	defer locked.Rollback()
	_, err = locked.Exec("SELECT id FROM shops WHERE id=$1 FOR UPDATE", shopID)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	now := time.Now()
	id := uuid.NewString()
	_, err = repo.CreateShopMessage(ctx, user, model.ShopMessages{ID: id, ShopID: shopID, UserID: user.UserID, Message: "cancelled", CreatedAt: &now})
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.NoError(t, locked.Rollback())
	require.Zero(t, assetCount(t, "shop_messages", "id", id))
}
