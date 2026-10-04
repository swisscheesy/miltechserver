package shops_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"miltechserver/api/shops/messages"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestSyncCommonIntegrityGuard(t *testing.T) {
	for _, corrupt := range []string{"null number", "null timestamp", "missing counter", "behind counter"} {
		t.Run(corrupt, func(t *testing.T) {
			clearShopTables(t, testDB)
			ensureUser(t, testDB, "integrity-owner")
			router := newTestRouterWithFlags(t, false, true)
			shop := createShop(t, router, "integrity-owner", "Integrity")
			old := insertMessageAt(t, shop, "integrity-owner", "old", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
			insertMessageAt(t, shop, "integrity-owner", "middle", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
			newest := insertMessageAt(t, shop, "integrity-owner", "new", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
			initial := decodeSyncPage(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shop, "initial", url.Values{"limit": {"1"}}), nil, "integrity-owner"))
			var err error
			switch corrupt {
			case "null number":
				_, err = testDB.Exec(`UPDATE shop_messages SET insertion_number=NULL WHERE id=$1`, old)
			case "null timestamp":
				_, err = testDB.Exec(`UPDATE shop_messages SET created_at=NULL WHERE id=$1`, old)
			case "missing counter":
				_, err = testDB.Exec(`DELETE FROM shop_message_counters WHERE shop_id=$1`, shop)
			case "behind counter":
				_, err = testDB.Exec(`UPDATE shop_message_counters SET last_number=1 WHERE shop_id=$1`, shop)
			}
			require.NoError(t, err)
			for _, op := range []string{"initial", "history", "catch-up", "reconcile", "empty reconcile"} {
				t.Run(op, func(t *testing.T) {
					method := http.MethodGet
					query := url.Values{"limit": {"1"}}
					var body any
					operation := op
					switch op {
					case "history":
						query.Set("cursor", *initial.OlderCursor)
					case "catch-up":
						query.Set("after", "2")
					case "reconcile":
						method = http.MethodPost
						query = nil
						body = map[string]any{"ids": []string{newest}}
					case "empty reconcile":
						operation = "reconcile"
						method = http.MethodPost
						query = nil
						body = map[string]any{"ids": []string{}}
					}
					requireFailure(t, doContractRequest(t, router, method, messageSyncPath(shop, operation, query), body, "integrity-owner"), 503, "unsupported_contract")
				})
			}
		})
	}
}

func TestSyncWatermarkResetRequired(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "reset-owner")
	router := newTestRouterWithFlags(t, false, true)
	shop := createShop(t, router, "reset-owner", "Reset")
	first := createMessage(t, router, "reset-owner", shop, "first")
	last := createMessage(t, router, "reset-owner", shop, "last")
	_, err := testDB.Exec(`DELETE FROM shop_messages WHERE id=$1`, last)
	require.NoError(t, err)
	page := catchUp(t, router, shop, "0", nil, 100, "reset-owner")
	require.Equal(t, "2", page.Through)
	require.Equal(t, []string{first}, rowIDs(page.Rows))
	for _, q := range []url.Values{{"after": {"3"}}, {"after": {"1"}, "through": {"3"}}} {
		requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shop, "catch-up", q), nil, "reset-owner"), 409, "message_sync_reset_required")
	}
	requireFailure(t, doContractRequest(t, router, http.MethodGet, messageSyncPath(shop, "catch-up", url.Values{"after": {"3"}, "through": {"1"}}), nil, "reset-owner"), 400, "invalid")
}

func TestCapabilitiesFlagAndReadiness(t *testing.T) {
	router, shop, vehicle := atomicFixture(t, "readiness-owner")
	require.Equal(t, false, capabilityValue(t, router, "atomic_notification_save"))
	enabled := newTestRouterWithFlags(t, true, true)
	require.Equal(t, true, capabilityValue(t, enabled, "atomic_notification_save"))
	_, err := testDB.Exec(`ALTER TABLE shop_notification_operations RENAME TO unavailable_operations`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := testDB.Exec(`ALTER TABLE unavailable_operations RENAME TO shop_notification_operations`)
		require.NoError(t, err)
	})
	require.Equal(t, false, capabilityValue(t, enabled, "atomic_notification_save"))
	requireFailure(t, doContract2JSONRequest(t, router, atomicNotificationRequest(shop, vehicle), "readiness-owner"), 503, "unsupported_contract")
}

func TestSyncReadinessRejectsWrongAllocatorDefinition(t *testing.T) {
	tx, err := testDB.Begin()
	require.NoError(t, err)
	var original string
	require.NoError(t, tx.QueryRow(`SELECT pg_get_functiondef('public.assign_shop_message_insertion_number()'::regprocedure)`).Scan(&original))
	require.NoError(t, tx.Rollback())
	_, err = testDB.Exec(`CREATE OR REPLACE FUNCTION public.assign_shop_message_insertion_number() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$`)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := testDB.Exec(original); require.NoError(t, err) })
	require.False(t, messages.SyncReadiness(testDB, true)(context.Background()))
}
