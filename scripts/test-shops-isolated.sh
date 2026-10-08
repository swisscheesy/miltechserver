#!/usr/bin/env bash
set -euo pipefail
# Never trace generated connection arguments, including under bash -x.
set +x
fail() { printf '%s\n' "$1" >&2; exit 1; }
[[ $# -gt 0 ]] || fail 'Usage: scripts/test-shops-isolated.sh <go test arguments>'
[[ -z ${TEST_DATABASE_URL+x} && -z ${TEST_DATABASE_MARKER+x} && -z ${TEST_DB_URL+x} ]] || fail 'Refusing externally supplied test database configuration; unset TEST_DATABASE_URL, TEST_DATABASE_MARKER and TEST_DB_URL.'
# Only these TestMain paths are in this wrapper's integration acceptance scope.
# Never allow ./... to reach independent dotenv-based database setups.
packages=0
needs_value=false
verify_notification_migration=false
verify_message_sync_migration=false
verify_remediation_migrations=false
verify_release_runner_only=false
workspace=
publish_candidate_generated=false
go_arguments=()
for argument in "$@"; do
  if [[ "$needs_value" == true ]]; then needs_value=false; go_arguments+=("$argument"); continue; fi
  case "$argument" in
    --publish-candidate-generated) publish_candidate_generated=true; continue ;;
    --verify-release-runner) verify_release_runner_only=true; continue ;;
    --verify-remediation-migrations) verify_remediation_migrations=true; continue ;;
    --verify-notification-migration) verify_notification_migration=true; continue ;;
    --verify-message-sync-migration) verify_message_sync_migration=true; continue ;;
    ./tests/shops|./tests/equipment_services) packages=$((packages + 1)) ;;
    -race|-v|-failfast|-short) ;;
    -count|-run|-timeout|-parallel|-shuffle) needs_value=true ;;
    -count=*|-run=*|-timeout=*|-parallel=*|-shuffle=*) ;;
    *) fail 'Unsupported test argument or package; wrapper permits only ./tests/shops and ./tests/equipment_services.' ;;
  esac
  go_arguments+=("$argument")
done
[[ "$needs_value" == false && "$packages" -gt 0 ]] || fail 'A supported test package and complete flag values are required.'
root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/verify-shops-remediation-migrations.sh"
collect_remediation_migrations
baseline="$root/tests/testutil/shops_schema_baseline.sql"
# The approved public-schema export is pinned; no environment override.
baseline_sha256='9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70'
# The approved physical snapshot has the validated migration-015 constraints.
# Only migrations 016, 017 and 018 follow it; history of earlier source files is absent.
notification_migrations=(
  "migrations/016_create_shop_notification_operations.sql"
  "migrations/017_add_shop_notification_item_nickname_uom.sql"
)
message_sync_migrations=(
  "migrations/018_add_shop_message_insertion_numbers.sql"
)
later_migrations=("${notification_migrations[@]}" "${message_sync_migrations[@]}")
[[ $(shasum -a 256 "$baseline" | cut -d ' ' -f 1) == "$baseline_sha256" ]] || fail 'Approved baseline checksum mismatch.'
for tool in initdb pg_ctl psql createdb python3 go awk sed rg git shasum; do
  command -v "$tool" >/dev/null || fail "Required local tool unavailable: $tool"
done
# Ignore inherited libpq connection defaults for local setup and tests.
for name in $(compgen -e); do
  case "$name" in PG*) unset "$name" ;; esac
done
umask 077
instance=$(mktemp -d /tmp/miltech-test.XXXXXXXX)
cleanup() {
  status=$?
  trap - EXIT INT TERM
  # A failed startup may leave no server or a running one. Inspect before
  # removing the temporary directory so both outcomes clean up safely.
  if pg_ctl -D "$instance/data" status >/dev/null 2>&1; then
    pg_ctl -D "$instance/data" -m immediate -w stop >/dev/null 2>&1 || {
      printf '%s\n' "Disposable server cleanup failed; inspect $instance" >&2
      exit 1
    }
  fi
  rm -rf "$instance"
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
marker=$(python3 -c 'import secrets; print(secrets.token_hex(32))')
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
initdb -D "$instance/data" -U postgres -A trust >"$instance/setup.log" 2>&1 || fail 'Disposable initdb failed.'
pg_ctl -D "$instance/data" -l "$instance/server.log" -o "-h 127.0.0.1 -p $port -k $instance" -w start >"$instance/setup.log" 2>&1 || fail 'Disposable PostgreSQL startup failed.'
# The private socket ties schema setup to this cluster even if TCP binding fails.
create_local_db() { createdb -h "$instance" -p "$port" -U postgres "$1" >"$instance/setup.log" 2>&1 || fail 'Disposable database creation failed.'; }
psql_local() { local database=$1; shift; psql -X -v ON_ERROR_STOP=1 -h "$instance" -p "$port" -U postgres -d "$database" "$@"; }
# pg_dump --schema=public includes CREATE SCHEMA public but omits pg_trgm, which
# its public.gin_trgm_ops index needs. Keep the approved file byte-for-byte:
# provision the extension in the disposable default schema, then omit only the
# redundant schema declaration from the restore stream.
schema_declarations=$(awk '/^CREATE SCHEMA public;$/ { count++ } END { print count+0 }' "$baseline")
[[ "$schema_declarations" == 1 ]] || fail 'Approved baseline public schema declaration mismatch.'
restore_baseline() {
  local database=$1 stored
  create_local_db "$database"
  psql_local "$database" -c 'CREATE EXTENSION pg_trgm WITH SCHEMA public' >"$instance/setup.log" 2>&1 || fail 'Disposable pg_trgm setup failed.'
  local extension_schema
  extension_schema=$(psql_local "$database" -Atc "SELECT extnamespace::regnamespace::text FROM pg_extension WHERE extname = 'pg_trgm'" 2>"$instance/setup.log") || fail 'Disposable pg_trgm verification failed.'
  [[ "$extension_schema" == public ]] || fail 'Disposable pg_trgm verification failed.'
  sed '/^CREATE SCHEMA public;$/d' "$baseline" | psql_local "$database" >"$instance/setup.log" 2>&1 || fail 'Approved baseline restore failed.'
  psql_local "$database" -v marker="$marker" >"$instance/setup.log" 2>&1 <<'SQL'
CREATE SCHEMA test_infrastructure;
CREATE TABLE test_infrastructure.disposable_instance (
    singleton boolean PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    marker text NOT NULL CHECK (marker ~ '^[0-9a-f]{64}$')
);
INSERT INTO test_infrastructure.disposable_instance (marker) VALUES (:'marker');
SQL
  stored=$(psql_local "$database" -Atc 'SELECT marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE' 2>"$instance/setup.log") || fail 'Disposable instance marker verification failed.'
  [[ "$stored" == "$marker" ]] || fail 'Disposable instance marker verification failed.'
  verify_disposable_identity "$database"
  [[ -n "$workspace" ]] || prepare_generation_workspace
  regenerate_stage "$database" approved-baseline
  psql_local "$database" -f "$workspace/tests/testutil/shops_generation_fixture.sql" >"$instance/setup.log" 2>&1 || fail 'Disposable generation supplement failed.'
  regenerate_stage "$database" generation-supplement
}
apply_later_migrations() {
  local database=$1
  local migration
  for migration in "${later_migrations[@]}"; do
    verify_disposable_identity "$database"
    psql_local "$database" -f "$workspace/$migration" >"$instance/setup.log" 2>&1 || fail 'Post-baseline migration failed.'
    regenerate_stage "$database" "${migration##*/}"
  done
}
restore_baseline miltech_test_shops
apply_later_migrations miltech_test_shops
apply_remediation_migrations miltech_test_shops
if [[ "$verify_notification_migration" == true ]]; then
  restore_baseline miltech_test_upgrade_rehearsal
  psql_local miltech_test_upgrade_rehearsal -f "$workspace/tests/testutil/shops_notification_upgrade_seed.sql" >"$instance/setup.log" 2>&1 || fail 'Populated upgrade seed failed.'
  apply_later_migrations miltech_test_upgrade_rehearsal
  psql_local miltech_test_upgrade_rehearsal -f "$workspace/tests/testutil/shops_notification_upgrade_assert.sql" >"$instance/setup.log" 2>&1 || fail 'Populated upgrade assertion failed.'
  printf '%s\n' 'Populated notification upgrade passed.'

  restore_baseline miltech_test_empty_reverse
  apply_later_migrations miltech_test_empty_reverse
  psql_local miltech_test_empty_reverse -f "$workspace/migrations/016_rollback_shop_notification_operations.sql" >"$instance/setup.log" 2>&1 || fail 'Empty notification reverse failed.'
  regenerate_stage miltech_test_empty_reverse notification-reverse
  [[ $(psql_local miltech_test_empty_reverse -Atc "SELECT to_regclass('public.shop_notification_operations') IS NULL") == t ]] || fail 'Empty notification reverse left a ledger table.'
  printf '%s\n' 'Empty notification reverse passed.'

  restore_baseline miltech_test_populated_reverse
  apply_later_migrations miltech_test_populated_reverse
  psql_local miltech_test_populated_reverse >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO public.users (uid, email, username, created_at, is_enabled)
VALUES ('reverse-user', 'reverse-user@example.com', 'reverse-user', now(), true);
INSERT INTO public.shop_notification_operations
  (user_id, operation_id, fingerprint, notification_id, committed_at)
VALUES ('reverse-user', '00000000-0000-4000-8000-000000000201', decode(repeat('01', 32), 'hex'),
        '00000000-0000-4000-8000-000000000202', now());
SQL
  if psql_local miltech_test_populated_reverse -f "$workspace/migrations/016_rollback_shop_notification_operations.sql" >"$instance/reverse.log" 2>&1; then
    fail 'Populated notification reverse unexpectedly succeeded.'
  fi
  rg -q -F 'notification operation receipts must be retained' "$instance/reverse.log" || fail 'Populated notification reverse failed for the wrong reason.'
  [[ $(psql_local miltech_test_populated_reverse -Atc "SELECT count(*) FROM public.shop_notification_operations WHERE user_id = 'reverse-user'") == 1 ]] || fail 'Populated notification reverse lost its receipt.'
  printf '%s\n' 'Populated notification reverse refused and retained its receipt.'
fi
if [[ "$verify_message_sync_migration" == true ]]; then
  apply_migration_files() {
    local database=$1; shift
    local migration
    for migration in "$@"; do
      verify_disposable_identity "$database"
      psql_local "$database" -f "$workspace/$migration" >"$instance/setup.log" 2>&1 || fail 'Post-baseline migration failed.'
      regenerate_stage "$database" "${migration##*/}"
    done
  }
  restore_baseline miltech_test_msgsync_upgrade
  apply_migration_files miltech_test_msgsync_upgrade "${notification_migrations[@]}"
  psql_local miltech_test_msgsync_upgrade -f "$workspace/tests/testutil/shops_message_sync_upgrade_seed.sql" >"$instance/setup.log" 2>&1 || fail 'Message sync upgrade seed failed.'
  apply_migration_files miltech_test_msgsync_upgrade "${message_sync_migrations[@]}"
  psql_local miltech_test_msgsync_upgrade -f "$workspace/tests/testutil/shops_message_sync_upgrade_assert.sql" >"$instance/setup.log" 2>&1 || fail 'Message sync upgrade assertion failed.'
  printf '%s\n' 'Populated message sync upgrade passed.'

  restore_baseline miltech_test_msgsync_null
  apply_migration_files miltech_test_msgsync_null "${notification_migrations[@]}"
  psql_local miltech_test_msgsync_null >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO public.users (uid, email, username, created_at, is_enabled) VALUES ('null-user', 'n@example.com', 'n', now(), true);
INSERT INTO public.shops (id, name, created_by, created_at) VALUES ('shop-null', 'Null', 'null-user', now());
INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at) VALUES ('m-null', 'shop-null', 'null-user', 'x', NULL);
SQL
  if psql_local miltech_test_msgsync_null -f "$workspace/${message_sync_migrations[0]}" >"$instance/null.log" 2>&1; then
    fail 'Migration 018 unexpectedly accepted a NULL created_at.'
  fi
  rg -q -F 'NULL created_at' "$instance/null.log" || fail 'NULL created_at refusal failed for the wrong reason.'
  [[ $(psql_local miltech_test_msgsync_null -Atc "SELECT count(*) FROM information_schema.columns WHERE table_name='shop_messages' AND column_name='insertion_number'") == 0 ]] || fail 'Refused migration left a column behind.'
  printf '%s\n' 'Migration 018 refused NULL created_at and changed nothing.'

  restore_baseline miltech_test_msgsync_reverse
  apply_later_migrations miltech_test_msgsync_reverse
  psql_local miltech_test_msgsync_reverse -f "$workspace/migrations/018_rollback_shop_message_insertion_numbers.sql" >"$instance/setup.log" 2>&1 || fail 'Message sync reverse failed.'
  regenerate_stage miltech_test_msgsync_reverse message-sync-reverse
  [[ $(psql_local miltech_test_msgsync_reverse -Atc "SELECT to_regclass('public.shop_message_counters') IS NULL") == t ]] || fail 'Reverse left the counter table.'
  psql_local miltech_test_msgsync_reverse -f "$workspace/${message_sync_migrations[0]}" >"$instance/setup.log" 2>&1 || fail 'Migration 018 was not re-appliable after reverse.'
  regenerate_stage miltech_test_msgsync_reverse message-sync-reapply
  printf '%s\n' 'Message sync reverse and re-apply passed.'
fi
verify_final_candidate
if [[ "$publish_candidate_generated" == true ]]; then
  python3 "$root/scripts/publish-shops-candidate-generated.py" "$root" "$workspace" "$instance/pmcs-manifest.json" "$instance/source-manifest.json"
fi
export TEST_DATABASE_URL="postgres://postgres@127.0.0.1:$port/miltech_test_shops?sslmode=disable&connect_timeout=5"
export TEST_DATABASE_MARKER="$marker"
export SHOPS_GENERATION_REFERENCE_MANIFEST="$instance/pmcs-manifest.json"
cd "$workspace"
env GOWORK=off GOFLAGS= GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -p 1 "${go_arguments[@]}"
