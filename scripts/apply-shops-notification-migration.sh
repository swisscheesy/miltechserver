#!/usr/bin/env bash
set -euo pipefail
set +x

fail() {
  printf 'Refused: %s\n' "$1" >&2
  exit 1
}

usage='Usage: apply-shops-notification-migration.sh {miltech_ng_test|miltech_ng} EXPECTED_SERVER_ADDRESS EXPECTED_PORT EXPECTED_ROLE EXPECTED_SCHEMA_SHA256'
[[ $# -eq 5 ]] || fail "$usage"

target=$1
expected_address=$2
expected_port=$3
expected_role=$4
expected_checksum=$5

case "$target" in
  miltech_ng_test) approved_checksum=9058c82a9a6de8b1215960c4ac38a5f19d8d79e552714781dbd8d92ff7130f70 ;;
  miltech_ng) approved_checksum=184eaa0cdb1f1671cbe4fb55eccdb0b5a2fce9bfb6ac7c858e6ab44e52c5deda ;;
  *) fail 'Target database is not approved.' ;;
esac

# These values belong to the independently reviewed 2026-09-27 exports.
[[ "$expected_address" == 192.168.20.70 ]] || fail 'Server address differs from the approved record.'
[[ "$expected_port" == 5432 ]] || fail 'Port differs from the approved record.'
[[ "$expected_role" == postgres ]] || fail 'Role differs from the approved record.'
[[ "$expected_checksum" == "$approved_checksum" ]] || fail 'Schema checksum differs from the approved target record.'
[[ -n ${PGSERVICE:-} && "$PGSERVICE" =~ ^[A-Za-z_][A-Za-z0-9_.-]*$ ]] || fail 'An operator-provisioned PGSERVICE name is required.'

command -v psql >/dev/null || fail 'psql is required.'
command -v pg_dump >/dev/null || fail 'pg_dump is required.'
command -v shasum >/dev/null || fail 'shasum is required.'

repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$repository_root"
[[ -f migrations/016_create_shop_notification_operations.sql ]] || fail 'Fixed migration file is missing.'

psql_args=(-X -w -v ON_ERROR_STOP=1 -v "expected_database=$target" -v "expected_address=$expected_address" -v "expected_port=$expected_port" -v "expected_role=$expected_role")

emit_guard() {
  cat <<'SQL'
SELECT COALESCE(
  current_database() = :'expected_database'
  AND inet_server_addr()::text = :'expected_address'
  AND inet_server_port() = :expected_port
  AND current_user = :'expected_role'
  AND current_setting('server_version_num')::integer = 140018
  AND to_regclass('public.shop_vehicle') IS NOT NULL
  AND to_regclass('public.shop_notification_operations') IS NULL
  AND (
    SELECT count(*) = 2
    FROM pg_constraint
    WHERE conrelid = 'public.shop_vehicle'::regclass
      AND contype = 'c'
      AND convalidated
      AND conname IN (
        'shop_vehicle_tracked_mileage_nonnegative',
        'shop_vehicle_tracked_hours_nonnegative'
      )
  ), false) AS approved \gset
\if :approved
\else
\echo 'Refused: live connection identity or migration prerequisites differ from the approved record.'
SELECT 1 / 0;
\endif
SQL
}

# Check identity before dumping so a misdirected service cannot even inspect an
# unintended schema. The guard is repeated in the migration psql session.
emit_guard | psql "${psql_args[@]}" -q || fail 'Initial connection guard failed.'

umask 077
schema_dump=$(mktemp "${TMPDIR:-/tmp}/shops-notification-schema.XXXXXXXX") || fail 'Could not create a private schema dump.'
trap 'rm -f "$schema_dump"' EXIT
pg_dump --schema-only --no-owner --no-acl --schema=public \
  --file="$schema_dump" 2>/dev/null || fail 'Schema export failed.'
actual_checksum=$(shasum -a 256 "$schema_dump" | cut -d ' ' -f 1)
[[ "$actual_checksum" == "$approved_checksum" ]] || fail 'Live schema fingerprint differs from the approved pre-migration export.'

# The final guard and the fixed include run on one psql connection. ON_ERROR_STOP
# exits before the include on any failed query or refused conditional branch.
{
  emit_guard
  printf '%s\n' '\i migrations/016_create_shop_notification_operations.sql'
} | psql "${psql_args[@]}" -q || fail 'Migration session failed.'

printf 'Migration 016 applied to approved target %s. Verify the resulting schema and rows before proceeding.\n' "$target"
