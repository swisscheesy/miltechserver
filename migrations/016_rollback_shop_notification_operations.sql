BEGIN;
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shop_notification_operations IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM public.shop_notification_operations) THEN
        RAISE EXCEPTION 'notification operation receipts must be retained';
    END IF;
END $$;
DROP TABLE public.shop_notification_operations;
COMMIT;
