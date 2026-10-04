package messages

import (
	"bytes"
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/require"
	"log/slog"
	"miltechserver/bootstrap"
	"testing"
	"time"
)

type cleanupRepoFake struct {
	jobs     []CleanupJob
	ready    bool
	trace    *[]string
	finished error
}

func (r *cleanupRepoFake) Claim(context.Context, int, time.Time) ([]CleanupJob, error) {
	*r.trace = append(*r.trace, "claim committed")
	return r.jobs, nil
}
func (r *cleanupRepoFake) Prepare(context.Context, CleanupJob) (Asset, bool, error) {
	*r.trace = append(*r.trace, "freeze committed")
	return Asset{ID: "trusted"}, r.ready, nil
}
func (r *cleanupRepoFake) Finish(_ context.Context, _ string, e error) error {
	r.finished = e
	*r.trace = append(*r.trace, "finish")
	return nil
}
func (r *cleanupRepoFake) Reconcile(context.Context) error {
	*r.trace = append(*r.trace, "reconcile")
	return nil
}

type cleanupBlobFake struct {
	trace     *[]string
	failure   error
	operation func(context.Context, Asset) error
}

func (b *cleanupBlobFake) Upload(context.Context, Asset, []byte, string) error { return nil }
func (b *cleanupBlobFake) Delete(ctx context.Context, a Asset) error {
	*b.trace = append(*b.trace, "delete "+a.ID)
	if b.operation != nil {
		return b.operation(ctx, a)
	}
	return b.failure
}
func (b *cleanupBlobFake) ListPrefix(context.Context, string, string, string, string) ([]string, string, error) {
	return nil, "", nil
}
func TestCleanupReferenceAndCrashRecovery(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "reference survives", true: "zero refs"}[ready], func(t *testing.T) {
			trace := []string{}
			repo := &cleanupRepoFake{jobs: []CleanupJob{{ID: "job"}}, ready: ready, trace: &trace}
			blob := &cleanupBlobFake{trace: &trace, failure: errors.New("transient")}
			w, err := NewCleanupWorker(repo, blob, DefaultCleanupConfig())
			require.NoError(t, err)
			require.NoError(t, w.runOnce(context.Background()))
			expected := []string{"reconcile", "claim committed", "freeze committed"}
			if ready {
				expected = append(expected, "delete trusted", "finish")
				require.Equal(t, blob.failure, repo.finished)
			}
			require.Equal(t, expected, trace)
		})
	}
}
func TestCleanupConfig(t *testing.T) {
	c := DefaultCleanupConfig()
	c.LeaseDuration = c.OperationTimeout
	_, err := NewCleanupWorker(&cleanupRepoFake{}, &cleanupBlobFake{}, c)
	require.Error(t, err)
}

func TestCleanup404AndCancellation(t *testing.T) {
	trace := []string{}
	repo := &cleanupRepoFake{jobs: []CleanupJob{{ID: "job"}}, ready: true, trace: &trace}
	blob := &cleanupBlobFake{trace: &trace, failure: &azcore.ResponseError{StatusCode: 404}}
	worker, err := NewCleanupWorker(repo, blob, DefaultCleanupConfig())
	require.NoError(t, err)
	require.NoError(t, worker.runOnce(context.Background()))
	require.NoError(t, repo.finished)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	trace = nil
	require.NoError(t, worker.Run(ctx))
	require.Empty(t, trace)
}

func TestCleanupBoundedOperationAndShutdown(t *testing.T) {
	trace := []string{}
	repo := &cleanupRepoFake{jobs: []CleanupJob{{ID: "job"}}, ready: true, trace: &trace}
	entered := make(chan struct{})
	finished := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Now()
	cfg := DefaultCleanupConfig()
	blob := &cleanupBlobFake{trace: &trace, operation: func(operation context.Context, _ Asset) error {
		deadline, ok := operation.Deadline()
		require.True(t, ok)
		require.Equal(t, now.Add(cfg.OperationTimeout), deadline)
		close(entered)
		<-operation.Done()
		return operation.Err()
	}}
	worker, err := NewCleanupWorker(repo, blob, cfg)
	require.NoError(t, err)
	worker.now = func() time.Time { return now }
	go func() { finished <- worker.Run(ctx) }()
	<-entered
	cancel()
	require.NoError(t, <-finished)
	require.Nil(t, repo.finished, "shutdown leaves the lease recoverable without detached acknowledgment")
}

type cleanupDiscardRepo struct{ Repository }

func (*cleanupDiscardRepo) DeleteMessageImageBlob(context.Context, *bootstrap.User, string, string) error {
	return nil
}
func TestCleanupDiscardLogsQueued(t *testing.T) {
	var output bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(old)
	require.NoError(t, NewService(&cleanupDiscardRepo{}, nil).DeleteMessageImage(context.Background(), &bootstrap.User{UserID: "actor"}, "shop", "asset"))
	require.Contains(t, output.String(), "shop_message_image_cleanup_queued")
	require.NotContains(t, output.String(), "image deleted")
}

// These callbacks inspect the contexts actually delivered to each blocking phase.
type budgetCleanupRepository struct {
	CleanupRepository
	reconcile func(context.Context) error
	claim     func(context.Context, int, time.Time) ([]CleanupJob, error)
	prepare   func(context.Context, CleanupJob) (Asset, bool, error)
	finish    func(context.Context, string, error) error
}

func (r budgetCleanupRepository) Reconcile(c context.Context) error { return r.reconcile(c) }
func (r budgetCleanupRepository) Claim(c context.Context, n int, d time.Time) ([]CleanupJob, error) {
	return r.claim(c, n, d)
}
func (r budgetCleanupRepository) Prepare(c context.Context, j CleanupJob) (Asset, bool, error) {
	return r.prepare(c, j)
}
func (r budgetCleanupRepository) Finish(c context.Context, id string, e error) error {
	return r.finish(c, id, e)
}
func TestCleanupDatabasePhaseBudgets(t *testing.T) {
	now := time.Now()
	cfg := DefaultCleanupConfig()
	lease := now.Add(5 * time.Second)
	check := func(c context.Context, want time.Time) {
		d, ok := c.Deadline()
		require.True(t, ok, "database phase must have deadline")
		require.Equal(t, want, d)
		require.NoError(t, c.Err())
	}
	var prepareContext context.Context
	var finishCalled bool
	repo := budgetCleanupRepository{
		reconcile: func(c context.Context) error { check(c, now.Add(cfg.OperationTimeout)); return nil },
		claim: func(c context.Context, _ int, _ time.Time) ([]CleanupJob, error) {
			check(c, now.Add(cfg.OperationTimeout))
			return []CleanupJob{{ID: "one", LeaseUntil: lease}}, nil
		},
		prepare: func(c context.Context, _ CleanupJob) (Asset, bool, error) {
			check(c, lease)
			prepareContext = c
			return Asset{}, true, nil
		},
		finish: func(c context.Context, _ string, _ error) error {
			check(c, lease)
			require.ErrorIs(t, prepareContext.Err(), context.Canceled)
			finishCalled = true
			return nil
		},
	}
	trace := []string{}
	worker, err := NewCleanupWorker(repo, &cleanupBlobFake{trace: &trace}, cfg)
	require.NoError(t, err)
	worker.now = func() time.Time { return now }
	require.NoError(t, worker.runOnce(context.Background()))
	require.True(t, finishCalled)
}
func TestCleanupMultiJobLeaseClock(t *testing.T) {
	now := time.Now()
	cfg := DefaultCleanupConfig()
	lease := now.Add(time.Second)
	prepared, finished, deleted := 0, 0, 0
	repo := budgetCleanupRepository{
		reconcile: func(context.Context) error { return nil },
		claim: func(context.Context, int, time.Time) ([]CleanupJob, error) {
			return []CleanupJob{{ID: "first", LeaseUntil: lease}, {ID: "expired", LeaseUntil: lease}}, nil
		},
		prepare: func(context.Context, CleanupJob) (Asset, bool, error) { prepared++; return Asset{}, true, nil },
		finish:  func(context.Context, string, error) error { finished++; return nil },
	}
	trace := []string{}
	blob := &cleanupBlobFake{trace: &trace, operation: func(context.Context, Asset) error {
		deleted++
		now = lease.Add(time.Second)
		return context.DeadlineExceeded
	}}
	worker, err := NewCleanupWorker(repo, blob, cfg)
	require.NoError(t, err)
	worker.now = func() time.Time { return now }
	require.NoError(t, worker.runOnce(context.Background()))
	require.Equal(t, 1, prepared, "expired second lease must not start preparation")
	require.Equal(t, 1, deleted)
	require.Zero(t, finished, "expired jobs remain recoverable; acknowledgment must not extend lease")
}

func TestCleanupFreshAcknowledgmentAfterCloudTimeout(t *testing.T) {
	cfg := DefaultCleanupConfig()
	cfg.OperationTimeout = 10 * time.Millisecond
	trace := []string{}
	finished := false
	repo := budgetCleanupRepository{
		reconcile: func(context.Context) error { return nil },
		claim: func(context.Context, int, time.Time) ([]CleanupJob, error) {
			return []CleanupJob{{ID: "one", LeaseUntil: time.Now().Add(time.Second)}}, nil
		},
		prepare: func(context.Context, CleanupJob) (Asset, bool, error) { return Asset{}, true, nil },
		finish: func(c context.Context, _ string, e error) error {
			require.NoError(t, c.Err())
			require.ErrorIs(t, e, context.DeadlineExceeded)
			_, bounded := c.Deadline()
			require.True(t, bounded)
			finished = true
			return nil
		},
	}
	blob := &cleanupBlobFake{trace: &trace, operation: func(c context.Context, _ Asset) error { <-c.Done(); return c.Err() }}
	worker, err := NewCleanupWorker(repo, blob, cfg)
	require.NoError(t, err)
	require.NoError(t, worker.runOnce(context.Background()))
	require.True(t, finished)
}
