package shops_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
)

const (
	legacyImagesForward = "../../migrations/024_register_legacy_shop_message_images.sql"
	legacyImagesReverse = "../../migrations/024_rollback_register_legacy_shop_message_images.sql"
	legacyUploader      = "legacy:unattributed"
)

// The pre-020 server built exactly this URL for every message image upload.
func legacyImageURL(shopID, imageID, extension string) string {
	return "https://owned.blob.core.windows.net/shop-message-images/" + shopID + "/" + imageID + extension
}

// Inserts bypass the repository so no reference exists, matching pre-020 rows.
func insertLegacyMessage(t *testing.T, shopID, userID, text string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := testDB.Exec(`INSERT INTO shop_messages(id,shop_id,user_id,message,created_at) VALUES($1,$2,$3,$4,now())`, id, shopID, userID, text)
	require.NoError(t, err)
	return id
}

// Each migration file owns its BEGIN/COMMIT; a failure leaves the session in
// an aborted transaction that must be closed before the connection is reused.
func runLegacyImagesMigration(t *testing.T, path string) error {
	t.Helper()
	sqlText, err := os.ReadFile(path)
	require.NoError(t, err)
	conn, err := testDB.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(context.Background(), string(sqlText))
	if err != nil {
		_, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK")
		require.NoError(t, rollbackErr)
	}
	return err
}

func tableCount(t *testing.T, table string) int {
	t.Helper()
	var count int
	// Table names are literal test code, never request inputs.
	require.NoError(t, testDB.QueryRow("SELECT count(*) FROM "+table).Scan(&count))
	return count
}

func TestLegacyImageBackfillRegistersAndCleansUp(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	sharedImage, singleImage := uuid.NewString(), uuid.NewString()
	first := insertLegacyMessage(t, shop, user.UserID, "look [IMAGE:"+legacyImageURL(shop, sharedImage, ".jpg")+"]")
	second := insertLegacyMessage(t, shop, user.UserID, "[IMAGE:"+legacyImageURL(shop, sharedImage, ".jpg")+"] again")
	insertLegacyMessage(t, shop, user.UserID, "[IMAGE:"+legacyImageURL(shop, singleImage, ".png")+"]")
	insertLegacyMessage(t, shop, user.UserID, "plain text")

	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward))

	require.Equal(t, 2, tableCount(t, "shop_message_uploads"))
	require.Equal(t, 3, tableCount(t, "shop_message_asset_references"))
	require.Zero(t, tableCount(t, "shop_message_blob_cleanup_jobs"))
	var uploader, blobKey, url, state string
	require.NoError(t, testDB.QueryRow("SELECT uploader_id, blob_key, url, state FROM shop_message_uploads WHERE id=$1", sharedImage).Scan(&uploader, &blobKey, &url, &state))
	require.Equal(t, legacyUploader, uploader)
	require.Equal(t, shop+"/"+sharedImage+".jpg", blobKey)
	require.Equal(t, legacyImageURL(shop, sharedImage, ".jpg"), url)
	require.Equal(t, "ready", state)
	require.Equal(t, 2, assetCount(t, "shop_message_asset_references", "upload_id", sharedImage))

	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward), "a repeated run must be a no-op")
	require.Equal(t, 2, tableCount(t, "shop_message_uploads"))
	require.Equal(t, 3, tableCount(t, "shop_message_asset_references"))

	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, first))
	cleanup := cleanupRepo(t, "owned")
	jobs := claimCleanup(t, cleanup)
	require.Len(t, jobs, 1)
	_, ready, err := cleanup.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.False(t, ready, "the second message still shows the shared image")
	require.Equal(t, "ready", cleanupState(t, sharedImage))

	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, second))
	jobs = claimCleanup(t, cleanup)
	require.Len(t, jobs, 1)
	target, ready, err := cleanup.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.True(t, ready)
	require.Equal(t, shop+"/"+sharedImage+".jpg", target.BlobKey)
	// The Azure store refuses any individual key not shaped shop/upload-id+extension.
	require.Equal(t, target.ShopID+"/"+target.ID+target.Extension, target.BlobKey)
	require.NoError(t, cleanup.Finish(context.Background(), jobs[0].ID, nil))
	require.NoError(t, cleanup.Reconcile(context.Background()))
	require.Equal(t, "deleted", cleanupState(t, sharedImage))
	require.Equal(t, "ready", cleanupState(t, singleImage))
}

func TestLegacyImageBackfillEditRemovingImageQueuesCleanup(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	image := uuid.NewString()
	message := insertLegacyMessage(t, shop, user.UserID, "[IMAGE:"+legacyImageURL(shop, image, ".webp")+"]")
	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward))

	now, edited := time.Now(), true
	require.NoError(t, repo.UpdateShopMessage(context.Background(), user, model.ShopMessages{ID: message, ShopID: shop, Message: "image removed", UpdatedAt: &now, IsEdited: &edited}))

	require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", image))
	require.Equal(t, "cleanup_pending", cleanupState(t, image))
}

func TestLegacyImageBackfillSkipsManagedUploads(t *testing.T) {
	_, user, repo, managed := cleanupAssetFixture(t)
	cleanupMessage(t, repo, user, managed, nil)

	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward))

	require.Equal(t, 1, tableCount(t, "shop_message_uploads"))
	require.Equal(t, 1, tableCount(t, "shop_message_asset_references"))
	var uploader string
	require.NoError(t, testDB.QueryRow("SELECT uploader_id FROM shop_message_uploads WHERE id=$1", managed.ID).Scan(&uploader))
	require.Equal(t, user.UserID, uploader)
}

func TestLegacyImageBackfillRefusesUnsafeReferences(t *testing.T) {
	cases := []struct {
		name   string
		seed   func(shop, userID, image string)
		reason string
	}{
		{
			name: "non-canonical marker",
			seed: func(shop, userID, image string) {
				insertLegacyMessage(t, shop, userID, "[IMAGE:"+legacyImageURL(shop, image, ".jpg")+"?size=small]")
			},
			reason: "024: non-canonical historical image reference",
		},
		{
			name: "image path from another shop",
			seed: func(shop, userID, image string) {
				insertLegacyMessage(t, shop, userID, "[IMAGE:"+legacyImageURL(uuid.NewString(), image, ".jpg")+"]")
			},
			reason: "024: historical image path belongs to a different shop",
		},
		{
			name: "identity mentioned outside a marker",
			seed: func(shop, userID, image string) {
				insertLegacyMessage(t, shop, userID, "[IMAGE:"+legacyImageURL(shop, image, ".jpg")+"]")
				insertLegacyMessage(t, shop, userID, "see image "+image)
			},
			reason: "024: historical image identity appears outside a canonical marker",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shop, user, _ := uploadFixture(t)
			tc.seed(shop, user.UserID, uuid.NewString())

			err := runLegacyImagesMigration(t, legacyImagesForward)

			require.ErrorContains(t, err, tc.reason)
			require.Zero(t, tableCount(t, "shop_message_uploads"))
			require.Zero(t, tableCount(t, "shop_message_asset_references"))
		})
	}
}

func TestLegacyImageBackfillReverse(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	image := uuid.NewString()
	message := insertLegacyMessage(t, shop, user.UserID, "[IMAGE:"+legacyImageURL(shop, image, ".gif")+"]")
	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward))

	require.NoError(t, runLegacyImagesMigration(t, legacyImagesReverse))
	require.Zero(t, tableCount(t, "shop_message_uploads"))
	require.Zero(t, tableCount(t, "shop_message_asset_references"))
	require.Zero(t, tableCount(t, "shop_message_blob_cleanup_jobs"), "reverse must not queue blob deletion")

	require.NoError(t, runLegacyImagesMigration(t, legacyImagesForward))
	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, message))
	err := runLegacyImagesMigration(t, legacyImagesReverse)
	require.ErrorContains(t, err, "024: legacy image cleanup has started")
	require.Equal(t, "ready", cleanupState(t, image))
	require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", image))
}
