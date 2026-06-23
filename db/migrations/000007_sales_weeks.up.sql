-- Tuần mở bán: mỗi xe nhập kho gắn với một tuần mở bán.
CREATE TABLE sales_weeks (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_start DATE        NOT NULL UNIQUE,  -- thứ Hai
    week_end   DATE        NOT NULL,         -- Chủ Nhật
    label      TEXT,
    created_by BIGINT      REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE inventory ADD COLUMN sales_week_id BIGINT REFERENCES sales_weeks(id);
CREATE INDEX idx_inventory_week ON inventory (sales_week_id);
