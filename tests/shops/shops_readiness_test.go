package shops_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"miltechserver/api/middleware"
	"miltechserver/api/shops"
	"miltechserver/api/shops/capabilities"
	"miltechserver/api/shops/messages"
	"miltechserver/bootstrap"
	"miltechserver/internal/jetgen"
	"miltechserver/tests/testutil"
	"os"
	"testing"
	"time"
)

func TestStartupRequiredShopsSchema(t *testing.T) {
	require.NoError(t, jetgen.ValidateStartupSchema(context.Background(), testDB))
	for _, tc := range []struct {
		name, sql string
		ready     bool
	}{
		{"compatible additive", `ALTER TABLE shop_message_uploads ADD COLUMN future_feature text`, true},
		{"optional receipts absent", `ALTER TABLE shop_notification_operations RENAME TO unavailable_operations`, true},
		{"optional sync absent", `ALTER TABLE shop_message_counters RENAME TO unavailable_counters`, true},
		{"missing asset table", `ALTER TABLE shop_message_uploads RENAME TO unavailable_uploads`, false},
		{"missing resolution version", `ALTER TABLE shop_notification_item_metadata DROP COLUMN resolution_version`, false},
		{"wrong metadata type", `ALTER TABLE shop_notification_item_metadata ALTER COLUMN unit_of_measure TYPE text`, false},
		{"wrong metadata nullability", `ALTER TABLE shop_notification_item_metadata ALTER COLUMN state DROP NOT NULL`, false},
		{"disabled lifecycle hook", `ALTER TABLE shop_message_asset_references DISABLE TRIGGER shop_message_asset_reference_removed`, false},
		{"wrong lifecycle function", `CREATE OR REPLACE FUNCTION public.protect_shop_message_upload_target() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$`, false},
		{"missing cleanup UUID default", `ALTER TABLE shop_message_blob_cleanup_jobs ALTER COLUMN id DROP DEFAULT`, false},
		{"missing resolution guard", `ALTER TABLE shop_notification_item_metadata DROP CONSTRAINT shop_notification_item_metadata_check`, false},
		{"wrong retention key", `ALTER TABLE shop_notification_item_metadata DROP CONSTRAINT shop_notification_item_metadata_pkey`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := testDB.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(tc.sql)
			require.NoError(t, err)
			err = jetgen.ValidateRequiredShopsSchema(context.Background(), tx)
			if tc.ready {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// Each pool is marker-verified before assuming a non-superuser role. A single
// connection keeps SET ROLE local to this test pool, never the shared test DB.
func readinessRoleDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	role := "readiness_" + uuid.NewString()
	quoted := pq.QuoteIdentifier(role)
	_, err := testDB.Exec(`CREATE ROLE ` + quoted + ` NOLOGIN NOSUPERUSER NOBYPASSRLS NOINHERIT`)
	require.NoError(t, err)
	db, err := testutil.OpenDisposableTestDB(os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_DATABASE_MARKER"))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
		_, err := testDB.Exec(`DROP OWNED BY ` + quoted + `; DROP ROLE ` + quoted)
		require.NoError(t, err)
	})
	_, err = testDB.Exec(`GRANT USAGE ON SCHEMA public TO ` + quoted)
	require.NoError(t, err)
	_, err = testDB.Exec(`GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA public TO ` + quoted)
	require.NoError(t, err)
	_, err = db.Exec(`SET ROLE ` + quoted)
	require.NoError(t, err)
	var current string
	var super bool
	require.NoError(t, db.QueryRow(`SELECT current_user,rolsuper FROM pg_roles WHERE rolname=current_user`).Scan(&current, &super))
	require.Equal(t, role, current)
	require.False(t, super)
	_, err = db.Exec(`SET default_transaction_read_only=on`)
	require.NoError(t, err)
	return db, quoted
}

func TestCapabilitiesCurrentRoleReadOnlyProbes(t *testing.T) {
	db, role := readinessRoleDB(t)
	ctx := context.Background()
	require.True(t, capabilities.AtomicReady(ctx, db))
	require.True(t, messages.SyncReadiness(db, true)(ctx))
	require.NoError(t, jetgen.ValidateStartupSchema(ctx, db))
	for _, tc := range []struct {
		name, privilege, table string
		atomic, sync           bool
	}{
		{"receipt update", "UPDATE", "shop_notification_operations", false, true},
		{"mandatory audit insert", "INSERT", "shop_vehicle_notification_changes", false, true},
		{"membership row lock", "UPDATE", "shop_members", false, true},
		{"counter update", "UPDATE", "shop_message_counters", true, false},
		{"shop row lock", "UPDATE", "shops", false, false},
		{"metadata read", "SELECT", "shop_notification_item_metadata", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testDB.Exec(`REVOKE ` + tc.privilege + ` ON ` + pq.QuoteIdentifier(tc.table) + ` FROM ` + role)
			require.NoError(t, err)
			defer func() {
				_, err := testDB.Exec(`GRANT ` + tc.privilege + ` ON ` + pq.QuoteIdentifier(tc.table) + ` TO ` + role)
				require.NoError(t, err)
			}()
			require.Equal(t, tc.atomic, capabilities.AtomicReady(ctx, db))
			require.Equal(t, tc.sync, messages.SyncReadiness(db, true)(ctx))
		})
	}
	t.Log("marker-verified restricted current role; default_transaction_read_only=on; independent grant revocations fail closed")
}

func TestReadinessProbePoolCancellationAndDeadline(t *testing.T) {
	db, _ := readinessRoleDB(t)
	held, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer held.Close()
	for _, tc := range []struct {
		name  string
		probe func(context.Context) bool
	}{
		{"atomic", func(ctx context.Context) bool { return capabilities.AtomicReady(ctx, db) }},
		{"sync", messages.SyncReadiness(db, true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			start := time.Now()
			require.False(t, tc.probe(ctx))
			require.Less(t, time.Since(start), time.Second)
			start = time.Now()
			require.False(t, tc.probe(context.Background()))
			require.Less(t, time.Since(start), 3*time.Second)
		})
	}
}

func TestAtomicHTTPUnavailableForCurrentRole(t *testing.T) {
	_, shop, vehicle := atomicFixture(t, "role-owner")
	db, role := readinessRoleDB(t)
	_, err := testDB.Exec(`REVOKE INSERT ON shop_vehicle_notification_changes FROM ` + role)
	require.NoError(t, err)
	router := gin.New()
	router.Use(middleware.ErrorHandler, testutil.FakeAuthMiddleware())
	shops.RegisterRoutes(shops.Dependencies{DB: db, Env: &bootstrap.Env{ShopsAtomicNotificationSaveEnabled: true}}, router.Group("/api/v1/auth"))
	require.Equal(t, false, capabilityValue(t, router, "atomic_notification_save"))
	requireFailure(t, doContract2JSONRequest(t, router, atomicNotificationRequest(shop, vehicle), "role-owner"), 503, "unsupported_contract")
	require.Equal(t, 0, atomicRowCount(t, "shop_notification_operations", "user_id=$1", "role-owner"))
}

func TestSyncReadinessRequiredSchema(t *testing.T) {
	for _, tc := range []struct{ name, change, restore string }{
		{"missing index", `ALTER INDEX idx_shop_messages_shop_created_id RENAME TO hidden_sync_index`, `ALTER INDEX hidden_sync_index RENAME TO idx_shop_messages_shop_created_id`},
		{"wrong number type", `ALTER TABLE shop_message_counters ALTER COLUMN last_number TYPE numeric`, `ALTER TABLE shop_message_counters ALTER COLUMN last_number TYPE bigint`},
		{"wrong timing", `DROP TRIGGER shop_messages_assign_insertion_number ON shop_messages; CREATE TRIGGER shop_messages_assign_insertion_number AFTER INSERT ON shop_messages FOR EACH ROW EXECUTE FUNCTION assign_shop_message_insertion_number()`, `DROP TRIGGER shop_messages_assign_insertion_number ON shop_messages; CREATE TRIGGER shop_messages_assign_insertion_number BEFORE INSERT ON shop_messages FOR EACH ROW EXECUTE FUNCTION assign_shop_message_insertion_number()`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testDB.Exec(tc.change)
			require.NoError(t, err)
			defer func() { _, err := testDB.Exec(tc.restore); require.NoError(t, err) }()
			require.False(t, messages.SyncReadiness(testDB, true)(context.Background()))
		})
	}
	require.True(t, messages.SyncReadiness(testDB, true)(context.Background()))
}

func TestSyncAllReadsHonorRequestCancellation(t *testing.T) {
	clearShopTables(t, testDB)
	ensureUser(t, testDB, "context-owner")
	shop := createShop(t, newTestRouter(t), "context-owner", "Context")
	db, _ := readinessRoleDB(t)
	repo := messages.NewRepository(db, nil, nil)
	user := &bootstrap.User{UserID: "context-owner"}
	cursor := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"v":2,"shop_id":%q,"created_at":"2026-01-01T00:00:00Z","id":%q}`, shop, uuid.NewString())))
	readers := []struct {
		name string
		read func(context.Context) error
	}{
		{"initial", func(ctx context.Context) error { _, err := repo.InitialMessages(ctx, user, shop, 1); return err }},
		{"history", func(ctx context.Context) error { _, err := repo.MessageHistory(ctx, user, shop, cursor, 1); return err }},
		{"catch-up", func(ctx context.Context) error {
			_, err := repo.CatchUpMessages(ctx, user, shop, "0", nil, 1)
			return err
		}},
		{"empty reconcile", func(ctx context.Context) error {
			_, err := repo.ReconcileMessages(ctx, user, shop, []string{})
			return err
		}},
	}
	for _, stage := range []string{"pool", "integrity query"} {
		t.Run(stage, func(t *testing.T) {
			if stage == "pool" {
				held, err := db.Conn(context.Background())
				require.NoError(t, err)
				defer held.Close()
			} else {
				lock, err := testDB.Begin()
				require.NoError(t, err)
				defer lock.Rollback()
				_, err = lock.Exec(`LOCK TABLE public.shop_message_counters IN ACCESS EXCLUSIVE MODE`)
				require.NoError(t, err)
			}
			for _, reader := range readers {
				t.Run(reader.name, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
					defer cancel()
					start := time.Now()
					err := reader.read(ctx)
					require.Error(t, err)
					require.Less(t, time.Since(start), time.Second)
				})
			}
		})
	}
}

func TestStartupSchemaWriteCompatibility(t *testing.T) {
	clearShopTables(t, testDB)
	for _, tc := range []struct {
		name, change string
		ready        bool
	}{
		{"required extra cleanup column", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_required text NOT NULL`, false},
		{"required extra upload column", `ALTER TABLE shop_message_uploads ADD COLUMN future_required text NOT NULL`, false},
		{"required extra reference column", `ALTER TABLE shop_message_asset_references ADD COLUMN future_required text NOT NULL`, false},
		{"required extra metadata column", `ALTER TABLE shop_notification_item_metadata ADD COLUMN future_required text NOT NULL`, false},
		{"nullable extra cleanup column", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_optional text`, true},
		{"defaulted extra cleanup column", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_default text NOT NULL DEFAULT 'compatible'`, true},
		{"generated extra cleanup column", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_generated text GENERATED ALWAYS AS (scope || '-future') STORED NOT NULL`, true},
		{"identity extra cleanup column", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_identity bigint GENERATED ALWAYS AS IDENTITY`, true},
		{"known generated reason", `ALTER TABLE shop_message_blob_cleanup_jobs DROP COLUMN reason; ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN reason text GENERATED ALWAYS AS ('fixed'::text) STORED NOT NULL`, false},
		{"known by-default metadata version", `ALTER TABLE shop_notification_item_metadata ALTER COLUMN version ADD GENERATED BY DEFAULT AS IDENTITY`, true},
		{"known identity metadata version", `ALTER TABLE shop_notification_item_metadata ALTER COLUMN version ADD GENERATED ALWAYS AS IDENTITY`, false},
		{"column restricted upload protection", `DROP TRIGGER shop_message_upload_target_immutable ON shop_message_uploads; CREATE TRIGGER shop_message_upload_target_immutable BEFORE UPDATE OF failure ON shop_message_uploads FOR EACH ROW EXECUTE FUNCTION protect_shop_message_upload_target()`, false},
		{"column restricted cleanup protection", `DROP TRIGGER shop_message_cleanup_target_immutable ON shop_message_blob_cleanup_jobs; CREATE TRIGGER shop_message_cleanup_target_immutable BEFORE UPDATE OF failure ON shop_message_blob_cleanup_jobs FOR EACH ROW EXECUTE FUNCTION protect_shop_message_cleanup_target()`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := testDB.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(tc.change)
			require.NoError(t, err)
			err = jetgen.ValidateRequiredShopsSchema(context.Background(), tx)
			if tc.ready {
				require.NoError(t, err)
			} else {
				require.Error(t, err, "incompatible compiled write schema must fail before generation")
			}
		})
	}
	require.NoError(t, jetgen.ValidateStartupSchema(context.Background(), testDB))
}

func TestAtomicSchemaWriteCompatibility(t *testing.T) {
	clearShopTables(t, testDB)
	for _, tc := range []struct {
		name, change string
		ready        bool
	}{
		{"required extra receipt column", `ALTER TABLE shop_notification_operations ADD COLUMN future_required text NOT NULL`, false},
		{"required extra notification column", `ALTER TABLE shop_vehicle_notifications ADD COLUMN future_required text NOT NULL`, false},
		{"required extra item column", `ALTER TABLE shop_notification_items ADD COLUMN future_required text NOT NULL`, false},
		{"required extra audit column", `ALTER TABLE shop_vehicle_notification_changes ADD COLUMN future_required text NOT NULL`, false},
		{"nullable extra receipt column", `ALTER TABLE shop_notification_operations ADD COLUMN future_optional text`, true},
		{"defaulted extra receipt column", `ALTER TABLE shop_notification_operations ADD COLUMN future_default text NOT NULL DEFAULT 'compatible'`, true},
		{"generated extra receipt column", `ALTER TABLE shop_notification_operations ADD COLUMN future_generated text GENERATED ALWAYS AS (user_id || '-future') STORED NOT NULL`, true},
		{"identity extra receipt column", `ALTER TABLE shop_notification_operations ADD COLUMN future_identity bigint GENERATED ALWAYS AS IDENTITY`, true},
		{"known generated notification ID", `ALTER TABLE shop_notification_operations DROP COLUMN notification_id; ALTER TABLE shop_notification_operations ADD COLUMN notification_id uuid GENERATED ALWAYS AS ('11111111-1111-4111-8111-111111111111'::uuid) STORED NOT NULL`, false},
		{"known by-default item quantity", `ALTER TABLE shop_notification_items ALTER COLUMN quantity DROP DEFAULT; ALTER TABLE shop_notification_items ALTER COLUMN quantity ADD GENERATED BY DEFAULT AS IDENTITY`, true},
		{"known identity item quantity", `ALTER TABLE shop_notification_items ALTER COLUMN quantity DROP DEFAULT; ALTER TABLE shop_notification_items ALTER COLUMN quantity ADD GENERATED ALWAYS AS IDENTITY`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := testDB.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(tc.change)
			require.NoError(t, err)
			require.Equal(t, tc.ready, capabilities.AtomicSchemaReady(context.Background(), tx), "atomic compiled INSERT contract")
		})
	}
	require.True(t, capabilities.AtomicReady(context.Background(), testDB))
}

func TestStartupSchemaCleanupInsertCompatibility(t *testing.T) {
	clearShopTables(t, testDB)
	for _, tc := range []struct{ name, change, code string }{
		{"required addition", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_required text NOT NULL`, "23502"},
		{"nullable addition", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_optional text`, ""},
		{"defaulted addition", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_default text NOT NULL DEFAULT 'compatible'`, ""},
		{"generated addition", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_generated text GENERATED ALWAYS AS (scope || '-future') STORED NOT NULL`, ""},
		{"identity addition", `ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN future_identity bigint GENERATED ALWAYS AS IDENTITY`, ""},
		{"known generated column", `ALTER TABLE shop_message_blob_cleanup_jobs DROP COLUMN reason; ALTER TABLE shop_message_blob_cleanup_jobs ADD COLUMN reason text GENERATED ALWAYS AS ('fixed'::text) STORED NOT NULL`, "428C9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := testDB.Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(tc.change)
			require.NoError(t, err)
			schemaErr := jetgen.ValidateRequiredShopsSchema(context.Background(), tx)
			// This is the column set of enqueueAsset and the 020 cleanup enqueue hook.
			_, err = tx.Exec(`INSERT INTO shop_message_blob_cleanup_jobs(upload_id,shop_id,scope,account,container,blob_key,reason) VALUES ($1,$2,'asset','account','shop-message-images','key','reference_removed')`, uuid.NewString(), uuid.NewString())
			if tc.code == "" {
				require.NoError(t, err)
				require.NoError(t, schemaErr)
			} else {
				var pgErr *pq.Error
				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, pq.ErrorCode(tc.code), pgErr.Code)
				require.Error(t, schemaErr)
			}
		})
	}
}

func TestStartupSchemaLifecycleTriggerCoverage(t *testing.T) {
	clearShopTables(t, testDB)
	for _, restricted := range []bool{false, true} {
		for _, table := range []string{"shop_message_uploads", "shop_message_blob_cleanup_jobs"} {
			t.Run(fmt.Sprintf("%s/restricted=%t", table, restricted), func(t *testing.T) {
				tx, err := testDB.Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				var trigger, function string
				id := uuid.NewString()
				if table == "shop_message_uploads" {
					trigger = "shop_message_upload_target_immutable"
					function = "protect_shop_message_upload_target"
					_, err = tx.Exec(`INSERT INTO shop_message_uploads(id,operation_id,shop_id,uploader_id,account,container,blob_key,url,extension,state,lease_until) VALUES ($1,$2,'shop','actor','account','shop-message-images','key','url','.jpg','uploading',now())`, id, uuid.NewString())
				} else {
					trigger = "shop_message_cleanup_target_immutable"
					function = "protect_shop_message_cleanup_target"
					_, err = tx.Exec(`INSERT INTO shop_message_blob_cleanup_jobs(id,shop_id,scope,account,container,blob_key,reason) VALUES ($1,'shop','shop_prefix','account','shop-message-images','key','test')`, id)
				}
				require.NoError(t, err)
				if restricted {
					_, err = tx.Exec(`DROP TRIGGER ` + pq.QuoteIdentifier(trigger) + ` ON ` + pq.QuoteIdentifier(table) + `; CREATE TRIGGER ` + pq.QuoteIdentifier(trigger) + ` BEFORE UPDATE OF failure ON ` + pq.QuoteIdentifier(table) + ` FOR EACH ROW EXECUTE FUNCTION ` + pq.QuoteIdentifier(function) + `()`)
					require.NoError(t, err)
				}
				schemaErr := jetgen.ValidateRequiredShopsSchema(context.Background(), tx)
				_, err = tx.Exec(`UPDATE `+pq.QuoteIdentifier(table)+` SET account='bypassed' WHERE id=$1`, id)
				if restricted {
					require.NoError(t, err, "restricted hook really bypasses immutable-target protection")
					require.Error(t, schemaErr)
				} else {
					require.Error(t, err, "original hook protects target")
					require.NoError(t, schemaErr)
				}
			})
		}
	}
}
