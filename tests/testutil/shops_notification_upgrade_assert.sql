DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM public.users
        WHERE uid = 'notification-upgrade-user' AND email = 'notification-upgrade@example.com'
    ) OR NOT EXISTS (
        SELECT 1 FROM public.shops
        WHERE id = '00000000-0000-4000-8000-000000000101'
          AND name = 'Upgrade Shop' AND created_by = 'notification-upgrade-user'
    ) OR NOT EXISTS (
        SELECT 1 FROM public.shop_vehicle
        WHERE id = '00000000-0000-4000-8000-000000000102'
          AND shop_id = '00000000-0000-4000-8000-000000000101'
          AND tracked_mileage = 12 AND tracked_hours = 34
    ) OR NOT EXISTS (
        SELECT 1 FROM public.shop_vehicle_notifications
        WHERE id = '00000000-0000-4000-8000-000000000103'
          AND shop_id = '00000000-0000-4000-8000-000000000101'
          AND vehicle_id = '00000000-0000-4000-8000-000000000102'
          AND title = 'Upgrade notification' AND description = 'preserve this row' AND type = 'PM'
    ) OR NOT EXISTS (
        SELECT 1 FROM public.shop_notification_items
        WHERE id = '00000000-0000-4000-8000-000000000104'
          AND shop_id = '00000000-0000-4000-8000-000000000101'
          AND notification_id = '00000000-0000-4000-8000-000000000103'
          AND niin = 'upgrade-niin' AND nomenclature = 'Upgrade item' AND quantity = 2
    ) THEN
        RAISE EXCEPTION 'populated notification upgrade changed seeded rows';
    END IF;

    IF (SELECT count(*) FROM public.users WHERE uid = 'notification-upgrade-user') != 1
       OR (SELECT count(*) FROM public.shops WHERE id = '00000000-0000-4000-8000-000000000101') != 1
       OR (SELECT count(*) FROM public.shop_vehicle WHERE id = '00000000-0000-4000-8000-000000000102') != 1
       OR (SELECT count(*) FROM public.shop_vehicle_notifications WHERE id = '00000000-0000-4000-8000-000000000103') != 1
       OR (SELECT count(*) FROM public.shop_notification_items WHERE id = '00000000-0000-4000-8000-000000000104') != 1
       OR to_regclass('public.shop_notification_operations') IS NULL THEN
        RAISE EXCEPTION 'populated notification upgrade lost rows or ledger';
    END IF;

    IF (SELECT count(*) FROM pg_catalog.pg_constraint
        WHERE conrelid = 'public.shop_vehicle'::regclass
          AND conname IN ('shop_vehicle_tracked_mileage_nonnegative', 'shop_vehicle_tracked_hours_nonnegative')
          AND convalidated) != 2 THEN
        RAISE EXCEPTION 'existing vehicle usage constraints changed';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM test_upgrade.baseline_constraints)
       OR EXISTS (
           SELECT relation_name, constraint_name, definition, validated
           FROM test_upgrade.baseline_constraints
           EXCEPT
           SELECT relation.relname, c.conname, pg_catalog.pg_get_constraintdef(c.oid), c.convalidated
           FROM pg_catalog.pg_constraint AS c
           JOIN pg_catalog.pg_class AS relation ON relation.oid = c.conrelid
           JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
           WHERE namespace.nspname = 'public'
       ) THEN
        RAISE EXCEPTION 'existing populated-schema constraints changed';
    END IF;
END $$;
