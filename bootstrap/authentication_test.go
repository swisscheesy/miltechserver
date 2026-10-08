package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Losing either initialization error would permit startup without identity.
func TestFirebaseInitializationFailure(t *testing.T) {
	for _, test := range []struct{ name, contents string }{
		{name: "missing credential file"},
		{name: "malformed credential file", contents: "invalid json"},
		{name: "auth client creation failure", contents: `{"type":"service_account","project_id":"unit-test-project","client_email":"unit@example.test","private_key":"invalid key"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "credentials.json")
			if test.contents != "" {
				require.NoError(t, os.WriteFile(path, []byte(test.contents), 0600))
			}
			t.Setenv("FIREBASE_AUTH_KEY", path)
			if test.name == "auth client creation failure" {
				app, err := NewFirebaseApp(context.Background())
				require.NoError(t, err)
				require.NotNil(t, app)
			}
			client, err := NewFireAuth(context.Background())
			require.Error(t, err)
			require.Nil(t, client)
			// Nil Env proves no SQL/Azure initialization is attempted after auth fails.
			app, err := App(context.Background(), nil)
			require.Error(t, err)
			require.Equal(t, Application{}, app)
		})
	}
}

func TestFirebaseMissingMountedCredentialsDoesNotExposePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-mount-location.json")
	t.Setenv("FIREBASE_AUTH_KEY", path)
	app, err := NewFirebaseApp(context.Background())
	require.Error(t, err)
	require.Nil(t, app)
	require.NotContains(t, err.Error(), path)
}
