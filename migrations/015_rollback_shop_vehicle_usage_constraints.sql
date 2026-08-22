BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_vehicle
  DROP CONSTRAINT shop_vehicle_tracked_mileage_nonnegative,
  DROP CONSTRAINT shop_vehicle_tracked_hours_nonnegative;

COMMIT;
