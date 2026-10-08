BEGIN;

LOCK TABLE public.shop_vehicle_notifications, public.shop_notification_items IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF to_regclass('public.shop_notification_item_metadata') IS NOT NULL THEN
        RAISE EXCEPTION '023: retention table already exists';
    END IF;
    IF (SELECT count(*) FROM pg_attribute WHERE attrelid='public.shop_notification_items'::regclass
        AND attname IN ('id','notification_id','niin') AND atttypid='text'::regtype
        AND atttypmod=-1 AND attnotnull AND NOT attisdropped) <> 3
        OR (SELECT count(*) FROM pg_attribute WHERE attrelid='public.shop_notification_items'::regclass
            AND ((attname='nickname' AND atttypid='text'::regtype AND atttypmod=-1)
                OR (attname='unit_of_measure' AND atttypid='varchar'::regtype AND atttypmod=54))
            AND NOT attnotnull AND NOT attisdropped) <> 2 THEN
        RAISE EXCEPTION '023: unexpected item metadata column shape';
    END IF;
    IF EXISTS (SELECT 1 FROM public.shop_notification_items a JOIN public.shop_notification_items b
        ON a.notification_id=b.notification_id AND a.niin=b.niin
        WHERE a.nickname IS DISTINCT FROM b.nickname OR a.unit_of_measure IS DISTINCT FROM b.unit_of_measure) THEN
        RAISE EXCEPTION '023: conflicting historical metadata; explicit resolution manifest required';
    END IF;
END $$;

CREATE TABLE public.shop_notification_item_metadata (
    notification_id text NOT NULL REFERENCES public.shop_vehicle_notifications(id) ON DELETE CASCADE,
    niin text NOT NULL,
    nickname text,
    unit_of_measure varchar(50),
    state text NOT NULL CHECK (state IN ('resolved','ambiguous')),
    candidates jsonb NOT NULL CHECK (jsonb_typeof(candidates)='array'),
    version bigint NOT NULL CHECK (version > 0),
    -- Backfill is evidence, not an owner-accepted explicit resolution.
    resolution_version bigint NOT NULL DEFAULT 0 CHECK (resolution_version >= 0 AND resolution_version <= version),
    PRIMARY KEY (notification_id, niin)
);
-- Grouping raw nullable values cannot select a winner: the preflight proved
-- that every physical candidate for this exact logical key agrees.
INSERT INTO public.shop_notification_item_metadata
    (notification_id,niin,nickname,unit_of_measure,state,candidates,version)
SELECT notification_id,niin,nickname,unit_of_measure,'resolved',
    jsonb_agg(jsonb_build_object('item_id',id,'nickname',nickname,'unit_of_measure',unit_of_measure,'version',1) ORDER BY id),1
FROM public.shop_notification_items
GROUP BY notification_id,niin,nickname,unit_of_measure;
COMMIT;
