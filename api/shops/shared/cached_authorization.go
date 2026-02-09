package shared

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"miltechserver/bootstrap"
)

const cachedAuthorizationKey = "cached_authorization"

type CachedAuthorization struct {
	inner       ShopAuthorization
	mu          sync.RWMutex
	boolCache   map[string]bool
	stringCache map[string]string
}

func NewCachedAuthorization(inner ShopAuthorization) *CachedAuthorization {
	return &CachedAuthorization{
		inner:       inner,
		boolCache:   make(map[string]bool),
		stringCache: make(map[string]string),
	}
}

func CachedAuthorizationFromContext(c *gin.Context, factory func() ShopAuthorization) *CachedAuthorization {
	if cached, exists := c.Get(cachedAuthorizationKey); exists {
		return cached.(*CachedAuthorization)
	}

	auth := NewCachedAuthorization(factory())
	c.Set(cachedAuthorizationKey, auth)
	return auth
}

func (auth *CachedAuthorization) cacheKey(operation string, parts ...string) string {
	if len(parts) == 0 {
		return operation
	}

	key := operation
	for _, part := range parts {
		key = fmt.Sprintf("%s:%s", key, part)
	}
	return key
}

func (auth *CachedAuthorization) getBool(key string) (bool, bool) {
	auth.mu.RLock()
	defer auth.mu.RUnlock()
	val, ok := auth.boolCache[key]
	return val, ok
}

func (auth *CachedAuthorization) setBool(key string, val bool) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	auth.boolCache[key] = val
}

func (auth *CachedAuthorization) getString(key string) (string, bool) {
	auth.mu.RLock()
	defer auth.mu.RUnlock()
	val, ok := auth.stringCache[key]
	return val, ok
}

func (auth *CachedAuthorization) setString(key string, val string) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	auth.stringCache[key] = val
}

func (auth *CachedAuthorization) IsUserMemberOfShop(user *bootstrap.User, shopID string) (bool, error) {
	key := auth.cacheKey("member", shopID, user.UserID)
	if cached, ok := auth.getBool(key); ok {
		return cached, nil
	}

	val, err := auth.inner.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return false, err
	}

	auth.setBool(key, val)
	return val, nil
}

func (auth *CachedAuthorization) IsUserShopAdmin(user *bootstrap.User, shopID string) (bool, error) {
	key := auth.cacheKey("admin", shopID, user.UserID)
	if cached, ok := auth.getBool(key); ok {
		return cached, nil
	}

	val, err := auth.inner.IsUserShopAdmin(user, shopID)
	if err != nil {
		return false, err
	}

	auth.setBool(key, val)
	return val, nil
}

func (auth *CachedAuthorization) GetUserRoleInShop(user *bootstrap.User, shopID string) (string, error) {
	key := auth.cacheKey("role", shopID, user.UserID)
	if cached, ok := auth.getString(key); ok {
		return cached, nil
	}

	val, err := auth.inner.GetUserRoleInShop(user, shopID)
	if err != nil {
		return "", err
	}

	auth.setString(key, val)
	return val, nil
}

func (auth *CachedAuthorization) CanUserModifyVehicle(user *bootstrap.User, vehicleID string) (bool, error) {
	key := auth.cacheKey("modify_vehicle", vehicleID, user.UserID)
	if cached, ok := auth.getBool(key); ok {
		return cached, nil
	}

	val, err := auth.inner.CanUserModifyVehicle(user, vehicleID)
	if err != nil {
		return false, err
	}

	auth.setBool(key, val)
	return val, nil
}

func (auth *CachedAuthorization) CanUserModifyList(user *bootstrap.User, listID string) (bool, error) {
	key := auth.cacheKey("modify_list", listID, user.UserID)
	if cached, ok := auth.getBool(key); ok {
		return cached, nil
	}

	val, err := auth.inner.CanUserModifyList(user, listID)
	if err != nil {
		return false, err
	}

	auth.setBool(key, val)
	return val, nil
}

func (auth *CachedAuthorization) CanUserModifyNotification(user *bootstrap.User, notificationID string) (bool, error) {
	key := auth.cacheKey("modify_notification", notificationID, user.UserID)
	if cached, ok := auth.getBool(key); ok {
		return cached, nil
	}

	val, err := auth.inner.CanUserModifyNotification(user, notificationID)
	if err != nil {
		return false, err
	}

	auth.setBool(key, val)
	return val, nil
}

func (auth *CachedAuthorization) RequireShopMember(user *bootstrap.User, shopID string) error {
	isMember, err := auth.IsUserMemberOfShop(user, shopID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrShopAccessDenied
	}
	return nil
}

func (auth *CachedAuthorization) RequireShopAdmin(user *bootstrap.User, shopID string) error {
	isAdmin, err := auth.IsUserShopAdmin(user, shopID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrShopAdminRequired
	}
	return nil
}
