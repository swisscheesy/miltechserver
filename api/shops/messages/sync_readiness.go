package messages

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// SyncReadiness checks the supported 018 schema plus the 019 allocator bridge
// under the current role. Data integrity is checked separately inside every
// per-Shop read snapshot; fleet/write/restore gates remain operational.
func SyncReadiness(db *sql.DB, enabled bool) func(context.Context) bool {
	return func(ctx context.Context) bool {
		if !enabled || db == nil {
			return false
		}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		var ready bool
		if err := db.QueryRowContext(ctx, syncReadinessQuery).Scan(&ready); err != nil {
			slog.Warn("message sync readiness unavailable", "category", "probe")
			return false
		}
		if !ready {
			slog.Warn("message sync readiness unavailable", "category", "schema_or_access")
		}
		return ready
	}
}

const syncReadinessQuery = `WITH required_columns(relation,name,kind,not_null) AS (VALUES
 ('shop_messages','id','text',true),('shop_messages','shop_id','text',true),('shop_messages','user_id','text',true),
 ('shop_messages','message','text',true),('shop_messages','created_at','timestamptz',false),('shop_messages','updated_at','timestamptz',false),
 ('shop_messages','is_edited','bool',false),('shop_messages','parent_id','text',false),('shop_messages','insertion_number','int8',false),
 ('shop_message_counters','shop_id','text',true),('shop_message_counters','last_number','int8',true)
 ) SELECT has_schema_privilege(current_user,'public','USAGE')
 AND NOT EXISTS (SELECT 1 FROM required_columns r
 LEFT JOIN pg_class c ON c.oid=to_regclass('public.'||r.relation)
 LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname=r.name AND NOT a.attisdropped
 WHERE a.attname IS NULL OR c.relkind <> 'r' OR c.relrowsecurity OR a.atttypid<>to_regtype(r.kind)
 OR (r.not_null AND NOT a.attnotnull) OR a.atttypmod NOT IN (-1,6))
 AND EXISTS (SELECT 1 FROM pg_trigger t JOIN pg_proc p ON p.oid=t.tgfoid JOIN pg_language l ON l.oid=p.prolang
 WHERE t.tgrelid=to_regclass('public.shop_messages') AND t.tgname='shop_messages_assign_insertion_number'
 AND t.tgfoid=to_regprocedure('public.assign_shop_message_insertion_number()')
 AND t.tgtype=7 AND t.tgenabled='O' AND NOT t.tgisinternal AND t.tgnargs=0 AND t.tgqual IS NULL
 AND p.prorettype='trigger'::regtype AND l.lanname='plpgsql' AND NOT p.prosecdef AND p.proconfig IS NULL
 AND p.provolatile='v' AND md5(p.prosrc)='42f7105c5298267751a040de0fe30012'
 AND has_function_privilege(current_user,p.oid,'EXECUTE'))
 AND EXISTS (SELECT 1 FROM pg_index WHERE indexrelid=to_regclass('public.shop_messages_shop_insertion_number_key')
 AND indrelid=to_regclass('public.shop_messages') AND indisunique AND indisvalid AND indisready
 AND indpred IS NULL AND indexprs IS NULL AND indnkeyatts=2
 AND pg_get_indexdef(indexrelid,1,true)='shop_id' AND pg_get_indexdef(indexrelid,2,true)='insertion_number')
 AND EXISTS (SELECT 1 FROM pg_index WHERE indexrelid=to_regclass('public.idx_shop_messages_shop_created_id')
 AND indrelid=to_regclass('public.shop_messages') AND indisvalid AND indisready AND indpred IS NULL AND indexprs IS NULL AND indnkeyatts=3
 AND pg_get_indexdef(indexrelid,1,true)='shop_id' AND pg_get_indexdef(indexrelid,2,true)='created_at' AND pg_get_indexdef(indexrelid,3,true)='id'
 AND indoption[0]=0 AND indoption[1]=3 AND indoption[2]=3)
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_counters') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (shop_id)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_counters') AND contype='f' AND convalidated AND pg_get_constraintdef(oid)='FOREIGN KEY (shop_id) REFERENCES shops(id) ON DELETE CASCADE')
 AND NOT EXISTS (SELECT 1 FROM (VALUES
 ('shop_messages','SELECT'),('shop_members','SELECT'),('users','SELECT'),
 ('shops','SELECT'),('shops','UPDATE'),('shop_message_counters','SELECT'),('shop_message_counters','INSERT'),('shop_message_counters','UPDATE')
 ) r(relation,privilege) LEFT JOIN pg_class c ON c.oid=to_regclass('public.'||r.relation)
 WHERE c.oid IS NULL OR c.relrowsecurity OR NOT has_table_privilege(current_user,c.oid,r.privilege))`
