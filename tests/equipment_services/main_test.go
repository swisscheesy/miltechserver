package equipment_services_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"miltechserver/tests/testutil"
)

const sharedShopTablesLockID int64 = 70020

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}

	unlock := lockSharedShopTables(testDB)
	exitCode := m.Run()
	unlock()

	if err := testDB.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}

	os.Exit(exitCode)
}

func lockSharedShopTables(db *sql.DB) func() {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Fatalf("failed to reserve shared shop table lock connection: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, sharedShopTablesLockID); err != nil {
		_ = conn.Close()
		log.Fatalf("failed to lock shared shop tables: %v", err)
	}
	return func() {
		if _, err := conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, sharedShopTablesLockID); err != nil {
			log.Printf("failed to unlock shared shop tables: %v", err)
		}
		if err := conn.Close(); err != nil {
			log.Printf("failed to close shared shop table lock connection: %v", err)
		}
	}
}
