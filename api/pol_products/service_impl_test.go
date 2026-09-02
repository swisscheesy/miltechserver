package pol_products

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type repoStub struct {
	resp PolProductsResponse
	err  error
}

func (r *repoStub) GetPolProducts() (PolProductsResponse, error) {
	return r.resp, r.err
}

func TestServiceReturnsProducts(t *testing.T) {
	repo := &repoStub{resp: PolProductsResponse{Count: 239}}
	svc := NewService(repo)

	resp, err := svc.GetPolProducts(context.Background(), )
	require.NoError(t, err)
	require.Equal(t, 239, resp.Count)
}

func TestServiceReturnsError(t *testing.T) {
	repo := &repoStub{err: errors.New("db down")}
	svc := NewService(repo)

	_, err := svc.GetPolProducts(context.Background(), )
	require.Error(t, err)
}
