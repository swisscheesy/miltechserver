BEGIN;

SET LOCAL lock_timeout = '5s';

-- ACCESS EXCLUSIVE: this migration blocks reads AND writes of shop_messages
-- until COMMIT. ADD COLUMN needs that lock anyway; the foreign key below takes
-- SHARE ROW EXCLUSIVE on shops, which blocks writes to shops (not reads); and the
-- backfill UPDATE and the non-concurrent CREATE INDEX statements run under
-- these locks. Taking the strongest lock up front makes the backfill, counter
-- seeding and trigger installation one gap-free transition: no message can be
-- committed unnumbered between the backfill and the trigger.
-- lock_timeout only bounds the wait to ACQUIRE the lock. It does not bound how
-- long the lock is held once acquired; the hold time is the duration of the
-- whole transaction (see docs/testing/shops-message-sync-measurements.md).
LOCK TABLE public.shop_messages IN ACCESS EXCLUSIVE MODE;

-- The sync reader treats NULL created_at as "unavailable"; refuse rather than
-- fabricate an ordering for such rows.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM public.shop_messages WHERE created_at IS NULL) THEN
        RAISE EXCEPTION 'shop_messages contains NULL created_at; resolve before enabling message sync';
    END IF;
END $$;

-- Nullable, no default: no table rewrite; released clients and binaries that
-- never mention the column keep inserting successfully.
ALTER TABLE public.shop_messages ADD COLUMN insertion_number bigint;

CREATE TABLE public.shop_message_counters (
    shop_id text PRIMARY KEY REFERENCES public.shops(id) ON DELETE CASCADE,
    last_number bigint NOT NULL DEFAULT 0 CHECK (last_number >= 0)
);

-- Deterministic backfill: per-shop order by (created_at, id), starting at 1.
UPDATE public.shop_messages AS m
SET insertion_number = ranked.number
FROM (
    SELECT id, row_number() OVER (PARTITION BY shop_id ORDER BY created_at, id) AS number
    FROM public.shop_messages
) AS ranked
WHERE m.id = ranked.id;

-- Every existing shop gets a counter, including shops with no messages.
INSERT INTO public.shop_message_counters (shop_id, last_number)
SELECT s.id, COALESCE(max(m.insertion_number), 0)
FROM public.shops AS s
LEFT JOIN public.shop_messages AS m ON m.shop_id = s.id
GROUP BY s.id;

-- Sole allocator. The counter increment commits or rolls back with the insert,
-- and its row lock is held to commit, so numbers are assigned in commit order.
-- Any value supplied by the writer is deliberately overwritten.
CREATE FUNCTION public.assign_shop_message_insertion_number() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
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

CREATE TRIGGER shop_messages_assign_insertion_number
    BEFORE INSERT ON public.shop_messages
    FOR EACH ROW EXECUTE FUNCTION public.assign_shop_message_insertion_number();

CREATE UNIQUE INDEX shop_messages_shop_insertion_number_key
    ON public.shop_messages (shop_id, insertion_number);

-- Serves the (created_at, id) tuple cursor and ORDER BY created_at DESC, id DESC.
CREATE INDEX idx_shop_messages_shop_created_id
    ON public.shop_messages (shop_id, created_at DESC, id DESC);

COMMIT;
