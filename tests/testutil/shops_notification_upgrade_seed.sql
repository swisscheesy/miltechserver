-- Synthetic rows in the approved pre-016 physical schema. No exported row data is used.
CREATE SCHEMA test_upgrade;
CREATE TABLE test_upgrade.baseline_constraints (
    relation_name text NOT NULL,
    constraint_name text NOT NULL,
    definition text NOT NULL,
    validated boolean NOT NULL,
    PRIMARY KEY (relation_name, constraint_name)
);
INSERT INTO test_upgrade.baseline_constraints
SELECT relation.relname, c.conname, pg_catalog.pg_get_constraintdef(c.oid), c.convalidated
FROM pg_catalog.pg_constraint AS c
JOIN pg_catalog.pg_class AS relation ON relation.oid = c.conrelid
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
WHERE namespace.nspname = 'public'
  AND relation.relname IN ('users', 'shops', 'shop_vehicle', 'shop_vehicle_notifications', 'shop_notification_items');

INSERT INTO public.users (uid, email, username, created_at, is_enabled)
VALUES ('notification-upgrade-user', 'notification-upgrade@example.com', 'upgrade-user', now(), true);

INSERT INTO public.shops (id, name, details, created_by)
VALUES ('00000000-0000-4000-8000-000000000101', 'Upgrade Shop', 'before migration 016', 'notification-upgrade-user');

INSERT INTO public.shop_vehicle (id, creator_id, admin, shop_id, tracked_mileage, tracked_hours)
VALUES ('00000000-0000-4000-8000-000000000102', 'notification-upgrade-user', 'notification-upgrade-user',
        '00000000-0000-4000-8000-000000000101', 12, 34);

INSERT INTO public.shop_vehicle_notifications (id, shop_id, vehicle_id, title, description, type)
VALUES ('00000000-0000-4000-8000-000000000103', '00000000-0000-4000-8000-000000000101',
        '00000000-0000-4000-8000-000000000102', 'Upgrade notification', 'preserve this row', 'PM');

INSERT INTO public.shop_notification_items (id, shop_id, notification_id, niin, nomenclature, quantity)
VALUES ('00000000-0000-4000-8000-000000000104', '00000000-0000-4000-8000-000000000101',
        '00000000-0000-4000-8000-000000000103', 'upgrade-niin', 'Upgrade item', 2);
