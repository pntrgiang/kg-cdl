DROP TABLE IF EXISTS voucher_vehicles;
ALTER TABLE vouchers
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS applies_to_all,
    DROP COLUMN IF EXISTS min_rank;
