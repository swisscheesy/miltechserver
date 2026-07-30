package persistence

import (
	"context"
	"database/sql"

	"github.com/stretchr/testify/require"
	"testing"
)

func TestAccountDeletionCleanerImplementsTransactionBoundary(t *testing.T) {
	t.Parallel()

	cleaner := NewAccountCleaner()
	require.NotNil(t, cleaner)

	var contract interface {
		CleanupAccount(context.Context, *sql.Tx, string) error
	} = cleaner
	require.NotNil(t, contract)
}
