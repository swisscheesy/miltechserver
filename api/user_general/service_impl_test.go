package user_general

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/api/auth"
	"miltechserver/bootstrap"
)

type repoStub struct {
	upsertErr  error
	deleteErr  error
	updateErr  error
	deletedUID string
	deleteCtx  context.Context
}

func (r *repoStub) UpsertUser(*bootstrap.User, auth.UserDto) error {
	return r.upsertErr
}

func (r *repoStub) DeleteUser(ctx context.Context, uid string) error {
	r.deleteCtx = ctx
	r.deletedUID = uid
	return r.deleteErr
}

func (r *repoStub) UpdateUserDisplayName(string, string) error {
	return r.updateErr
}

func TestServiceDeleteUserReturnsError(t *testing.T) {
	repo := &repoStub{deleteErr: errors.New("boom")}
	svc := NewService(repo)

	ctx := context.WithValue(context.Background(), struct{}{}, "request")
	err := svc.DeleteUser(ctx, "uid")
	require.Error(t, err)
	require.Same(t, ctx, repo.deleteCtx)
	require.Equal(t, "uid", repo.deletedUID)
}
