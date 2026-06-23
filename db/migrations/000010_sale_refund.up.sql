-- Hoàn trả giao dịch bán sai (quản lý). Giữ bản ghi để truy vết.
ALTER TABLE sales
    ADD COLUMN refunded_at   TIMESTAMPTZ,
    ADD COLUMN refunded_by   BIGINT REFERENCES users(id),
    ADD COLUMN refund_reason TEXT;
