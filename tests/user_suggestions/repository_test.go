package user_suggestions_test

import (
	"testing"

	user_suggestions "miltechserver/api/user_suggestions"

	"github.com/stretchr/testify/require"
)

// TestGetAllWithScores_ShowFilter verifies that the repository only returns
// suggestions where show IS TRUE, excluding show=false and show=NULL rows.
func TestGetAllWithScores_ShowFilter(t *testing.T) {
	clearTables(t, testDB)

	userID := "sugg-show-filter"
	ensureUser(t, testDB, userID)

	insertSuggestion(t, testDB, "Visible Feature", userID, boolPtr(true))
	insertSuggestion(t, testDB, "Hidden Feature", userID, boolPtr(false))
	insertSuggestion(t, testDB, "Unset Feature", userID, nil)

	repo := user_suggestions.NewRepository(testDB)
	results, err := repo.GetAllWithScores("")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Visible Feature", results[0].Title)
}

// TestGetAllWithScores_EmptyWhenAllHidden verifies an empty result when every
// suggestion is hidden, rather than an error.
func TestGetAllWithScores_EmptyWhenAllHidden(t *testing.T) {
	clearTables(t, testDB)

	userID := "sugg-all-hidden"
	ensureUser(t, testDB, userID)

	insertSuggestion(t, testDB, "Hidden A", userID, boolPtr(false))
	insertSuggestion(t, testDB, "Hidden B", userID, boolPtr(false))

	repo := user_suggestions.NewRepository(testDB)
	results, err := repo.GetAllWithScores("")
	require.NoError(t, err)
	require.Empty(t, results)
}

// TestGetAllWithScores_MultipleVisible verifies all show=true rows are returned.
func TestGetAllWithScores_MultipleVisible(t *testing.T) {
	clearTables(t, testDB)

	userID := "sugg-multi-visible"
	ensureUser(t, testDB, userID)

	insertSuggestion(t, testDB, "Feature One", userID, boolPtr(true))
	insertSuggestion(t, testDB, "Feature Two", userID, boolPtr(true))
	insertSuggestion(t, testDB, "Feature Hidden", userID, boolPtr(false))

	repo := user_suggestions.NewRepository(testDB)
	results, err := repo.GetAllWithScores("")
	require.NoError(t, err)
	require.Len(t, results, 2)
}
