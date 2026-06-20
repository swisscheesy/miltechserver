package core

import (
	"context"
	"errors"
	"testing"

	"miltechserver/api/response"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type overviewRepositoryStub struct {
	Repository
	getOverview func(context.Context, *bootstrap.User) ([]response.ShopEquipmentOverview, error)
}

func (stub overviewRepositoryStub) GetShopEquipmentOverview(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
	return stub.getOverview(ctx, user)
}

func TestGetShopEquipmentOverview(t *testing.T) {
	requestContext := context.Background()
	repository := overviewRepositoryStub{getOverview: func(ctx context.Context, user *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
		require.Equal(t, requestContext, ctx)
		require.Equal(t, "user-1", user.UserID)
		return []response.ShopEquipmentOverview{
			{ID: "shop-1", Equipment: nil},
			{ID: "shop-2", Equipment: []response.ShopEquipmentSummary{{ID: "equipment-1"}}},
		}, nil
	}}
	service := NewService(repository, nil)

	result, err := service.GetShopEquipmentOverview(requestContext, &bootstrap.User{UserID: "user-1"})
	require.NoError(t, err)
	require.NotNil(t, result.Shops[0].Equipment)
	require.Empty(t, result.Shops[0].Equipment)
	require.Equal(t, 0, result.Shops[0].EquipmentCount)
	require.Equal(t, 1, result.Shops[1].EquipmentCount)
}

func TestGetShopEquipmentOverviewRejectsMissingUser(t *testing.T) {
	service := NewService(overviewRepositoryStub{}, nil)
	result, err := service.GetShopEquipmentOverview(context.Background(), nil)
	require.Nil(t, result)
	require.EqualError(t, err, "unauthorized user")
}

func TestGetShopEquipmentOverviewHidesRepositoryError(t *testing.T) {
	repository := overviewRepositoryStub{getOverview: func(context.Context, *bootstrap.User) ([]response.ShopEquipmentOverview, error) {
		return nil, errors.New("database host and query details")
	}}
	service := NewService(repository, nil)
	result, err := service.GetShopEquipmentOverview(context.Background(), &bootstrap.User{UserID: "user-1"})
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrShopEquipmentOverviewUnavailable)
	require.NotContains(t, err.Error(), "database host")
}
