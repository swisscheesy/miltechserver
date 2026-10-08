#!/usr/bin/env bash
set -euo pipefail
set +x
# No dotenv loading, inherited DSN, target discovery, or automatic retries.
exec python3 - "${BASH_SOURCE[0]}" "$@" <<'PYTHON'
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import subprocess
import sys

# Reviewed, target-specific, per-stage records only. Pinning requires the owner
# gates in docs/testing/shops-server-remediation-release.md; never infer live pins.
TARGET_PINS = {"miltech_ng_test": "UNPINNED", "miltech_ng": "UNPINNED"}
SOURCE_PINS = {'015_add_shop_vehicle_usage_constraints.sql': '6d65e52cae06395cf0bc92757f4d664843b22f8471d2529c7c49f5cbec7d6a93', '015_rollback_shop_vehicle_usage_constraints.sql': 'd2d33f05a848285d02a5a77679ebc6927a7b5cca6a37e884dcd42a6cecfc3595', '016_create_shop_notification_operations.sql': '32b7a07f383aa1c6be028df865871bb0034e8df42d6f8e8474bff3aa139087de', '016_rollback_shop_notification_operations.sql': 'a7aca23e2309eb8d8247abafe4c33691adb79f467d036e3a9265218628dbb5fa', '017_add_shop_notification_item_nickname_uom.sql': '9fd74ee58d083a3e5a223d4a155ac3115d0d15986745425e0de6c8c78a6b7c94', '017_rollback_shop_notification_item_nickname_uom.sql': '6e2b1de535b200a5295f999552ca1e7da0cd1d54fa1afdbf4b964eba8210193e', '018_add_shop_message_insertion_numbers.sql': 'b0e8809dca1d8e109cc3072c13c9033f38f40bbe4a086bffd8322f57617777d7', '018_rollback_shop_message_insertion_numbers.sql': '775a51afebf059dbc5d7a733a47fdd4fc598a03a6100443e9e31a60077371ecf', '019_fix_shop_message_allocator_lock_order.sql': 'd50416fc7294e2eff5a98ec98039dbc18dc32968963925fe3ef9261c3d9847ca', '019_rollback_fix_shop_message_allocator_lock_order.sql': '0653a07f5d87d44be8d2cd7c55555a4cb93b01da8afdfe6fe9fc1e5d30c99a8f', '020_create_shop_message_asset_lifecycle.sql': 'ab18c2b9a92c4d09ea1235e002bb551bf81a0952d0ed7c4c725780b4832f903a', '020_rollback_shop_message_asset_lifecycle.sql': 'd8003e69aae701e8df15b0ee4ab9a7aa2b7c9ebcf2f2a91b96e35bf7039a1f6a', '021_enforce_shop_message_parent_ownership.sql': '5c3d6f46679ab6760d6e0f4d100c14813357da2886fb041e72fc64550fc86319', '021_rollback_enforce_shop_message_parent_ownership.sql': 'f55205688d9f6f9251b11c2c617237461b60e16ffcecb55901f7b6516bbcf238', '022_add_shop_vehicle_base_usage_constraints.sql': 'eba4513f600297b67ff72fc75352fa146ab85f150777d0d2b1af70e5afc252f8', '022_rollback_shop_vehicle_base_usage_constraints.sql': '9ab681091e2606690339127e0cf1b13236d15bf7123911cc601387fda2cc2a59', '023_create_shop_notification_item_metadata.sql': '9ca02f397517396c9e221ac761de477686fc86519728a2df298c70485c5bfe31', '023_rollback_shop_notification_item_metadata.sql': '9702cb44ba7da8a5356fb3a3e6d21f786654a36d4d8b45f8341664eddb1fdfa3', '024_register_legacy_shop_message_images.sql': 'c0c97ca90509bf957fd26ff6f6f8abda248f424ffdacb01358dac11923298242', '024_rollback_register_legacy_shop_message_images.sql': '6496f26d186acabaa4e38604f1c067f89f558e8c15fbd3c533cb9796af001e8d'}
CATALOG_SQL = "SELECT COALESCE(string_agg(record,E'\\n' ORDER BY record),'') FROM (\n SELECT 'column:'||c.relname||':'||a.attnum||':'||a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||a.attnotnull||':'||COALESCE(pg_get_expr(d.adbin,d.adrelid),'') record FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_attribute a ON a.attrelid=c.oid LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum WHERE n.nspname='public' AND a.attnum>0 AND NOT a.attisdropped\n UNION ALL SELECT 'relation:'||c.relname||':'||c.relkind||':'||CASE WHEN c.relkind IN ('v','m') THEN pg_get_viewdef(c.oid) ELSE '' END FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'\n UNION ALL SELECT 'constraint:'||c.conname||':'||pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace WHERE n.nspname='public'\n UNION ALL SELECT 'index:'||indexname||':'||indexdef FROM pg_indexes WHERE schemaname='public'\n UNION ALL SELECT 'function:'||p.proname||':'||pg_get_functiondef(p.oid) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public' AND p.prokind IN ('f','p')\n UNION ALL SELECT 'trigger:'||c.relname||':'||t.tgenabled||':'||pg_get_triggerdef(t.oid) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND NOT t.tgisinternal\n UNION ALL SELECT 'enum:'||t.typname||':'||e.enumsortorder||':'||e.enumlabel FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace JOIN pg_enum e ON e.enumtypid=t.oid WHERE n.nspname='public'\n UNION ALL SELECT 'sequence:'||sequencename||':'||data_type||':'||start_value||':'||min_value||':'||max_value||':'||increment_by||':'||cycle FROM pg_sequences WHERE schemaname='public'\n ) catalog"

# Full IDs are hashed in-session and never emitted. Owner inventory remains in
# access-controlled release evidence; count-only approvals are insufficient.
DATA_SQL = r"""SELECT jsonb_build_object(
 'orphan_shops', (SELECT coalesce(jsonb_agg(s.id ORDER BY s.id),'[]') FROM public.shops s WHERE NOT EXISTS(SELECT 1 FROM public.shop_members m WHERE m.shop_id=s.id)),
 'adminless_shops', (SELECT coalesce(jsonb_agg(s.id ORDER BY s.id),'[]') FROM public.shops s WHERE NOT EXISTS(SELECT 1 FROM public.shop_members m WHERE m.shop_id=s.id AND m.role='admin')),
 'foreign_parents', (SELECT coalesce(jsonb_agg(m.id ORDER BY m.id),'[]') FROM public.shop_messages m LEFT JOIN public.shop_messages p ON p.id=m.parent_id WHERE m.parent_id IS NOT NULL AND (p.id IS NULL OR p.shop_id<>m.shop_id)),
 'negative_bases', (SELECT coalesce(jsonb_agg(v.id ORDER BY v.id),'[]') FROM public.shop_vehicle v WHERE v.mileage<0 OR v.hours<0),
 'ambiguous_metadata', (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.notification_id,x.niin),'[]') FROM (SELECT notification_id,niin FROM public.shop_notification_items GROUP BY notification_id,niin HAVING count(DISTINCT jsonb_build_array(nickname,unit_of_measure))>1) x),
 'parent_actions', (SELECT coalesce(jsonb_agg(jsonb_build_array(c.conname,pg_get_constraintdef(c.oid)) ORDER BY c.conname),'[]') FROM pg_constraint c WHERE c.conrelid='public.shop_messages'::regclass AND c.contype='f'),
 'protected_historical_assets', (SELECT coalesce(jsonb_agg(jsonb_build_array(m.id,m.message) ORDER BY m.id),'[]') FROM public.shop_messages m WHERE m.message LIKE '%shop-message-images%')
)::text"""


def refuse(reason):
    raise SystemExit('Refused: '+reason)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def secure_read(path):
    if not path.is_file() or any(p.is_symlink() for p in (path,*path.parents)):
        refuse('missing or redirected input')
    return path.read_bytes()


def guard(expression, reason):
    return "SELECT COALESCE(("+expression+"),false) AS approved \\gset\n\\if :approved\n\\else\n\\echo 'Refused: "+reason+"'\nSELECT 1/0;\n\\endif\n"


def verify_generation(path, pin, target, stage):
    raw=secure_read(path)
    if digest(raw)!=pin['generation_evidence_sha256']:
        refuse('preceding tagged-generation evidence checksum mismatch')
    proof=json.loads(raw)
    manifest=proof['generation_manifest']
    expected_stage=str(int(stage)-1).zfill(3)
    if proof['stage']!=expected_stage or proof['source_sha256']!=pin['source_sha256']:
        refuse('preceding generation stage or source mismatch')
    if (manifest['database'],str(ipaddress.ip_interface(manifest['address']).ip),manifest['port'],manifest['user'],manifest['schema'],manifest['catalog_sha256']) != (target,pin['address'],pin['port'],pin['application_role'],'public',pin['catalog_sha256']):
        refuse('preceding generation target/schema identity mismatch')
    before,after=proof['pmcs_before'],proof['pmcs_after']
    if len(before)!=32 or before!=after or digest(json.dumps(before,sort_keys=True).encode())!=pin['pmcs_sha256']:
        refuse('32 PMCS baseline hashes mismatch')
    if any(not re.fullmatch(r'[0-9a-f]{64}',v) for v in before.values()):
        refuse('invalid PMCS hash evidence')
    if proof['generated_packages_compile']!='PASS':
        refuse('preceding generated package compilation not proven')


def run():
    disposable=len(sys.argv)>2 and sys.argv[2]=='--disposable-rehearsal'
    if disposable: del sys.argv[2]
    if len(sys.argv)!=10:
        refuse('Usage: runner TARGET forward-NNN AUTHORIZATION ADDRESS PORT MIGRATION_ROLE CATALOG_SHA256 PRECEDING_GENERATION_JSON')
    script,target,action,authorization,address,port,role,catalog,proof_path=sys.argv[1:]
    if disposable:
        if not re.fullmatch(r'miltech_test_[a-z0-9_]+',target) or address not in ('127.0.0.1','::1') or not re.fullmatch(r'[0-9a-f]{64}',os.environ.get('SHOPS_RELEASE_TEST_MARKER','')):
            refuse('disposable mode requires loopback, miltech_test_* and protected wrapper marker')
    elif target not in ('miltech_ng_test','miltech_ng'):
        refuse('named mode refuses disposable/unknown target')
    if target not in TARGET_PINS:
        refuse('target is not allowlisted')
    if not re.fullmatch(r'forward-02[0-4]|forward-019',action):
        refuse('only one forward 019-024 action is supported; reverse needs separate owner resolution authorization')
    stage=action[-3:]
    records=TARGET_PINS[target]
    if records=='UNPINNED' or records.get(stage,'UNPINNED')=='UNPINNED':
        refuse('target/stage is UNPINNED')
    pin=records[stage]
    required=('address','port','role','version','application_role','service','catalog_sha256','data_sha256','authorization','generation_evidence_sha256','source_sha256','pmcs_sha256','probe_shop_id','probe_user_id','test_target_evidence')
    if any(k not in pin or pin[k] in ('','UNPINNED',None) for k in required):
        refuse('incomplete reviewed target/stage pin')
    for key in ('catalog_sha256','data_sha256','generation_evidence_sha256','source_sha256','pmcs_sha256'):
        if not re.fullmatch(r'[0-9a-f]{64}',pin[key]): refuse('malformed reviewed checksum')
    if (authorization,address,port,role,catalog)!=(pin['authorization'],pin['address'],str(pin['port']),pin['role'],pin['catalog_sha256']):
        refuse('arguments differ from approved target/action authorization or identity')
    if os.environ.get('PGSERVICE')!=pin['service'] or os.environ.get('DB_USERNAME')!=pin['application_role']:
        refuse('PGSERVICE or actual DB_USERNAME differs from reviewed identity')
    # Reject even empty selectors. A service file/password file is provisioned by
    # the operator, never printed or copied; all other libpq settings are refused.
    permitted={'PGSERVICE','PGSERVICEFILE','PGPASSFILE'}
    if any(k.startswith('PG') and k not in permitted for k in os.environ) or any(k in os.environ for k in ('JET_DSN','DATABASE_URL','DB_DSN')):
        refuse('unset inherited connection selectors/options/DSNs')
    if not re.fullmatch(r'[A-Za-z_][A-Za-z0-9_.-]*',pin['service']): refuse('invalid service name')
    verify_generation(Path(proof_path).absolute(),pin,target,stage)
    root=Path(script).resolve().parents[1]
    sql_by_name={}
    for name,checksum in SOURCE_PINS.items():
        raw=secure_read(root/'migrations'/name)
        if digest(raw)!=checksum: refuse('migration source differs from reviewed 015-024 pins')
        sql_by_name[name]=raw.decode()
    name=next(n for n in sql_by_name if n.startswith(stage+'_') and '_rollback_' not in n)
    migration=sql_by_name[name]
    # Keep the reviewed SQL body verbatim in the guarded transaction, replacing
    # only its single outer transaction envelope. Never accept arbitrary SQL.
    if len(re.findall(r'^BEGIN;$',migration,re.MULTILINE))!=1 or not migration.endswith('COMMIT;\n'):
        refuse('unexpected migration transaction envelope')
    migration=re.sub(r'^BEGIN;\n','',migration,count=1,flags=re.MULTILINE).removesuffix('COMMIT;\n')
    sql="BEGIN;\nSET LOCAL search_path=pg_catalog,public;\nSET LOCAL client_min_messages=warning;\n"
    identity="current_database()=:'database' AND host(inet_server_addr())=:'address' AND inet_server_port()=:'port'::integer AND current_user=:'role' AND current_setting('server_version_num')=:'version'"
    sql+=guard(identity,'connection identity mismatch')
    if disposable:
        sql+=guard("(SELECT marker=:'marker' FROM test_infrastructure.disposable_instance WHERE singleton=true)",'disposable marker mismatch')
    sql+="SELECT pg_backend_pid() AS identity_session_id \\gset\nSET LOCAL lock_timeout='5s';\n"
    # Fencing every writer remains an owner gate; locks stabilize the local
    # preflight rows/catalog used by these actions until the migration commits.
    sql+="LOCK TABLE public.shops,public.shop_members,public.shop_messages,public.shop_message_counters,public.shop_vehicle,public.shop_notification_items IN SHARE ROW EXCLUSIVE MODE;\n"
    sql+="SELECT encode(sha256(convert_to(("+CATALOG_SQL+"),'UTF8')),'hex') AS catalog_sha256 \\gset\n"
    sql+=guard(":'catalog_sha256'=:'catalog'",'catalog checksum mismatch')
    sql+="SELECT ("+DATA_SQL+") AS data_manifest \\gset\n"
    sql+=guard("encode(sha256(convert_to(:'data_manifest','UTF8')),'hex')=:'data'",'data manifest checksum mismatch')
    for field in ('orphan_shops','adminless_shops','foreign_parents','negative_bases','ambiguous_metadata'):
        sql+=guard("jsonb_array_length(:'data_manifest'::jsonb->'"+field+"')=0",'unresolved '+field)
    sql+=legacy_proof()
    sql+="SELECT pg_backend_pid() AS migration_session_id \\gset\n"
    sql+=guard(":'identity_session_id'=:'migration_session_id'",'session changed')
    sql+='-- BEGIN REVIEWED MIGRATION '+name+'\n'+migration
    sql+=legacy_proof()
    sql+="COMMIT;\nSELECT 'SQL_APPLIED_GENERATION_PENDING|'||:'identity_session_id'||'|'||pg_backend_pid()::text;\n"
    args=['psql','-X','-w','-q','-A','-t','-v','ON_ERROR_STOP=1']
    values={'database':target,'address':address,'port':port,'role':role,'version':str(pin['version']),'catalog':catalog,'data':pin['data_sha256'],'app_role':pin['application_role'],'probe_shop':pin['probe_shop_id'],'probe_user':pin['probe_user_id']}
    if disposable: values['marker']=os.environ['SHOPS_RELEASE_TEST_MARKER']
    for key,value in values.items(): args+=['-v',key+'='+value]
    result=subprocess.run(args,input=sql,text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    if result.returncode:
        refuse('migration session failed; inspect status with owner before retry; no automatic retry or raw SQL diagnostics')
    receipt=re.search(r'^SQL_APPLIED_GENERATION_PENDING\|([0-9]+)\|([0-9]+)$',result.stdout,re.MULTILINE)
    if not receipt or receipt[1]!=receipt[2]:
        refuse('SQL commit may have occurred but session receipt unavailable; owner inspection required; do not retry')
    print('identity_session_id='+receipt[1]+' migration_session_id='+receipt[2])
    print('SQL applied: '+target+' '+action+'. Generation is PENDING; do not retry this action.')
    print('STOP: tagged generation from the verified intended target is required now, with generated-package compilation and unchanged 32 PMCS hashes. Next action requires reviewed checkpoint pins. Final 024 also requires this checkpoint; this is not release approval.')


def legacy_proof():
    sql='SAVEPOINT application_role_proof;\nSET LOCAL ROLE :"app_role";\n'
    sql+=guard("current_user=:'app_role' AND has_schema_privilege(current_user,'public','USAGE')",'actual application role unavailable')
    for table,privileges in {'shops':('SELECT','UPDATE'),'shop_messages':('SELECT','INSERT'),'shop_message_counters':('SELECT','INSERT','UPDATE'),'shop_notification_operations':('SELECT','INSERT','UPDATE')}.items():
        for privilege in privileges:
            sql+=guard("has_table_privilege(current_user,'public."+table+"','"+privilege+"')",'required application privilege unavailable')
    sql+="INSERT INTO public.shop_messages(id,shop_id,user_id,message) VALUES(gen_random_uuid()::text,:'probe_shop',:'probe_user','release legacy write proof') RETURNING insertion_number > 0 AS numbered \\gset\n"
    sql+=guard(":'numbered'::boolean",'legacy insert was not numbered')
    sql+='ROLLBACK TO SAVEPOINT application_role_proof;\nRELEASE SAVEPOINT application_role_proof;\n'
    return sql


try:
    run()
except (KeyError,ValueError,TypeError,OSError,StopIteration):
    refuse('invalid/missing reviewed evidence or unavailable tool; no database/configuration details emitted')
PYTHON
