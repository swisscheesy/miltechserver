package shops_test

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// insertMessageNumber inserts with the exact column list released writers use
// and returns the number the trigger assigned.
func insertMessageNumber(t *testing.T, q queryRower, shopID, userID string) int64 {
	t.Helper()
	var n sql.NullInt64
	require.NoError(t, q.QueryRow(
		`INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at, updated_at, is_edited)
		 VALUES ($1, $2, $3, 'legacy writer', now(), now(), false)
		 RETURNING insertion_number`,
		uuid.NewString(), shopID, userID).Scan(&n))
	require.True(t, n.Valid, "trigger must assign insertion_number")
	return n.Int64
}

func TestMessageSyncSchemaObjects(t *testing.T) {
	var column string
	require.NoError(t, testDB.QueryRow(`
		SELECT data_type || ':' || is_nullable FROM information_schema.columns
		WHERE table_schema='public' AND table_name='shop_messages' AND column_name='insertion_number'`).Scan(&column))
	require.Equal(t, "bigint:YES", column)

	var counterFK string
	require.NoError(t, testDB.QueryRow(`
		SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid='public.shop_message_counters'::regclass AND contype='f'`).Scan(&counterFK))
	require.Equal(t, "FOREIGN KEY (shop_id) REFERENCES shops(id) ON DELETE CASCADE", counterFK)

	for _, index := range []string{"shop_messages_shop_insertion_number_key", "idx_shop_messages_shop_created_id"} {
		var exists bool
		require.NoError(t, testDB.QueryRow(`SELECT to_regclass('public.' || $1) IS NOT NULL`, index).Scan(&exists))
		require.True(t, exists, index)
	}

	var enabled string
	require.NoError(t, testDB.QueryRow(`
		SELECT tgenabled::text FROM pg_trigger
		WHERE tgrelid='public.shop_messages'::regclass AND tgname='shop_messages_assign_insertion_number'
		  AND (tgtype & 2) = 2 AND (tgtype & 4) = 4`).Scan(&enabled)) // BEFORE + INSERT
	require.Equal(t, "O", enabled)
}

func TestMessageSyncLegacyInsertsAreNumberedPerShop(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shopA := createShop(t, router, "user-1", "Shop A")
	shopB := createShop(t, router, "user-1", "Shop B")

	require.Equal(t, int64(1), insertMessageNumber(t, testDB, shopA, "user-1"))
	require.Equal(t, int64(2), insertMessageNumber(t, testDB, shopA, "user-1"))
	require.Equal(t, int64(1), insertMessageNumber(t, testDB, shopB, "user-1"))
	require.Equal(t, int64(3), insertMessageNumber(t, testDB, shopA, "user-1"))

	// A writer that tries to choose its own number must not win.
	var n int64
	require.NoError(t, testDB.QueryRow(
		`INSERT INTO public.shop_messages (id, shop_id, user_id, message, insertion_number)
		 VALUES ($1, $2, 'user-1', 'forged', 999) RETURNING insertion_number`,
		uuid.NewString(), shopA).Scan(&n))
	require.Equal(t, int64(4), n)

	var last int64
	require.NoError(t, testDB.QueryRow(`SELECT last_number FROM public.shop_message_counters WHERE shop_id=$1`, shopA).Scan(&last))
	require.Equal(t, int64(4), last)
}

func TestMessageSyncCounterIsDeletedWithShop(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Doomed")
	insertMessageNumber(t, testDB, shopID, "user-1")
	_, err := testDB.Exec(`DELETE FROM public.shops WHERE id=$1`, shopID)
	require.NoError(t, err)
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM public.shop_message_counters WHERE shop_id=$1`, shopID).Scan(&count))
	require.Zero(t, count)
}
