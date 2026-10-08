BEGIN;

-- Registers pre-020 message images so deleting or editing their messages queues
-- blob cleanup like any managed upload. Only data changes; no schema changes.
-- EXCLUSIVE leaves plain reads running but waits out every writer and row-lock
-- holder first, so FK checks below cannot deadlock against a message edit.
-- Order follows the runtime: messages, then uploads, then references.
SET LOCAL lock_timeout = '5s';
LOCK TABLE public.shop_messages IN EXCLUSIVE MODE;
LOCK TABLE public.shop_message_uploads IN EXCLUSIVE MODE;
LOCK TABLE public.shop_message_asset_references IN EXCLUSIVE MODE;

-- The pre-020 server named blobs {shop_id}/{uuid.New()}{extension}, which is the
-- exact key shape the cleanup worker accepts, so the key's UUID becomes the
-- upload ID and no blob is renamed. Only this canonical marker form is accepted.
CREATE TEMPORARY TABLE legacy_message_image_markers ON COMMIT DROP AS
SELECT m.id AS message_id,
       m.shop_id AS message_shop_id,
       m.created_at AS message_created_at,
       marker[1] AS account,
       marker[2] AS key_shop_id,
       marker[3] AS upload_id,
       marker[4] AS extension
FROM public.shop_messages m
CROSS JOIN LATERAL regexp_matches(
    m.message,
    '\[IMAGE:https://([a-z0-9][a-z0-9-]*)\.blob\.core\.windows\.net/shop-message-images/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(\.jpg|\.png|\.gif|\.webp)\]',
    'g') AS marker;

DO $$
BEGIN
    -- A missed reference would let the worker delete a blob that a message still
    -- shows, so any container mention the strict pattern did not capture refuses.
    IF EXISTS (
        SELECT 1
        FROM public.shop_messages m
        WHERE (length(lower(m.message)) - length(replace(lower(m.message), 'shop-message-images', '')))
              / length('shop-message-images')
           <> (SELECT count(*) FROM legacy_message_image_markers k WHERE k.message_id = m.id)) THEN
        RAISE EXCEPTION '024: non-canonical historical image reference; explicit mapping required';
    END IF;
    -- References are same-shop by FK; a copied cross-shop link has no safe owner.
    IF EXISTS (SELECT 1 FROM legacy_message_image_markers WHERE key_shop_id <> message_shop_id) THEN
        RAISE EXCEPTION '024: historical image path belongs to a different shop; explicit mapping required';
    END IF;
    IF EXISTS (
        SELECT 1 FROM legacy_message_image_markers
        GROUP BY upload_id
        HAVING count(DISTINCT (account, key_shop_id, extension)) > 1) THEN
        RAISE EXCEPTION '024: conflicting storage targets for one historical image identity';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM legacy_message_image_markers k
        JOIN public.shop_message_uploads u ON u.id = k.upload_id::uuid
        WHERE (u.account, u.container, u.blob_key)
              IS DISTINCT FROM (k.account, 'shop-message-images', k.key_shop_id || '/' || k.upload_id || k.extension)) THEN
        RAISE EXCEPTION '024: historical image identity is registered to a different storage target';
    END IF;
END $$;

-- Targets the registry already owns are maintained by the runtime; this also
-- makes a repeated run a no-op.
CREATE TEMPORARY TABLE legacy_message_image_candidates ON COMMIT DROP AS
SELECT k.*
FROM legacy_message_image_markers k
WHERE NOT EXISTS (
    SELECT 1 FROM public.shop_message_uploads u
    WHERE u.account = k.account
      AND u.container = 'shop-message-images'
      AND u.blob_key = k.key_shop_id || '/' || k.upload_id || k.extension);

DO $$
BEGIN
    -- Every message naming a new identity must hold a reference to it; otherwise
    -- deleting the referencing messages could remove an image still on display.
    IF EXISTS (
        SELECT 1
        FROM (SELECT DISTINCT upload_id FROM legacy_message_image_candidates) c
        JOIN public.shop_messages m ON strpos(lower(m.message), c.upload_id) > 0
        WHERE NOT EXISTS (
            SELECT 1 FROM legacy_message_image_candidates k
            WHERE k.upload_id = c.upload_id AND k.message_id = m.id)) THEN
        RAISE EXCEPTION '024: historical image identity appears outside a canonical marker; explicit mapping required';
    END IF;
END $$;

-- Uploader is deliberately not inferred from message authors. The placeholder
-- cannot equal a Firebase UID, so only a current admin may discard an image once
-- it is unreferenced; while referenced, nobody can. The lease is already expired.
INSERT INTO public.shop_message_uploads
    (id, operation_id, shop_id, uploader_id, account, container, blob_key, url,
     extension, state, lease_until, created_at, updated_at, failure)
SELECT c.upload_id::uuid,
       gen_random_uuid(),
       c.key_shop_id,
       'legacy:unattributed',
       c.account,
       'shop-message-images',
       c.key_shop_id || '/' || c.upload_id || c.extension,
       'https://' || c.account || '.blob.core.windows.net/shop-message-images/' || c.key_shop_id || '/' || c.upload_id || c.extension,
       c.extension,
       'ready',
       COALESCE(min(c.message_created_at), now()),
       COALESCE(min(c.message_created_at), now()),
       now(),
       ''
FROM legacy_message_image_candidates c
GROUP BY c.upload_id, c.key_shop_id, c.account, c.extension;

INSERT INTO public.shop_message_asset_references (message_id, upload_id, shop_id)
SELECT DISTINCT c.message_id, c.upload_id::uuid, c.message_shop_id
FROM legacy_message_image_candidates c;
COMMIT;
