package flags

import (
	"database/sql"
	"fmt"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"miltechserver/.gen/miltech_ng/public/model"
	. "miltechserver/.gen/miltech_ng/public/table"
)

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) Create(flag model.MaterialImagesFlags) error {
	stmt := MaterialImagesFlags.INSERT(
		MaterialImagesFlags.ImageID,
		MaterialImagesFlags.UserID,
		MaterialImagesFlags.Reason,
		MaterialImagesFlags.Description,
	).VALUES(
		flag.ImageID,
		flag.UserID,
		flag.Reason,
		flag.Description,
	)

	_, err := stmt.Exec(r.db)
	if err != nil {
		return fmt.Errorf("failed to create flag: %w", err)
	}

	return nil
}

func (r *RepositoryImpl) GetByImage(imageID string) ([]model.MaterialImagesFlags, error) {
	stmt := SELECT(
		MaterialImagesFlags.AllColumns,
	).FROM(
		MaterialImagesFlags,
	).WHERE(
		MaterialImagesFlags.ImageID.EQ(UUID(uuid.MustParse(imageID))),
	).ORDER_BY(
		MaterialImagesFlags.CreatedAt.DESC(),
	)

	var flags []model.MaterialImagesFlags
	err := stmt.Query(r.db, &flags)
	if err != nil {
		return nil, fmt.Errorf("failed to get flags: %w", err)
	}

	return flags, nil
}

func (r *RepositoryImpl) CountByImage(imageID string) (int, error) {
	stmt := SELECT(
		COUNT(MaterialImagesFlags.ID).AS("count"),
	).FROM(
		MaterialImagesFlags,
	).WHERE(
		MaterialImagesFlags.ImageID.EQ(UUID(uuid.MustParse(imageID))),
	)

	var count struct {
		Count int32 `sql:"count"`
	}
	err := stmt.Query(r.db, &count)
	if err != nil {
		return 0, fmt.Errorf("failed to get flag count: %w", err)
	}

	return int(count.Count), nil
}
