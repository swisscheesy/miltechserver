BEGIN;

LOCK TABLE public.shop_vehicle IN ACCESS EXCLUSIVE MODE;
DO $$
DECLARE
    reading text;
    column_number smallint;
BEGIN
    IF (SELECT count(*) FROM pg_attribute WHERE attrelid='public.shop_vehicle'::regclass
        AND attname IN ('mileage','hours','tracked_mileage','tracked_hours')
        AND atttypid='integer'::regtype AND atttypmod=-1 AND NOT attisdropped) <> 4
        OR (SELECT count(*) FROM pg_attribute WHERE attrelid='public.shop_vehicle'::regclass
            AND attname IN ('mileage','hours') AND attnotnull) <> 2
        OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid='public.shop_vehicle'::regclass
            AND attname IN ('tracked_mileage','tracked_hours') AND attnotnull)
        OR (SELECT count(*) FROM pg_attrdef d JOIN pg_attribute a ON a.attrelid=d.adrelid AND a.attnum=d.adnum
            WHERE d.adrelid='public.shop_vehicle'::regclass AND a.attname IN ('mileage','hours')
            AND pg_get_expr(d.adbin,d.adrelid)='0') <> 2 THEN
        RAISE EXCEPTION '022: unexpected vehicle usage column shape';
    END IF;
    -- Migration 015 owns these nullable-reading checks; require and retain them.
    FOR reading IN SELECT unnest(ARRAY['tracked_mileage','tracked_hours']) LOOP
        SELECT attnum INTO column_number FROM pg_attribute WHERE attrelid='public.shop_vehicle'::regclass AND attname=reading;
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.shop_vehicle'::regclass
            AND conname='shop_vehicle_'||reading||'_nonnegative' AND contype='c' AND conkey=ARRAY[column_number]
            AND convalidated AND NOT connoinherit
            AND pg_get_expr(conbin,conrelid)='(('||reading||' IS NULL) OR ('||reading||' >= 0))') THEN
            RAISE EXCEPTION '022: unexpected migration 015 tracked constraint';
        END IF;
    END LOOP;
    FOR reading IN SELECT unnest(ARRAY['mileage','hours']) LOOP
        SELECT attnum INTO column_number FROM pg_attribute WHERE attrelid='public.shop_vehicle'::regclass AND attname=reading;
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.shop_vehicle'::regclass
            AND conname='shop_vehicle_'||reading||'_nonnegative' AND contype='c' AND conkey=ARRAY[column_number]
            AND convalidated AND NOT connoinherit AND pg_get_expr(conbin,conrelid)='('||reading||' >= 0)') THEN
            RAISE EXCEPTION '022: unexpected base usage constraint';
        END IF;
    END LOOP;
END $$;

-- These two constraints belong to 022; migration 015 is untouched.
ALTER TABLE public.shop_vehicle
    DROP CONSTRAINT shop_vehicle_mileage_nonnegative,
    DROP CONSTRAINT shop_vehicle_hours_nonnegative;
COMMIT;
