package ratelimit

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/go-jet/jet/v2/qrm"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
)

const (
	maxUploadsPerHour = 3
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) CheckLimit(userID string, niin string) (bool, *time.Time, error) {
	stmt := SELECT(
		MaterialImagesUploadLimits.AllColumns,
	).FROM(
		MaterialImagesUploadLimits,
	).WHERE(
		MaterialImagesUploadLimits.UserID.EQ(String(userID)).
			AND(MaterialImagesUploadLimits.Niin.EQ(String(niin))),
	)

	var limit model.MaterialImagesUploadLimits
	err := stmt.Query(r.db, &limit)
	if err != nil {
		if err == qrm.ErrNoRows {
			return true, nil, nil
		}
		return false, nil, fmt.Errorf("failed to check upload limit: %w", err)
	}

	now := time.Now().UTC()
	windowStart := limit.LastUploadTime.Add(-1 * time.Hour)

	if now.After(limit.LastUploadTime.Add(1 * time.Hour)) {
		return true, nil, nil
	}

	if limit.UploadCount < maxUploadsPerHour {
		return true, nil, nil
	}

	nextAllowedTime := windowStart.Add(1 * time.Hour)
	return false, &nextAllowedTime, nil
}

func (r *RepositoryImpl) UpdateLimit(userID string, niin string) error {
	now := time.Now().UTC()

	stmt := MaterialImagesUploadLimits.INSERT(
		MaterialImagesUploadLimits.UserID,
		MaterialImagesUploadLimits.Niin,
		MaterialImagesUploadLimits.LastUploadTime,
		MaterialImagesUploadLimits.UploadCount,
	).VALUES(
		userID,
		niin,
		TimestampT(now),
		1,
	).ON_CONFLICT(
		MaterialImagesUploadLimits.UserID,
		MaterialImagesUploadLimits.Niin,
	).DO_UPDATE(
		SET(
			MaterialImagesUploadLimits.LastUploadTime.SET(TimestampT(now)),
			MaterialImagesUploadLimits.UploadCount.SET(
				MaterialImagesUploadLimits.UploadCount.ADD(Int(1)),
			),
		),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to update upload limit: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) CleanupOld(olderThan time.Time) error {
	stmt := MaterialImagesUploadLimits.DELETE().WHERE(
		MaterialImagesUploadLimits.LastUploadTime.LT(TimestampT(olderThan)),
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to cleanup old limits: %w", err)
	}

	return nil
}
