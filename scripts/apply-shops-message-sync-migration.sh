#!/usr/bin/env bash
set -euo pipefail
set +x

fail() {
  printf 'Refused: %s\n' "$1" >&2
  exit 1
}

usage='Usage: apply-shops-message-sync-migration.sh {miltech_ng_test|miltech_ng} EXPECTED_SERVER_ADDRESS EXPECTED_PORT EXPECTED_ROLE EXPECTED_SCHEMA_SHA256'
[[ $# -eq 5 ]] || fail "$usage"

target=$1
expected_address=$2
expected_port=$3
expected_role=$4
expected_checksum=$5

# Pre-018 public-schema fingerprints (pg_dump 14.18 --schema-only --no-owner
# --no-acl --schema=public | shasum -a 256) must be captured read-only by the
# operator after migration 016 (and 017, if it is applied to that database) and
# recorded in docs/testing/shops-database.md before these two constants are
# replaced in one reviewed commit. They are deliberately NOT guessed: the
# pre-016 hashes do not describe a database that already has 016, and whether
# 017 is applied live is unrecorded.
unpinned_checksum=UNPINNED
case "$target" in
  miltech_ng_test) approved_checksum=$unpinned_checksum ;;
  miltech_ng) approved_checksum=$unpinned_checksum ;;
  *) fail 'Target database is not approved.' ;;
esac
# Refuse before any argument comparison, tool lookup or database contact.
[[ "$approved_checksum" != "$unpinned_checksum" ]] || fail "pre-018 schema checksum for $target has not been pinned; see docs/testing/shops-database.md"
[[ "$approved_checksum" =~ ^[0-9a-f]{64}$ ]] || fail "pinned pre-018 schema checksum for $target is malformed; see docs/testing/shops-database.md"

# Connection identity of the reviewed 2026-09-27 exports (unchanged since).
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
approved_migration_checksum=b0e8809dca1d8e109cc3072c13c9033f38f40bbe4a086bffd8322f57617777d7
umask 077
migration_dir=$(mktemp -d /private/tmp/shops-message-sync-migration.XXXXXXXX) || fail 'Could not create a private migration directory.'
schema_dump=
cleanup() {
  [[ -z "$schema_dump" ]] || rm -f "$schema_dump"
  rm -rf "$migration_dir"
}
trap cleanup EXIT
migration_sql="$migration_dir/018_add_shop_message_insertion_numbers.sql"
cp migrations/018_add_shop_message_insertion_numbers.sql "$migration_sql" 2>/dev/null || fail 'Fixed migration file is missing.'
chmod 600 "$migration_sql"
[[ $(shasum -a 256 "$migration_sql" | cut -d ' ' -f 1) == "$approved_migration_checksum" ]] || fail 'Fixed migration checksum differs from the reviewed SQL.'

psql_args=(-X -w -v ON_ERROR_STOP=1 -v "expected_database=$target" -v "expected_address=$expected_address" -v "expected_port=$expected_port" -v "expected_role=$expected_role")

emit_guard() {
  cat <<'SQL'
SELECT COALESCE(
  current_database() = :'expected_database'
  AND host(inet_server_addr()) = :'expected_address'
  AND inet_server_port() = :expected_port
  AND current_user = :'expected_role'
  AND current_setting('server_version_num')::integer = 140018
  AND to_regclass('public.shop_vehicle') IS NOT NULL
  AND to_regclass('public.shop_messages') IS NOT NULL
  AND to_regclass('public.shop_notification_operations') IS NOT NULL
  AND to_regclass('public.shop_message_counters') IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'shop_messages'
      AND column_name = 'insertion_number'
  )
  AND (
    SELECT NOT relrowsecurity FROM pg_class WHERE oid = to_regclass('public.shops')
  ) IS TRUE
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

schema_dump=$(mktemp "${TMPDIR:-/tmp}/shops-message-sync-schema.XXXXXXXX") || fail 'Could not create a private schema dump.'
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

printf 'Migration 018 applied to approved target %s.\n' "$target"
cat <<'NOTE'
Apply only in a low-traffic window: the migration blocks message reads and writes (and shops writes) until it commits.
If a run fails with "deadlock detected" or a lock timeout, the migration is one transaction, so nothing changed; it is safe to re-run.
Verify before proceeding (see docs/testing/shops-database.md, "Message sync migration 018"):
  - shop_message_counters row count equals the shops row count
  - count(*) WHERE insertion_number IS NULL on shop_messages is 0
  - shop_messages_assign_insertion_number trigger is enabled
  - both new indexes are valid
  - max(insertion_number) per shop equals its counter
  - shop_messages row count is unchanged
  - a message posted through the application is numbered, and the app role can write shop_message_counters
Capture the post-migration schema SHA-256 independently.
NOTE
