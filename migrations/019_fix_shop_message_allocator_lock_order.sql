-- Forward accepts only the pinned 018 function body. No data or grants change.
BEGIN;
SET LOCAL lock_timeout = '5s';

-- Match cascading deletion and fence writers while validating the allocator.
-- External counter-first writers must be fenced before this migration starts.
LOCK TABLE public.shops IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE public.shop_messages IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.shop_message_counters IN ACCESS EXCLUSIVE MODE;

DO $guard$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc p JOIN pg_language l ON l.oid = p.prolang
        WHERE p.oid = to_regprocedure('public.assign_shop_message_insertion_number()')
          AND md5(p.prosrc) = '5b2f595523c9653cf80ec52148537550'
          AND l.lanname = 'plpgsql' AND p.prorettype = 'trigger'::regtype
          AND NOT p.prosecdef AND p.proconfig IS NULL AND p.provolatile = 'v'
    ) THEN
        RAISE EXCEPTION '019 refuses unexpected allocator definition or execution settings';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgrelid = 'public.shop_messages'::regclass
          AND tgname = 'shop_messages_assign_insertion_number'
          AND tgfoid = 'public.assign_shop_message_insertion_number()'::regprocedure
          AND tgtype = 7 AND tgenabled = 'O' AND NOT tgisinternal
          AND tgnargs = 0 AND tgqual IS NULL
    ) THEN
        RAISE EXCEPTION '019 refuses unexpected allocator trigger';
    END IF;
    IF EXISTS (
        SELECT 1 FROM public.shop_messages m
        LEFT JOIN public.shop_message_counters c ON c.shop_id = m.shop_id
        WHERE m.insertion_number IS NULL OR m.insertion_number <= 0
           OR c.shop_id IS NULL OR c.last_number < m.insertion_number
    ) OR EXISTS (
        SELECT 1 FROM public.shop_message_counters c
        LEFT JOIN public.shops s ON s.id = c.shop_id
        WHERE s.id IS NULL OR c.last_number IS NULL OR c.last_number < 0
    ) OR EXISTS (
        SELECT 1 FROM public.shop_messages
        GROUP BY shop_id, insertion_number HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION '019 refuses invalid message numbering or counters; reviewed repair required';
    END IF;
END $guard$;

CREATE OR REPLACE FUNCTION public.assign_shop_message_insertion_number() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER AS $$
BEGIN
    -- Legacy inserts must join the same Shop-before-counter order as current
    -- writers. KEY SHARE conflicts with their FOR UPDATE authorization lock.
    PERFORM 1 FROM public.shops WHERE id = NEW.shop_id FOR KEY SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'shop does not exist' USING ERRCODE = '23503';
    END IF;

    INSERT INTO public.shop_message_counters (shop_id, last_number)
    VALUES (NEW.shop_id, 0)
    ON CONFLICT (shop_id) DO NOTHING;

    UPDATE public.shop_message_counters
    SET last_number = last_number + 1
    WHERE shop_id = NEW.shop_id
    RETURNING last_number INTO NEW.insertion_number;

    -- Fail closed: NULL only if the counter row vanished between the upsert
    -- and the UPDATE. An unnumbered message would be invisible to catch-up.
    IF NEW.insertion_number IS NULL THEN
        RAISE EXCEPTION 'shop_message_counters row missing for shop %', NEW.shop_id;
    END IF;

    RETURN NEW;
END $$;

COMMIT;
