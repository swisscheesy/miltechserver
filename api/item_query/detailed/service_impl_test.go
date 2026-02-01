package detailed

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/api/response"
)

type repoStub struct {
	resp response.DetailedResponse
	err  error
}

func (r *repoStub) GetDetailedItemData(ctx context.Context, niin string) (response.DetailedResponse, error) {
	return r.resp, r.err
}

func TestFindDetailedItemReturnsRepoData(t *testing.T) {
	stub := &repoStub{resp: response.DetailedResponse{}}
	svc := NewService(stub)

	_, err := svc.FindDetailedItem(context.Background(), "123")
	require.NoError(t, err)
}

func TestFindDetailedItemReturnsRepoError(t *testing.T) {
	stub := &repoStub{err: errors.New("boom")}
	svc := NewService(stub)

	_, err := svc.FindDetailedItem(context.Background(), "123")
	require.Error(t, err)
}

func TestFindDetailedItemUsesCacheOnHit(t *testing.T) {
	stub := &repoStub{resp: response.DetailedResponse{}}
	svc := NewService(stub)

	// First call - cache miss, hits repo
	_, err := svc.FindDetailedItem(context.Background(), "123")
	require.NoError(t, err)

	// Second call - should hit cache, not repo
	stub.err = errors.New("should not be called")
	_, err = svc.FindDetailedItem(context.Background(), "123")
	require.NoError(t, err) // No error because we hit cache
}
