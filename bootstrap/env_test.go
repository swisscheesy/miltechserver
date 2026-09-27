package bootstrap

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEnvRejectsMalformedOrOverflowingUserPmcsValues(t *testing.T) {
	if os.Getenv("BOOTSTRAP_NEW_ENV_HELPER") == "1" {
		NewEnv()
		return
	}

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "malformed owned checklist limit", key: "USER_PMCS_MAX_OWNED_CHECKLISTS", value: "not-a-number"},
		{name: "overflowing mutation body limit", key: "USER_PMCS_MAX_MUTATION_BODY_BYTES", value: "999999999999999999999999999999"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestNewEnvRejectsMalformedOrOverflowingUserPmcsValues$")
			command.Env = environmentWithout(os.Environ(), "BOOTSTRAP_NEW_ENV_HELPER", "DEBUG", test.key)
			command.Env = append(command.Env, "BOOTSTRAP_NEW_ENV_HELPER=1", "DEBUG=false", test.key+"="+test.value)

			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), test.key)
		})
	}
}

func TestShopsAtomicNotificationSaveEnvironment(t *testing.T) {
	const key = "SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED"
	tests := []struct {
		name    string
		value   string
		present bool
		want    bool
		wantErr bool
	}{
		{name: "absent"},
		{name: "false", value: "false", present: true},
		{name: "true", value: "true", present: true, want: true},
		{name: "empty", present: true, wantErr: true},
		{name: "malformed", value: "TRUE", present: true, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(key, test.value)
			if !test.present {
				require.NoError(t, os.Unsetenv(key))
			}
			got, err := shopsAtomicNotificationSaveEnabledFromEnvironment()
			if test.wantErr {
				require.ErrorContains(t, err, key)
				require.False(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestNewEnvLoadsShopsAtomicNotificationSaveFlag(t *testing.T) {
	t.Setenv("DEBUG", "false")
	t.Setenv("SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED", "true")
	require.True(t, NewEnv().ShopsAtomicNotificationSaveEnabled)
	t.Setenv("SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED", "false")
	require.False(t, NewEnv().ShopsAtomicNotificationSaveEnabled)
}

func TestNewEnvRejectsInvalidShopsAtomicNotificationSaveFlag(t *testing.T) {
	if os.Getenv("BOOTSTRAP_ATOMIC_FLAG_HELPER") == "1" {
		NewEnv()
		return
	}
	for _, value := range []string{"", "TRUE", "1"} {
		t.Run("value="+value, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestNewEnvRejectsInvalidShopsAtomicNotificationSaveFlag$")
			command.Env = environmentWithout(os.Environ(), "BOOTSTRAP_ATOMIC_FLAG_HELPER", "DEBUG", "SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED")
			command.Env = append(command.Env, "BOOTSTRAP_ATOMIC_FLAG_HELPER=1", "DEBUG=false", "SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED="+value)
			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "SHOPS_ATOMIC_NOTIFICATION_SAVE_ENABLED")
		})
	}
}

func environmentWithout(environment []string, keys ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, value := range environment {
		key, _, _ := strings.Cut(value, "=")
		if contains(keys, key) {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
