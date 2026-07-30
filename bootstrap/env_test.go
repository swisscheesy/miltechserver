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
