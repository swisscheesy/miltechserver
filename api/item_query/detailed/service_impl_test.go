package detailed

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"miltechserver/api/response"
)

type repoStub struct {
	resp response.DetailedResponse
	err  error
}

func (r *repoStub) GetDetailedItemData(string) (response.DetailedResponse, error) {
	return r.resp, r.err
}

func TestFindDetailedItemReturnsRepoData(t *testing.T) {
	stub := &repoStub{resp: response.DetailedResponse{}}
	svc := NewService(stub)

	_, err := svc.FindDetailedItem("123")
	require.NoError(t, err)
}

func TestFindDetailedItemReturnsRepoError(t *testing.T) {
	stub := &repoStub{err: errors.New("boom")}
	svc := NewService(stub)

	_, err := svc.FindDetailedItem("123")
	require.Error(t, err)
}
