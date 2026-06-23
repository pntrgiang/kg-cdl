ALTER TABLE sales
    DROP COLUMN IF EXISTS voucher_id,
    DROP COLUMN IF EXISTS voucher_discount,
    DROP COLUMN IF EXISTS prize_win_id,
    DROP COLUMN IF EXISTS is_gift;

DROP TABLE IF EXISTS customer_vouchers;
DROP TABLE IF EXISTS event_winners;

ALTER TABLE events
    DROP COLUMN IF EXISTS register_deadline,
    DROP COLUMN IF EXISTS prize_type,
    DROP COLUMN IF EXISTS voucher_id,
    DROP COLUMN IF EXISTS prize_vehicle_catalog_id,
    DROP COLUMN IF EXISTS winners_count,
    DROP COLUMN IF EXISTS draw_status;

DROP TABLE IF EXISTS vouchers;
