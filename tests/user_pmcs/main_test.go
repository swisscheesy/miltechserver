package user_pmcs_test

import (
	"database/sql"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	if err := loadEnv(); err != nil {
		log.Fatalf("failed to load test environment: %v", err)
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		log.Fatal("TEST_DATABASE_URL is not set")
	}
	dsn = disableSSLWhenUnspecified(dsn)

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}
	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}

	exitCode := m.Run()

	if err := testDB.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}
	os.Exit(exitCode)
}

func disableSSLWhenUnspecified(dsn string) string {
	databaseURL, err := url.Parse(dsn)
	if err != nil {
		log.Fatalf("failed to parse TEST_DATABASE_URL: %v", err)
	}

	query := databaseURL.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
		databaseURL.RawQuery = query.Encode()
	}
	return databaseURL.String()
}

func loadEnv() error {
	if os.Getenv("TEST_DATABASE_URL") != "" {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	current := wd
	for {
		envPath := filepath.Join(current, ".env")
		if _, statErr := os.Stat(envPath); statErr == nil {
			return godotenv.Load(envPath)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func TestMainReportsLoadEnvError(t *testing.T) {
	const helperEnvironmentVariable = "USER_PMCS_TEST_MAIN_LOAD_ENV_HELPER"
	if os.Getenv(helperEnvironmentVariable) == "1" {
		return
	}

	testDirectory, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	envPath := filepath.Join(testDirectory, ".env")
	require.NoError(t, os.Mkdir(envPath, 0o700))

	expectedLoadError := godotenv.Load(envPath)
	require.Error(t, expectedLoadError)

	executable, err := os.Executable()
	require.NoError(t, err)

	command := exec.Command(executable, "-test.run=^TestMainReportsLoadEnvError$")
	command.Dir = testDirectory
	for _, environmentVariable := range os.Environ() {
		if strings.HasPrefix(environmentVariable, "TEST_DATABASE_URL=") ||
			strings.HasPrefix(environmentVariable, helperEnvironmentVariable+"=") {
			continue
		}
		command.Env = append(command.Env, environmentVariable)
	}
	command.Env = append(command.Env, helperEnvironmentVariable+"=1")

	output, err := command.CombinedOutput()
	require.Error(t, err)
	require.Contains(
		t,
		string(output),
		"failed to load test environment: "+expectedLoadError.Error(),
	)
}
