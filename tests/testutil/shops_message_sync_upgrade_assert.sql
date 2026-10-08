DO $$
DECLARE got text;
BEGIN
  SELECT string_agg(message || '=' || insertion_number, ',' ORDER BY insertion_number)
    INTO got FROM public.shop_messages WHERE shop_id = 'shop-a';
  IF got IS DISTINCT FROM 'first=1,tie-b=2,tie-d=3,third=4' THEN
    RAISE EXCEPTION 'backfill order wrong: %', got;
  END IF;
  IF (SELECT insertion_number FROM public.shop_messages WHERE id = '00000000-0000-4000-8000-0000000000b1') <> 1 THEN
    RAISE EXCEPTION 'shop-b not numbered independently';
  END IF;
  SELECT string_agg(shop_id || '=' || last_number, ',' ORDER BY shop_id) INTO got FROM public.shop_message_counters;
  IF got IS DISTINCT FROM 'shop-a=4,shop-b=1,shop-empty=0' THEN
    RAISE EXCEPTION 'counters wrong: %', got;
  END IF;
  INSERT INTO public.shop_messages (id, shop_id, user_id, message, created_at)
    VALUES ('00000000-0000-4000-8000-0000000000e1', 'shop-a', 'seed-user', 'after', now());
  IF (SELECT insertion_number FROM public.shop_messages WHERE id = '00000000-0000-4000-8000-0000000000e1') <> 5 THEN
    RAISE EXCEPTION 'post-migration insert not numbered 5';
  END IF;
END $$;
