package shops_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const sharedShopTablesLockID int64 = 70020

var testDB *sql.DB

func TestMain(m *testing.M) {
	_ = loadEnv()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		log.Fatal("TEST_DATABASE_URL is not set")
	}

	var err error
	testDB, err = sql.Open("postgres", "postgres://postgres:potato123@192.168.20.70/miltech_ng_test?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to open test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
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
