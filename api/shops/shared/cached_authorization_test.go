package shared

import (
	"errors"
	"testing"

	"miltechserver/bootstrap"

	"github.com/stretchr/testify/require"
)

type fakeAuthorization struct {
	memberCalls int
	adminCalls  int
	roleCalls   int
	memberErr   error
	adminErr    error
	roleErr     error
	memberVal   bool
	adminVal    bool
	roleVal     string
}

func (auth *fakeAuthorization) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	auth.memberCalls++
	if auth.memberErr != nil {
		return false, auth.memberErr
	}
	return auth.memberVal, nil
}

func (auth *fakeAuthorization) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	auth.adminCalls++
	if auth.adminErr != nil {
		return false, auth.adminErr
	}
	return auth.adminVal, nil
}

func (auth *fakeAuthorization) GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error) {
	auth.roleCalls++
	if auth.roleErr != nil {
		return "", auth.roleErr
	}
	return auth.roleVal, nil
}

func (auth *fakeAuthorization) CanUserModifyVehicle(user *bootstrap.User, vehicleID string) (bool, error) {
	return false, nil
}

func (auth *fakeAuthorization) CanUserModifyList(user *bootstrap.User, listID string) (bool, error) {
	return false, nil
}

func (auth *fakeAuthorization) CanUserModifyNotification(user *bootstrap.User, notificationID string) (bool, error) {
	return false, nil
}

func (auth *fakeAuthorization) RequireShopMember(user *bootstrap.User, shopID string) error {
	return nil
}

func (auth *fakeAuthorization) RequireShopAdmin(user *bootstrap.User, shopID string) error {
	return nil
}

func TestCachedAuthorizationCachesSuccessfulCalls(t *testing.T) {
	inner := &fakeAuthorization{
		memberVal: true,
		adminVal:  true,
		roleVal:   "admin",
	}

	cached := NewCachedAuthorization(inner)
	user := &bootstrap.User{UserID: "user-1"}

	val, err := cached.IsUserMemberOfShop(user, "shop-1")
	require.NoError(t, err)
	require.True(t, val)

	val, err = cached.IsUserMemberOfShop(user, "shop-1")
	require.NoError(t, err)
	require.True(t, val)

	require.Equal(t, 1, inner.memberCalls)

	role, err := cached.GetUserRoleInShop(user, "shop-1")
	require.NoError(t, err)
	require.Equal(t, "admin", role)

	role, err = cached.GetUserRoleInShop(user, "shop-1")
	require.NoError(t, err)
	require.Equal(t, "admin", role)

	require.Equal(t, 1, inner.roleCalls)
}

func TestCachedAuthorizationDoesNotCacheErrors(t *testing.T) {
	inner := &fakeAuthorization{
		memberErr: errors.New("boom"),
	}

	cached := NewCachedAuthorization(inner)
	user := &bootstrap.User{UserID: "user-1"}

	_, err := cached.IsUserMemberOfShop(user, "shop-1")
	require.Error(t, err)

	_, err = cached.IsUserMemberOfShop(user, "shop-1")
	require.Error(t, err)

	require.Equal(t, 2, inner.memberCalls)
}
