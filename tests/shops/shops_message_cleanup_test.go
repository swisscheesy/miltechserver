package shops_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"miltechserver/.gen/miltech_ng/public/model"
	"miltechserver/api/shops/core"
	"miltechserver/api/shops/members"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCleanupShopDeletionDurable(t *testing.T) {
	shop, user, _ := uploadFixture(t)
	assets := messages.NewAssetRepository(testDB, messages.AssetStorage{Account: "owned", Container: "shop-message-images"})
	var a messages.Asset
	require.NoError(t, assetTransaction(func(tx *sql.Tx) error {
		var err error
		a, err = assets.Reserve(context.Background(), tx, user, shop, ".jpg")
		return err
	}))
	require.NoError(t, core.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"}).DeleteShop(context.Background(), user, shop))
	require.Greater(t, assetCount(t, "shop_message_blob_cleanup_jobs", "upload_id", a.ID), 0, "Shop deletion must durably preserve interrupted upload cleanup")
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", a.ID).Scan(&state))
	require.Equal(t, "cleanup_pending", state)
}

func cleanupRepo(t *testing.T, account string) messages.CleanupRepository {
	t.Helper()
	r, err := messages.NewCleanupRepository(testDB, messages.AssetStorage{Account: account, Container: "shop-message-images"}, messages.DefaultCleanupConfig())
	require.NoError(t, err)
	return r
}
func claimCleanup(t *testing.T, r messages.CleanupRepository) []messages.CleanupJob {
	t.Helper()
	jobs, err := r.Claim(context.Background(), 10, time.Now().Add(2*time.Minute))
	require.NoError(t, err)
	return jobs
}
func cleanupAssetFixture(t *testing.T) (string, *bootstrap.User, *messages.RepositoryImpl, messages.Asset) {
	t.Helper()
	shop, user, repo := uploadFixture(t)
	a, err := repo.ReserveMessageImage(context.Background(), user, shop, ".jpg")
	require.NoError(t, err)
	require.NoError(t, repo.FinalizeMessageImage(context.Background(), user, a.ID))
	return shop, user, repo, a
}
func cleanupMessage(t *testing.T, repo *messages.RepositoryImpl, user *bootstrap.User, a messages.Asset, parent *string) string {
	t.Helper()
	now := time.Now()
	row, err := repo.CreateShopMessage(context.Background(), user, model.ShopMessages{ID: uuid.NewString(), ShopID: a.ShopID, UserID: user.UserID, Message: "[IMAGE:" + a.URL + "]", ParentID: parent, CreatedAt: &now, UpdatedAt: &now})
	require.NoError(t, err)
	return row.ID
}
func cleanupState(t *testing.T, id string) string {
	t.Helper()
	var state string
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_uploads WHERE id=$1", id).Scan(&state))
	return state
}
func expireCleanupUpload(t *testing.T, id string) {
	t.Helper()
	_, err := testDB.Exec("UPDATE shop_message_uploads SET lease_until=now()-interval '1 second' WHERE id=$1", id)
	require.NoError(t, err)
}
func assertCleanupOutsideLocks(t *testing.T, a messages.Asset) {
	t.Helper()
	var active int
	require.NoError(t, testDB.QueryRow("SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND xact_start IS NOT NULL").Scan(&active))
	require.Zero(t, active, "cloud calls under database lock")
	tx, err := testDB.Begin()
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.Exec("SELECT id FROM shops WHERE id=$1 FOR UPDATE NOWAIT", a.ShopID)
	require.NoError(t, err)
	if a.ID != "" {
		_, err = tx.Exec("SELECT id FROM shop_message_uploads WHERE id=$1 FOR UPDATE NOWAIT", a.ID)
		require.NoError(t, err)
	}
	_, err = tx.Exec("SELECT id FROM shop_message_blob_cleanup_jobs FOR UPDATE NOWAIT")
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
}
func TestCleanupReferenceAndCrashRecovery(t *testing.T) {
	_, user, repo, a := cleanupAssetFixture(t)
	first := cleanupMessage(t, repo, user, a, nil)
	second := cleanupMessage(t, repo, user, a, nil)
	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, first))
	r := cleanupRepo(t, "owned")
	jobs := claimCleanup(t, r)
	require.Len(t, jobs, 1)
	_, ready, err := r.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.False(t, ready, "surviving reference must prevent deletion")
	require.Equal(t, "ready", cleanupState(t, a.ID))
	require.NoError(t, repo.DeleteShopMessage(context.Background(), user, second))
	jobs = claimCleanup(t, r)
	require.Len(t, jobs, 1)
	target, ready, err := r.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.True(t, ready)
	require.Equal(t, a.Account, target.Account)
	require.Equal(t, a.BlobKey, target.BlobKey)
	require.Equal(t, "deleting", cleanupState(t, a.ID))
	assertCleanupOutsideLocks(t, target)
	objects := map[string]bool{a.BlobKey: true}
	deleteCalls := 0
	cloud := &cleanupCloud{delete: func(ctx context.Context, asset messages.Asset) error {
		assertCleanupOutsideLocks(t, asset)
		require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", asset.ID))
		deleteCalls++
		if !objects[asset.BlobKey] {
			return &azcore.ResponseError{StatusCode: 404}
		}
		delete(objects, asset.BlobKey)
		return nil
	}}
	require.NoError(t, cloud.Delete(context.Background(), target))
	require.Empty(t, objects)
	// The process crashes after cloud deletion but before acknowledgement.
	_, err = testDB.Exec("UPDATE shop_message_blob_cleanup_jobs SET lease_until=now()-interval '1 second' WHERE id=$1", jobs[0].ID)
	require.NoError(t, err)
	recovered := cleanupRepo(t, "owned")
	jobs = claimCleanup(t, recovered)
	require.Len(t, jobs, 1)
	_, ready, err = recovered.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.True(t, ready)
	require.Error(t, r.Finish(context.Background(), jobs[0].ID, nil), "old owner must not acknowledge recovered lease")
	require.Error(t, cloud.Delete(context.Background(), target), "recovered deletion finds a 404")
	require.Equal(t, 2, deleteCalls)
	// Queue acknowledgement must not wait on any upload resource lock.
	held, err := testDB.Begin()
	require.NoError(t, err)
	defer held.Rollback()
	_, err = held.Exec("SELECT id FROM shop_message_uploads WHERE id=$1 FOR UPDATE", a.ID)
	require.NoError(t, err)
	ackCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, recovered.Finish(ackCtx, jobs[0].ID, nil))
	require.NoError(t, held.Rollback())
	require.Equal(t, "deleting", cleanupState(t, a.ID))
	require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
	require.Equal(t, "deleted", cleanupState(t, a.ID))
	require.Error(t, repo.FinalizeMessageImage(context.Background(), user, a.ID))
}
func TestCleanupLatePutTombstone(t *testing.T) {
	shop, user, repo, a := cleanupAssetFixture(t)
	require.NoError(t, repo.DeleteMessageImageBlob(context.Background(), user, a.ID, shop))
	expireCleanupUpload(t, a.ID)
	r := cleanupRepo(t, "owned")
	jobs := claimCleanup(t, r)
	require.Len(t, jobs, 1)
	_, ready, err := r.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.True(t, ready)
	require.NoError(t, r.Finish(context.Background(), jobs[0].ID, nil))
	require.Equal(t, "deleting", cleanupState(t, a.ID))
	require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
	// The cloud object reappears after the first acknowledged delete.
	objects := map[string]bool{a.BlobKey: true}
	cloud := &cleanupCloud{delete: func(ctx context.Context, asset messages.Asset) error {
		assertCleanupOutsideLocks(t, asset)
		delete(objects, asset.BlobKey)
		return nil
	}}
	// Advance persisted reconciliation timestamps, never sleep.
	_, err = testDB.Exec("UPDATE shop_message_blob_cleanup_jobs SET updated_at=now()-interval '6 minutes'")
	require.NoError(t, err)
	require.NoError(t, r.Reconcile(context.Background()))
	jobs = claimCleanup(t, r)
	require.Len(t, jobs, 1)
	target, ready, err := r.Prepare(context.Background(), jobs[0])
	require.NoError(t, err)
	require.True(t, ready)
	require.Equal(t, a.BlobKey, target.BlobKey)
	require.Equal(t, "deleted", target.State)
	assertCleanupOutsideLocks(t, target)
	require.NoError(t, cloud.Delete(context.Background(), target))
	require.Empty(t, objects)
	require.NoError(t, r.Finish(context.Background(), jobs[0].ID, nil))
	require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
	require.Equal(t, "deleted", cleanupState(t, a.ID))
}
func TestCleanupAccountMismatch(t *testing.T) {
	shop, user, repo, a := cleanupAssetFixture(t)
	require.NoError(t, repo.DeleteMessageImageBlob(context.Background(), user, a.ID, shop))
	expireCleanupUpload(t, a.ID)
	r := cleanupRepo(t, "different")
	jobs := claimCleanup(t, r)
	require.Len(t, jobs, 1)
	_, ready, err := r.Prepare(context.Background(), jobs[0])
	require.Error(t, err)
	require.False(t, ready)
	require.NoError(t, r.Finish(context.Background(), jobs[0].ID, err))
	var state, failure string
	require.NoError(t, testDB.QueryRow("SELECT state,failure FROM shop_message_blob_cleanup_jobs WHERE id=$1", jobs[0].ID).Scan(&state, &failure))
	require.Equal(t, "manual-review", state)
	require.Equal(t, "target_or_retry_limit", failure)
}
func TestCleanupLiveUploadLeaseAndReadyDraft(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	a, err := repo.ReserveMessageImage(context.Background(), user, shop, ".jpg")
	require.NoError(t, err)
	require.NoError(t, core.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"}).DeleteShop(context.Background(), user, shop))
	r := cleanupRepo(t, "owned")
	jobs := claimCleanup(t, r)
	require.Len(t, jobs, 2)
	for _, j := range jobs {
		_, ready, err := r.Prepare(context.Background(), j)
		require.NoError(t, err)
		require.False(t, ready, "both prefix and asset must wait for pending live PUT")
	}
	expireCleanupUpload(t, a.ID)
	_, err = testDB.Exec("UPDATE shop_message_blob_cleanup_jobs SET next_attempt=now()-interval '1 second'")
	require.NoError(t, err)
	for _, j := range claimCleanup(t, r) {
		_, ready, err := r.Prepare(context.Background(), j)
		require.NoError(t, err)
		require.True(t, ready)
		require.NoError(t, r.Finish(context.Background(), j.ID, nil))
	}
	require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
	require.Equal(t, "deleted", cleanupState(t, a.ID))
	_, _, _, draft := cleanupAssetFixture(t)
	expireCleanupUpload(t, draft.ID)
	require.NoError(t, r.Reconcile(context.Background()))
	require.Empty(t, claimCleanup(t, r))
	require.Equal(t, "ready", cleanupState(t, draft.ID))
}
func TestCleanupExpiredReservationAndRetryLimit(t *testing.T) {
	shop, user, repo := uploadFixture(t)
	a, err := repo.ReserveMessageImage(context.Background(), user, shop, ".jpg")
	require.NoError(t, err)
	expireCleanupUpload(t, a.ID)
	r := cleanupRepo(t, "owned")
	require.NoError(t, r.Reconcile(context.Background()))
	for attempt := 1; attempt <= 10; attempt++ {
		jobs := claimCleanup(t, r)
		require.Len(t, jobs, 1)
		_, ready, err := r.Prepare(context.Background(), jobs[0])
		require.NoError(t, err)
		require.True(t, ready)
		require.NoError(t, r.Finish(context.Background(), jobs[0].ID, errors.New("SECRET remote error")))
		var state, failure string
		var attempts int
		var delay float64
		require.NoError(t, testDB.QueryRow("SELECT state,failure,attempts,EXTRACT(EPOCH FROM next_attempt-updated_at) FROM shop_message_blob_cleanup_jobs WHERE id=$1", jobs[0].ID).Scan(&state, &failure, &attempts, &delay))
		require.Equal(t, attempt, attempts)
		require.NotContains(t, failure, "SECRET")
		if attempt == 10 {
			require.Equal(t, "manual-review", state)
		} else {
			require.Equal(t, "retryable", state)
			expected := 5 * float64(int64(1)<<(attempt-1))
			if expected > 300 {
				expected = 300
			}
			require.InDelta(t, expected, delay, .1)
		}
		_, err = testDB.Exec("UPDATE shop_message_blob_cleanup_jobs SET next_attempt=now()-interval '1 second'")
		require.NoError(t, err)
	}
	require.NoError(t, r.Reconcile(context.Background()))
	require.Empty(t, claimCleanup(t, r))
}
func TestCleanupCascadeAndFinalMember(t *testing.T) {
	for _, kind := range []string{"reply", "account", "final_member"} {
		t.Run(kind, func(t *testing.T) {
			shop, user, repo, a := cleanupAssetFixture(t)
			author := user
			if kind == "account" || kind == "final_member" {
				ensureUser(t, testDB, "cleanup-other")
				_, err := testDB.Exec("INSERT INTO shop_members(id,shop_id,user_id,role) VALUES($1,$2,'cleanup-other','admin')", uuid.NewString(), shop)
				require.NoError(t, err)
				author = &bootstrap.User{UserID: "cleanup-other"}
			}
			parent := cleanupMessage(t, repo, author, a, nil)
			cleanupMessage(t, repo, author, a, &parent)
			switch kind {
			case "reply":
				require.NoError(t, repo.DeleteShopMessage(context.Background(), author, parent))
			case "account":
				_, err := testDB.Exec("DELETE FROM users WHERE uid=$1", author.UserID)
				require.NoError(t, err)
			case "final_member":
				m := members.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
				require.NoError(t, m.LeaveShop(context.Background(), user, shop))
				require.NoError(t, m.LeaveShop(context.Background(), author, shop))
			}
			require.Zero(t, assetCount(t, "shop_message_asset_references", "upload_id", a.ID))
			expireCleanupUpload(t, a.ID)
			r := cleanupRepo(t, "owned")
			jobs := claimCleanup(t, r)
			require.NotEmpty(t, jobs)
			for _, job := range jobs {
				target, ready, err := r.Prepare(context.Background(), job)
				require.NoError(t, err)
				require.True(t, ready)
				assertCleanupOutsideLocks(t, target)
				require.NoError(t, r.Finish(context.Background(), job.ID, nil))
			}
			require.NoError(t, cleanupRepo(t, "owned").Reconcile(context.Background()))
			require.Equal(t, "deleted", cleanupState(t, a.ID))
		})
	}
}

func TestCleanup020PopulatedReverseRefuses(t *testing.T) {
	_, _, _, a := cleanupAssetFixture(t)
	sqlText, err := os.ReadFile("../../migrations/020_rollback_shop_message_asset_lifecycle.sql")
	require.NoError(t, err)
	conn, err := testDB.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(context.Background(), string(sqlText))
	require.ErrorContains(t, err, "message asset registry and cleanup proof must be retained")
	_, err = conn.ExecContext(context.Background(), "ROLLBACK")
	require.NoError(t, err)
	require.Equal(t, "ready", cleanupState(t, a.ID))
}

type cleanupSinglePass struct {
	messages.CleanupRepository
	cancel context.CancelFunc
}

func (r *cleanupSinglePass) Prepare(ctx context.Context, j messages.CleanupJob) (messages.Asset, bool, error) {
	a, ready, err := r.CleanupRepository.Prepare(ctx, j)
	if !ready && err == nil {
		r.cancel()
	}
	return a, ready, err
}
func (r *cleanupSinglePass) Finish(ctx context.Context, id string, e error) error {
	err := r.CleanupRepository.Finish(ctx, id, e)
	r.cancel()
	return err
}

type cleanupCloud struct {
	delete func(context.Context, messages.Asset) error
	list   func(context.Context, string, string, string, string) ([]string, string, error)
}

func (b *cleanupCloud) Upload(context.Context, messages.Asset, []byte, string) error { return nil }
func (b *cleanupCloud) Delete(ctx context.Context, a messages.Asset) error           { return b.delete(ctx, a) }
func (b *cleanupCloud) ListPrefix(ctx context.Context, a, c, p, s string) ([]string, string, error) {
	return b.list(ctx, a, c, p, s)
}
func runCleanupPass(t *testing.T, r messages.CleanupRepository, b messages.BlobStore) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := messages.DefaultCleanupConfig()
	cfg.BatchSize = 1
	worker, err := messages.NewCleanupWorker(&cleanupSinglePass{r, cancel}, b, cfg)
	require.NoError(t, err)
	require.NoError(t, worker.Run(ctx))
}
func TestCleanupPrefixPagingAndRollback(t *testing.T) {
	shop, user, _ := uploadFixture(t)
	// A failure after enqueue must roll back both prefix proof and aggregate deletion.
	_, err := testDB.Exec(`CREATE FUNCTION cleanup_test_refuse() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'delete refused'; END $$; CREATE TRIGGER cleanup_test_refuse BEFORE DELETE ON shops FOR EACH ROW EXECUTE FUNCTION cleanup_test_refuse()`)
	require.NoError(t, err)
	coreRepo := core.NewRepository(testDB, nil, &bootstrap.Env{BlobAccountName: "owned"})
	require.Error(t, coreRepo.DeleteShop(context.Background(), user, shop))
	require.Zero(t, assetCount(t, "shop_message_blob_cleanup_jobs", "shop_id", shop))
	require.Equal(t, 1, assetCount(t, "shops", "id", shop))
	_, err = testDB.Exec(`DROP TRIGGER cleanup_test_refuse ON shops; DROP FUNCTION cleanup_test_refuse()`)
	require.NoError(t, err)
	require.NoError(t, coreRepo.DeleteShop(context.Background(), user, shop))
	require.Equal(t, 1, assetCount(t, "shop_message_blob_cleanup_jobs", "shop_id", shop))
	r := cleanupRepo(t, "owned")
	deleted := []string{}
	fail := true
	cursors := []string{}
	cloud := &cleanupCloud{list: func(ctx context.Context, account, container, prefix, cursor string) ([]string, string, error) {
		require.Equal(t, "owned", account)
		require.Equal(t, "shop-message-images", container)
		require.Equal(t, shop+"/", prefix)
		assertCleanupOutsideLocks(t, messages.Asset{ShopID: shop})
		cursors = append(cursors, cursor)
		if cursor == "" {
			return []string{shop + "/historical image.jpg"}, "page2", nil
		}
		require.Equal(t, "page2", cursor)
		return []string{shop + "/last.png"}, "", nil
	}, delete: func(ctx context.Context, a messages.Asset) error {
		assertCleanupOutsideLocks(t, a)
		if fail {
			return errors.New("transient cloud error")
		}
		deleted = append(deleted, a.BlobKey)
		return nil
	}}
	runCleanupPass(t, r, cloud)
	var cursor, state string
	require.NoError(t, testDB.QueryRow("SELECT cursor,state FROM shop_message_blob_cleanup_jobs WHERE shop_id=$1", shop).Scan(&cursor, &state))
	require.Empty(t, cursor)
	require.Equal(t, "retryable", state)
	_, err = testDB.Exec("UPDATE shop_message_blob_cleanup_jobs SET next_attempt=now()-interval '1 second'")
	require.NoError(t, err)
	fail = false
	runCleanupPass(t, r, cloud)
	require.NoError(t, testDB.QueryRow("SELECT cursor,state FROM shop_message_blob_cleanup_jobs WHERE shop_id=$1", shop).Scan(&cursor, &state))
	require.Equal(t, "page2", cursor)
	require.Equal(t, "pending", state)
	// Construct a different worker/repository to prove paging survives process restart.
	runCleanupPass(t, cleanupRepo(t, "owned"), cloud)
	require.NoError(t, testDB.QueryRow("SELECT state FROM shop_message_blob_cleanup_jobs WHERE shop_id=$1", shop).Scan(&state))
	require.Equal(t, "completed", state)
	require.Equal(t, []string{"", "", "page2"}, cursors)
	require.Equal(t, []string{shop + "/historical image.jpg", shop + "/last.png"}, deleted)
}

func TestCleanup020ReverseSerializesWriters(t *testing.T) {
	for _, kind := range []string{"reservation", "job"} {
		t.Run(kind, func(t *testing.T) {
			shop, user, _ := uploadFixture(t)
			raw, err := os.ReadFile("../../migrations/020_rollback_shop_message_asset_lifecycle.sql")
			require.NoError(t, err)
			// Exercise the exact lock-and-proof prefix without dropping the shared fixture.
			prefix := strings.TrimPrefix(strings.Split(string(raw), "DROP TABLE")[0], "BEGIN;")
			writer, err := testDB.Begin()
			require.NoError(t, err)
			defer writer.Rollback()
			var pid int
			require.NoError(t, writer.QueryRow("SELECT pg_backend_pid()").Scan(&pid))
			if kind == "reservation" {
				assets := messages.NewAssetRepository(testDB, messages.AssetStorage{Account: "owned", Container: "shop-message-images"})
				_, err = assets.Reserve(context.Background(), writer, user, shop, ".jpg")
				require.NoError(t, err)
			} else {
				_, err = writer.Exec(`INSERT INTO shop_message_blob_cleanup_jobs(scope,shop_id,account,container,blob_key,reason) VALUES('shop_prefix',$1,'owned','shop-message-images',$1||'/','shop_deleted')`, shop)
				require.NoError(t, err)
			}
			reverse, err := testDB.Begin()
			require.NoError(t, err)
			defer reverse.Rollback()
			done := make(chan error, 1)
			go func() { _, e := reverse.Exec(prefix); done <- e }()
			blocked := false
			deadline := time.After(time.Second)
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
		wait:
			for {
				select {
				case err = <-done:
					break wait
				case <-deadline:
					break wait
				case <-ticker.C:
					var waiting bool
					require.NoError(t, testDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting))
					if waiting {
						blocked = true
						break wait
					}
				}
			}
			require.NoError(t, writer.Commit())
			if blocked {
				select {
				case err = <-done:
				case <-time.After(time.Second):
					t.Fatal("rollback did not finish after writer commit")
				}
			}
			require.True(t, blocked, "rollback must block before checking emptiness while a lifecycle writer is in flight")
			require.ErrorContains(t, err, "message asset registry and cleanup proof must be retained")
			require.NoError(t, reverse.Rollback())
			table := "shop_message_uploads"
			if kind == "job" {
				table = "shop_message_blob_cleanup_jobs"
			}
			require.Equal(t, 1, assetCount(t, table, "shop_id", shop))
		})
	}
}
