-- Run only with SHOPS_MESSAGE_SYNC_ENABLED=false on every instance. Numbers
-- already served to clients as watermarks become meaningless after this; clients
-- re-initialise on their next load because the capability reverts to false.
-- ACCESS EXCLUSIVE: DROP COLUMN and DROP INDEX need it, so reads and writes of
-- shop_messages block until COMMIT. lock_timeout only bounds the wait to
-- acquire the lock, not how long it is held.
BEGIN;
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shop_messages IN ACCESS EXCLUSIVE MODE;
DROP TRIGGER IF EXISTS shop_messages_assign_insertion_number ON public.shop_messages;
DROP FUNCTION IF EXISTS public.assign_shop_message_insertion_number();
DROP INDEX IF EXISTS public.idx_shop_messages_shop_created_id;
DROP INDEX IF EXISTS public.shop_messages_shop_insertion_number_key;
DROP TABLE IF EXISTS public.shop_message_counters;
ALTER TABLE public.shop_messages DROP COLUMN IF EXISTS insertion_number;
COMMIT;
