package settings

import (
	"context"
	"testing"

	"miltechserver/api/request"
	"miltechserver/api/shops/shared"
	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type contextAuthorization struct {
	shared.ShopAuthorization
	t   *testing.T
	ctx context.Context
}

func (a contextAuthorization) IsUserMemberOfShop(ctx context.Context, user *bootstrap.User, shop string) (bool, error) {
	require.Same(a.t, a.ctx, ctx)
	return true, nil
}
func (a contextAuthorization) IsUserShopAdmin(ctx context.Context, user *bootstrap.User, shop string) (bool, error) {
	require.Same(a.t, a.ctx, ctx)
	return true, nil
}

type contextRepository struct {
	Repository
	t     *testing.T
	ctx   context.Context
	value bool
}

func (r *contextRepository) GetShopAdminOnlyListsSetting(ctx context.Context, shop string) (bool, error) {
	require.Same(r.t, r.ctx, ctx)
	return r.value, nil
}
func (r *contextRepository) GetShopSettings(ctx context.Context, shop string) (*request.ShopSettings, error) {
	require.Same(r.t, r.ctx, ctx)
	return &request.ShopSettings{AdminOnlyLists: r.value}, nil
}
func (r *contextRepository) UpdateShopAdminOnlyListsSetting(ctx context.Context, user *bootstrap.User, shop string, value bool) error {
	require.Same(r.t, r.ctx, ctx)
	r.value = value
	return nil
}
func (r *contextRepository) UpdateShopSettings(ctx context.Context, user *bootstrap.User, shop string, updates request.UpdateShopSettingsRequest) error {
	require.Same(r.t, r.ctx, ctx)
	r.value = *updates.AdminOnlyLists
	return nil
}

func TestSettingsPropagateRequestContextAndFalse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := &contextRepository{t: t, ctx: ctx, value: true}
	service := NewService(repo, contextAuthorization{t: t, ctx: ctx})
	user := &bootstrap.User{UserID: "admin"}
	value, err := service.GetShopAdminOnlyListsSetting(ctx, user, "shop")
	require.NoError(t, err)
	require.True(t, value)
	settings, err := service.GetShopSettings(ctx, user, "shop")
	require.NoError(t, err)
	require.True(t, settings.AdminOnlyLists)
	require.NoError(t, service.UpdateShopAdminOnlyListsSetting(ctx, user, "shop", false))
	require.False(t, repo.value)
	repo.value = true
	disabled := false
	settings, err = service.UpdateShopSettings(ctx, user, "shop", request.UpdateShopSettingsRequest{AdminOnlyLists: &disabled})
	require.NoError(t, err)
	require.False(t, settings.AdminOnlyLists)
}
