package jetgen

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ValidateStartupSchema checks the mandatory 020/023 contracts of this binary
// before generation publishes models or the server registers routes. Generic
// Generate deliberately remains usable at earlier migration rehearsal stages.
func ValidateStartupSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("required startup schema unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return errors.New("required startup schema unavailable")
	}
	defer tx.Rollback()
	if err = ValidateRequiredShopsSchema(ctx, tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return errors.New("required startup schema unavailable")
	}
	return nil
}

// ValidateRequiredShopsSchema also serves atomic readiness inside its single
// bounded, read-only current-role snapshot. Compatible additions are allowed;
// required columns, lifecycle functions and ownership keys must retain meaning.
// Additional columns must be omittable by the compiled INSERTs: nullable or
// supplied by a default, generation expression or identity. Known fields remain
// writable (BY DEFAULT identity accepts explicit values; ALWAYS does not).
// This structural check does not certify arbitrary default/expression functions;
// deployment still requires real writes under the actual application role.
func ValidateRequiredShopsSchema(ctx context.Context, tx *sql.Tx) error {
	var ready bool
	if err := tx.QueryRowContext(ctx, requiredShopsSchemaQuery).Scan(&ready); err != nil || !ready {
		return errors.New("required Shops schema or access unavailable")
	}
	return nil
}

// Function hashes pin only the supported 020 bodies and execution properties,
// not the entire schema fingerprint. They must change with a reviewed migration.
const requiredShopsSchemaQuery = `WITH required_columns(relation, name, kind, not_null) AS (VALUES
('shop_message_uploads','id','uuid',true),
('shop_message_uploads','operation_id','uuid',true),
('shop_message_uploads','shop_id','text',true),
('shop_message_uploads','uploader_id','text',true),
('shop_message_uploads','account','text',true),
('shop_message_uploads','container','text',true),
('shop_message_uploads','blob_key','text',true),
('shop_message_uploads','url','text',true),
('shop_message_uploads','extension','text',true),
('shop_message_uploads','state','text',true),
('shop_message_uploads','lease_until','timestamp with time zone',true),
('shop_message_uploads','created_at','timestamp with time zone',true),
('shop_message_uploads','updated_at','timestamp with time zone',true),
('shop_message_uploads','failure','text',true),
('shop_message_asset_references','message_id','text',true),
('shop_message_asset_references','upload_id','uuid',true),
('shop_message_asset_references','shop_id','text',true),
('shop_message_blob_cleanup_jobs','id','uuid',true),
('shop_message_blob_cleanup_jobs','upload_id','uuid',false),
('shop_message_blob_cleanup_jobs','shop_id','text',true),
('shop_message_blob_cleanup_jobs','scope','text',true),
('shop_message_blob_cleanup_jobs','account','text',true),
('shop_message_blob_cleanup_jobs','container','text',true),
('shop_message_blob_cleanup_jobs','blob_key','text',true),
('shop_message_blob_cleanup_jobs','reason','text',true),
('shop_message_blob_cleanup_jobs','state','text',true),
('shop_message_blob_cleanup_jobs','attempts','integer',true),
('shop_message_blob_cleanup_jobs','next_attempt','timestamp with time zone',true),
('shop_message_blob_cleanup_jobs','lease_owner','text',true),
('shop_message_blob_cleanup_jobs','lease_until','timestamp with time zone',false),
('shop_message_blob_cleanup_jobs','failure','text',true),
('shop_message_blob_cleanup_jobs','cursor','text',true),
('shop_message_blob_cleanup_jobs','created_at','timestamp with time zone',true),
('shop_message_blob_cleanup_jobs','updated_at','timestamp with time zone',true),
('shop_notification_item_metadata','notification_id','text',true),
('shop_notification_item_metadata','niin','text',true),
('shop_notification_item_metadata','nickname','text',false),
('shop_notification_item_metadata','unit_of_measure','character varying(50)',false),
('shop_notification_item_metadata','state','text',true),
('shop_notification_item_metadata','candidates','jsonb',true),
('shop_notification_item_metadata','version','bigint',true),
('shop_notification_item_metadata','resolution_version','bigint',true)), required_functions(name, source_hash) AS (VALUES
('protect_shop_message_upload_target','b202f12292eb210772a6387861c2e773'),
('protect_shop_message_cleanup_target','56e73f8148d08bc29f7fcc0202f20c48'),
('enqueue_shop_message_asset_candidate','2f9c50c17bf6b59b67bb505b03d3a414'))
SELECT has_schema_privilege(current_user,'public','USAGE')
 AND NOT EXISTS (
 SELECT 1 FROM required_columns r
 LEFT JOIN pg_class c ON c.oid=to_regclass('public.'||r.relation)
 LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname=r.name AND NOT a.attisdropped
 WHERE a.attname IS NULL OR c.relkind <> 'r' OR c.relrowsecurity
 OR format_type(a.atttypid,a.atttypmod) <> r.kind OR a.attnotnull <> r.not_null
 OR a.attgenerated <> '' OR a.attidentity = 'a')
 AND NOT EXISTS (
 SELECT 1 FROM (SELECT DISTINCT relation FROM required_columns) r
 JOIN pg_attribute a ON a.attrelid=to_regclass('public.'||r.relation)
 WHERE a.attnum>0 AND NOT a.attisdropped AND a.attnotnull
 AND NOT a.atthasdef AND a.attgenerated='' AND a.attidentity=''
 AND NOT EXISTS (SELECT 1 FROM required_columns known WHERE known.relation=r.relation AND known.name=a.attname))
 AND NOT EXISTS (
 SELECT 1 FROM required_functions r LEFT JOIN pg_proc p ON p.oid=to_regprocedure('public.'||r.name||'()')
 LEFT JOIN pg_language l ON l.oid=p.prolang
 WHERE p.oid IS NULL OR p.prorettype <> 'trigger'::regtype OR l.lanname <> 'plpgsql'
 OR p.prosecdef OR p.proconfig IS NOT NULL OR p.provolatile <> 'v'
 OR md5(p.prosrc) <> r.source_hash OR NOT has_function_privilege(current_user,p.oid,'EXECUTE'))
 AND NOT EXISTS (
 SELECT 1 FROM (VALUES
 ('shop_message_uploads','shop_message_upload_target_immutable','protect_shop_message_upload_target',19),
 ('shop_message_blob_cleanup_jobs','shop_message_cleanup_target_immutable','protect_shop_message_cleanup_target',19),
 ('shop_message_asset_references','shop_message_asset_reference_removed','enqueue_shop_message_asset_candidate',9)
 ) r(relation,name,function_name,kind)
 LEFT JOIN pg_trigger t ON t.tgrelid=to_regclass('public.'||r.relation) AND t.tgname=r.name
 WHERE t.oid IS NULL OR t.tgfoid <> to_regprocedure('public.'||r.function_name||'()')
 OR t.tgtype <> r.kind OR t.tgenabled <> 'O' OR t.tgisinternal OR t.tgnargs <> 0 OR t.tgqual IS NOT NULL
 OR cardinality(t.tgattr::smallint[]) <> 0)
 AND NOT EXISTS (
 SELECT 1 FROM (VALUES
 ('shop_message_uploads','SELECT'),('shop_message_uploads','INSERT'),('shop_message_uploads','UPDATE'),
 ('shop_message_asset_references','SELECT'),('shop_message_asset_references','INSERT'),('shop_message_asset_references','DELETE'),
 ('shop_message_blob_cleanup_jobs','SELECT'),('shop_message_blob_cleanup_jobs','INSERT'),('shop_message_blob_cleanup_jobs','UPDATE'),
 ('shop_notification_item_metadata','SELECT'),('shop_notification_item_metadata','INSERT'),('shop_notification_item_metadata','UPDATE'),('shop_notification_item_metadata','DELETE')
 ) r(relation,privilege)
 WHERE NOT COALESCE(has_table_privilege(current_user,to_regclass('public.'||r.relation),r.privilege),false))
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_notification_item_metadata') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (notification_id, niin)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_notification_item_metadata') AND contype='f' AND convalidated AND confrelid=to_regclass('public.shop_vehicle_notifications') AND confdeltype='c' AND pg_get_constraintdef(oid)='FOREIGN KEY (notification_id) REFERENCES shop_vehicle_notifications(id) ON DELETE CASCADE')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_asset_references') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (message_id, upload_id)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_asset_references') AND contype='f' AND convalidated AND pg_get_constraintdef(oid)='FOREIGN KEY (message_id, shop_id) REFERENCES shop_messages(id, shop_id) ON DELETE CASCADE')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_asset_references') AND contype='f' AND convalidated AND pg_get_constraintdef(oid)='FOREIGN KEY (upload_id, shop_id) REFERENCES shop_message_uploads(id, shop_id) ON DELETE RESTRICT')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_uploads') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (id)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_uploads') AND contype='u' AND convalidated AND pg_get_constraintdef(oid)='UNIQUE (operation_id)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_uploads') AND contype='u' AND convalidated AND pg_get_constraintdef(oid)='UNIQUE (account, container, blob_key)')
 AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.shop_message_blob_cleanup_jobs') AND contype='p' AND convalidated AND pg_get_constraintdef(oid)='PRIMARY KEY (id)')
 AND COALESCE(has_function_privilege(current_user,to_regprocedure('gen_random_uuid()'),'EXECUTE'),false)

 AND NOT EXISTS (SELECT 1 FROM (VALUES
('shop_message_blob_cleanup_jobs','CHECK ((((scope = ''asset''::text) AND (upload_id IS NOT NULL)) OR ((scope = ''shop_prefix''::text) AND (upload_id IS NULL))))'),
('shop_message_blob_cleanup_jobs','CHECK (((account <> ''''::text) AND (blob_key <> ''''::text)))'),
('shop_message_blob_cleanup_jobs','CHECK ((attempts >= 0))'),
('shop_message_blob_cleanup_jobs','CHECK ((container = ''shop-message-images''::text))'),
('shop_message_blob_cleanup_jobs','CHECK ((scope = ANY (ARRAY[''asset''::text, ''shop_prefix''::text])))'),
('shop_message_blob_cleanup_jobs','CHECK ((state = ANY (ARRAY[''pending''::text, ''leased''::text, ''retryable''::text, ''completed''::text, ''manual-review''::text])))'),
('shop_message_uploads','CHECK ((account <> ''''::text))'),
('shop_message_uploads','CHECK ((blob_key <> ''''::text))'),
('shop_message_uploads','CHECK ((container = ''shop-message-images''::text))'),
('shop_message_uploads','CHECK ((extension = ANY (ARRAY[''.jpg''::text, ''.png''::text, ''.gif''::text, ''.webp''::text])))'),
('shop_message_uploads','CHECK ((state = ANY (ARRAY[''uploading''::text, ''ready''::text, ''cleanup_pending''::text, ''deleting''::text, ''deleted''::text])))'),
('shop_notification_item_metadata','CHECK (((resolution_version >= 0) AND (resolution_version <= version)))'),
('shop_notification_item_metadata','CHECK ((jsonb_typeof(candidates) = ''array''::text))'),
('shop_notification_item_metadata','CHECK ((state = ANY (ARRAY[''resolved''::text, ''ambiguous''::text])))'),
('shop_notification_item_metadata','CHECK ((version > 0))')) r(relation,definition)
 WHERE NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.'||r.relation)
 AND c.contype='c' AND c.convalidated AND pg_get_constraintdef(c.oid)=r.definition))

 AND NOT EXISTS (SELECT 1 FROM (VALUES
('shop_message_uploads','created_at','now()'),
('shop_message_uploads','updated_at','now()'),
('shop_message_uploads','failure','''''::text'),
('shop_message_blob_cleanup_jobs','id','gen_random_uuid()'),
('shop_message_blob_cleanup_jobs','state','''pending''::text'),
('shop_message_blob_cleanup_jobs','attempts','0'),
('shop_message_blob_cleanup_jobs','next_attempt','now()'),
('shop_message_blob_cleanup_jobs','lease_owner','''''::text'),
('shop_message_blob_cleanup_jobs','failure','''''::text'),
('shop_message_blob_cleanup_jobs','cursor','''''::text'),
('shop_message_blob_cleanup_jobs','created_at','now()'),
('shop_message_blob_cleanup_jobs','updated_at','now()'),
('shop_notification_item_metadata','resolution_version','0')) r(relation,name,definition)
 LEFT JOIN pg_attribute a ON a.attrelid=to_regclass('public.'||r.relation) AND a.attname=r.name AND NOT a.attisdropped
 LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
 WHERE d.oid IS NULL OR pg_get_expr(d.adbin,d.adrelid)<>r.definition)
`
