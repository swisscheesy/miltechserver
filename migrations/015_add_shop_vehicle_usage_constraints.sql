BEGIN;

SET LOCAL lock_timeout = '5s';

ALTER TABLE public.shop_vehicle
  ADD CONSTRAINT shop_vehicle_tracked_mileage_nonnegative
  CHECK (tracked_mileage IS NULL OR tracked_mileage >= 0) NOT VALID,
  ADD CONSTRAINT shop_vehicle_tracked_hours_nonnegative
  CHECK (tracked_hours IS NULL OR tracked_hours >= 0) NOT VALID;

ALTER TABLE public.shop_vehicle
  VALIDATE CONSTRAINT shop_vehicle_tracked_mileage_nonnegative;

ALTER TABLE public.shop_vehicle
  VALIDATE CONSTRAINT shop_vehicle_tracked_hours_nonnegative;

COMMIT;
