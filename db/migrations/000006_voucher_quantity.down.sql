ALTER TABLE vouchers
    DROP COLUMN IF EXISTS quantity,
    DROP COLUMN IF EXISTS used_count;
