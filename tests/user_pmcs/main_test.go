package user_pmcs_test

import (
	"database/sql"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	_ = loadEnv()

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
