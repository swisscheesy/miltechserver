package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/stretchr/testify/require"
	userpmcsshared "miltechserver/api/user_pmcs/shared"
	"miltechserver/bootstrap"
	"miltechserver/internal/jetgen"
	"miltechserver/internal/testsql"
	"net/url"
	"testing"
)

func TestStartupJetUsesApplicationPort(t *testing.T) {
	env := &bootstrap.Env{Host: "127.0.0.1", Port: "55439", Username: "app", DBName: "miltech_test_fixture", DBSchema: "public", SslMode: "disable"}
	env.UserPmcs = bootstrap.UserPmcsConfig(userpmcsshared.DefaultConfig())
	behavior := &testsql.Behavior{Columns: []string{"ready"}, Rows: [][]driver.Value{{true}}}
	db := testsql.Open(t, behavior)
	calls := 0
	applicationPort := ""
	generatorPort := ""
	server, err := setupEngine(context.Background(), env, func(got *sql.DB, cfg jetgen.Config) error {
		calls++
		require.Equal(t, int64(1), behavior.Queries.Load(), "mandatory schema validated before generator")
		require.True(t, behavior.BeginReadOnly.Load())
		require.Same(t, db, got)
		require.Equal(t, "public", cfg.Schema)
		generatorPort = applicationPort
		return nil
	}, func(_ context.Context, got *bootstrap.Env) (bootstrap.Application, error) {
		dsn, err := bootstrap.DatabaseDSN(got)
		require.NoError(t, err)
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		applicationPort = u.Port()
		return bootstrap.Application{Db: db}, nil
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	require.Equal(t, "55439", generatorPort)
	require.Equal(t, applicationPort, generatorPort)
	require.Equal(t, 1, calls)
}
func TestStartupJetStopsBeforeRoutesOnFailure(t *testing.T) {
	secret := errors.New("password=private-source")
	db := testsql.Open(t, &testsql.Behavior{Columns: []string{"ready"}, Rows: [][]driver.Value{{true}}})
	env := &bootstrap.Env{DBSchema: "public"}
	for _, appFails := range []bool{false, true} {
		calls := 0
		server, err := setupEngine(context.Background(), env, func(*sql.DB, jetgen.Config) error { calls++; return secret }, func(context.Context, *bootstrap.Env) (bootstrap.Application, error) {
			if appFails {
				return bootstrap.Application{}, secret
			}
			return bootstrap.Application{Db: db}, nil
		})
		require.Error(t, err)
		require.Nil(t, server)
		require.NotContains(t, err.Error(), "private-source")
		if appFails {
			require.Zero(t, calls)
		} else {
			require.Equal(t, 1, calls)
		}
	}
}

func TestStartupJetRejectsDifferentCompiledSchema(t *testing.T) {
	called := false
	engine, err := setupEngine(context.Background(), &bootstrap.Env{DBSchema: "different"}, func(*sql.DB, jetgen.Config) error { called = true; return nil }, func(context.Context, *bootstrap.Env) (bootstrap.Application, error) {
		called = true
		return bootstrap.Application{}, nil
	})
	require.Error(t, err)
	require.Nil(t, engine)
	require.False(t, called)
}

func TestCleanupWorkerStopsBeforeDatabase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	released := make(chan struct{})
	go func() { <-ctx.Done(); close(released); close(done) }()
	require.NoError(t, stopCleanupBeforeDatabase(cancel, done, func() error {
		select {
		case <-released:
		default:
			t.Fatal("database closed before worker joined")
		}
		return nil
	}))
}

func TestStartupSchemaFailurePreventsGenerationAndRoutes(t *testing.T) {
	db := testsql.Open(t, &testsql.Behavior{QueryErr: errors.New("schema unavailable")})
	env := &bootstrap.Env{DBSchema: "public"}
	env.UserPmcs = bootstrap.UserPmcsConfig(userpmcsshared.DefaultConfig())
	generated := false
	engine, err := setupEngine(context.Background(), env, func(*sql.DB, jetgen.Config) error { generated = true; return nil }, func(context.Context, *bootstrap.Env) (bootstrap.Application, error) {
		return bootstrap.Application{Db: db}, nil
	})
	require.Error(t, err)
	require.Nil(t, engine)
	require.False(t, generated)
}
