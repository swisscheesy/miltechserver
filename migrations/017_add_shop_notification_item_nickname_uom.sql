BEGIN;

SET LOCAL lock_timeout = '5s';

-- Nullable with no default: metadata-only (no table rewrite), and released
-- clients that never send these columns keep inserting successfully.
-- Types match shop_list_items.nickname / unit_of_measure.
ALTER TABLE public.shop_notification_items
  ADD COLUMN IF NOT EXISTS nickname text,
  ADD COLUMN IF NOT EXISTS unit_of_measure character varying(50);

COMMIT;
