-- Voucher nâng cấp: hạn sử dụng, phạm vi xe áp dụng, hạng tối thiểu.
ALTER TABLE vouchers
    ADD COLUMN expires_at     TIMESTAMPTZ,                 -- hạn sử dụng
    ADD COLUMN applies_to_all BOOLEAN NOT NULL DEFAULT TRUE, -- áp dụng mọi xe
    ADD COLUMN min_rank       customer_rank NOT NULL DEFAULT 'regular'; -- hạng tối thiểu được dùng

-- Voucher áp dụng cho các xe cụ thể (khi applies_to_all = false).
CREATE TABLE voucher_vehicles (
    voucher_id BIGINT NOT NULL REFERENCES vouchers(id) ON DELETE CASCADE,
    catalog_id BIGINT NOT NULL REFERENCES vehicle_catalog(id),
    PRIMARY KEY (voucher_id, catalog_id)
);
