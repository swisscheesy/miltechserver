package user_general

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type repoStub struct {
	upsertErr error
	deleteErr error
	updateErr error
}

func (r *repoStub) UpsertUser(*bootstrap.User, auth.UserDto) error {
	return r.upsertErr
}

func (r *repoStub) DeleteUser(string) error {
	return r.deleteErr
}

func (r *repoStub) UpdateUserDisplayName(string, string) error {
	return r.updateErr
}

func TestServiceDeleteUserReturnsError(t *testing.T) {
	repo := &repoStub{deleteErr: errors.New("boom")}
	svc := NewService(repo)

	err := svc.DeleteUser("uid")
	require.Error(t, err)
}
