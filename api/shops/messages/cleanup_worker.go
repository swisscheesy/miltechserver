package messages

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"log/slog"
	"strings"
	"time"
)

type CleanupJob struct {
	ID, AssetID, Scope, Account, Container, BlobKey, ShopID, State string
	LeaseUntil                                                     time.Time
}
type CleanupRepository interface {
	Claim(context.Context, int, time.Time) ([]CleanupJob, error)
	Prepare(context.Context, CleanupJob) (Asset, bool, error)
	Finish(context.Context, string, error) error
	Reconcile(context.Context) error
}
type CleanupConfig struct {
	BatchSize, MaxAttempts                                    int
	PollInterval, OperationTimeout, LeaseDuration, MaxBackoff time.Duration
}

func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{10, 10, 5 * time.Second, 30 * time.Second, 2 * time.Minute, 5 * time.Minute}
}

type CleanupWorker struct {
	repo  CleanupRepository
	blobs BlobStore
	cfg   CleanupConfig
	now   func() time.Time
}

var errCleanupTarget = errors.New("cleanup target requires manual review")

type prefixProgress struct{ cursor string }

func (e *prefixProgress) Error() string { return "prefix page completed" }
func validateCleanupConfig(c CleanupConfig) error {
	if c.BatchSize <= 0 || c.MaxAttempts <= 0 || c.PollInterval <= 0 || c.OperationTimeout <= 0 || c.LeaseDuration <= c.OperationTimeout || c.MaxBackoff <= 0 {
		return errors.New("invalid cleanup configuration")
	}
	return nil
}
func NewCleanupWorker(r CleanupRepository, b BlobStore, c CleanupConfig) (*CleanupWorker, error) {
	if r == nil || b == nil {
		return nil, errors.New("cleanup dependencies required")
	}
	if err := validateCleanupConfig(c); err != nil {
		return nil, err
	}
	return &CleanupWorker{r, b, c, time.Now}, nil
}
func (w *CleanupWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := w.runOnce(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("shop_image_cleanup_iteration_failed", "category", "database_or_storage_unavailable")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (w *CleanupWorker) runOnce(ctx context.Context) error {
	phase, cancel := w.phaseContext(ctx, time.Time{})
	err := w.repo.Reconcile(phase)
	cancel()
	if err != nil {
		return err
	}
	phase, cancel = w.phaseContext(ctx, time.Time{})
	jobs, err := w.repo.Claim(phase, w.cfg.BatchSize, w.now().Add(w.cfg.LeaseDuration))
	cancel()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !job.LeaseUntil.IsZero() && !w.now().Before(job.LeaseUntil) {
			continue
		}
		phase, cancel = w.phaseContext(ctx, job.LeaseUntil)
		asset, ready, err := w.repo.Prepare(phase, job)
		cancel()
		if err != nil {
			if finishErr := w.finish(ctx, job, err); finishErr != nil {
				return finishErr
			}
			continue
		}
		if !ready {
			continue
		}
		operation, cancel := w.phaseContext(ctx, job.LeaseUntil)
		if operation.Err() != nil {
			cancel()
			continue
		}
		if job.Scope == "shop_prefix" {
			err = w.deletePrefixPage(operation, asset)
		} else {
			err = w.blobs.Delete(operation, asset)
		}
		cancel()
		var response *azcore.ResponseError
		if errors.As(err, &response) && response.StatusCode == 404 {
			err = nil
		}
		if finishErr := w.finish(ctx, job, err); finishErr != nil {
			return finishErr
		}
	}
	return nil
}

// Every blocking phase uses the existing operation budget. Job phases also
// respect the lease: an expired job is left for durable recovery.
func (w *CleanupWorker) phaseContext(ctx context.Context, lease time.Time) (context.Context, context.CancelFunc) {
	deadline := w.now().Add(w.cfg.OperationTimeout)
	if !lease.IsZero() && lease.Before(deadline) {
		deadline = lease
	}
	return context.WithDeadline(ctx, deadline)
}
func (w *CleanupWorker) finish(ctx context.Context, job CleanupJob, operationErr error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !job.LeaseUntil.IsZero() && !w.now().Before(job.LeaseUntil) {
		return nil
	}
	// A cloud timeout must not poison acknowledgment, but shutdown still cancels it.
	phase, cancel := w.phaseContext(ctx, job.LeaseUntil)
	defer cancel()
	return w.repo.Finish(phase, job.ID, operationErr)
}

func (w *CleanupWorker) deletePrefixPage(ctx context.Context, asset Asset) error {
	keys, next, err := w.blobs.ListPrefix(ctx, asset.Account, asset.Container, asset.BlobKey, asset.OperationID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, asset.BlobKey) || key == asset.BlobKey {
			return errCleanupTarget
		}
		target := asset
		target.BlobKey = key
		if err := w.blobs.Delete(ctx, target); err != nil {
			var response *azcore.ResponseError
			if !errors.As(err, &response) || response.StatusCode != 404 {
				return err
			}
		}
	}
	if next != "" {
		if next == asset.OperationID {
			return errCleanupTarget
		}
		return &prefixProgress{cursor: next}
	}
	return nil
}
