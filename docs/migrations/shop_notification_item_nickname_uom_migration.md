# Shop Notification Item Nickname and Unit of Measure

**Migration Date**: TBD (apply before deploying the server build that reads these columns)
**Purpose**: Store an optional nickname and unit of measure on direct notification items, matching `shop_list_items`
**Repo files**: `migrations/017_add_shop_notification_item_nickname_uom.sql`, `migrations/017_rollback_shop_notification_item_nickname_uom.sql`

---

## Overview

Direct notification items (`shop_notification_items`) only stored NIIN,
nomenclature and quantity. The mobile app now lets users choose a unit of
measure (EA, DZ, BX, ...) and an optional nickname for these items, the same
way shop list items already work.

**Why the columns are nullable with no default**:
- `ADD COLUMN` with no default is a metadata-only change. There is no table
  rewrite, and the lock is held only briefly.
- Released app versions never send these fields. The server treats a missing
  field as "leave unchanged" on update and stores `NULL` on insert.
- Clients show a `NULL` unit as `EA` and a `NULL` nickname as empty. This is
  the same fallback already used for `shop_list_items`.

---

## Migration Instructions

1. Obtain separate owner authorization for the existing `miltech_ng_test`
   target and pin its current database/address/port/migration role/version/schema
   and application `DB_USERNAME`. Check identity and the approved SQL in the same
   application session. Prior 016 hashes do not identify a pre-017 schema.
2. Apply the reviewed 017 forward file to **`miltech_ng_test` first**. Immediately
   regenerate with the tagged `go run ./tools/jetregen` workflow using the
   separately verified intended source; never use the plain `jet` CLI. Compare
   all 32 tracked `user_pmcs_*` hashes, compile generated packages, and prove a
   real legacy application-role write and enrichment preservation.
3. Only after the test evidence is accepted, obtain a **separate** authorization
   and same-session identity/SQL checks for `miltech_ng`, then apply once and
   repeat tagged generation and verification. This document authorizes neither
   target contact nor application; current target identity/stage is UNPINNED.
4. Deploying, enabling flags, or reversing is a separate owner gate. A migration
   already present must be inspected, not blindly reapplied. `IF NOT EXISTS`
   alone does not prove compatible types, grants or historical data.

See [the remediation release gate sheet](../testing/shops-server-remediation-release.md).
The new runner covers 019–023 only; 017 requires an explicitly reviewed,
identity-checked target-specific procedure. SQL examples below preserve the
historical migration explanation and are not an executable target selection.

---

## Migration Script

```sql
BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_notification_items
  ADD COLUMN IF NOT EXISTS nickname text,
  ADD COLUMN IF NOT EXISTS unit_of_measure character varying(50);

COMMIT;
```

`IF NOT EXISTS` avoids duplicate-column errors but does not validate an existing
column definition. After a failure, establish the committed state with the owner
before an explicitly approved retry during a quieter period.

---

## Verification

```sql
SELECT column_name, data_type, character_maximum_length, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'shop_notification_items'
ORDER BY ordinal_position;
```

Expected new rows:

- `nickname`: `text`, max length `NULL`, nullable `YES`
- `unit_of_measure`: `character varying`, max length `50`, nullable `YES`

---

## Rollback

Reverse requires a separate explicit owner resolution authorization and a
retention/export manifest; it is never the default recovery action. Regenerate
through the tagged workflow immediately after any approved reverse or repair.
Roll back only **after** rolling back to a server build that doesn't
reference these columns. Dropping the columns deletes any nicknames and units
users have saved.

```sql
BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_notification_items
  DROP COLUMN IF EXISTS unit_of_measure,
  DROP COLUMN IF EXISTS nickname;

COMMIT;
```
