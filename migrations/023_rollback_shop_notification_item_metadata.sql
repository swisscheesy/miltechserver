BEGIN;
LOCK TABLE public.shop_notification_item_metadata IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.shop_notification_item_metadata) THEN
        RAISE EXCEPTION '023: populated retention reverse requires explicit resolution manifest';
    END IF;
END $$;
DROP TABLE public.shop_notification_item_metadata;
COMMIT;
