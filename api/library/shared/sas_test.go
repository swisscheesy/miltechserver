package shared

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/stretchr/testify/require"
)

func TestUDKCacheHit(t *testing.T) {
	// Reset package cache state before test.
	packageUDKCache = udkEntry{}

	// Manually populate the cache with a key that expires far in the future.
	fakeKey := &service.UserDelegationCredential{}
	packageUDKCache.key = fakeKey
	packageUDKCache.expiresAt = time.Now().Add(30 * time.Minute)

	// getOrRefreshUDK should return the cached key without calling Azure.
	// We pass a nil svcClient — if it tries to call Azure it will panic.
	got, err := getOrRefreshUDK(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, fakeKey, got)
}

func TestUDKCacheMiss_ExpiredKey(t *testing.T) {
	// Reset package cache state before test.
	packageUDKCache = udkEntry{}

	// Populate the cache with an already-expired key.
	packageUDKCache.key = &service.UserDelegationCredential{}
	packageUDKCache.expiresAt = time.Now().Add(-1 * time.Minute)

	// getOrRefreshUDK should detect the expiry and attempt to call Azure.
	// Since svcClient is nil it will panic — which proves the cache miss path
	// was taken. Recover the panic to assert it happened.
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic from nil svcClient on cache miss")
	}()

	//nolint:staticcheck
	_, _ = getOrRefreshUDK(context.Background(), nil)
}

func TestUDKCacheMiss_NearExpiry(t *testing.T) {
	// Reset package cache state before test.
	packageUDKCache = udkEntry{}

	// Populate the cache with a key that expires in 3 minutes (inside the 5-min margin).
	packageUDKCache.key = &service.UserDelegationCredential{}
	packageUDKCache.expiresAt = time.Now().Add(3 * time.Minute)

	// Should be treated as a miss — attempt to call Azure (nil panic expected).
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic from nil svcClient on near-expiry miss")
	}()

	//nolint:staticcheck
	_, _ = getOrRefreshUDK(context.Background(), nil)
}

func TestUDKCacheHit_ExactlyAboveMargin(t *testing.T) {
	// Reset package cache state before test.
	packageUDKCache = udkEntry{}

	// Key expires in 6 minutes — just above the 5-min margin. Should be a cache hit.
	fakeKey := &service.UserDelegationCredential{}
	packageUDKCache.key = fakeKey
	packageUDKCache.expiresAt = time.Now().Add(6 * time.Minute)

	got, err := getOrRefreshUDK(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, fakeKey, got)
}
