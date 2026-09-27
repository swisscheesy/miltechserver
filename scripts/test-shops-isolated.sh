#!/usr/bin/env bash
set -euo pipefail
# Never trace generated connection arguments, including under bash -x.
set +x
fail() { printf '%s\n' "$1" >&2; exit 1; }
[[ $# -gt 0 ]] || fail 'Usage: scripts/test-shops-isolated.sh <go test arguments>'
[[ -z ${TEST_DATABASE_URL+x} && -z ${TEST_DATABASE_MARKER+x} ]] || fail 'Refusing externally supplied test database configuration; unset TEST_DATABASE_URL and TEST_DATABASE_MARKER.'
# Only these TestMain paths are in this wrapper's integration acceptance scope.
# Never allow ./... to reach independent dotenv-based database setups.
packages=0
needs_value=false
verify_notification_migration=false
go_arguments=()
for argument in "$@"; do
  if [[ "$needs_value" == true ]]; then needs_value=false; go_arguments+=("$argument"); continue; fi
  case "$argument" in
    --verify-notification-migration) verify_notification_migration=true; continue ;;
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
baseline="$root/tests/testutil/shops_schema_baseline.sql"
# The approved public-schema export is pinned; no environment override.
baseline_sha256='9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70'
# The approved physical snapshot has the validated migration-015 constraints.
# Only migration 016 follows it; history of earlier source files is absent.
later_migrations=("migrations/016_create_shop_notification_operations.sql")
[[ $(shasum -a 256 "$baseline" | cut -d ' ' -f 1) == "$baseline_sha256" ]] || fail 'Approved baseline checksum mismatch.'
for tool in initdb pg_ctl psql createdb python3 go awk sed rg; do
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
  local database=$1
  create_local_db "$database"
  psql_local "$database" -c 'CREATE EXTENSION pg_trgm WITH SCHEMA public' >"$instance/setup.log" 2>&1 || fail 'Disposable pg_trgm setup failed.'
  local extension_schema
  extension_schema=$(psql_local "$database" -Atc "SELECT extnamespace::regnamespace::text FROM pg_extension WHERE extname = 'pg_trgm'" 2>"$instance/setup.log") || fail 'Disposable pg_trgm verification failed.'
  [[ "$extension_schema" == public ]] || fail 'Disposable pg_trgm verification failed.'
  sed '/^CREATE SCHEMA public;$/d' "$baseline" | psql_local "$database" >"$instance/setup.log" 2>&1 || fail 'Approved baseline restore failed.'
}
apply_later_migrations() {
  local database=$1
  local migration
  for migration in "${later_migrations[@]}"; do
    psql_local "$database" -f "$root/$migration" >"$instance/setup.log" 2>&1 || fail 'Post-baseline migration failed.'
  done
}
restore_baseline miltech_test_shops
apply_later_migrations miltech_test_shops
if [[ "$verify_notification_migration" == true ]]; then
  restore_baseline miltech_upgrade_rehearsal
  psql_local miltech_upgrade_rehearsal -f "$root/tests/testutil/shops_notification_upgrade_seed.sql" >"$instance/setup.log" 2>&1 || fail 'Populated upgrade seed failed.'
  apply_later_migrations miltech_upgrade_rehearsal
  psql_local miltech_upgrade_rehearsal -f "$root/tests/testutil/shops_notification_upgrade_assert.sql" >"$instance/setup.log" 2>&1 || fail 'Populated upgrade assertion failed.'
  printf '%s\n' 'Populated notification upgrade passed.'

  restore_baseline miltech_empty_reverse
  apply_later_migrations miltech_empty_reverse
  psql_local miltech_empty_reverse -f "$root/migrations/016_rollback_shop_notification_operations.sql" >"$instance/setup.log" 2>&1 || fail 'Empty notification reverse failed.'
  [[ $(psql_local miltech_empty_reverse -Atc "SELECT to_regclass('public.shop_notification_operations') IS NULL") == t ]] || fail 'Empty notification reverse left a ledger table.'
  printf '%s\n' 'Empty notification reverse passed.'

  restore_baseline miltech_populated_reverse
  apply_later_migrations miltech_populated_reverse
  psql_local miltech_populated_reverse >"$instance/setup.log" 2>&1 <<'SQL'
INSERT INTO public.users (uid, email, username, created_at, is_enabled)
VALUES ('reverse-user', 'reverse-user@example.com', 'reverse-user', now(), true);
INSERT INTO public.shop_notification_operations
  (user_id, operation_id, fingerprint, notification_id, committed_at)
VALUES ('reverse-user', '00000000-0000-4000-8000-000000000201', decode(repeat('01', 32), 'hex'),
        '00000000-0000-4000-8000-000000000202', now());
SQL
  if psql_local miltech_populated_reverse -f "$root/migrations/016_rollback_shop_notification_operations.sql" >"$instance/reverse.log" 2>&1; then
    fail 'Populated notification reverse unexpectedly succeeded.'
  fi
  rg -q -F 'notification operation receipts must be retained' "$instance/reverse.log" || fail 'Populated notification reverse failed for the wrong reason.'
  [[ $(psql_local miltech_populated_reverse -Atc "SELECT count(*) FROM public.shop_notification_operations WHERE user_id = 'reverse-user'") == 1 ]] || fail 'Populated notification reverse lost its receipt.'
  printf '%s\n' 'Populated notification reverse refused and retained its receipt.'
fi
psql_local miltech_test_shops -v marker="$marker" >"$instance/setup.log" 2>&1 <<'SQL'
CREATE SCHEMA test_infrastructure;
CREATE TABLE test_infrastructure.disposable_instance (
    singleton boolean PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    marker text NOT NULL CHECK (marker ~ '^[0-9a-f]{64}$')
);
INSERT INTO test_infrastructure.disposable_instance (marker) VALUES (:'marker');
SQL
stored=$(psql_local miltech_test_shops -Atc 'SELECT marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE' 2>"$instance/setup.log") || fail 'Disposable instance marker verification failed.'
[[ "$stored" == "$marker" ]] || fail 'Disposable instance marker verification failed.'
export TEST_DATABASE_URL="postgres://postgres@127.0.0.1:$port/miltech_test_shops?sslmode=disable&connect_timeout=5"
export TEST_DATABASE_MARKER="$marker"
cd "$root"
go test "${go_arguments[@]}"
