package capabilities

import (
	"context"
	"database/sql"
	"log/slog"
	"miltechserver/internal/jetgen"
	"time"
)

// AtomicReady uses the caller's role and one bounded read-only snapshot. It is
// independent of advertisement, and never called from pure atomic services.
func AtomicReady(ctx context.Context, db *sql.DB) bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		slog.Warn("atomic readiness unavailable", "category", "begin")
		return false
	}
	defer tx.Rollback()
	if err = jetgen.ValidateRequiredShopsSchema(ctx, tx); err != nil {
		slog.Warn("atomic readiness unavailable", "category", "required_schema_or_access")
		return false
	}
	if !AtomicSchemaReady(ctx, tx) {
		slog.Warn("atomic readiness unavailable", "category", "atomic_schema_or_access")
		return false
	}
	if err = tx.Commit(); err != nil {
		slog.Warn("atomic readiness unavailable", "category", "commit")
		return false
	}
	return true
}

// AtomicSchemaReady checks the atomic-specific contract in the caller's snapshot.
// The caller owns the deadline and transaction; AtomicReady also checks the
// mandatory Shops schema before this query. Omitted columns of INSERT targets
// must be nullable/defaulted/generated/identity; explicitly written columns must
// accept values. Defaults/expressions are structurally supported, not executed
// or certified here: actual-role write verification remains a release gate.
func AtomicSchemaReady(ctx context.Context, tx *sql.Tx) bool {
	var ready bool
	return tx.QueryRowContext(ctx, atomicReadinessQuery).Scan(&ready) == nil && ready
}

const atomicReadinessQuery = `WITH required_columns(relation,name,kind,not_null) AS (VALUES
('shops','id','text',true),
('shops','details','text',false),
('shops','created_by','text',true),
('shops','admin_only_lists','bool',true),
('shop_members','id','text',true),
('shop_members','shop_id','text',true),
('shop_members','user_id','text',true),
('shop_members','role','text',true),
('shop_vehicle','id','text',true),
('shop_vehicle','creator_id','text',true),
('shop_vehicle','niin','text',true),
('shop_vehicle','admin','text',true),
('shop_vehicle','model','text',true),
('shop_vehicle','serial','text',true),
('shop_vehicle','uoc','text',true),
('shop_vehicle','mileage','int4',true),
('shop_vehicle','hours','int4',true),
('shop_vehicle','comment','text',true),
('shop_vehicle','save_time','timestamptz',true),
('shop_vehicle','last_updated','timestamptz',true),
('shop_vehicle','shop_id','text',true),
('shop_vehicle','tracked_mileage','int4',false),
('shop_vehicle','tracked_hours','int4',false),
('shop_lists','id','text',true),
('shop_lists','shop_id','text',true),
('shop_lists','created_by','text',true),
('shop_lists','description','text',true),
('shop_lists','created_at','timestamptz',true),
('shop_lists','updated_at','timestamptz',true),
('shop_list_items','id','text',true),
('shop_list_items','list_id','text',true),
('shop_list_items','niin','text',true),
('shop_list_items','nomenclature','text',true),
('shop_list_items','quantity','int4',true),
('shop_list_items','added_by','text',true),
('shop_list_items','created_at','timestamptz',true),
('shop_list_items','updated_at','timestamptz',true),
('shop_list_items','nickname','text',false),
('shop_list_items','unit_of_measure','varchar',false),
('shop_vehicle_notifications','id','text',true),
('shop_vehicle_notifications','shop_id','text',true),
('shop_vehicle_notifications','vehicle_id','text',true),
('shop_vehicle_notifications','title','text',true),
('shop_vehicle_notifications','description','text',true),
('shop_vehicle_notifications','type','text',true),
('shop_vehicle_notifications','completed','bool',true),
('shop_vehicle_notifications','save_time','timestamptz',true),
('shop_vehicle_notifications','last_updated','timestamptz',true),
('shop_vehicle_notifications','attached_shop_list','text',false),
('shop_notification_items','id','text',true),
('shop_notification_items','shop_id','text',true),
('shop_notification_items','notification_id','text',true),
('shop_notification_items','niin','text',true),
('shop_notification_items','nomenclature','text',true),
('shop_notification_items','quantity','int4',true),
('shop_notification_items','save_time','timestamptz',true),
('shop_vehicle_notification_changes','id','text',true),
('shop_vehicle_notification_changes','notification_id','text',false),
('shop_vehicle_notification_changes','shop_id','text',true),
('shop_vehicle_notification_changes','vehicle_id','text',false),
('shop_vehicle_notification_changes','changed_by','text',false),
('shop_vehicle_notification_changes','changed_at','timestamptz',true),
('shop_vehicle_notification_changes','change_type','text',true),
('shop_vehicle_notification_changes','field_changes','jsonb',true),
('shop_vehicle_notification_changes','notification_title','text',false),
('shop_vehicle_notification_changes','notification_type','text',false),
('shop_vehicle_notification_changes','vehicle_admin','text',false),
('shop_notification_operations','user_id','text',true),
('shop_notification_operations','operation_id','uuid',true),
('shop_notification_operations','fingerprint','bytea',true),
('shop_notification_operations','notification_id','uuid',true),
('shop_notification_operations','committed_at','timestamptz',true),
('shop_notification_items','nickname','text',false),
('shop_notification_items','unit_of_measure','varchar',false))
SELECT has_schema_privilege(current_user,'public','USAGE')
 AND NOT EXISTS (SELECT 1 FROM required_columns r
 LEFT JOIN pg_class c ON c.oid=to_regclass('public.'||r.relation)
 LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname=r.name AND NOT a.attisdropped
 WHERE a.attname IS NULL OR c.relkind <> 'r' OR c.relrowsecurity OR a.atttypid<>to_regtype(r.kind) OR a.attnotnull<>r.not_null
 OR (r.kind='varchar' AND a.atttypmod<>54)
 OR (r.relation IN ('shop_notification_operations','shop_vehicle_notifications','shop_notification_items','shop_vehicle_notification_changes')
 AND NOT (r.relation='shop_vehicle_notification_changes' AND r.name IN ('id','changed_at'))
 AND (a.attgenerated<>'' OR a.attidentity='a')))
 AND NOT EXISTS (SELECT 1 FROM (VALUES
 ('shop_notification_operations'),('shop_vehicle_notifications'),('shop_notification_items'),('shop_vehicle_notification_changes')
 ) r(relation) JOIN pg_attribute a ON a.attrelid=to_regclass('public.'||r.relation)
 WHERE a.attnum>0 AND NOT a.attisdropped AND a.attnotnull
 AND NOT a.atthasdef AND a.attgenerated='' AND a.attidentity=''
 AND NOT EXISTS (SELECT 1 FROM required_columns known WHERE known.relation=r.relation AND known.name=a.attname
 AND NOT (known.relation='shop_vehicle_notification_changes' AND known.name IN ('id','changed_at'))))
 AND NOT EXISTS (SELECT 1 FROM (VALUES
('shops','SELECT'),
('shops','UPDATE'),
('shop_members','SELECT'),
('shop_members','UPDATE'),
('shop_vehicle','SELECT'),
('shop_vehicle','UPDATE'),
('shop_lists','SELECT'),
('shop_lists','UPDATE'),
('shop_list_items','SELECT'),
('shop_vehicle_notifications','SELECT'),
('shop_vehicle_notifications','INSERT'),
('shop_vehicle_notifications','UPDATE'),
('shop_notification_items','SELECT'),
('shop_notification_items','INSERT'),
('shop_notification_items','UPDATE'),
('shop_notification_items','DELETE'),
('shop_vehicle_notification_changes','INSERT'),
('shop_notification_operations','SELECT'),
('shop_notification_operations','INSERT'),
('shop_notification_operations','UPDATE')) r(relation,privilege)
 WHERE NOT COALESCE(has_table_privilege(current_user,to_regclass('public.'||r.relation),r.privilege),false))
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_notification_operations') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (user_id, operation_id)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_notification_operations') AND contype='f' AND convalidated AND pg_get_constraintdef(oid)='FOREIGN KEY (user_id) REFERENCES users(uid) ON DELETE CASCADE')
 AND NOT EXISTS (SELECT 1 FROM (VALUES
('shop_notification_operations','CHECK ((octet_length(fingerprint) = 32))')) r(relation,definition)
 WHERE NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.'||r.relation)
 AND c.contype='c' AND c.convalidated AND pg_get_constraintdef(c.oid)=r.definition))
`
