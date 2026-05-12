package sb_700_20_test

import (
	"database/sql"
	"log"
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

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}

	exitCode := m.Run()
	_ = testDB.Close()
	os.Exit(exitCode)
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
