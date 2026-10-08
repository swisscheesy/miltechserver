BEGIN;

-- Prevent writers from changing the historical relationship during preflight.
LOCK TABLE public.shop_messages IN ACCESS EXCLUSIVE MODE;
DO $$
DECLARE
    id_column smallint;
    shop_column smallint;
    parent_column smallint;
BEGIN
    IF (SELECT count(*) FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass
        AND attname IN ('id','shop_id','parent_id') AND atttypid='text'::regtype AND atttypmod=-1 AND NOT attisdropped) <> 3
        OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='id' AND attnotnull)
        OR NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='shop_id' AND attnotnull)
        OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='parent_id' AND attnotnull) THEN
        RAISE EXCEPTION '021: unexpected message identity column types or nullability';
    END IF;
    SELECT attnum INTO id_column FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='id';
    SELECT attnum INTO shop_column FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='shop_id';
    SELECT attnum INTO parent_column FROM pg_attribute WHERE attrelid='public.shop_messages'::regclass AND attname='parent_id';
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.shop_messages'::regclass
        AND conname='shop_messages_id_shop_unique' AND contype='u' AND conkey=ARRAY[id_column,shop_column]
        AND convalidated AND NOT condeferrable AND NOT condeferred) THEN
        RAISE EXCEPTION '021: expected migration 020 composite message key';
    END IF;
    IF EXISTS (SELECT 1 FROM public.shop_messages child LEFT JOIN public.shop_messages parent ON parent.id=child.parent_id
        WHERE child.parent_id IS NOT NULL AND (parent.id IS NULL OR parent.shop_id IS DISTINCT FROM child.shop_id)) THEN
        RAISE EXCEPTION '021: foreign or missing reply parent; explicit repair manifest required';
    END IF;
    IF (SELECT count(*) FROM pg_constraint WHERE conrelid='public.shop_messages'::regclass
        AND contype='f' AND parent_column=ANY(conkey)) <> 1
        OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.shop_messages'::regclass
            AND conname='shop_message_parent_id_fkey' AND contype='f' AND confrelid='public.shop_messages'::regclass
            AND conkey=ARRAY[parent_column] AND confkey=ARRAY[id_column]
            AND confdeltype='c' AND confupdtype='a' AND confmatchtype='s'
            AND convalidated AND NOT condeferrable AND NOT condeferred)
        OR EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.shop_messages'::regclass AND conname='shop_message_parent_shop_fkey') THEN
        RAISE EXCEPTION '021: unexpected reply FK shape or action; explicit conversion manifest required';
    END IF;
END $$;

ALTER TABLE public.shop_messages DROP CONSTRAINT shop_message_parent_id_fkey;
ALTER TABLE public.shop_messages ADD CONSTRAINT shop_message_parent_shop_fkey
    FOREIGN KEY (parent_id,shop_id) REFERENCES public.shop_messages(id,shop_id) ON DELETE CASCADE;
-- shop_messages_id_shop_unique belongs to migration 020 and is retained.
COMMIT;
