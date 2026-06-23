ALTER TABLE sales
    DROP COLUMN IF EXISTS refunded_at,
    DROP COLUMN IF EXISTS refunded_by,
    DROP COLUMN IF EXISTS refund_reason;
