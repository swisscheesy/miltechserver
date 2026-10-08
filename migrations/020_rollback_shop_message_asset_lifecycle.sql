BEGIN;
-- Match lifecycle resource order: assets, references, then cleanup jobs.
-- Hold all writer-blocking locks before observing emptiness. A concurrent
-- reservation/cascade/claim must serialize here or cause bounded refusal.
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shop_message_uploads IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_message_asset_references IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_message_blob_cleanup_jobs IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
    IF EXISTS(SELECT 1 FROM public.shop_message_uploads) OR
       EXISTS(SELECT 1 FROM public.shop_message_asset_references) OR
       EXISTS(SELECT 1 FROM public.shop_message_blob_cleanup_jobs) THEN
        RAISE EXCEPTION 'message asset registry and cleanup proof must be retained';
    END IF;
END $$;
DROP TABLE public.shop_message_asset_references;
DROP FUNCTION public.enqueue_shop_message_asset_candidate();
DROP TABLE public.shop_message_blob_cleanup_jobs;
DROP FUNCTION public.protect_shop_message_cleanup_target();
DROP TABLE public.shop_message_uploads;
DROP FUNCTION public.protect_shop_message_upload_target();
ALTER TABLE public.shop_messages DROP CONSTRAINT shop_messages_id_shop_unique;
COMMIT;
