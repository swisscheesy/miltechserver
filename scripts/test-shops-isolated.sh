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
for argument in "$@"; do
  if [[ "$needs_value" == true ]]; then needs_value=false; continue; fi
  case "$argument" in
    ./tests/shops|./tests/equipment_services) packages=$((packages + 1)) ;;
    -race|-v|-failfast|-short) ;;
    -count|-run|-timeout|-parallel|-shuffle) needs_value=true ;;
    -count=*|-run=*|-timeout=*|-parallel=*|-shuffle=*) ;;
    *) fail 'Unsupported test argument or package; wrapper permits only ./tests/shops and ./tests/equipment_services.' ;;
  esac
done
[[ "$needs_value" == false && "$packages" -gt 0 ]] || fail 'A supported test package and complete flag values are required.'
root=$(cd "$(dirname "$0")/.." && pwd)
baseline="$root/tests/testutil/shops_schema_baseline.sql"
# Fill these only in a reviewed baseline-provenance change. No environment override.
baseline_sha256=''
later_migrations=()
[[ -n "$baseline_sha256" ]] || fail 'BLOCKED: owner-approved schema baseline and migration boundary are unavailable; see docs/testing/shops-database.md.'
[[ $(shasum -a 256 "$baseline" | cut -d ' ' -f 1) == "$baseline_sha256" ]] || fail 'Approved baseline checksum mismatch.'
for tool in initdb pg_ctl psql createdb python3 go; do
  command -v "$tool" >/dev/null || fail "Required local tool unavailable: $tool"
done
# Ignore inherited libpq connection defaults for local setup and tests.
for name in $(compgen -e); do
  case "$name" in PG*) unset "$name" ;; esac
done
umask 077
instance=$(mktemp -d /tmp/miltech-test.XXXXXXXX)
started=false
cleanup() {
  status=$?
  trap - EXIT INT TERM
  if [[ "$started" == true ]]; then
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
started=true
pg_ctl -D "$instance/data" -l "$instance/server.log" -o "-h 127.0.0.1 -p $port -k $instance" -w start >"$instance/setup.log" 2>&1 || fail 'Disposable PostgreSQL startup failed.'
# The private socket ties schema setup to this cluster even if TCP binding fails.
createdb -h "$instance" -p "$port" -U postgres miltech_test_shops >"$instance/setup.log" 2>&1 || fail 'Disposable database creation failed.'
psql_local() { psql -X -v ON_ERROR_STOP=1 -h "$instance" -p "$port" -U postgres -d miltech_test_shops "$@"; }
psql_local -f "$baseline" >"$instance/setup.log" 2>&1 || fail 'Approved baseline restore failed.'
# Bash 3.2 treats an empty array as unset under nounset.
for migration in ${later_migrations[@]+"${later_migrations[@]}"}; do
  psql_local -f "$root/$migration" >"$instance/setup.log" 2>&1 || fail 'Post-baseline migration failed.'
done
psql_local -v marker="$marker" >"$instance/setup.log" 2>&1 <<'SQL'
CREATE SCHEMA test_infrastructure;
CREATE TABLE test_infrastructure.disposable_instance (
    singleton boolean PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    marker text NOT NULL CHECK (marker ~ '^[0-9a-f]{64}$')
);
INSERT INTO test_infrastructure.disposable_instance (marker) VALUES (:'marker');
SQL
stored=$(psql_local -Atc 'SELECT marker FROM test_infrastructure.disposable_instance WHERE singleton = TRUE' 2>"$instance/setup.log") || fail 'Disposable instance marker verification failed.'
[[ "$stored" == "$marker" ]] || fail 'Disposable instance marker verification failed.'
export TEST_DATABASE_URL="postgres://postgres@127.0.0.1:$port/miltech_test_shops?sslmode=disable&connect_timeout=5"
export TEST_DATABASE_MARKER="$marker"
cd "$root"
go test "$@"
