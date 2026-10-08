package bootstrap

import (
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestDatabaseDSN(t *testing.T) {
	env := &Env{Host: "127.0.0.1", Port: "55439", Username: "app role", Password: "a \\ ' @/?secret", DBName: "miltech_test_fixture", DBSchema: "public", SslMode: "require"}
	dsn, err := DatabaseDSN(env)
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "55439", parsed.Port())
	require.Equal(t, "app role", parsed.User.Username())
	password, _ := parsed.User.Password()
	require.Equal(t, env.Password, password)
	require.Equal(t, "require", parsed.Query().Get("sslmode"))
	require.Equal(t, "public", parsed.Query().Get("search_path"))
	for _, port := range []string{"", "0", "65536", "bad"} {
		env.Port = port
		_, err = DatabaseDSN(env)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestDatabaseDSNRejectsInvalidSchema(t *testing.T) {
	env := &Env{Host: "127.0.0.1", Port: "5432", Username: "app", Password: "private", DBName: "miltech_test_fixture", DBSchema: "public,private", SslMode: "disable"}
	_, err := DatabaseDSN(env)
	require.Error(t, err)
}
