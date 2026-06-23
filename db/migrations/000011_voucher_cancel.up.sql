ALTER TABLE vouchers
    ADD COLUMN cancelled_at  timestamptz,
    ADD COLUMN cancelled_by  bigint REFERENCES users(id),
    ADD COLUMN cancel_reason text;
