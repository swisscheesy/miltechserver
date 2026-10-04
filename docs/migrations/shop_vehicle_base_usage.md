# Shop vehicle base usage integrity (migration 022)

Migration 022 adds validated `shop_vehicle_mileage_nonnegative` and
`shop_vehicle_hours_nonnegative` checks on `public.shop_vehicle`. Base readings
are non-null integers with zero defaults. Tracked readings retain their nullable
meaning and the existing migration-015 checks, names, definitions and identities.
The forward and reverse scripts verify this actual column/constraint shape.

## Historical gate

Inventory the actual target before approving any application:

```sql
SELECT id, shop_id, mileage, hours
FROM public.shop_vehicle
WHERE mileage < 0 OR hours < 0
ORDER BY shop_id, id;
```

The forward script obtains an exclusive table lock, checks shape and the
migration-015 constraints, rejects existing 022 checks, and refuses every negative
historical base reading in the same transaction. It never sets a reading to zero,
reinterprets a tracked reading, or repairs a row. A refusal rolls back completely.
An invalid historical row remains usable for unrelated tracked writes before
hardening; deployment must wait for a separately authorized repair manifest.
`NOT VALID` is deliberately not used to sidestep the inventory gate: a new check
would still reject unrelated updates to an invalid row.

An actual repair manifest must identify each row, old values, replacement values,
and evidence for the replacements. This task authorizes no target inventory,
repair, or migration. Schedule the exclusive lock and obtain separate approval
for each existing named target, with same-session identity checks.

## Forward, reverse and generation

Apply `migrations/022_add_shop_vehicle_base_usage_constraints.sql` after the
approved migration prefix. Both checks are fully validated in the forward
transaction. Repeated forward refuses. The reverse script verifies the exact
022 definitions and removes only its two checks; it refuses missing or drifted
checks. It preserves all rows and the 015 checks. Repeated reverse refuses;
forward is re-applicable after a valid reverse. An interrupted reverse rolls back.

Use `go run ./tools/jetregen` through the tagged generation workflow after every
migration/rehearsal/repair. Never run plain Jet or hand-edit generated files.
Check all 32 tracked `user_pmcs_*` generated files byte-for-byte; stop on drift.
022 changes constraints only, so it adds no generated model imports. The
throwaway wrapper regenerates fresh intermediate and candidate model packages,
then builds the full candidate. Worktree canonical generation need not be
published for this constraint-only change.

The authorized disposable verification command is:

```sh
rtk proxy env -u TEST_DATABASE_URL -u TEST_DATABASE_MARKER -u TEST_DB_URL \
  bash scripts/test-shops-isolated.sh \
  --verify-notification-migration --verify-message-sync-migration \
  --verify-remediation-migrations -v -count=1 \
  ./tests/shops ./tests/equipment_services
```

The 022 verifier covers populated forward/reverse/reapply, interrupted reverse,
negative mileage/hours refusal with identical schema/data, incorrect types,
nullability/defaults, 015 definition drift, repeat application and reverse
constraint drift. Intentionally incompatible type fixtures and their failing
migration run inside one rollback transaction; normal compatibility generation
runs afterwards rather than bypassing its guards. No existing target, real
Azure, credentials or capability flags are involved.

## Request behavior

The server retains the existing JSON names, value types and response envelopes.
At least one non-null tracked reading with absent/empty NIIN, model, serial, UOC
and comment selects legacy usage intent before role branching. Every member,
creator and admin then writes tracked readings only; Admin/base snapshots are
ignored, including empty Admin and negative base snapshots.

Other edits require current creator/admin authority and SET only supplied
non-null metadata fields, including Admin. Null/omitted metadata and null/omitted
tracked values preserve the row. NIIN/model/serial/comment may explicitly clear;
empty UOC keeps the established `UNK` default; actual empty Admin edits remain
invalid. Authorized mixed metadata and non-null tracked edits remain one atomic
write under the existing current-authority transaction. New base values on
create and actual metadata intent must be nonnegative. Zero is valid.

Presence preservation does not add version/conflict resolution for stale
absolute PUT usage versus newer PATCH adjustments. Effective-reading fallback,
int32 bounds and usage adjustment arithmetic remain unchanged. Full host tests,
physical migrations, fleet compatibility, named target repairs, signed artifacts,
released clients and deployment are separate acceptance gates.
