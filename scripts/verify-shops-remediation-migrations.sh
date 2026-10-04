#!/usr/bin/env bash
# Sourced by the disposable wrapper; no DSN, target selector or SQL argument API.
if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
  printf '%s\n' 'Run this helper through scripts/test-shops-isolated.sh --verify-remediation-migrations; wrapper context required.' >&2
  exit 1
fi

collect_remediation_migrations() {
  remediation_migrations=()
  local pair forward reverse missing=false
  for pair in \
    '019_fix_shop_message_allocator_lock_order.sql|019_rollback_fix_shop_message_allocator_lock_order.sql' \
    '020_create_shop_message_asset_lifecycle.sql|020_rollback_shop_message_asset_lifecycle.sql' \
    '021_enforce_shop_message_parent_ownership.sql|021_rollback_enforce_shop_message_parent_ownership.sql' \
    '022_add_shop_vehicle_base_usage_constraints.sql|022_rollback_shop_vehicle_base_usage_constraints.sql' \
    '023_create_shop_notification_item_metadata.sql|023_rollback_shop_notification_item_metadata.sql'; do
    forward=${pair%%|*}; reverse=${pair#*|}
    if [[ ! -e "$root/migrations/$forward" && ! -L "$root/migrations/$forward" && ! -e "$root/migrations/$reverse" && ! -L "$root/migrations/$reverse" ]]; then
      missing=true
      continue
    fi
    [[ -f "$root/migrations/$forward" && -f "$root/migrations/$reverse" && ! -L "$root/migrations/$forward" && ! -L "$root/migrations/$reverse" ]] || fail 'Remediation migration pair is incomplete or redirected.'
    [[ "$missing" == false ]] || fail 'Remediation migration pairs must form an ordered prefix starting at 019.'
    remediation_migrations+=("migrations/$forward")
  done
}

verify_disposable_identity() {
  local database=$1 stored
  [[ "$database" == miltech_test_* && "$marker" =~ ^[0-9a-f]{64}$ ]] || fail 'Disposable database name or marker is invalid.'
  stored=$(psql_local "$database" -Atc "SELECT current_database() || '|' || current_user || '|' || marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE" 2>"$instance/setup.log") || fail 'Disposable identity/marker verification failed.'
  [[ "$stored" == "$database|postgres|$marker" ]] || fail 'Disposable identity/marker verification failed.'
}

prepare_generation_workspace() {
  workspace="$instance/source"
  # Use current candidate files, including intended uncommitted additions. Never
  # copy generated output, dotenv/configuration, Git metadata or scratch caches.
  python3 - "$root" "$workspace" "$instance" <<'PYTHON'
import hashlib, json, pathlib, shutil, sys
root, output, instance = map(pathlib.Path, sys.argv[1:])
sys.path.insert(0, str(root / "scripts"))
from shops_candidate_inputs import collect_candidate_inputs
output.mkdir()
inputs, pmcs = collect_candidate_inputs(root)
for name in inputs:
    destination = output / name
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(root / name, destination)
(instance / "source-manifest.json").write_text(json.dumps({
    "inputs": inputs,
    "fixture_provenance": {
        "lookup_lin_niin_mat": "Documented nsn/army_lin_to_niin SQL; disposable supplement only",
        "tmde_interval_mat": "Synthetic empty compile ABI; baseline types plus NULL::text item_name WITH NO DATA; no target/runtime certification",
    },
}, sort_keys=True))
(instance / "pmcs-manifest.json").write_text(json.dumps(pmcs, sort_keys=True))
print("Candidate source SHA-256: " + hashlib.sha256(json.dumps(inputs, sort_keys=True).encode()).hexdigest())
PYTHON
  (cd "$workspace" && build_go build -o "$instance/jetregen" ./tools/jetregen) || fail 'Isolated tagged generator build failed.'
}

build_go() {
  env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
    -u JET_DSN -u JET_OUTPUT_DIR -u JET_SCHEMA \
    GOWORK=off GOFLAGS= GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go "$@"
}

regenerate_stage() {
  local database=$1 stage=$2 tcp_identity
  verify_disposable_identity "$database"
  # Independently bind the generator's TCP endpoint to the marked private socket
  # database, so a port collision cannot direct generation at another cluster.
  tcp_identity=$(psql -X -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "$port" -U postgres -d "$database" -Atc "SELECT current_database() || '|' || current_user || '|' || marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE" 2>"$instance/setup.log") || fail 'Disposable TCP identity/marker verification failed.'
  [[ "$tcp_identity" == "$database|postgres|$marker" ]] || fail 'Disposable TCP identity/marker verification failed.'
  # A fresh output for every step makes carried-forward models impossible.
  rm -rf "$workspace/.gen"
  JET_DSN="postgres://postgres@127.0.0.1:$port/$database?sslmode=disable&connect_timeout=5" \
    JET_SCHEMA=public JET_OUTPUT_DIR="$workspace/.gen" "$instance/jetregen" >"$instance/generation.log" 2>&1 || fail "Tagged generation failed at $stage."
  python3 - "$workspace" "$instance/pmcs-manifest.json" "$database" "$port" <<'PYTHON'
import hashlib, ipaddress, json, pathlib, sys
workspace, reference = map(pathlib.Path, sys.argv[1:3])
manifest = json.loads((workspace / ".gen/miltech_ng/public/generation-manifest.json").read_text())
if (manifest["database"], str(ipaddress.ip_interface(manifest["address"]).ip), manifest["port"], manifest["user"], manifest["schema"]) != (sys.argv[3], "127.0.0.1", int(sys.argv[4]), "postgres", "public"):
    raise SystemExit("Generated source identity mismatch")
for name, digest in json.loads(reference.read_text()).items():
    generated = workspace / name
    if not generated.is_file() or hashlib.sha256(generated.read_bytes()).hexdigest() != digest:
        raise SystemExit("Tracked user_pmcs generated contract changed: " + name)
print("Verified disposable identity: " + manifest["database"] + " 127.0.0.1:" + str(manifest["port"]) + " postgres; marker matched; catalog=" + manifest["catalog_sha256"])
PYTHON
  (cd "$workspace" && build_go build ./.gen/miltech_ng/public/model ./.gen/miltech_ng/public/table ./.gen/miltech_ng/public/view) || fail "Generated packages failed at $stage."
  printf 'Generated packages PASS: %s / %s\n' "$database" "$stage"
}

apply_remediation_migrations() {
  local database=$1 migration
  # The empty-array expansion is required by macOS Bash 3.2 with nounset.
  for migration in ${remediation_migrations[@]+"${remediation_migrations[@]}"}; do
    verify_disposable_identity "$database"
    psql_local "$database" -f "$workspace/$migration" >"$instance/setup.log" 2>&1 || fail "Remediation migration failed: $migration"
    regenerate_stage "$database" "${migration##*/}"
    printf 'Remediation forward PASS: %s\n' "$migration"
  done
}

verify_final_candidate() {
  if [[ "$verify_remediation_migrations" == true || "${verify_release_runner_only:-false}" == true ]]; then
    verify_release_runner
  fi
  if [[ "$verify_remediation_migrations" == true && -f "$workspace/migrations/019_fix_shop_message_allocator_lock_order.sql" ]]; then
    verify_allocator_bridge_migration
  fi
  if [[ "$verify_remediation_migrations" == true && -f "$workspace/migrations/020_create_shop_message_asset_lifecycle.sql" ]]; then
    verify_asset_lifecycle_migration
  fi
  if [[ "$verify_remediation_migrations" == true && -f "$workspace/migrations/021_enforce_shop_message_parent_ownership.sql" ]]; then
    verify_message_parent_migration
  fi
  if [[ "$verify_remediation_migrations" == true && -f "$workspace/migrations/022_add_shop_vehicle_base_usage_constraints.sql" ]]; then
    verify_vehicle_base_usage_migration
  fi
  if [[ "$verify_remediation_migrations" == true && -f "$workspace/migrations/023_create_shop_notification_item_metadata.sql" ]]; then
    verify_notification_metadata_migration
  fi
  regenerate_stage miltech_test_shops final-available-candidate
  (cd "$workspace" && build_go build ./...) || fail 'Fresh full candidate build failed.'
  printf '%s\n' 'Fresh full available candidate build PASS (not final all-migrations acceptance).'
  if [[ "$verify_remediation_migrations" == true ]]; then
    printf 'Remediation pairs available: %s/5; populated/refusal/reverse acceptance belongs to each migration task and Task 28.\n' "${#remediation_migrations[@]}"
  fi
}

# All helpers below inherit only the wrapper's marked private cluster. No target
# or arbitrary SQL arguments are exposed to callers outside this sourced file.
verify_allocator_bridge_migration() {
  local database=miltech_test_allocator_bridge
  local forward=migrations/019_fix_shop_message_allocator_lock_order.sql
  local reverse=migrations/019_rollback_fix_shop_message_allocator_lock_order.sql
  local before after
  restore_baseline "$database"
  apply_later_migrations "$database"
  verify_disposable_identity "$database"
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO public.users (uid, email, username, created_at, is_enabled)
VALUES ('allocator-user', 'allocator@example.com', 'allocator', now(), true);
INSERT INTO public.shops (id, name, created_by, created_at)
VALUES ('allocator-shop', 'Allocator', 'allocator-user', now());
INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at)
VALUES ('allocator-one', 'allocator-shop', 'allocator-user', 'retained', now()),
       ('allocator-two', 'allocator-shop', 'allocator-user', 'deleted', now());
DELETE FROM public.shop_messages WHERE id='allocator-two';
SQL
  before=$(allocator_data_fingerprint)
  allocator_migrate "$forward" allocator-populated-forward
  [[ "$(allocator_data_fingerprint)" == "$before" ]] || fail '019 forward changed messages/counters.'
  allocator_expect_refusal "$forward" 'unexpected allocator definition' allocator-repeated-forward

  # Simulate interruption after function replacement, before COMMIT. The
  # rollback must leave the bridge and all persisted rows exactly intact.
  verify_disposable_identity "$database"
  {
    printf '%s\n' 'BEGIN;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$reverse"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/setup.log" 2>&1 || fail '019 interrupted reverse rehearsal failed.'
  regenerate_stage "$database" allocator-interrupted-reverse-rollback
  [[ $(psql_local "$database" -Atc "SELECT md5(prosrc) FROM pg_proc WHERE oid='public.assign_shop_message_insertion_number()'::regprocedure") == 42f7105c5298267751a040de0fe30012 ]] || fail 'Interrupted reverse changed allocator.'
  [[ "$(allocator_data_fingerprint)" == "$before" ]] || fail 'Interrupted reverse changed rows.'

  allocator_migrate "$reverse" allocator-reverse
  [[ $(psql_local "$database" -Atc "SELECT md5(prosrc) FROM pg_proc WHERE oid='public.assign_shop_message_insertion_number()'::regprocedure") == 5b2f595523c9653cf80ec52148537550 ]] || fail '019 reverse did not restore pinned 018 body.'
  [[ "$(allocator_data_fingerprint)" == "$before" ]] || fail '019 reverse changed rows.'
  allocator_expect_refusal "$reverse" 'unexpected allocator definition' allocator-repeated-reverse

  # Save the verified definition for exact restoration after deliberate drift.
  psql_local "$database" -Atc "SELECT pg_get_functiondef('public.assign_shop_message_insertion_number()'::regprocedure)" >"$instance/allocator-original.sql"
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
CREATE OR REPLACE FUNCTION public.assign_shop_message_insertion_number() RETURNS trigger
LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
SQL
  regenerate_stage "$database" allocator-definition-drift-fixture
  allocator_expect_refusal "$forward" 'unexpected allocator definition' allocator-definition-refusal
  allocator_migrate "$instance/allocator-original.sql" allocator-definition-repair

  psql_local "$database" -c 'ALTER FUNCTION public.assign_shop_message_insertion_number() SECURITY DEFINER' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-security-drift-fixture
  allocator_expect_refusal "$forward" 'unexpected allocator definition' allocator-security-refusal
  psql_local "$database" -c 'ALTER FUNCTION public.assign_shop_message_insertion_number() SECURITY INVOKER' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-security-repair

  psql_local "$database" -c 'ALTER TABLE public.shop_messages DISABLE TRIGGER shop_messages_assign_insertion_number' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-trigger-drift-fixture
  allocator_expect_refusal "$forward" 'unexpected allocator trigger' allocator-trigger-refusal
  psql_local "$database" -c 'ALTER TABLE public.shop_messages ENABLE TRIGGER shop_messages_assign_insertion_number' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-trigger-repair

  psql_local "$database" -c "UPDATE public.shop_message_counters SET last_number=0 WHERE shop_id='allocator-shop'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-counter-drift-fixture
  allocator_expect_refusal "$forward" 'invalid message numbering or counters' allocator-counter-refusal
  psql_local "$database" -c "UPDATE public.shop_message_counters SET last_number=2 WHERE shop_id='allocator-shop'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-counter-repair

  psql_local "$database" -c "UPDATE public.shop_messages SET insertion_number=NULL WHERE id='allocator-one'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-number-drift-fixture
  allocator_expect_refusal "$forward" 'invalid message numbering or counters' allocator-number-refusal
  psql_local "$database" -c "UPDATE public.shop_messages SET insertion_number=1 WHERE id='allocator-one'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-number-repair

  psql_local "$database" -c "DELETE FROM public.shop_message_counters WHERE shop_id='allocator-shop'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-missing-counter-fixture
  allocator_expect_refusal "$forward" 'invalid message numbering or counters' allocator-missing-counter-refusal
  psql_local "$database" -c "INSERT INTO public.shop_message_counters VALUES ('allocator-shop',2)" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" allocator-missing-counter-repair

  allocator_migrate "$forward" allocator-reapply
  after=$(allocator_data_fingerprint)
  [[ "$after" == "$before" ]] || fail '019 reapply changed messages/counters.'
  [[ $(psql_local "$database" -Atc "INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at, insertion_number) VALUES ('allocator-three', 'allocator-shop', 'allocator-user', 'next', now(), 999) RETURNING insertion_number" | head -n 1) == 3 ]] || fail '019 reapply lost high watermark or accepted caller allocation.'
  printf '%s\n' '019 populated forward/refusal/interrupted rollback/reverse/reapply PASS; data preserved; deleted-number high watermark retained.'
}

allocator_migrate() {
  local file=$1 stage=$2
  # Only the fixed verified backup path can bypass the candidate migration root.
  if [[ "$file" != "$instance/allocator-original.sql" ]]; then file="$workspace/$file"; fi
  verify_disposable_identity "$database"
  psql_local "$database" -f "$file" >"$instance/setup.log" 2>&1 || fail "019 rehearsal migration failed: $stage"
  regenerate_stage "$database" "$stage"
}

allocator_data_fingerprint() {
  verify_disposable_identity "$database"
  psql_local "$database" -Atc "SELECT md5(jsonb_build_array(
    (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM public.shop_messages m),
    (SELECT jsonb_agg(to_jsonb(c) ORDER BY shop_id) FROM public.shop_message_counters c)
  )::text)"
}

allocator_expect_refusal() {
  local migration=$1 reason=$2 stage=$3 before_state after_state
  verify_disposable_identity "$database"
  before_state=$(allocator_rehearsal_state)
  if psql_local "$database" -f "$workspace/$migration" >"$instance/allocator-refusal.log" 2>&1; then
    fail "019 unexpectedly accepted: $stage"
  fi
  rg -q -F "$reason" "$instance/allocator-refusal.log" || fail "019 refusal failed for wrong reason: $stage"
  after_state=$(allocator_rehearsal_state)
  [[ "$after_state" == "$before_state" ]] || fail "019 refusal changed function, trigger or rows: $stage"
  regenerate_stage "$database" "$stage"
  printf '019 refusal PASS: %s; function/trigger/data unchanged\n' "$stage"
}

allocator_rehearsal_state() {
  printf '%s|' "$(allocator_data_fingerprint)"
  psql_local "$database" -Atc "SELECT md5(pg_get_functiondef('public.assign_shop_message_insertion_number()'::regprocedure)) || '|' || tgenabled
    FROM pg_trigger WHERE tgrelid='public.shop_messages'::regclass AND tgname='shop_messages_assign_insertion_number'"
}

verify_asset_lifecycle_migration() {
  local database=miltech_test_asset_lifecycle
  local forward=migrations/020_create_shop_message_asset_lifecycle.sql
  local reverse=migrations/020_rollback_shop_message_asset_lifecycle.sql
  restore_baseline "$database"
  apply_later_migrations "$database"
  verify_disposable_identity "$database"
  psql_local "$database" -f "$workspace/migrations/019_fix_shop_message_allocator_lock_order.sql" >"$instance/setup.log" 2>&1 || fail '020 prerequisite failed.'
  regenerate_stage "$database" asset-prerequisite
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO users(uid,email,username,created_at,is_enabled) VALUES('asset-migration','a@example.com','a',now(),true);
INSERT INTO shops(id,name,created_by,created_at) VALUES('asset-shop','A','asset-migration',now());
INSERT INTO shop_messages(id,shop_id,user_id,message,created_at) VALUES('asset-message','asset-shop','asset-migration','[IMAGE:https://unverified.example/old.jpg]',now());
SQL
  psql_local "$database" -f "$workspace/$forward" >"$instance/setup.log" 2>&1 || fail '020 populated forward failed.'
  regenerate_stage "$database" asset-populated-forward
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM shop_message_uploads") == 0 ]] || fail '020 inferred historical ownership.'
  [[ $(psql_local "$database" -Atc "SELECT message FROM shop_messages WHERE id='asset-message'") == '[IMAGE:https://unverified.example/old.jpg]' ]] || fail '020 changed historical text.'
  if psql_local "$database" -f "$workspace/$forward" >"$instance/asset-refusal.log" 2>&1; then fail '020 repeated forward unexpectedly succeeded.'; fi
  rg -q -F 'already exists' "$instance/asset-refusal.log" || fail '020 repeat refused for wrong reason.'
  regenerate_stage "$database" asset-repeated-forward-refusal
  psql_local "$database" -f "$workspace/$reverse" >"$instance/setup.log" 2>&1 || fail '020 empty registry reverse failed.'
  regenerate_stage "$database" asset-empty-reverse
  [[ $(psql_local "$database" -Atc "SELECT to_regclass('public.shop_message_uploads') IS NULL") == t ]] || fail '020 reverse left registry.'
  psql_local "$database" -f "$workspace/$forward" >"$instance/setup.log" 2>&1 || fail '020 reapply failed.'
  regenerate_stage "$database" asset-reapply
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO shop_message_uploads(id,operation_id,shop_id,uploader_id,account,container,blob_key,url,extension,state,lease_until)
VALUES('00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000021','asset-shop','asset-migration','owned','shop-message-images','asset-shop/a.jpg','https://owned.blob.core.windows.net/shop-message-images/asset-shop/a.jpg','.jpg','ready',now());
INSERT INTO shop_message_asset_references VALUES('asset-message','00000000-0000-4000-8000-000000000020','asset-shop');
SQL
  if psql_local "$database" -f "$workspace/$reverse" >"$instance/asset-refusal.log" 2>&1; then fail '020 populated reverse unexpectedly succeeded.'; fi
  rg -q -F 'message asset registry and cleanup proof must be retained' "$instance/asset-refusal.log" || fail '020 populated reverse refused for wrong reason.'
  [[ $(psql_local "$database" -Atc 'SELECT count(*) FROM shop_message_asset_references') == 1 ]] || fail '020 refused reverse changed references.'
  regenerate_stage "$database" asset-populated-reverse-refusal
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
CREATE ROLE asset_deletion_role NOLOGIN;
GRANT USAGE ON SCHEMA public TO asset_deletion_role;
GRANT SELECT,DELETE ON shop_messages TO asset_deletion_role;
GRANT SELECT ON shop_message_uploads TO asset_deletion_role;
SQL
  if psql_local "$database" -c "SET ROLE asset_deletion_role; DELETE FROM shop_messages WHERE id='asset-message'" >"$instance/asset-refusal.log" 2>&1; then fail '020 unprivileged enqueue unexpectedly succeeded.'; fi
  rg -q -F 'permission denied for table shop_message_blob_cleanup_jobs' "$instance/asset-refusal.log" || fail '020 missing enqueue privilege failed for wrong reason.'
  [[ $(psql_local "$database" -Atc 'SELECT count(*) FROM shop_message_asset_references') == 1 ]] || fail '020 missing privilege lost reference.'
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
GRANT INSERT ON shop_message_blob_cleanup_jobs TO asset_deletion_role;
SET ROLE asset_deletion_role;
DELETE FROM shop_messages WHERE id='asset-message';
RESET ROLE;
SQL
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM shop_message_blob_cleanup_jobs WHERE blob_key='asset-shop/a.jpg' AND state='pending'") == 1 ]] || fail '020 invoker hook failed to retain exact target.'
  [[ $(psql_local "$database" -Atc 'SELECT count(*) FROM shop_message_asset_references') == 0 ]] || fail '020 message cascade retained reference.'
  regenerate_stage "$database" asset-invoker-cascade-proof
  printf '%s\n' '020 empty/populated forward, repeated-forward refusal, reverse/reapply, populated reverse refusal, invoker-role denial/rollback and privileged cascade PASS; historical text unchanged and no inferred owners.'
}


verify_message_parent_migration() {
  local database=miltech_test_message_parent
  local forward=migrations/021_enforce_shop_message_parent_ownership.sql
  local reverse=migrations/021_rollback_enforce_shop_message_parent_ownership.sql
  local migration before key_oid
  restore_baseline "$database"
  apply_later_migrations "$database"
  for migration in migrations/019_fix_shop_message_allocator_lock_order.sql migrations/020_create_shop_message_asset_lifecycle.sql; do
    parent_migrate "$migration" "parent-prerequisite-${migration##*/}"
  done
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO users(uid,email,username,created_at,is_enabled) VALUES('parent-user','parent@example.com','parent',now(),true);
INSERT INTO shops(id,name,created_by,created_at) VALUES('parent-shop','A','parent-user',now()),('other-shop','B','parent-user',now());
INSERT INTO shop_messages(id,shop_id,user_id,message) VALUES('parent','parent-shop','parent-user','parent');
INSERT INTO shop_messages(id,shop_id,user_id,message,parent_id) VALUES('reply','parent-shop','parent-user','reply','parent'),('foreign','other-shop','parent-user','foreign','parent');
SQL
  parent_expect_refusal "$forward" 'foreign or missing reply parent' TestMessageParentMigrationRefusesForeignRows
  psql_local "$database" -c "DELETE FROM shop_messages WHERE id='foreign'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-disposable-foreign-fixture-removal

  # These are deliberately corrupted disposable fixtures, never target repairs.
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
ALTER TABLE shop_messages DROP CONSTRAINT shop_message_parent_id_fkey;
INSERT INTO shop_messages(id,shop_id,user_id,message,parent_id) VALUES('missing','parent-shop','parent-user','missing','absent');
ALTER TABLE shop_messages ADD CONSTRAINT shop_message_parent_id_fkey FOREIGN KEY(parent_id) REFERENCES shop_messages(id) ON DELETE CASCADE NOT VALID;
SQL
  regenerate_stage "$database" parent-missing-fixture
  parent_expect_refusal "$forward" 'foreign or missing reply parent' parent-missing-refusal
  psql_local "$database" -c "DELETE FROM shop_messages WHERE id='missing'; ALTER TABLE shop_messages VALIDATE CONSTRAINT shop_message_parent_id_fkey" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-missing-fixture-restoration

  psql_local "$database" -c 'ALTER TABLE shop_messages DROP CONSTRAINT shop_message_parent_id_fkey; ALTER TABLE shop_messages ADD CONSTRAINT shop_message_parent_id_fkey FOREIGN KEY(parent_id) REFERENCES shop_messages(id) ON DELETE SET NULL' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-action-fixture
  parent_expect_refusal "$forward" 'unexpected reply FK shape or action' parent-unapproved-action-refusal
  psql_local "$database" -c 'ALTER TABLE shop_messages DROP CONSTRAINT shop_message_parent_id_fkey; ALTER TABLE shop_messages ADD CONSTRAINT shop_message_parent_id_fkey FOREIGN KEY(parent_id) REFERENCES shop_messages(id) ON DELETE CASCADE DEFERRABLE' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-shape-fixture
  parent_expect_refusal "$forward" 'unexpected reply FK shape or action' parent-shape-refusal
  psql_local "$database" -c 'ALTER TABLE shop_messages ALTER CONSTRAINT shop_message_parent_id_fkey NOT DEFERRABLE' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-shape-fixture-restoration
  # The generator intentionally refuses non-text legacy columns. Keep this
  # corrupted type inside the failing migration transaction, then regenerate
  # the rolled-back schema rather than publishing incompatible models.
  before=$(parent_rehearsal_state)
  if {
    printf '%s\n' 'BEGIN;' 'ALTER TABLE shop_messages ALTER COLUMN parent_id TYPE varchar;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$forward"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/parent-refusal.log" 2>&1; then
    fail '021 unexpectedly accepted parent-type-refusal'
  fi
  rg -q -F 'unexpected message identity column types' "$instance/parent-refusal.log" || fail '021 type refusal failed for wrong reason.'
  [[ "$(parent_rehearsal_state)" == "$before" ]] || fail '021 type refusal changed schema/data.'
  regenerate_stage "$database" parent-type-refusal-rollback
  printf '%s\n' '021 refusal PASS: parent-type-refusal; schema/data unchanged'

  before=$(parent_data_fingerprint)
  key_oid=$(psql_local "$database" -Atc "SELECT oid FROM pg_constraint WHERE conrelid='shop_messages'::regclass AND conname='shop_messages_id_shop_unique'")
  parent_migrate "$forward" parent-populated-forward
  [[ "$(parent_data_fingerprint)" == "$before" ]] || fail '021 forward changed rows.'
  parent_expect_refusal "$forward" 'unexpected reply FK shape or action' parent-repeated-forward-refusal
  # Dependency/ownership proof: 021 rollback must retain the exact 020 key and
  # the asset FK that depends on it, without rebuilding either.
  parent_migrate "$reverse" parent-populated-reverse
  [[ "$(parent_data_fingerprint)" == "$before" ]] || fail '021 reverse changed rows.'
  [[ $(psql_local "$database" -Atc "SELECT oid FROM pg_constraint WHERE conrelid='shop_messages'::regclass AND conname='shop_messages_id_shop_unique'") == "$key_oid" ]] || fail '021 reverse replaced or dropped the 020 key.'
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM pg_constraint WHERE conrelid='shop_message_asset_references'::regclass AND confrelid='shop_messages'::regclass AND contype='f' AND confdeltype='c' AND convalidated") == 1 ]] || fail '021 reverse damaged asset dependency.'
  parent_expect_refusal "$reverse" 'unexpected reply FK shape or action' parent-repeated-reverse-refusal
  parent_migrate "$forward" parent-reapply
  before=$(parent_rehearsal_state)
  {
    printf '%s\n' 'BEGIN;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$reverse"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/setup.log" 2>&1 || fail '021 interrupted reverse failed.'
  regenerate_stage "$database" parent-interrupted-reverse
  [[ "$(parent_rehearsal_state)" == "$before" ]] || fail '021 interrupted reverse changed schema/data.'
  psql_local "$database" -c 'ALTER TABLE shop_messages ALTER CONSTRAINT shop_message_parent_shop_fkey DEFERRABLE' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-reverse-shape-fixture
  parent_expect_refusal "$reverse" 'unexpected reply FK shape or action' parent-reverse-shape-refusal
  psql_local "$database" -c 'ALTER TABLE shop_messages ALTER CONSTRAINT shop_message_parent_shop_fkey NOT DEFERRABLE' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" parent-reverse-shape-restoration
  [[ $(psql_local "$database" -Atc "SELECT confdeltype FROM pg_constraint WHERE conrelid='shop_messages'::regclass AND conname='shop_message_parent_shop_fkey'") == c ]] || fail '021 did not preserve cascade.'
  psql_local "$database" -c "DELETE FROM shop_messages WHERE id='parent'" >"$instance/setup.log" 2>&1
  [[ $(psql_local "$database" -Atc 'SELECT count(*) FROM shop_messages') == 0 ]] || fail '021 same-Shop cascade failed.'
  printf '%s\n' '021 populated forward/reverse/reapply, foreign/missing/action/shape/type refusals, interrupted reverse and 020 key ownership PASS.'
}

parent_migrate() {
  local file=$1 stage=$2
  verify_disposable_identity "$database"
  psql_local "$database" -f "$workspace/$file" >"$instance/setup.log" 2>&1 || fail "021 rehearsal failed: $stage"
  regenerate_stage "$database" "$stage"
}

parent_data_fingerprint() {
  psql_local "$database" -Atc "SELECT md5(jsonb_build_array(
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM shop_messages x),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY shop_id) FROM shop_message_counters x),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM shop_message_uploads x),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY message_id,upload_id) FROM shop_message_asset_references x),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM shop_message_blob_cleanup_jobs x))::text)"
}

parent_rehearsal_state() {
  verify_disposable_identity "$database"
  printf '%s|' "$(parent_data_fingerprint)"
  # Compare physical schema metadata as well as definitions. Refusal cannot
  # silently rebuild a same-looking constraint or remove an asset dependency.
  psql_local "$database" -Atc "SELECT md5(jsonb_build_array(
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY oid) FROM pg_constraint x WHERE connamespace='public'::regnamespace),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY attrelid,attnum) FROM pg_attribute x WHERE attrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace)),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY indexrelid) FROM pg_index x WHERE indrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace)),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY oid) FROM pg_trigger x WHERE tgrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace)))::text)"
}

parent_expect_refusal() {
  local file=$1 reason=$2 stage=$3 before_state
  verify_disposable_identity "$database"
  before_state=$(parent_rehearsal_state)
  if psql_local "$database" -f "$workspace/$file" >"$instance/parent-refusal.log" 2>&1; then fail "021 unexpectedly accepted $stage"; fi
  rg -q -F "$reason" "$instance/parent-refusal.log" || fail "021 refused for wrong reason: $stage"
  [[ "$(parent_rehearsal_state)" == "$before_state" ]] || fail "021 refusal changed schema/data: $stage"
  regenerate_stage "$database" "$stage"
  printf '021 refusal PASS: %s; schema/data unchanged\n' "$stage"
}


verify_vehicle_base_usage_migration() {
  local database=miltech_test_vehicle_base_usage
  local forward=migrations/022_add_shop_vehicle_base_usage_constraints.sql
  local reverse=migrations/022_rollback_shop_vehicle_base_usage_constraints.sql
  local before tracked_oid reading
  restore_baseline "$database"
  apply_later_migrations "$database"
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO users(uid,email,username,created_at,is_enabled) VALUES('base-user','base@example.com','base',now(),true);
INSERT INTO shops(id,name,created_by,created_at) VALUES('base-shop','Base','base-user',now());
INSERT INTO shop_vehicle(id,shop_id,creator_id,admin,mileage,hours) VALUES('base-vehicle','base-shop','base-user','Base',100,50);
SQL
  # A bad historical row remains usable before hardening; no implicit repair.
  for reading in mileage hours; do
    psql_local "$database" -c "UPDATE shop_vehicle SET $reading=-1 WHERE id='base-vehicle'" >"$instance/setup.log" 2>&1
    regenerate_stage "$database" "base-negative-$reading-fixture"
    psql_local "$database" -c "UPDATE shop_vehicle SET tracked_mileage=120 WHERE id='base-vehicle'" >"$instance/setup.log" 2>&1
    base_expect_refusal "$forward" 'negative historical base usage; explicit repair manifest required' "base-negative-$reading-refusal"
    [[ $(psql_local "$database" -Atc "SELECT $reading FROM shop_vehicle WHERE id='base-vehicle'") == -1 ]] || fail '022 repaired a historical reading.'
    # Deliberate disposable fixture restoration is not a target data repair.
    psql_local "$database" -c "UPDATE shop_vehicle SET $reading=0 WHERE id='base-vehicle'" >"$instance/setup.log" 2>&1
    regenerate_stage "$database" "base-negative-$reading-fixture-restoration"
  done
  before=$(base_rehearsal_state)
  # The intentionally incompatible type is kept inside a failed transaction;
  # tagged generation still runs against its rolled-back compatible schema.
  if {
    printf '%s\n' 'BEGIN;' 'ALTER TABLE shop_vehicle ALTER COLUMN mileage TYPE bigint;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$forward"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/base-refusal.log" 2>&1; then fail '022 accepted incompatible base column type.'; fi
  rg -q -F 'unexpected vehicle usage column shape' "$instance/base-refusal.log" || fail '022 type refusal failed for wrong reason.'
  [[ "$(base_rehearsal_state)" == "$before" ]] || fail '022 type refusal changed schema/data.'
  regenerate_stage "$database" base-type-refusal-rollback

  psql_local "$database" -c 'ALTER TABLE shop_vehicle ALTER COLUMN hours DROP NOT NULL' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-nullability-fixture
  base_expect_refusal "$forward" 'unexpected vehicle usage column shape' base-nullability-refusal
  psql_local "$database" -c 'ALTER TABLE shop_vehicle ALTER COLUMN hours SET NOT NULL; ALTER TABLE shop_vehicle ALTER COLUMN mileage SET DEFAULT 1' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-default-fixture
  base_expect_refusal "$forward" 'unexpected vehicle usage column shape' base-default-refusal
  psql_local "$database" -c 'ALTER TABLE shop_vehicle ALTER COLUMN mileage SET DEFAULT 0; ALTER TABLE shop_vehicle DROP CONSTRAINT shop_vehicle_tracked_hours_nonnegative; ALTER TABLE shop_vehicle ADD CONSTRAINT shop_vehicle_tracked_hours_nonnegative CHECK (tracked_hours >= -1)' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-tracked-constraint-fixture
  base_expect_refusal "$forward" 'unexpected migration 015 tracked constraint' base-tracked-constraint-refusal
  psql_local "$database" -c 'ALTER TABLE shop_vehicle DROP CONSTRAINT shop_vehicle_tracked_hours_nonnegative; ALTER TABLE shop_vehicle ADD CONSTRAINT shop_vehicle_tracked_hours_nonnegative CHECK (tracked_hours IS NULL OR tracked_hours >= 0)' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-tracked-constraint-restoration

  before=$(base_data_fingerprint)
  tracked_oid=$(base_tracked_constraint_identity)
  base_migrate "$forward" base-populated-forward
  [[ "$(base_data_fingerprint)" == "$before" ]] || fail '022 forward changed historical data.'
  [[ "$(base_tracked_constraint_identity)" == "$tracked_oid" ]] || fail '022 forward changed 015 constraints.'
  base_expect_refusal "$forward" 'base usage constraints already exist' base-repeated-forward-refusal
  # Interruption cannot drop the guards, mutate data or rebuild the 015 checks.
  before=$(base_rehearsal_state)
  {
    printf '%s\n' 'BEGIN;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$reverse"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/setup.log" 2>&1 || fail '022 interrupted reverse failed.'
  regenerate_stage "$database" base-interrupted-reverse-rollback
  [[ "$(base_rehearsal_state)" == "$before" ]] || fail '022 interrupted reverse changed schema/data.'

  before=$(base_data_fingerprint)
  base_migrate "$reverse" base-populated-reverse
  [[ "$(base_data_fingerprint)" == "$before" ]] || fail '022 reverse changed data.'
  [[ "$(base_tracked_constraint_identity)" == "$tracked_oid" ]] || fail '022 reverse changed 015 constraints.'
  base_expect_refusal "$reverse" 'unexpected base usage constraint' base-repeated-reverse-refusal
  base_migrate "$forward" base-reapply
  psql_local "$database" -c 'ALTER TABLE shop_vehicle DROP CONSTRAINT shop_vehicle_hours_nonnegative; ALTER TABLE shop_vehicle ADD CONSTRAINT shop_vehicle_hours_nonnegative CHECK (hours >= -1)' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-reverse-definition-fixture
  base_expect_refusal "$reverse" 'unexpected base usage constraint' base-reverse-definition-refusal
  psql_local "$database" -c 'ALTER TABLE shop_vehicle DROP CONSTRAINT shop_vehicle_hours_nonnegative; ALTER TABLE shop_vehicle ADD CONSTRAINT shop_vehicle_hours_nonnegative CHECK (hours >= 0)' >"$instance/setup.log" 2>&1
  regenerate_stage "$database" base-reverse-definition-restoration
  [[ "$(base_tracked_constraint_identity)" == "$tracked_oid" ]] || fail '022 rehearsal changed 015 constraints.'
  printf '%s\n' '022 populated forward/reverse/reapply, negative historical/type/nullability/default/015-definition/repeated/reverse-definition refusals and interrupted reverse PASS; rows unchanged, no automatic repair, exact 015 constraints retained.'
}

base_migrate() {
  local file=$1 stage=$2
  verify_disposable_identity "$database"
  psql_local "$database" -f "$workspace/$file" >"$instance/setup.log" 2>&1 || fail "022 rehearsal failed: $stage"
  regenerate_stage "$database" "$stage"
}

base_data_fingerprint() {
  psql_local "$database" -Atc "SELECT md5(coalesce(jsonb_agg(to_jsonb(x) ORDER BY id)::text,'[]')) FROM shop_vehicle x"
}

base_tracked_constraint_identity() {
  psql_local "$database" -Atc "SELECT md5(jsonb_agg(to_jsonb(x) ORDER BY oid)::text) FROM pg_constraint x WHERE conrelid='shop_vehicle'::regclass AND conname IN ('shop_vehicle_tracked_mileage_nonnegative','shop_vehicle_tracked_hours_nonnegative')"
}

base_rehearsal_state() {
  verify_disposable_identity "$database"
  printf '%s|' "$(base_data_fingerprint)"
  psql_local "$database" -Atc "SELECT md5(jsonb_build_array(
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY oid) FROM pg_constraint x WHERE conrelid='shop_vehicle'::regclass),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY attnum) FROM pg_attribute x WHERE attrelid='shop_vehicle'::regclass),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY oid) FROM pg_attrdef x WHERE adrelid='shop_vehicle'::regclass))::text)"
}

base_expect_refusal() {
  local file=$1 reason=$2 stage=$3 before_state
  verify_disposable_identity "$database"
  before_state=$(base_rehearsal_state)
  if psql_local "$database" -f "$workspace/$file" >"$instance/base-refusal.log" 2>&1; then fail "022 unexpectedly accepted $stage"; fi
  rg -q -F "$reason" "$instance/base-refusal.log" || fail "022 refused for wrong reason: $stage"
  [[ "$(base_rehearsal_state)" == "$before_state" ]] || fail "022 refusal changed schema/data: $stage"
  regenerate_stage "$database" "$stage"
  printf '022 refusal PASS: %s; schema/data unchanged\n' "$stage"
}


verify_notification_metadata_migration() {
  local database=miltech_test_notification_metadata
  local forward=migrations/023_create_shop_notification_item_metadata.sql
  local reverse=migrations/023_rollback_shop_notification_item_metadata.sql
  local migration before original_constraints
  restore_baseline "$database"
  apply_later_migrations "$database"
  for migration in "${remediation_migrations[@]}"; do
    [[ "$migration" != "$forward" ]] || break
    metadata_migrate "$migration" "metadata-prerequisite-${migration##*/}"
  done
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO users(uid,email,username,created_at,is_enabled) VALUES('metadata-user','metadata@example.com','metadata',now(),true);
INSERT INTO shops(id,name,created_by) VALUES('metadata-shop','Metadata','metadata-user');
INSERT INTO shop_vehicle(id,shop_id,creator_id,admin) VALUES('metadata-vehicle','metadata-shop','metadata-user','Metadata');
INSERT INTO shop_vehicle_notifications(id,shop_id,vehicle_id,title,description,type)
VALUES('metadata-notification','metadata-shop','metadata-vehicle','Metadata','Retained','PM');
INSERT INTO shop_notification_items(id,shop_id,notification_id,niin,nomenclature,quantity,nickname,unit_of_measure) VALUES
('metadata-a','metadata-shop','metadata-notification','exact','Hub',99,'Front hub','KT'),
('metadata-b','historical-other','metadata-notification','exact','Other',3,'Other hub','EA'),
('metadata-null','metadata-shop','metadata-notification','null','Null',7,NULL,NULL),
('metadata-default','metadata-shop','metadata-notification','default','Default',8,'','EA'),
('metadata-raw','metadata-shop','metadata-notification',' exact ','Raw',9,'',' raw-code ');
SQL
  original_constraints=$(metadata_item_constraints)
  metadata_expect_refusal "$forward" 'conflicting historical metadata; explicit resolution manifest required' metadata-conflicting-backfill-refusal
  [[ $(psql_local "$database" -Atc "SELECT nickname FROM shop_notification_items WHERE id='metadata-b'") == 'Other hub' ]] || fail '023 repaired conflicting metadata.'
  # Synthetic fixture changes only: null/empty disagreement is also ambiguous.
  psql_local "$database" -c "UPDATE shop_notification_items SET nickname=NULL,unit_of_measure=NULL WHERE id='metadata-a'; UPDATE shop_notification_items SET nickname='',unit_of_measure=NULL WHERE id='metadata-b'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" metadata-null-empty-conflict-fixture
  metadata_expect_refusal "$forward" 'conflicting historical metadata; explicit resolution manifest required' metadata-null-empty-backfill-refusal
  psql_local "$database" -c "UPDATE shop_notification_items SET nickname='Front hub',unit_of_measure='KT' WHERE id IN ('metadata-a','metadata-b')" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" metadata-unambiguous-fixture-restoration

  before=$(metadata_rehearsal_state)
  if {
    printf '%s\n' 'BEGIN;' 'ALTER TABLE shop_notification_items ALTER COLUMN unit_of_measure TYPE text;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$forward"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/metadata-refusal.log" 2>&1; then fail '023 accepted incompatible unit column shape.'; fi
  rg -q -F 'unexpected item metadata column shape' "$instance/metadata-refusal.log" || fail '023 type refusal failed for wrong reason.'
  [[ "$(metadata_rehearsal_state)" == "$before" ]] || fail '023 type refusal changed schema/data.'
  regenerate_stage "$database" metadata-type-refusal-rollback
  before=$(metadata_item_fingerprint)
  metadata_migrate "$forward" metadata-populated-forward
  [[ "$(metadata_item_fingerprint)" == "$before" ]] || fail '023 forward changed physical items.'
  [[ "$(metadata_item_constraints)" == "$original_constraints" ]] || fail '023 changed active-item constraints.'
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM shop_notification_item_metadata") == 4 ]] || fail '023 backfill logical key count mismatch.'
  [[ $(psql_local "$database" -Atc "SELECT nickname || '|' || unit_of_measure || '|' || state || '|' || jsonb_array_length(candidates) || '|' || version FROM shop_notification_item_metadata WHERE niin='exact'") == 'Front hub|KT|resolved|2|1' ]] || fail '023 unambiguous duplicate backfill failed.'
  [[ $(psql_local "$database" -Atc "SELECT (nickname IS NULL AND unit_of_measure IS NULL) FROM shop_notification_item_metadata WHERE niin='null'") == t ]] || fail '023 normalized null metadata.'
  [[ $(psql_local "$database" -Atc "SELECT nickname || '|' || unit_of_measure FROM shop_notification_item_metadata WHERE niin='default'") == '|EA' ]] || fail '023 normalized default metadata.'
  [[ $(psql_local "$database" -Atc "SELECT unit_of_measure FROM shop_notification_item_metadata WHERE niin=' exact '") == ' raw-code ' ]] || fail '023 normalized exact identity/raw unit.'
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM information_schema.columns WHERE table_name='shop_notification_item_metadata' AND column_name='quantity'") == 0 ]] || fail '023 retained quantity.'
  [[ $(psql_local "$database" -Atc "SELECT count(*) FROM shop_notification_item_metadata WHERE resolution_version <> 0") == 0 ]] || fail '023 backfill invented explicit resolution provenance.'
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL'
DO $$ BEGIN
    BEGIN
        UPDATE shop_notification_item_metadata SET resolution_version=-1 WHERE niin='exact';
        RAISE EXCEPTION '023 accepted a negative resolution version';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
    BEGIN
        UPDATE shop_notification_item_metadata SET resolution_version=version+1 WHERE niin='exact';
        RAISE EXCEPTION '023 accepted a future resolution version';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
END $$;
SQL
  metadata_expect_refusal "$forward" 'retention table already exists' metadata-repeated-forward-refusal
  metadata_expect_refusal "$reverse" 'populated retention reverse requires explicit resolution manifest' metadata-populated-reverse-refusal
  # Notification deletion is the sole supported retention expiry. Fixture reset
  # supplies a legitimately empty table for reverse/interruption/reapply proofs.
  psql_local "$database" -c "DELETE FROM shop_vehicle_notifications WHERE id='metadata-notification'" >"$instance/setup.log" 2>&1
  regenerate_stage "$database" metadata-notification-expiry-fixture
  [[ $(psql_local "$database" -Atc 'SELECT count(*) FROM shop_notification_item_metadata') == 0 ]] || fail '023 notification cascade did not end retention.'
  before=$(metadata_rehearsal_state)
  {
    printf '%s\n' 'BEGIN;'
    sed '/^BEGIN;$/d; /^COMMIT;$/d' "$workspace/$reverse"
    printf '%s\n' 'ROLLBACK;'
  } | psql_local "$database" >"$instance/setup.log" 2>&1 || fail '023 interrupted reverse failed.'
  regenerate_stage "$database" metadata-interrupted-reverse-rollback
  [[ "$(metadata_rehearsal_state)" == "$before" ]] || fail '023 interrupted reverse changed schema/data.'
  metadata_migrate "$reverse" metadata-empty-reverse
  [[ $(psql_local "$database" -Atc "SELECT to_regclass('public.shop_notification_item_metadata') IS NULL") == t ]] || fail '023 reverse retained table.'
  metadata_expect_refusal "$reverse" 'does not exist' metadata-repeated-reverse-refusal
  metadata_migrate "$forward" metadata-empty-reapply
  [[ "$(metadata_item_constraints)" == "$original_constraints" ]] || fail '023 rehearsals changed active-item constraints.'
  printf '%s\n' '023 populated/null/default/raw/duplicate backfill, conflicting/null-empty/type/repeated/populated-reverse refusals, notification cascade, interrupted reverse, empty reverse/reapply PASS; no quantity or active uniqueness changes; zero backfill resolution provenance and resolution-version bounds verified.'
}

metadata_migrate() {
  local file=$1 stage=$2
  verify_disposable_identity "$database"
  psql_local "$database" -f "$workspace/$file" >"$instance/setup.log" 2>&1 || fail "023 rehearsal failed: $stage"
  regenerate_stage "$database" "$stage"
}
metadata_item_fingerprint() {
  psql_local "$database" -Atc "SELECT md5(coalesce(jsonb_agg(to_jsonb(x) ORDER BY id)::text,'[]')) FROM shop_notification_items x"
}
metadata_item_constraints() {
  psql_local "$database" -Atc "SELECT md5(jsonb_agg(to_jsonb(x) ORDER BY oid)::text) FROM pg_constraint x WHERE conrelid='shop_notification_items'::regclass"
}
metadata_rehearsal_state() {
  verify_disposable_identity "$database"
  printf '%s|' "$(metadata_item_fingerprint)"
  if [[ $(psql_local "$database" -Atc "SELECT to_regclass('public.shop_notification_item_metadata') IS NOT NULL") == t ]]; then
    psql_local "$database" -Atc "SELECT md5(coalesce(jsonb_agg(to_jsonb(x) ORDER BY notification_id,niin)::text,'[]')) FROM shop_notification_item_metadata x"
  fi
  psql_local "$database" -Atc "SELECT md5(jsonb_build_array(
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY oid) FROM pg_constraint x WHERE connamespace='public'::regnamespace),
    (SELECT jsonb_agg(to_jsonb(x) ORDER BY attrelid,attnum) FROM pg_attribute x WHERE attrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace)))::text)"
}
metadata_expect_refusal() {
  local file=$1 reason=$2 stage=$3 before_state
  before_state=$(metadata_rehearsal_state)
  if psql_local "$database" -f "$workspace/$file" >"$instance/metadata-refusal.log" 2>&1; then fail "023 unexpectedly accepted $stage"; fi
  rg -q -F "$reason" "$instance/metadata-refusal.log" || fail "023 refused for wrong reason: $stage"
  [[ "$(metadata_rehearsal_state)" == "$before_state" ]] || fail "023 refusal changed schema/data: $stage"
  regenerate_stage "$database" "$stage"
  printf '023 refusal PASS: %s; schema/data unchanged\n' "$stage"
}


# A dedicated fixture keeps runner application distinct from each migration's
# existing populated/forward/refusal/reverse matrix. No named target is aliased.
verify_release_runner() {
  local database=miltech_test_release_runner stage
  restore_baseline "$database"
  apply_later_migrations "$database"
  verify_disposable_identity "$database"
  psql_local "$database" >"$instance/setup.log" 2>&1 <<'SQL' || fail 'Disposable release fixture setup failed.'
INSERT INTO users(uid,email,username,created_at,is_enabled) VALUES('release-user','release@example.invalid','release',now(),true);
INSERT INTO shops(id,name,created_by) VALUES('release-shop','Release fixture','release-user');
INSERT INTO shop_members(id,shop_id,user_id,role) VALUES('release-member','release-shop','release-user','admin');
SQL
  regenerate_stage "$database" release-owner-approved-fixture
  for stage in 019 020 021 022 023; do
    verify_disposable_identity "$database"
    SHOPS_RELEASE_TEST_MARKER="$marker" python3 "$root/scripts/apply-shops-remediation-migrations_test.py" \
      --rehearse-disposable "$root" "$instance" "$database" "$port" "$stage" || fail "Disposable release runner failed at $stage."
    regenerate_stage "$database" "release-runner-$stage"
  done
  printf '%s\n' 'Release runner complete disposable checkpoint PASS: all five actions regenerated; 32 PMCS hashes unchanged; no named target contact.'
}
