package shops_test

import (
	"context"
	"database/sql"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/messages"
	"testing"
	"time"
)

type observedCleanupRepository struct {
	messages.CleanupRepository
	before func(string)
	after  func(string, error)
}

func (r observedCleanupRepository) Reconcile(c context.Context) error {
	r.before("reconcile")
	e := r.CleanupRepository.Reconcile(c)
	r.after("reconcile", e)
	return e
}
func (r observedCleanupRepository) Claim(c context.Context, n int, d time.Time) ([]messages.CleanupJob, error) {
	r.before("claim")
	j, e := r.CleanupRepository.Claim(c, n, d)
	r.after("claim", e)
	return j, e
}
func (r observedCleanupRepository) Prepare(c context.Context, j messages.CleanupJob) (messages.Asset, bool, error) {
	r.before("prepare")
	a, b, e := r.CleanupRepository.Prepare(c, j)
	r.after("prepare", e)
	return a, b, e
}
func (r observedCleanupRepository) Finish(c context.Context, id string, e error) error {
	r.before("finish")
	err := r.CleanupRepository.Finish(c, id, e)
	r.after("finish", err)
	return err
}
func TestCleanupDatabaseBlockingBudgetAndRecovery(t *testing.T) {
	for _, phase := range []string{"reconcile", "claim", "prepare", "finish"} {
		t.Run(phase, func(t *testing.T) {
			shop, user, repo, a := cleanupAssetFixture(t)
			b, err := repo.ReserveMessageImage(context.Background(), user, shop, ".png")
			require.NoError(t, err)
			require.NoError(t, repo.FinalizeMessageImage(context.Background(), user, b.ID))
			for _, asset := range []messages.Asset{a, b} {
				require.NoError(t, repo.DeleteMessageImageBlob(context.Background(), user, asset.ID, shop))
				expireCleanupUpload(t, asset.ID)
			}
			database := aggregateSnapshotDatabase(t, &aggregateSnapshotProbe{})
			database.SetMaxOpenConns(1)
			cfg := messages.DefaultCleanupConfig()
			cfg.OperationTimeout = 100 * time.Millisecond
			cfg.LeaseDuration = time.Second
			cfg.BatchSize = 1
			real, err := messages.NewCleanupRepository(database, messages.AssetStorage{Account: "owned", Container: "shop-message-images"}, cfg)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var held *sql.Tx
			var conn *sql.Conn
			var observed error
			t.Cleanup(func() {
				if held != nil {
					_ = held.Rollback()
				}
				if conn != nil {
					_ = conn.Close()
				}
			})
			wrapped := observedCleanupRepository{CleanupRepository: real, before: func(current string) {
				if current != phase {
					return
				}
				if phase == "reconcile" || phase == "claim" {
					conn, err = database.Conn(context.Background())
					require.NoError(t, err)
					return
				}
				held, err = testDB.Begin()
				require.NoError(t, err)
				query := "SELECT id FROM shop_message_uploads FOR UPDATE"
				if phase == "finish" {
					query = "SELECT id FROM shop_message_blob_cleanup_jobs FOR UPDATE"
				}
				_, err = held.Exec(query)
				require.NoError(t, err)
			}, after: func(current string, e error) {
				if current == phase {
					observed = e
					cancel()
				}
			}}
			cloud := &cleanupCloud{delete: func(c context.Context, asset messages.Asset) error { assertCleanupOutsideLocks(t, asset); return nil }}
			worker, err := messages.NewCleanupWorker(wrapped, cloud, cfg)
			require.NoError(t, err)
			start := time.Now()
			require.NoError(t, worker.Run(ctx))
			require.Error(t, observed)
			require.Less(t, time.Since(start), time.Second, "worker phase must end on operation budget, not outer watchdog")
			if held != nil {
				require.NoError(t, held.Rollback())
			}
			if conn != nil {
				require.NoError(t, conn.Close())
			}
			require.Eventually(t, func() bool { return database.Stats().InUse == 0 }, time.Second, time.Millisecond)
			_, err = testDB.Exec(`UPDATE shop_message_blob_cleanup_jobs SET lease_until=now()-interval '1 second',next_attempt=now()-interval '1 second'`)
			require.NoError(t, err)
			recovered := cleanupRepo(t, "owned")
			deleted := map[string]bool{}
			cloud.delete = func(c context.Context, asset messages.Asset) error {
				assertCleanupOutsideLocks(t, asset)
				deleted[asset.ID] = true
				return nil
			}
			runCleanupPass(t, recovered, cloud)
			runCleanupPass(t, recovered, cloud)
			require.NoError(t, recovered.Reconcile(context.Background()))
			require.Len(t, deleted, 2, "blocked job and another job both make progress after recovery")
			require.Equal(t, "deleted", cleanupState(t, a.ID))
			require.Equal(t, "deleted", cleanupState(t, b.ID))
		})
	}
}
