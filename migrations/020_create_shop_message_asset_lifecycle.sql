BEGIN;

-- Existing targets are deliberately not inferred from historical message text.
CREATE TABLE public.shop_message_uploads (
    id uuid PRIMARY KEY,
    operation_id uuid NOT NULL UNIQUE,
    shop_id text NOT NULL,
    uploader_id text NOT NULL,
    account text NOT NULL CHECK (account <> ''),
    container text NOT NULL CHECK (container = 'shop-message-images'),
    blob_key text NOT NULL CHECK (blob_key <> ''),
    url text NOT NULL,
    extension text NOT NULL CHECK (extension IN ('.jpg','.png','.gif','.webp')),
    state text NOT NULL CHECK (state IN ('uploading','ready','cleanup_pending','deleting','deleted')),
    lease_until timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    failure text NOT NULL DEFAULT '',
    UNIQUE (id, shop_id),
    UNIQUE (account, container, blob_key)
);
CREATE INDEX shop_message_uploads_shop_idx ON public.shop_message_uploads(shop_id, id);
CREATE INDEX shop_message_uploads_reconcile_idx ON public.shop_message_uploads(state, lease_until);
ALTER TABLE public.shop_messages ADD CONSTRAINT shop_messages_id_shop_unique UNIQUE(id,shop_id);
CREATE TABLE public.shop_message_asset_references (
    message_id text NOT NULL,
    upload_id uuid NOT NULL,
    shop_id text NOT NULL,
    PRIMARY KEY(message_id,upload_id),
    FOREIGN KEY(message_id,shop_id) REFERENCES public.shop_messages(id,shop_id) ON DELETE CASCADE,
    FOREIGN KEY(upload_id,shop_id) REFERENCES public.shop_message_uploads(id,shop_id) ON DELETE RESTRICT
);
CREATE INDEX shop_message_asset_references_upload_idx ON public.shop_message_asset_references(upload_id);
CREATE TABLE public.shop_message_blob_cleanup_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id uuid,
    shop_id text NOT NULL,
    scope text NOT NULL CHECK (scope IN ('asset','shop_prefix')),
    account text NOT NULL,
    container text NOT NULL CHECK (container = 'shop-message-images'),
    blob_key text NOT NULL,
    reason text NOT NULL,
    state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','leased','retryable','completed','manual-review')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt timestamptz NOT NULL DEFAULT now(),
    lease_owner text NOT NULL DEFAULT '',
    lease_until timestamptz,
    failure text NOT NULL DEFAULT '',
    cursor text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((scope='asset' AND upload_id IS NOT NULL) OR (scope='shop_prefix' AND upload_id IS NULL)),
    CHECK (account <> '' AND blob_key <> '')
);
CREATE INDEX shop_message_blob_cleanup_jobs_due_idx ON public.shop_message_blob_cleanup_jobs(next_attempt,id) WHERE state IN ('pending','retryable');
CREATE INDEX shop_message_blob_cleanup_jobs_lease_idx ON public.shop_message_blob_cleanup_jobs(lease_until) WHERE state='leased';
CREATE INDEX shop_message_blob_cleanup_jobs_upload_idx ON public.shop_message_blob_cleanup_jobs(upload_id);

CREATE FUNCTION public.protect_shop_message_upload_target() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.id,NEW.operation_id,NEW.shop_id,NEW.uploader_id,NEW.account,NEW.container,NEW.blob_key,NEW.url,NEW.extension)
       IS DISTINCT FROM
       (OLD.id,OLD.operation_id,OLD.shop_id,OLD.uploader_id,OLD.account,OLD.container,OLD.blob_key,OLD.url,OLD.extension) THEN
        RAISE EXCEPTION 'message upload target is immutable';
    END IF;
    IF NEW.state <> OLD.state AND NOT (
        (OLD.state='uploading' AND NEW.state IN ('ready','cleanup_pending')) OR
        (OLD.state='ready' AND NEW.state='cleanup_pending') OR
        (OLD.state='cleanup_pending' AND NEW.state='deleting') OR
        (OLD.state='deleting' AND NEW.state='deleted')) THEN
        RAISE EXCEPTION 'invalid message upload state transition';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER shop_message_upload_target_immutable BEFORE UPDATE ON public.shop_message_uploads
FOR EACH ROW EXECUTE FUNCTION public.protect_shop_message_upload_target();

CREATE FUNCTION public.protect_shop_message_cleanup_target() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (NEW.id,NEW.upload_id,NEW.shop_id,NEW.scope,NEW.account,NEW.container,NEW.blob_key)
       IS DISTINCT FROM (OLD.id,OLD.upload_id,OLD.shop_id,OLD.scope,OLD.account,OLD.container,OLD.blob_key) THEN
        RAISE EXCEPTION 'message cleanup target is immutable';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER shop_message_cleanup_target_immutable BEFORE UPDATE ON public.shop_message_blob_cleanup_jobs
FOR EACH ROW EXECUTE FUNCTION public.protect_shop_message_cleanup_target();

-- Invoker security makes missing queue privileges fail the deleting transaction.
-- No Shop/asset lock is taken here: FK/account cascades must not invert lock order.
CREATE FUNCTION public.enqueue_shop_message_asset_candidate() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO public.shop_message_blob_cleanup_jobs(upload_id,shop_id,scope,account,container,blob_key,reason)
    SELECT id,shop_id,'asset',account,container,blob_key,'reference_removed'
    FROM public.shop_message_uploads WHERE id=OLD.upload_id AND shop_id=OLD.shop_id;
    RETURN OLD;
END $$;
CREATE TRIGGER shop_message_asset_reference_removed AFTER DELETE ON public.shop_message_asset_references
FOR EACH ROW EXECUTE FUNCTION public.enqueue_shop_message_asset_candidate();
COMMIT;
