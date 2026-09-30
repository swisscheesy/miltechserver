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

1. Apply to the local `miltech_ng` database, then regenerate jet models:
   `jet -dsn="postgresql://postgres:<password>@localhost:5432/miltech_ng?sslmode=disable" -schema=public -path=./.gen`
2. Apply to `miltech_ng_test`.
3. Apply to production **before** deploying the server build that reads the
   new columns.
4. Run the verification query after each apply.

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

`IF NOT EXISTS` makes the script safe to re-run. If it fails with a lock
timeout, retry it during a quieter period.

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
