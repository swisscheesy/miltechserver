package eic_test

import (
	"database/sql"
	"testing"

	"miltechserver/api/eic"

	"github.com/stretchr/testify/require"
)

func TestEICRepositoryValidationErrors(t *testing.T) {
	repo := eic.NewRepository((*sql.DB)(nil))

	_, err := repo.GetByNIIN(" ")
	require.ErrorIs(t, err, eic.ErrEmptyParam)

	_, err = repo.GetByLIN("\t")
	require.ErrorIs(t, err, eic.ErrEmptyParam)

	_, err = repo.GetByFSCPaginated("", 1)
	require.ErrorIs(t, err, eic.ErrEmptyParam)

	_, err = repo.GetByFSCPaginated("ABCD", 0)
	require.ErrorIs(t, err, eic.ErrInvalidPage)

	_, err = repo.GetAllPaginated(0, "")
	require.ErrorIs(t, err, eic.ErrInvalidPage)
}
