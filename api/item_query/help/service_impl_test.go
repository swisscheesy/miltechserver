package help

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/.gen/miltech_ng/public/model"
)

type repoStub struct {
	rows     []model.Help
	err      error
	lastCode string
}

func (r *repoStub) FindByCode(code string) ([]model.Help, error) {
	r.lastCode = code
	return r.rows, r.err
}

func TestFindByCodeNormalizesAndReturnsFirstRow(t *testing.T) {
	repo := &repoStub{
		rows: []model.Help{
			{Code: "AB12", Description: "A description"},
			{Code: "AB12", Description: "B description"},
		},
	}
	svc := NewService(repo)

	result, err := svc.FindByCode("  ab12 ")
	require.NoError(t, err)
	require.Equal(t, "AB12", repo.lastCode)
	require.Equal(t, "A description", result.Description)
}

func TestFindByCodeReturnsInvalidCodeForEmptyInput(t *testing.T) {
	svc := NewService(&repoStub{})

	_, err := svc.FindByCode("   ")
	require.ErrorIs(t, err, ErrInvalidCode)
}

func TestFindByCodePropagatesRepositoryErrors(t *testing.T) {
	expectedErr := errors.New("db down")
	svc := NewService(&repoStub{err: expectedErr})

	_, err := svc.FindByCode("abc")
	require.ErrorIs(t, err, expectedErr)
}
