-- Run only with SHOPS_MESSAGE_SYNC_ENABLED=false on every instance. Numbers
-- already served to clients as watermarks become meaningless after this; clients
-- re-initialise on their next load because the capability reverts to false.
-- Lock order: shops first, then shop_messages, the same order as a cascading
-- DELETE FROM shops, so this cannot deadlock against one.
-- shops, ACCESS EXCLUSIVE: observed in a scratch probe to be what DROP TABLE of
-- the foreign-key-bearing counter table takes on shops (it removes the FK
-- triggers there). Reads and writes of shops block until COMMIT.
-- shop_messages, ACCESS EXCLUSIVE: DROP COLUMN and DROP INDEX need it. Reads and
-- writes of shop_messages block until COMMIT.
-- lock_timeout bounds each lock wait, not how long a lock is held. A timed-out
-- or deadlock-victim run rolls back atomically and can be re-run.
BEGIN;
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shops IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_messages IN ACCESS EXCLUSIVE MODE;
DROP TRIGGER IF EXISTS shop_messages_assign_insertion_number ON public.shop_messages;
DROP FUNCTION IF EXISTS public.assign_shop_message_insertion_number();
DROP INDEX IF EXISTS public.idx_shop_messages_shop_created_id;
DROP INDEX IF EXISTS public.shop_messages_shop_insertion_number_key;
DROP TABLE IF EXISTS public.shop_message_counters;
ALTER TABLE public.shop_messages DROP COLUMN IF EXISTS insertion_number;
COMMIT;
