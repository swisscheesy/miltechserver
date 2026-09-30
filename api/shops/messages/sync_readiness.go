package messages

import (
	"context"
	"database/sql"
	"log/slog"
)

// SyncReadiness reports whether message_sync may be advertised: the operator
// flag is on and the physical schema has the counter table plus an enabled
// BEFORE INSERT allocator trigger. It cannot prove backfill or that every
// serving instance is upgraded; those remain release-runbook gates.
func SyncReadiness(db *sql.DB, enabled bool) func(context.Context) bool {
	return func(ctx context.Context) bool {
		if !enabled {
			return false
		}
		var ready bool
		// The 'public.shop_messages'::regclass cast errors when the table is
		// missing, which lands in the error branch and fails closed.
		err := db.QueryRowContext(ctx, `
			SELECT to_regclass('public.shop_message_counters') IS NOT NULL
			   AND EXISTS (
			     SELECT 1 FROM pg_trigger
			     WHERE tgrelid = 'public.shop_messages'::regclass
			       AND tgname = 'shop_messages_assign_insertion_number'
			       AND tgenabled = 'O' AND NOT tgisinternal)`).Scan(&ready)
		if err != nil {
			slog.Error("message sync readiness probe failed", "error", err)
			return false
		}
		return ready
	}
}
