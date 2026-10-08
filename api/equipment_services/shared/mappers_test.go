package shared

import (
	"bytes"
	"context"
	"database/sql/driver"
	"github.com/stretchr/testify/require"
	"log/slog"
	"miltechserver/internal/testsql"
	"testing"
)

func TestUsernameCacheCancellationDoesNotPoisonLookup(t *testing.T) {
	db := testsql.Open(t, &testsql.Behavior{Columns: []string{"users.username"}, Rows: [][]driver.Value{{"actual user"}}})
	cache := NewUsernameCache(NewUsernameRepository(db))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cache.GetUsernameByUserID(ctx, "actor")
	require.ErrorIs(t, err, context.Canceled)
	username, err := cache.GetUsernameByUserID(context.Background(), "actor")
	require.NoError(t, err)
	require.Equal(t, "actual user", username)
}

func TestUsernameCacheHitHonorsCancellation(t *testing.T) {
	db := testsql.Open(t, &testsql.Behavior{Columns: []string{"users.username"}, Rows: [][]driver.Value{{"actual user"}}})
	cache := NewUsernameCache(NewUsernameRepository(db))
	_, err := cache.GetUsernameByUserID(context.Background(), "actor")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = cache.GetUsernameByUserID(ctx, "actor")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUsernameMissingAndEmptyPreserveFallback(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	for _, rows := range [][][]driver.Value{nil, {{""}}} {
		db := testsql.Open(t, &testsql.Behavior{Columns: []string{"users.username"}, Rows: rows})
		username, err := NewUsernameRepository(db).GetUsernameByUserID(context.Background(), "missing-or-empty")
		require.NoError(t, err)
		require.Equal(t, "Unknown User", username)
	}

	require.Contains(t, logs.String(), "Failed to get username for user")
	require.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("level=WARN")))
}
