package tmde_test

import (
	"database/sql"
	"testing"

	"miltechserver/api/tmde"

	"github.com/stretchr/testify/require"
)

func TestRepositoryValidationErrors(t *testing.T) {
	repo := tmde.NewRepository((*sql.DB)(nil))

	_, err := repo.GetByNIIN("  ")
	require.ErrorIs(t, err, tmde.ErrEmptyParam)

	_, err = repo.GetByNIIN("\t")
	require.ErrorIs(t, err, tmde.ErrEmptyParam)

	_, err = repo.GetAllPaginated(0)
	require.ErrorIs(t, err, tmde.ErrInvalidPage)

	_, err = repo.GetAllPaginated(-1)
	require.ErrorIs(t, err, tmde.ErrInvalidPage)
}
