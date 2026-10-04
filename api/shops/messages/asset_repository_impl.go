package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"log/slog"
	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
	dbutil "miltechserver/api/shared/db"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"
	"time"
)

type assetRepository struct{ storage AssetStorage }

func NewAssetRepository(_ *sql.DB, storage AssetStorage) AssetRepository {
	return &assetRepository{storage: storage}
}

func requireAssetActor(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID string) (bool, error) {
	if tx == nil || user == nil || user.UserID == "" {
		return false, shared.ErrShopAccessDenied
	}
	admin, _, err := shared.LockShopMutation(ctx, tx, shopID, user.UserID)
	return admin, err
}

func assetFromRow(a model.ShopMessageUploads) Asset {
	return Asset{ID: a.ID.String(), OperationID: a.OperationID.String(), ShopID: a.ShopID, UploaderID: a.UploaderID, Account: a.Account, Container: a.Container, BlobKey: a.BlobKey, URL: a.URL, Extension: a.Extension, State: a.State, LeaseUntil: a.LeaseUntil}
}

func loadAsset(ctx context.Context, tx *sql.Tx, id string, lock bool) (model.ShopMessageUploads, error) {
	var a model.ShopMessageUploads
	parsed, err := uuid.Parse(id)
	if err != nil {
		return a, errors.New("invalid asset ID")
	}
	stmt := SELECT(ShopMessageUploads.AllColumns).FROM(ShopMessageUploads).WHERE(ShopMessageUploads.ID.EQ(UUID(parsed)))
	if lock {
		stmt = stmt.FOR(UPDATE())
	}
	err = stmt.QueryContext(ctx, tx, &a)
	return a, err
}

func (r *assetRepository) Reserve(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID, extension string) (Asset, error) {
	if !r.storage.valid() {
		return Asset{}, errors.New("asset storage unavailable")
	}
	if _, err := uuid.Parse(shopID); err != nil {
		return Asset{}, errors.New("invalid shop ID")
	}
	switch extension {
	case ".jpg", ".png", ".gif", ".webp":
	default:
		return Asset{}, errors.New("unsupported image extension")
	}
	if _, err := requireAssetActor(ctx, tx, user, shopID); err != nil {
		return Asset{}, err
	}
	now := time.Now().UTC()
	id := uuid.New()
	row := model.ShopMessageUploads{ID: id, OperationID: uuid.New(), ShopID: shopID, UploaderID: user.UserID, Account: r.storage.Account, Container: r.storage.Container, BlobKey: shopID + "/" + id.String() + extension, Extension: extension, State: "uploading", LeaseUntil: now.Add(UploadLeaseDuration), CreatedAt: now, UpdatedAt: now}
	row.URL = fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", row.Account, row.Container, row.BlobKey)
	_, err := ShopMessageUploads.INSERT(ShopMessageUploads.AllColumns).MODEL(row).ExecContext(ctx, tx)
	return assetFromRow(row), err
}

func (r *assetRepository) Finalize(ctx context.Context, tx *sql.Tx, user *bootstrap.User, id string) error {
	if tx == nil {
		return errors.New("asset transaction required")
	}
	a, err := loadAsset(ctx, tx, id, false)
	if err != nil {
		return err
	}
	if _, err = requireAssetActor(ctx, tx, user, a.ShopID); err != nil {
		return err
	}
	a, err = loadAsset(ctx, tx, id, true)
	if err != nil {
		return err
	}
	if a.UploaderID != user.UserID || a.State != "uploading" || !a.LeaseUntil.After(time.Now()) {
		return errors.New("asset finalization unavailable")
	}
	return updateAssetState(ctx, tx, a.ID, "ready", "")
}

func updateAssetState(ctx context.Context, tx *sql.Tx, id uuid.UUID, state, reason string) error {
	_, err := ShopMessageUploads.UPDATE(ShopMessageUploads.State, ShopMessageUploads.UpdatedAt, ShopMessageUploads.Failure).SET(String(state), TimestampzT(time.Now().UTC()), String(reason)).WHERE(ShopMessageUploads.ID.EQ(UUID(id))).ExecContext(ctx, tx)
	return err
}

func enqueueAsset(ctx context.Context, tx *sql.Tx, a model.ShopMessageUploads, reason string) error {
	_, err := ShopMessageBlobCleanupJobs.INSERT(ShopMessageBlobCleanupJobs.UploadID, ShopMessageBlobCleanupJobs.ShopID, ShopMessageBlobCleanupJobs.Scope, ShopMessageBlobCleanupJobs.Account, ShopMessageBlobCleanupJobs.Container, ShopMessageBlobCleanupJobs.BlobKey, ShopMessageBlobCleanupJobs.Reason).VALUES(a.ID, a.ShopID, "asset", a.Account, a.Container, a.BlobKey, reason).ExecContext(ctx, tx)
	return err
}

func (r *assetRepository) FailReserved(ctx context.Context, tx *sql.Tx, reserved Asset, _ string) error {
	if tx == nil {
		return errors.New("asset transaction required")
	}
	a, err := loadAsset(ctx, tx, reserved.ID, true)
	if err != nil {
		return err
	}
	if a.OperationID.String() != reserved.OperationID || a.ShopID != reserved.ShopID || a.UploaderID != reserved.UploaderID || a.Account != reserved.Account || a.Container != reserved.Container || a.BlobKey != reserved.BlobKey {
		return errors.New("reservation proof mismatch")
	}
	if a.State != "uploading" {
		return nil
	}
	if err = updateAssetState(ctx, tx, a.ID, "cleanup_pending", "upload_not_finalized"); err != nil {
		return err
	}
	return enqueueAsset(ctx, tx, a, "upload_not_finalized")
}

func assetReferenceCount(ctx context.Context, tx *sql.Tx, id uuid.UUID) (int64, error) {
	var row struct{ Count int64 }
	err := SELECT(COUNT(ShopMessageAssetReferences.UploadID).AS("count")).FROM(ShopMessageAssetReferences).WHERE(ShopMessageAssetReferences.UploadID.EQ(UUID(id))).QueryContext(ctx, tx, &row)
	return row.Count, err
}

func (r *assetRepository) Discard(ctx context.Context, tx *sql.Tx, user *bootstrap.User, shopID, id string) error {
	admin, err := requireAssetActor(ctx, tx, user, shopID)
	if err != nil {
		return err
	}
	a, err := loadAsset(ctx, tx, id, true)
	if err != nil {
		return err
	}
	if a.ShopID != shopID || (!admin && a.UploaderID != user.UserID) {
		return shared.ErrShopAccessDenied
	}
	count, err := assetReferenceCount(ctx, tx, a.ID)
	if err != nil {
		return err
	}
	if count != 0 {
		return errors.New("published asset cannot be discarded")
	}
	if a.State != "ready" {
		return errors.New("asset is not ready for discard")
	}
	if err = updateAssetState(ctx, tx, a.ID, "cleanup_pending", ""); err != nil {
		return err
	}
	return enqueueAsset(ctx, tx, a, "discard")
}

func (r *assetRepository) ReplaceReferences(ctx context.Context, tx *sql.Tx, user *bootstrap.User, messageID, text string) error {
	if tx == nil || user == nil {
		return shared.ErrShopAccessDenied
	}
	if err := shared.AuthorizeOwnedMutation(ctx, tx, user.UserID, "message", messageID, false); err != nil {
		return err
	}
	var message model.ShopMessages
	if err := SELECT(legacyMessageColumns()).FROM(ShopMessages).WHERE(ShopMessages.ID.EQ(String(messageID))).QueryContext(ctx, tx, &message); err != nil {
		return err
	}
	targets := managedMarkerTargets(text)
	var existing []model.ShopMessageAssetReferences
	if err := SELECT(ShopMessageAssetReferences.AllColumns).FROM(ShopMessageAssetReferences).WHERE(ShopMessageAssetReferences.MessageID.EQ(String(messageID))).QueryContext(ctx, tx, &existing); err != nil {
		return err
	}
	prior := map[uuid.UUID]bool{}
	condition := Bool(false)
	for _, ref := range existing {
		prior[ref.UploadID] = true
		condition = condition.OR(ShopMessageUploads.ID.EQ(UUID(ref.UploadID)))
	}
	for target := range targets {
		condition = condition.OR(ShopMessageUploads.Account.EQ(String(target.Account)).AND(ShopMessageUploads.Container.EQ(String(target.Container))).AND(ShopMessageUploads.BlobKey.EQ(String(target.BlobKey))))
	}
	var affected []model.ShopMessageUploads
	if err := SELECT(ShopMessageUploads.AllColumns).FROM(ShopMessageUploads).WHERE(ShopMessageUploads.ShopID.EQ(String(message.ShopID)).AND(condition)).ORDER_BY(ShopMessageUploads.ID.ASC()).FOR(UPDATE()).QueryContext(ctx, tx, &affected); err != nil {
		return err
	}
	for _, a := range affected {
		wanted := targets[assetTarget{a.Account, a.Container, a.BlobKey}]
		if wanted {
			if a.State != "ready" {
				return errors.New("managed image is unavailable")
			}
			if !prior[a.ID] {
				if _, err := ShopMessageAssetReferences.INSERT(ShopMessageAssetReferences.AllColumns).MODEL(model.ShopMessageAssetReferences{MessageID: messageID, UploadID: a.ID, ShopID: message.ShopID}).ExecContext(ctx, tx); err != nil {
					return err
				}
			}
		} else if prior[a.ID] {
			if _, err := ShopMessageAssetReferences.DELETE().WHERE(ShopMessageAssetReferences.MessageID.EQ(String(messageID)).AND(ShopMessageAssetReferences.UploadID.EQ(UUID(a.ID)))).ExecContext(ctx, tx); err != nil {
				return err
			}
			count, err := assetReferenceCount(ctx, tx, a.ID)
			if err != nil {
				return err
			}
			if count == 0 && a.State == "ready" {
				if err = updateAssetState(ctx, tx, a.ID, "cleanup_pending", ""); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Caller owns the already authorized Shop deletion transaction. Registry rows and
// independent jobs survive the parent and retain interrupted-upload proof.
func (r *assetRepository) EnqueueShopCleanup(ctx context.Context, tx *sql.Tx, shopID string) error {
	if tx == nil {
		return errors.New("asset transaction required")
	}
	var shop model.Shops
	if err := SELECT(Shops.AllColumns).FROM(Shops).WHERE(Shops.ID.EQ(String(shopID))).FOR(UPDATE()).QueryContext(ctx, tx, &shop); err != nil {
		return err
	}
	if r.storage.valid() {
		parsed, err := uuid.Parse(shop.ID)
		if err != nil || parsed.String() != shop.ID {
			return errors.New("invalid persisted shop identity")
		}
		if _, err = ShopMessageBlobCleanupJobs.INSERT(ShopMessageBlobCleanupJobs.ShopID, ShopMessageBlobCleanupJobs.Scope, ShopMessageBlobCleanupJobs.Account, ShopMessageBlobCleanupJobs.Container, ShopMessageBlobCleanupJobs.BlobKey, ShopMessageBlobCleanupJobs.Reason).VALUES(shop.ID, "shop_prefix", r.storage.Account, r.storage.Container, shop.ID+"/", "shop_deleted").ExecContext(ctx, tx); err != nil {
			return err
		}
	}

	var assets []model.ShopMessageUploads
	if err := SELECT(ShopMessageUploads.AllColumns).FROM(ShopMessageUploads).WHERE(ShopMessageUploads.ShopID.EQ(String(shop.ID))).ORDER_BY(ShopMessageUploads.ID.ASC()).FOR(UPDATE()).QueryContext(ctx, tx, &assets); err != nil {
		return err
	}
	for _, a := range assets {
		if a.State == "uploading" || a.State == "ready" {
			if err := updateAssetState(ctx, tx, a.ID, "cleanup_pending", ""); err != nil {
				return err
			}
		}
		if err := enqueueAsset(ctx, tx, a, "shop_deleted"); err != nil {
			return err
		}
	}
	return nil
}

// Each repository owns its claims. A different process can recover an expired
// lease, but cannot acknowledge another owner's work.
type cleanupRepository struct {
	db      *sql.DB
	storage AssetStorage
	cfg     CleanupConfig
	owner   string
	now     func() time.Time
}

func NewCleanupRepository(database *sql.DB, storage AssetStorage, cfg CleanupConfig) (CleanupRepository, error) {
	if database == nil {
		return nil, errors.New("cleanup database required")
	}
	if err := validateCleanupConfig(cfg); err != nil {
		return nil, err
	}
	return &cleanupRepository{database, storage, cfg, uuid.NewString(), time.Now}, nil
}
func cleanupJob(row model.ShopMessageBlobCleanupJobs) CleanupJob {
	j := CleanupJob{ID: row.ID.String(), Scope: row.Scope, Account: row.Account, Container: row.Container, BlobKey: row.BlobKey, ShopID: row.ShopID, State: row.State}
	if row.UploadID != nil {
		j.AssetID = row.UploadID.String()
	}
	if row.LeaseUntil != nil {
		j.LeaseUntil = *row.LeaseUntil
	}
	return j
}
func (r *cleanupRepository) Claim(ctx context.Context, limit int, leaseUntil time.Time) ([]CleanupJob, error) {
	if limit <= 0 || limit > r.cfg.BatchSize || !leaseUntil.After(r.now()) {
		return nil, errors.New("invalid cleanup claim")
	}
	jobs := []CleanupJob{}
	err := dbutil.WithTxContext(ctx, r.db, func(tx *sql.Tx) error {
		j := ShopMessageBlobCleanupJobs
		now := TimestampzT(r.now())
		due := j.State.IN(String("pending"), String("retryable")).AND(j.NextAttempt.LT_EQ(now)).OR(j.State.EQ(String("leased")).AND(j.LeaseUntil.LT_EQ(now)))
		var rows []model.ShopMessageBlobCleanupJobs
		if err := SELECT(j.AllColumns).FROM(j).WHERE(due).ORDER_BY(j.NextAttempt.ASC(), j.ID.ASC()).LIMIT(int64(limit)).FOR(UPDATE().SKIP_LOCKED()).QueryContext(ctx, tx, &rows); err != nil {
			return err
		}
		for _, row := range rows {
			if row.State == "leased" {
				slog.Warn("shop_image_cleanup_lease_recovered", "job_id", row.ID.String())
			}
			if _, err := j.UPDATE(j.State, j.LeaseOwner, j.LeaseUntil, j.UpdatedAt).SET(String("leased"), String(r.owner), TimestampzT(leaseUntil), now).WHERE(j.ID.EQ(UUID(row.ID))).ExecContext(ctx, tx); err != nil {
				return err
			}
			row.State = "leased"
			row.LeaseUntil = &leaseUntil
			jobs = append(jobs, cleanupJob(row))
		}
		return nil
	})
	return jobs, err
}
func (r *cleanupRepository) loadClaim(ctx context.Context, tx *sql.Tx, id string, lock bool) (model.ShopMessageBlobCleanupJobs, error) {
	var row model.ShopMessageBlobCleanupJobs
	parsed, err := uuid.Parse(id)
	if err != nil {
		return row, err
	}
	j := ShopMessageBlobCleanupJobs
	stmt := SELECT(j.AllColumns).FROM(j).WHERE(j.ID.EQ(UUID(parsed)).AND(j.State.EQ(String("leased"))).AND(j.LeaseOwner.EQ(String(r.owner))).AND(j.LeaseUntil.GT(TimestampzT(r.now()))))
	if lock {
		stmt = stmt.FOR(UPDATE())
	}
	err = stmt.QueryContext(ctx, tx, &row)
	return row, err
}
func (r *cleanupRepository) settle(ctx context.Context, tx *sql.Tx, row model.ShopMessageBlobCleanupJobs, state string, next time.Time, failure, cursor string, attempts int32) error {
	j := ShopMessageBlobCleanupJobs
	result, err := j.UPDATE(j.State, j.NextAttempt, j.Failure, j.Cursor, j.Attempts, j.LeaseOwner, j.LeaseUntil, j.UpdatedAt).SET(String(state), TimestampzT(next), String(failure), String(cursor), Int32(attempts), String(""), NULL, TimestampzT(r.now())).WHERE(j.ID.EQ(UUID(row.ID)).AND(j.State.EQ(String("leased"))).AND(j.LeaseOwner.EQ(String(r.owner))).AND(j.LeaseUntil.GT(TimestampzT(r.now())))).ExecContext(ctx, tx)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("cleanup lease lost")
	}
	return nil
}
func (r *cleanupRepository) Prepare(ctx context.Context, job CleanupJob) (Asset, bool, error) {
	var target Asset
	ready := false
	err := dbutil.WithTxContext(ctx, r.db, func(tx *sql.Tx) error {
		row, err := r.loadClaim(ctx, tx, job.ID, false)
		if err != nil {
			return err
		}
		if row.Account != r.storage.Account || row.Container != r.storage.Container || !r.storage.valid() {
			return errCleanupTarget
		}
		if row.Scope == "shop_prefix" {
			return r.preparePrefix(ctx, tx, row, &target, &ready)
		}
		a, err := loadAsset(ctx, tx, row.UploadID.String(), true)
		if err != nil {
			return err
		}
		if a.Account != row.Account || a.Container != row.Container || a.BlobKey != row.BlobKey || a.ShopID != row.ShopID {
			return errCleanupTarget
		}
		count, err := assetReferenceCount(ctx, tx, a.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return r.settle(ctx, tx, row, "completed", r.now(), "references_survive", row.Cursor, row.Attempts)
		}
		// Shop deletion retains the upload lease when moving uploading to pending.
		if (a.State == "uploading" || a.State == "cleanup_pending") && a.LeaseUntil.After(r.now()) {
			return r.settle(ctx, tx, row, "pending", a.LeaseUntil, "upload_lease_live", row.Cursor, row.Attempts)
		}
		if a.State == "uploading" || a.State == "ready" {
			if err := updateAssetState(ctx, tx, a.ID, "cleanup_pending", ""); err != nil {
				return err
			}
			a.State = "cleanup_pending"
		}
		if a.State == "cleanup_pending" {
			if err := updateAssetState(ctx, tx, a.ID, "deleting", ""); err != nil {
				return err
			}
			a.State = "deleting"
		}
		if _, err := r.loadClaim(ctx, tx, job.ID, true); err != nil {
			return err
		}
		target = assetFromRow(a)
		ready = true
		return nil
	})
	return target, ready, err
}
func (r *cleanupRepository) preparePrefix(ctx context.Context, tx *sql.Tx, row model.ShopMessageBlobCleanupJobs, target *Asset, ready *bool) error {
	id, err := uuid.Parse(row.ShopID)
	if err != nil || id.String() != row.ShopID || row.BlobKey != row.ShopID+"/" {
		return errCleanupTarget
	}
	var shops []model.Shops
	if err := SELECT(Shops.AllColumns).FROM(Shops).WHERE(Shops.ID.EQ(String(row.ShopID))).QueryContext(ctx, tx, &shops); err != nil {
		return err
	}
	if len(shops) != 0 {
		return errCleanupTarget
	}
	var assets []model.ShopMessageUploads
	if err := SELECT(ShopMessageUploads.AllColumns).FROM(ShopMessageUploads).WHERE(ShopMessageUploads.ShopID.EQ(String(row.ShopID))).ORDER_BY(ShopMessageUploads.ID.ASC()).FOR(UPDATE()).QueryContext(ctx, tx, &assets); err != nil {
		return err
	}
	for _, a := range assets {
		count, err := assetReferenceCount(ctx, tx, a.ID)
		if err != nil {
			return err
		}
		if count != 0 {
			return errCleanupTarget
		}
		if (a.State == "uploading" || a.State == "cleanup_pending") && a.LeaseUntil.After(r.now()) {
			return r.settle(ctx, tx, row, "pending", a.LeaseUntil, "upload_lease_live", row.Cursor, row.Attempts)
		}
	}
	for _, a := range assets {
		if a.State == "ready" || a.State == "uploading" {
			if err := updateAssetState(ctx, tx, a.ID, "cleanup_pending", ""); err != nil {
				return err
			}
			a.State = "cleanup_pending"
		}
		if a.State == "cleanup_pending" {
			if err := updateAssetState(ctx, tx, a.ID, "deleting", ""); err != nil {
				return err
			}
		}
	}
	if _, err := r.loadClaim(ctx, tx, row.ID.String(), true); err != nil {
		return err
	}
	*target = Asset{ShopID: row.ShopID, Account: row.Account, Container: row.Container, BlobKey: row.BlobKey, State: "shop_prefix", OperationID: row.Cursor}
	*ready = true
	return nil
}
func (r *cleanupRepository) Finish(ctx context.Context, jobID string, deleteErr error) error {
	return dbutil.WithTxContext(ctx, r.db, func(tx *sql.Tx) error {
		row, err := r.loadClaim(ctx, tx, jobID, true)
		if err != nil {
			return err
		}
		state, failure, cursor := "completed", "", row.Cursor
		attempts := row.Attempts
		next := r.now()
		var progress *prefixProgress
		if errors.As(deleteErr, &progress) {
			state = "pending"
			cursor = progress.cursor
		} else if deleteErr != nil {
			attempts++
			state = "retryable"
			failure = "storage_operation_failed"
			delay := 5 * time.Second
			for n := int32(1); n < attempts && delay < r.cfg.MaxBackoff; n++ {
				delay *= 2
			}
			if delay > r.cfg.MaxBackoff {
				delay = r.cfg.MaxBackoff
			}
			next = next.Add(delay)
			if errors.Is(deleteErr, errCleanupTarget) || int(attempts) >= r.cfg.MaxAttempts {
				state = "manual-review"
				failure = "target_or_retry_limit"
				slog.Warn("shop_image_cleanup_manual_review", "job_id", jobID, "attempts", attempts)
			}
		}
		// Acknowledgement holds only this queue row. Reconcile materializes the
		// terminal tombstone from committed success proof without queue locks.

		return r.settle(ctx, tx, row, state, next, failure, cursor, attempts)
	})
}
func (r *cleanupRepository) Reconcile(ctx context.Context) error {
	return dbutil.WithTxContext(ctx, r.db, func(tx *sql.Tx) error {
		a := ShopMessageUploads
		j := ShopMessageBlobCleanupJobs
		successfulDelete := EXISTS(SELECT(j.ID).FROM(j).WHERE(j.UploadID.EQ(a.ID).AND(j.Scope.EQ(String("asset"))).AND(j.State.EQ(String("completed"))).AND(j.Failure.EQ(String(""))).AND(j.ShopID.EQ(a.ShopID)).AND(j.Account.EQ(a.Account)).AND(j.Container.EQ(a.Container)).AND(j.BlobKey.EQ(a.BlobKey))))
		var acknowledged []model.ShopMessageUploads
		if err := SELECT(a.AllColumns).FROM(a).WHERE(a.State.EQ(String("deleting")).AND(successfulDelete)).ORDER_BY(a.ID.ASC()).LIMIT(int64(r.cfg.BatchSize)).FOR(UPDATE().SKIP_LOCKED()).QueryContext(ctx, tx, &acknowledged); err != nil {
			return err
		}
		for _, row := range acknowledged {
			if err := updateAssetState(ctx, tx, row.ID, "deleted", ""); err != nil {
				return err
			}
		}
		// Existing pending/leased/manual-review work retains ownership. Completed
		// targets periodically run again because a canceled PUT can arrive late.
		active := EXISTS(SELECT(j.ID).FROM(j).WHERE(j.UploadID.EQ(a.ID).AND(j.State.NOT_EQ(String("completed")))))
		candidates := a.State.EQ(String("uploading")).AND(a.LeaseUntil.LT_EQ(TimestampzT(r.now()))).OR(a.State.IN(String("cleanup_pending"), String("deleting"), String("deleted")))
		recent := EXISTS(SELECT(j.ID).FROM(j).WHERE(j.UploadID.EQ(a.ID).AND(j.UpdatedAt.GT(TimestampzT(r.now().Add(-r.cfg.MaxBackoff))))))
		var rows []model.ShopMessageUploads
		if err := SELECT(a.AllColumns).FROM(a).WHERE(candidates.AND(NOT(active)).AND(NOT(recent))).ORDER_BY(a.UpdatedAt.ASC(), a.ID.ASC()).LIMIT(int64(r.cfg.BatchSize)).FOR(UPDATE().SKIP_LOCKED()).QueryContext(ctx, tx, &rows); err != nil {
			return err
		}
		for _, row := range rows {
			if row.State == "uploading" {
				if err := updateAssetState(ctx, tx, row.ID, "cleanup_pending", "upload_lease_expired"); err != nil {
					return err
				}
			}
			if err := enqueueAsset(ctx, tx, row, "reconcile"); err != nil {
				return err
			}
		}
		return nil
	})
}
