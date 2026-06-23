-- ── Voucher (do quản lý tạo) ─────────────────────────────────────
CREATE TABLE vouchers (
    id               BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name             TEXT        NOT NULL,
    discount_percent NUMERIC(5,2) NOT NULL CHECK (discount_percent > 0 AND discount_percent <= 100),
    max_amount       NUMERIC(14,2) NOT NULL DEFAULT 0,  -- 0 = không giới hạn mức giảm
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_by       BIGINT      REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Bổ sung cột cho sự kiện quay số trúng thưởng ─────────────────
-- draw_status: NULL = sự kiện thường; 'open' | 'drawn' | 'published'
ALTER TABLE events
    ADD COLUMN register_deadline        TIMESTAMPTZ,
    ADD COLUMN prize_type               TEXT,        -- 'voucher' | 'vehicle'
    ADD COLUMN voucher_id               BIGINT REFERENCES vouchers(id),
    ADD COLUMN prize_vehicle_catalog_id BIGINT REFERENCES vehicle_catalog(id),
    ADD COLUMN winners_count            INT,
    ADD COLUMN draw_status              TEXT;

-- ── Người trúng giải của một lần quay ────────────────────────────
CREATE TABLE event_winners (
    id                BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id          BIGINT      NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    customer_id       BIGINT      NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    status            TEXT        NOT NULL DEFAULT 'pending',  -- 'pending' | 'confirmed'
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at      TIMESTAMPTZ,
    -- với thưởng là xe: đánh dấu đã giao khi bán
    fulfilled_at      TIMESTAMPTZ,
    fulfilled_sale_id BIGINT      REFERENCES sales(id),
    UNIQUE (event_id, customer_id)
);

-- ── Voucher đã phát cho khách (dùng khi mua xe) ──────────────────
CREATE TABLE customer_vouchers (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id  BIGINT      NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    voucher_id   BIGINT      NOT NULL REFERENCES vouchers(id),
    event_id     BIGINT      REFERENCES events(id) ON DELETE SET NULL,
    status       TEXT        NOT NULL DEFAULT 'available',  -- 'available' | 'used'
    used_sale_id BIGINT      REFERENCES sales(id),
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cust_vouchers_customer ON customer_vouchers (customer_id) WHERE status = 'available';

-- ── Bán xe có thể dùng voucher / là xe tặng trúng thưởng ─────────
ALTER TABLE sales
    ADD COLUMN voucher_id       BIGINT REFERENCES vouchers(id),
    ADD COLUMN voucher_discount NUMERIC(14,2) NOT NULL DEFAULT 0,
    ADD COLUMN prize_win_id     BIGINT REFERENCES event_winners(id),  -- != NULL: xe tặng trúng thưởng (giá 0)
    ADD COLUMN is_gift          BOOLEAN NOT NULL DEFAULT FALSE;
