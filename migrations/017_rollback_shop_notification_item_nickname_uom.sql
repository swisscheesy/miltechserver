BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_notification_items
  DROP COLUMN IF EXISTS unit_of_measure,
  DROP COLUMN IF EXISTS nickname;

COMMIT;
