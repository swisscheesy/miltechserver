package shops_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/messages"
	"testing"
	"time"
)

func TestMessageParentSameShop(t *testing.T) {
	shop, user, repo, asset := cleanupAssetFixture(t)
	parent := cleanupMessage(t, repo, user, asset, nil)
	foreignShop := createShop(t, newTestRouter(t), user.UserID, "Also a member")
	foreign, err := repo.ReserveMessageImage(context.Background(), user, foreignShop, ".jpg")
	require.NoError(t, err)
	require.NoError(t, repo.FinalizeMessageImage(context.Background(), user, foreign.ID))
	now := time.Now()
	id := uuid.NewString()
	_, err = repo.CreateShopMessage(context.Background(), user, model.ShopMessages{ID: id, ShopID: foreignShop, UserID: user.UserID, ParentID: &parent, Message: "[IMAGE:" + foreign.URL + "]", CreatedAt: &now})
	require.ErrorContains(t, err, "reply parent is unavailable in this shop")
	require.Zero(t, assetCount(t, "shop_messages", "id", id))
	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", foreign.ID))
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "shop_id", foreignShop))
	require.Equal(t, "ready", cleanupState(t, foreign.ID))
	child := cleanupMessage(t, repo, user, asset, &parent)
	require.Equal(t, 1, assetCount(t, "shop_messages", "id", child))
	require.Equal(t, 2, assetCount(t, "shop_messages", "shop_id", shop))
	missing := uuid.NewString()
	_, err = repo.CreateShopMessage(context.Background(), user, model.ShopMessages{ID: uuid.NewString(), ShopID: shop, UserID: user.UserID, ParentID: &missing, Message: "missing", CreatedAt: &now})
	require.ErrorContains(t, err, "reply parent is unavailable in this shop")
}

func TestMessageParentDatabaseOwnership(t *testing.T) {
	shop, user, repo, asset := cleanupAssetFixture(t)
	parent := cleanupMessage(t, repo, user, asset, nil)
	foreignShop := createShop(t, newTestRouter(t), user.UserID, "Foreign")
	_, err := testDB.Exec("INSERT INTO shop_messages(id,shop_id,user_id,message,parent_id) VALUES($1,$2,$3,'foreign',$4)", uuid.NewString(), foreignShop, user.UserID, parent)
	require.Error(t, err, "database must reject a foreign parent")
	require.Equal(t, 1, assetCount(t, "shop_messages", "shop_id", shop))
	require.Zero(t, assetCount(t, "shop_messages", "shop_id", foreignShop))
}

func TestMessageParentConcurrentDeletion(t *testing.T) {
	for _, outcome := range []string{"commit", "rollback", "cancel"} {
		t.Run(outcome, func(t *testing.T) {
			shop, user, repo, asset := cleanupAssetFixture(t)
			parent := cleanupMessage(t, repo, user, asset, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tx, err := testDB.BeginTx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(ctx, "DELETE FROM shop_messages WHERE id=$1", parent)
			require.NoError(t, err)
			id := uuid.NewString()
			done := make(chan error, 1)
			requestCtx, stop := context.WithCancel(ctx)
			defer stop()
			go func() {
				_, e := repo.CreateShopMessage(requestCtx, user, model.ShopMessages{ID: id, ShopID: shop, UserID: user.UserID, Message: "[IMAGE:" + asset.URL + "]", ParentID: &parent})
				done <- e
			}()
			waitAuthorityBlock(t, tx, ctx)
			switch outcome {
			case "commit":
				require.NoError(t, tx.Commit())
				require.ErrorContains(t, <-done, "reply parent is unavailable in this shop")
				require.Zero(t, assetCount(t, "shop_messages", "id", id))
				require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", asset.ID))
			case "rollback":
				require.NoError(t, tx.Rollback())
				require.NoError(t, <-done)
				require.Equal(t, 2, assetCount(t, "shop_message_asset_references", "upload_id", asset.ID))
			case "cancel":
				stop()
				select {
				case e := <-done:
					require.Error(t, e)
				case <-ctx.Done():
					t.Fatal("parent lookup ignored cancellation")
				}
				require.NoError(t, tx.Rollback())
				require.Zero(t, assetCount(t, "shop_messages", "id", id))
				require.Equal(t, 1, assetCount(t, "shop_message_asset_references", "upload_id", asset.ID))
			}
		})
	}
}

func TestMessageParentCascadeAssets(t *testing.T) {
	shop, user, repo, asset := cleanupAssetFixture(t)
	parent := cleanupMessage(t, repo, user, asset, nil)
	child := cleanupMessage(t, repo, user, asset, &parent)
	grandchild := cleanupMessage(t, repo, user, asset, &child)
	foreignShop := createShop(t, newTestRouter(t), user.UserID, "Untouched")
	foreign, err := repo.ReserveMessageImage(context.Background(), user, foreignShop, ".png")
	require.NoError(t, err)
	require.NoError(t, repo.FinalizeMessageImage(context.Background(), user, foreign.ID))
	foreignMessage := cleanupMessage(t, repo, user, foreign, nil)
	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, parent))
	for _, id := range []string{parent, child, grandchild} {
		require.Zero(t, assetCount(t, "shop_messages", "id", id))
	}
	require.Zero(t, assetCount(t, "shop_message_asset_references", "shop_id", shop))
	require.Equal(t, 1, assetCount(t, "shop_message_uploads", "id", asset.ID))
	require.Greater(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", asset.ID), 0)
	expireCleanupUpload(t, asset.ID)
	r := cleanupRepo(t, "owned")
	targets := []string{}
	cloud := &cleanupCloud{delete: func(ctx context.Context, a messages.Asset) error {
		assertCleanupOutsideLocks(t, a)
		require.Equal(t, shop, a.ShopID)
		require.Equal(t, asset.BlobKey, a.BlobKey)
		require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
		targets = append(targets, a.BlobKey)
		return nil
	}}
	for _, job := range claimCleanup(t, r) {
		target, ready, e := r.Prepare(context.Background(), job)
		require.NoError(t, e)
		require.True(t, ready)
		require.NoError(t, cloud.Delete(context.Background(), target))
		require.NoError(t, r.Finish(context.Background(), job.ID, nil))
	}
	require.NotEmpty(t, targets)
	require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
	require.Equal(t, "deleted", cleanupState(t, asset.ID))
	require.Equal(t, "ready", cleanupState(t, foreign.ID))
	require.Equal(t, 1, assetCount(t, "shop_messages", "id", foreignMessage))
	require.Equal(t, 1, assetCount(t, "shop_message_asset_references", "upload_id", foreign.ID))
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "shop_id", foreignShop))
}
