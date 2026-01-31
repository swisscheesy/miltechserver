package quick_lists

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type repoStub struct {
	clothingResp  QuickListsClothingResponse
	clothingErr   error
	wheelsResp    QuickListsWheelsResponse
	wheelsErr     error
	batteriesResp QuickListsBatteryResponse
	batteriesErr  error
}

func (r *repoStub) GetQuickListClothing() (QuickListsClothingResponse, error) {
	return r.clothingResp, r.clothingErr
}

func (r *repoStub) GetQuickListWheels() (QuickListsWheelsResponse, error) {
	return r.wheelsResp, r.wheelsErr
}

func (r *repoStub) GetQuickListBatteries() (QuickListsBatteryResponse, error) {
	return r.batteriesResp, r.batteriesErr
}

func TestServiceReturnsClothing(t *testing.T) {
	repo := &repoStub{clothingResp: QuickListsClothingResponse{Count: 1}}
	svc := NewService(repo)

	resp, err := svc.GetQuickListClothing()
	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
}

func TestServiceReturnsError(t *testing.T) {
	repo := &repoStub{wheelsErr: errors.New("db down")}
	svc := NewService(repo)

	_, err := svc.GetQuickListWheels()
	require.Error(t, err)
}
