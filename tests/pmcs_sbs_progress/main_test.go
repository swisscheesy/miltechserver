package pmcs_sbs_progress_test

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"miltechserver/tests/testutil"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("postgres", testutil.TestDSN)
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
