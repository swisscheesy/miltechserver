package user_suggestions_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func clearTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE
			user_suggestion_votes,
			user_suggestions
		RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err)
}

func ensureUser(t *testing.T, db *sql.DB, userID string) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO users (uid, email, username, created_at, is_enabled)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uid) DO NOTHING`,
		userID,
		userID+"@example.com",
		"test-user",
		time.Now().UTC(),
		true,
	)
	require.NoError(t, err)
}

// insertSuggestion inserts a row directly into user_suggestions with a specific show value.
// Pass nil for show to insert a NULL.
func insertSuggestion(t *testing.T, db *sql.DB, title, userID string, show *bool) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := db.Exec(
		`INSERT INTO user_suggestions (id, user_id, title, description, status, show)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id,
		userID,
		title,
		"Test description",
		"Submitted",
		show,
	)
	require.NoError(t, err)
	return id
}

func boolPtr(v bool) *bool {
	return &v
}
