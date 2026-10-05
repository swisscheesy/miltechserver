package shops_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"testing"
	"time"
)

type uploadBlobStore struct {
	put            func(context.Context, messages.Asset) error
	calls, deletes int
}

func (b *uploadBlobStore) Upload(ctx context.Context, a messages.Asset, _ []byte, _ string) error {
	b.calls++
	return b.put(ctx, a)
}
func (b *uploadBlobStore) Delete(context.Context, messages.Asset) error { b.deletes++; return nil }

type uploadFinalizationRepository struct {
	messages.Repository
	failure                 string
	compensationUnavailable bool
}

func (r *uploadFinalizationRepository) FinalizeMessageImage(ctx context.Context, u *bootstrap.User, id string) error {
	if r.failure == "before_commit" {
		return errors.New("finalization unavailable")
	}
	if err := r.Repository.FinalizeMessageImage(ctx, u, id); err != nil {
		return err
	}
	if r.failure == "ambiguous_commit" {
		return errors.New("commit acknowledgement lost")
	}
	return nil
}
func (r *uploadFinalizationRepository) FailMessageImage(ctx context.Context, a messages.Asset) error {
	if r.compensationUnavailable {
		return errors.New("database unavailable")
	}
	return r.Repository.FailMessageImage(ctx, a)
}
func uploadFixture(t *testing.T) (string, *bootstrap.User, *messages.RepositoryImpl) {
	t.Helper()
	clearShopTables(t, testDB)
	router := newTestRouter(t)
	ensureUser(t, testDB, "upload-owner")
	shop := createShop(t, router, "upload-owner", "Upload")
	return shop, &bootstrap.User{UserID: "upload-owner"}, messages.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
}
func assertUploadCommittedWithoutLocks(t *testing.T, ctx context.Context, a messages.Asset) {
	t.Helper()
	var state string
	require.NoError(t, testDB.QueryRowContext(ctx, "SELECT state FROM shop_message_uploads WHERE id=$1", a.ID).Scan(&state))
	require.Equal(t, "uploading", state)
	var active int
	require.NoError(t, testDB.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND xact_start IS NOT NULL").Scan(&active))
	require.Zero(t, active, "PUT must not overlap a SQL transaction")
	tx, e := testDB.BeginTx(ctx, nil)
	require.NoError(t, e)
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, "SELECT id FROM shops WHERE id=$1 FOR UPDATE NOWAIT", a.ShopID)
	require.NoError(t, e)
	_, e = tx.ExecContext(ctx, "SELECT id FROM shop_message_uploads WHERE id=$1 FOR UPDATE NOWAIT", a.ID)
	require.NoError(t, e)
	require.NoError(t, tx.Rollback())
}
func TestUploadRevokedAfterPut(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	var asset messages.Asset
	blob := &uploadBlobStore{put: func(ctx context.Context, a messages.Asset) error {
		asset = a
		assertUploadCommittedWithoutLocks(t, ctx, a)
		_, e := testDB.ExecContext(ctx, "DELETE FROM shop_members WHERE shop_id=$1 AND user_id=$2", shop, user.UserID)
		return e
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, e := messages.NewService(repo, nil).WithBlobStore(blob).UploadMessageImage(ctx, user, shop, []byte("image"), "image/jpeg")
	require.Error(t, e)
	require.Empty(t, result)
	require.Equal(t, 1, blob.calls)
	require.Zero(t, blob.deletes)
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", asset.ID).Scan(&state))
	require.Equal(t, "cleanup_pending", state)
	require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", asset.ID))
}
func TestUploadLegacyResponseKeys(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	for _, format := range []struct{ mime, ext string }{{"image/jpeg", ".jpg"}, {"image/png", ".png"}, {"image/gif", ".gif"}, {"image/webp", ".webp"}, {"", ".jpg"}} {
		t.Run(format.ext+format.mime, func(t *testing.T) {
			blob := &uploadBlobStore{put: func(ctx context.Context, a messages.Asset) error {
				assertUploadCommittedWithoutLocks(t, ctx, a)
				return nil
			}}
			result, e := messages.NewService(repo, nil).WithBlobStore(blob).UploadMessageImage(context.Background(), user, shop, []byte("image"), format.mime)
			require.NoError(t, e)
			require.Equal(t, format.ext, result.FileExtension)
			require.Equal(t, shop, result.ShopID)
			data, e := json.Marshal(result)
			require.NoError(t, e)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(data, &fields))
			keys := []string{}
			for k := range fields {
				keys = append(keys, k)
			}
			require.ElementsMatch(t, []string{"message_id", "shop_id", "image_url", "file_extension"}, keys)
			var state string
			require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", result.MessageID).Scan(&state))
			require.Equal(t, "ready", state)
			require.Zero(t, assetCount(t, "shop_messages", "id", result.MessageID))
			require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", result.MessageID))
			require.Zero(t, blob.deletes)
		})
	}
}
func TestUploadFailureRecovery(t *testing.T) {
	for _, kind := range []string{"put_failure", "finalize_failure", "canceled_put", "canceled_after_put", "canceled_before_reserve", "compensation_unavailable"} {
		t.Run(kind, func(t *testing.T) {
			shop, user, repo := uploadFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var asset messages.Asset
			wrapped := &uploadFinalizationRepository{Repository: repo}
			if kind == "compensation_unavailable" {
				wrapped.compensationUnavailable = true
				wrapped.failure = "before_commit"
			}
			if kind == "finalize_failure" {
				wrapped.failure = "before_commit"
			}
			blob := &uploadBlobStore{put: func(ctx context.Context, a messages.Asset) error {
				asset = a
				assertUploadCommittedWithoutLocks(t, ctx, a)
				if kind == "canceled_put" || kind == "canceled_after_put" {
					cancel()
					if kind == "canceled_put" {
						return ctx.Err()
					}
					return nil
				}
				if kind == "put_failure" {
					return errors.New("PUT failed")
				}
				return nil
			}}
			if kind == "canceled_before_reserve" {
				cancel()
			}
			result, e := messages.NewService(wrapped, nil).WithBlobStore(blob).UploadMessageImage(ctx, user, shop, []byte("image"), "image/jpeg")
			require.Error(t, e)
			require.Empty(t, result)
			require.Zero(t, blob.deletes)
			if kind == "canceled_before_reserve" {
				require.Zero(t, blob.calls)
				require.Zero(t, assetCount(t, "shop_message_uploads", "shop_id", shop))
				return
			}
			var state string
			var lease time.Time
			require.NoError(t, testDB.QueryRow("SELECT state,lease_until FROM shop_message_uploads WHERE id=$1", asset.ID).Scan(&state, &lease))
			if kind == "canceled_put" || kind == "canceled_after_put" || kind == "compensation_unavailable" {
				require.Equal(t, "uploading", state)
				require.True(t, lease.After(time.Now()))
				require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", asset.ID))
			} else {
				require.Equal(t, "cleanup_pending", state)
				require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", asset.ID))
			}
		})
	}
}
func TestUploadAmbiguousFinalizeProtectsReady(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	var asset messages.Asset
	blob := &uploadBlobStore{put: func(ctx context.Context, a messages.Asset) error {
		asset = a
		assertUploadCommittedWithoutLocks(t, ctx, a)
		return nil
	}}
	wrapped := &uploadFinalizationRepository{Repository: repo, failure: "ambiguous_commit"}
	result, e := messages.NewService(wrapped, nil).WithBlobStore(blob).UploadMessageImage(context.Background(), user, shop, []byte("image"), "image/jpeg")
	require.Error(t, e)
	require.Empty(t, result)
	require.Zero(t, blob.deletes)
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", asset.ID).Scan(&state))
	require.Equal(t, "ready", state)
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", asset.ID))
}
func TestUploadRejectedWithoutCloud(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	for _, tc := range []struct {
		name, shop, actor string
		size              int
	}{{"zero", uuid.Nil.String(), user.UserID, 16}, {"invalid", "bad", user.UserID, 16}, {"nonmember", shop, "outside", 16}, {"missing_shop", uuid.NewString(), user.UserID, 16}, {"empty", shop, user.UserID, 0}, {"large", shop, user.UserID, 15*1024*1024 + 1}} {
		t.Run(tc.name, func(t *testing.T) {
			blob := &uploadBlobStore{put: func(context.Context, messages.Asset) error { return nil }}
			result, e := messages.NewService(repo, nil).WithBlobStore(blob).UploadMessageImage(context.Background(), &bootstrap.User{UserID: tc.actor}, tc.shop, make([]byte, tc.size), "image/jpeg")
			require.Error(t, e)
			require.Empty(t, result)
			require.Zero(t, blob.calls)
			require.Zero(t, assetCount(t, "shop_message_uploads", "shop_id", shop))
		})
	}
}

func (b *uploadBlobStore) ListPrefix(context.Context, string, string, string, string) ([]string, string, error) {
	return nil, "", nil
}
