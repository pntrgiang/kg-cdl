ALTER TABLE vehicle_catalog
  DROP COLUMN IF EXISTS seats,
  DROP COLUMN IF EXISTS rate_speed,
  DROP COLUMN IF EXISTS rate_accel,
  DROP COLUMN IF EXISTS rate_braking,
  DROP COLUMN IF EXISTS rate_traction;
