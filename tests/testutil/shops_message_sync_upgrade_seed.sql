INSERT INTO public.users (uid, email, username, created_at, is_enabled)
VALUES ('seed-user', 'seed@example.com', 'seed', now(), true);
-- Columns match the baseline public.shops definition (id, name, created_by are NOT NULL).
INSERT INTO public.shops (id, name, created_by, created_at) VALUES
  ('shop-a', 'A', 'seed-user', now()), ('shop-b', 'B', 'seed-user', now()), ('shop-empty', 'E', 'seed-user', now());
-- Out-of-order and equal created_at values; ids chosen so the (created_at, id) order is known.
INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at) VALUES
  ('00000000-0000-4000-8000-00000000000c', 'shop-a', 'seed-user', 'third',  '2026-01-01T00:00:03Z'),
  ('00000000-0000-4000-8000-00000000000a', 'shop-a', 'seed-user', 'first',  '2026-01-01T00:00:01Z'),
  ('00000000-0000-4000-8000-00000000000b', 'shop-a', 'seed-user', 'tie-b',  '2026-01-01T00:00:02Z'),
  ('00000000-0000-4000-8000-00000000000d', 'shop-a', 'seed-user', 'tie-d',  '2026-01-01T00:00:02Z'),
  ('00000000-0000-4000-8000-0000000000b1', 'shop-b', 'seed-user', 'only-b', '2026-01-01T00:00:09Z');
