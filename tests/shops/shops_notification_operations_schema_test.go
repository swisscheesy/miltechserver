package shops_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationOperationsSchema(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "user-1")
	ensureUser(t, testDB, "user-2")

	var tableExists bool
	require.NoError(t, testDB.QueryRow(`SELECT to_regclass('public.shop_notification_operations') IS NOT NULL`).Scan(&tableExists))
	require.True(t, tableExists, "notification operation ledger must exist")

	var primaryKeyColumns string
	require.NoError(t, testDB.QueryRow(`
		SELECT string_agg(attribute.attname, ',' ORDER BY key.ordinality)
		FROM pg_catalog.pg_constraint AS c
		JOIN pg_catalog.pg_class AS relation ON relation.oid = c.conrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		JOIN unnest(c.conkey) WITH ORDINALITY AS key(attnum, ordinality) ON TRUE
		JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = relation.oid AND attribute.attnum = key.attnum
		WHERE namespace.nspname = 'public' AND relation.relname = 'shop_notification_operations'
		  AND c.contype = 'p'
		GROUP BY c.oid`).Scan(&primaryKeyColumns))
	require.Equal(t, "user_id,operation_id", primaryKeyColumns)

	var foreignKeyDefinition string
	require.NoError(t, testDB.QueryRow(`
		SELECT pg_catalog.pg_get_constraintdef(c.oid)
		FROM pg_catalog.pg_constraint AS c
		WHERE c.conrelid = 'public.shop_notification_operations'::regclass
		  AND c.contype = 'f'`).Scan(&foreignKeyDefinition))
	require.Equal(t, "FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE", foreignKeyDefinition)

	router := newTestRouter(t)
	shopID := createShop(t, router, "user-1", "Ledger Shop")
	vehicleID := createVehicle(t, router, "user-1", shopID)
	notificationID := createNotification(t, router, "user-1", shopID, vehicleID, "Ledger notification")
	operationID := "00000000-0000-4000-8000-000000000001"
	fingerprint := bytes.Repeat([]byte{0x1}, 32)
	insert := `INSERT INTO public.shop_notification_operations
		(user_id, operation_id, fingerprint, notification_id, committed_at)
		VALUES ($1, $2, $3, $4, now())`
	_, err := testDB.Exec(insert, "user-1", operationID, fingerprint, notificationID)
	require.NoError(t, err)
	_, err = testDB.Exec(insert, "user-1", operationID, fingerprint, notificationID)
	require.Error(t, err, "same user and operation must be unique")
	_, err = testDB.Exec(insert, "user-2", operationID, fingerprint, notificationID)
	require.NoError(t, err, "a second user may reuse an operation ID")
	for _, length := range []int{31, 33} {
		_, err = testDB.Exec(insert, "user-1", "00000000-0000-4000-8000-000000000003", bytes.Repeat([]byte{0x2}, length), notificationID)
		require.Error(t, err, "fingerprints must contain exactly 32 bytes")
	}

	_, err = testDB.Exec(`DELETE FROM public.shops WHERE id = $1`, shopID)
	require.NoError(t, err)
	var count int
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM public.shop_notification_operations WHERE operation_id = $1`, operationID).Scan(&count))
	require.Equal(t, 2, count, "shop and notification deletion must retain receipts")
	_, err = testDB.Exec(`DELETE FROM public.users WHERE uid = $1`, "user-1")
	require.NoError(t, err)
	require.NoError(t, testDB.QueryRow(`SELECT count(*) FROM public.shop_notification_operations WHERE operation_id = $1`, operationID).Scan(&count))
	require.Equal(t, 1, count, "account deletion must cascade only its own receipts")
}
