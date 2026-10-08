BEGIN;
-- Match lifecycle resource order: assets, references, then cleanup jobs. Hold
-- every writer-blocking lock before the checks so no cleanup can start mid-way.
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shop_message_uploads IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_message_asset_references IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_message_blob_cleanup_jobs IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    -- Any queued, completed or in-flight cleanup is proof that must be retained.
    IF EXISTS (SELECT 1 FROM public.shop_message_uploads
               WHERE uploader_id = 'legacy:unattributed' AND state <> 'ready')
       OR EXISTS (SELECT 1 FROM public.shop_message_blob_cleanup_jobs j
                  JOIN public.shop_message_uploads u ON u.id = j.upload_id
                  WHERE u.uploader_id = 'legacy:unattributed') THEN
        RAISE EXCEPTION '024: legacy image cleanup has started; registry and cleanup proof must be retained';
    END IF;
END $$;

CREATE TEMPORARY TABLE legacy_message_image_uploads ON COMMIT DROP AS
SELECT id FROM public.shop_message_uploads WHERE uploader_id = 'legacy:unattributed';

DELETE FROM public.shop_message_asset_references
WHERE upload_id IN (SELECT id FROM legacy_message_image_uploads);
-- The 020 trigger queued one candidate per removed reference above. None of
-- them may reach storage: the blobs return to unregistered, protected status.
DELETE FROM public.shop_message_blob_cleanup_jobs
WHERE upload_id IN (SELECT id FROM legacy_message_image_uploads);
DELETE FROM public.shop_message_uploads
WHERE id IN (SELECT id FROM legacy_message_image_uploads);
COMMIT;
