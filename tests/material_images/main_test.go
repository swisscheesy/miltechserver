package material_images_test

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"miltechserver/tests/testutil"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}

	exitCode := m.Run()

	if err := testDB.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}

	os.Exit(exitCode)
}
