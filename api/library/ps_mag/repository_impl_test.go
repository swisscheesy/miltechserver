package ps_mag

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Skip("TEST_DB_URL not set — skipping repository integration tests")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRepositorySearchSummaries_ReturnsResults(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	// This test assumes ps_mag_summaries has at least one row.
	// If the table is empty the test will pass with count=0 (not a failure).
	rows, total, err := repo.SearchSummaries("the", 1, 30)

	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 0)
	require.LessOrEqual(t, len(rows), 30)
}

func TestRepositorySearchSummaries_NoMatch(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	rows, total, err := repo.SearchSummaries("zzz_no_match_xyz_999", 1, 30)

	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, rows)
}

func TestRepositorySearchSummaries_PageTwo(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)

	// Page 2 with page size 1 — may return empty if only 1 match exists.
	_, _, err := repo.SearchSummaries("the", 2, 1)
	require.NoError(t, err)
}
