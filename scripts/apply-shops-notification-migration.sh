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
for selector in PGDATABASE PGHOST PGHOSTADDR PGPORT PGUSER; do
  [[ -z ${!selector+x} ]] || fail "Unset $selector; connection identity must come from PGSERVICE."
done

command -v psql >/dev/null || fail 'psql is required.'
command -v pg_dump >/dev/null || fail 'pg_dump is required.'
command -v shasum >/dev/null || fail 'shasum is required.'
command -v python3 >/dev/null || fail 'python3 is required to resolve the runner path.'

script_path=$(python3 -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")
repository_root=$(cd "$(dirname "$script_path")/.." && pwd -P)
cd "$repository_root"

# A private, hash-checked copy prevents a symlinked invocation or a later
# replacement of the source file from changing what the final psql session reads.
approved_migration_checksum=32b7a07f383aa1c6be028df865871bb0034e8df42d6f8e8474bff3aa139087de
umask 077
migration_dir=$(mktemp -d /private/tmp/shops-notification-migration.XXXXXXXX) || fail 'Could not create a private migration directory.'
schema_dump=
cleanup() {
  [[ -z "$schema_dump" ]] || rm -f "$schema_dump"
  rm -rf "$migration_dir"
}
trap cleanup EXIT
migration_sql="$migration_dir/016_create_shop_notification_operations.sql"
cp migrations/016_create_shop_notification_operations.sql "$migration_sql" 2>/dev/null || fail 'Fixed migration file is missing.'
chmod 600 "$migration_sql"
[[ $(shasum -a 256 "$migration_sql" | cut -d ' ' -f 1) == "$approved_migration_checksum" ]] || fail 'Fixed migration checksum differs from the reviewed SQL.'

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

schema_dump=$(mktemp "${TMPDIR:-/tmp}/shops-notification-schema.XXXXXXXX") || fail 'Could not create a private schema dump.'
pg_dump --schema-only --no-owner --no-acl --schema=public \
  --file="$schema_dump" 2>/dev/null || fail 'Schema export failed.'
actual_checksum=$(shasum -a 256 "$schema_dump" | cut -d ' ' -f 1)
[[ "$actual_checksum" == "$approved_checksum" ]] || fail 'Live schema fingerprint differs from the approved pre-migration export.'

# The final guard and the fixed include run on one psql connection. ON_ERROR_STOP
# exits before the include on any failed query or refused conditional branch.
[[ $(shasum -a 256 "$migration_sql" | cut -d ' ' -f 1) == "$approved_migration_checksum" ]] || fail 'Private migration copy changed before include.'
{
  emit_guard
  printf '\\i %s\n' "$migration_sql"
} | psql "${psql_args[@]}" -q || fail 'Migration session failed.'

printf 'Migration 016 applied to approved target %s. Verify the resulting schema and rows before proceeding.\n' "$target"
