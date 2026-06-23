-- KG Car Dealer — initial schema
-- PostgreSQL 16

BEGIN;

-- ───────────────────────────── enums ─────────────────────────────
CREATE TYPE user_role        AS ENUM ('dev', 'manager', 'staff');
CREATE TYPE customer_rank    AS ENUM ('regular', 'vip', 'svip');
CREATE TYPE vehicle_status   AS ENUM ('upcoming', 'on_sale', 'hidden', 'sold_out');
CREATE TYPE event_type       AS ENUM ('lucky_wheel', 'discount_campaign');

-- ───────────────────────────── users (staff) ─────────────────────
CREATE TABLE users (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    display_name  TEXT        NOT NULL,
    role          user_role   NOT NULL DEFAULT 'staff',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ───────────────────────────── customers ─────────────────────────
CREATE TABLE customers (
    id              BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username        TEXT        UNIQUE,            -- nullable: khách do nhân viên tạo có thể chưa có tài khoản
    password_hash   TEXT,
    full_name       TEXT        NOT NULL,
    phone           TEXT        NOT NULL,
    national_id     TEXT        NOT NULL UNIQUE,   -- số căn cước
    rank            customer_rank NOT NULL DEFAULT 'regular',
    total_spent     NUMERIC(14,2) NOT NULL DEFAULT 0,
    last_purchase_at TIMESTAMPTZ,
    created_by      BIGINT      REFERENCES users(id),  -- nhân viên tạo (null nếu khách tự đăng ký)
    claimed_at      TIMESTAMPTZ,                        -- thời điểm khách "claim" tài khoản (set username/password)
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_customers_total_spent ON customers (total_spent DESC, last_purchase_at ASC);

CREATE TABLE customer_rank_history (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id BIGINT       NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    old_rank    customer_rank,
    new_rank    customer_rank NOT NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ───────────────────────────── refresh tokens ────────────────────
-- subject_type phân biệt token của user (staff) hay customer
CREATE TABLE refresh_tokens (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_type TEXT        NOT NULL CHECK (subject_type IN ('user','customer')),
    subject_id   BIGINT      NOT NULL,
    token_hash   TEXT        NOT NULL UNIQUE,     -- lưu hash, không lưu token thô
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    user_agent   TEXT,
    ip           INET,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_subject ON refresh_tokens (subject_type, subject_id);

-- ───────────────────────────── vehicle catalog ───────────────────
-- danh mục mẫu xe: data GTA5 + xe mod
CREATE TABLE vehicle_catalog (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    model_code   TEXT        UNIQUE,              -- spawn name GTA, vd 'adder'; null cho mod tự do
    name         TEXT        NOT NULL,
    brand        TEXT,
    class        TEXT,                            -- Super, Sports, SUV...
    image_url    TEXT,
    description  TEXT,
    is_mod       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by   BIGINT      REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_catalog_name ON vehicle_catalog (name);

-- ───────────────────────────── inventory ─────────────────────────
-- xe thực tế trong kho
CREATE TABLE inventory (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    catalog_id   BIGINT      NOT NULL REFERENCES vehicle_catalog(id),
    base_price   NUMERIC(14,2) NOT NULL CHECK (base_price >= 0),
    quantity     INT         NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    status       vehicle_status NOT NULL DEFAULT 'upcoming',
    on_sale_at   TIMESTAMPTZ,                     -- thời điểm mở bán (cho xe upcoming)
    note         TEXT,
    created_by   BIGINT      REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_inventory_status ON inventory (status);

-- ───────────────────────────── discounts ─────────────────────────
CREATE TABLE discounts (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inventory_id  BIGINT      NOT NULL REFERENCES inventory(id) ON DELETE CASCADE,
    percent       NUMERIC(5,2) NOT NULL CHECK (percent > 0 AND percent <= 100),
    starts_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at       TIMESTAMPTZ,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_by    BIGINT      REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_discount_inventory ON discounts (inventory_id) WHERE is_active;

-- ───────────────────────────── sales ─────────────────────────────
-- lưu giá tại thời điểm bán (không tham chiếu sống)
CREATE TABLE sales (
    id              BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inventory_id    BIGINT      NOT NULL REFERENCES inventory(id),
    catalog_id      BIGINT      NOT NULL REFERENCES vehicle_catalog(id),
    customer_id     BIGINT      NOT NULL REFERENCES customers(id),
    sold_by         BIGINT      NOT NULL REFERENCES users(id),
    original_price  NUMERIC(14,2) NOT NULL,       -- giá trước giảm
    discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    final_price     NUMERIC(14,2) NOT NULL,       -- giá khách thực trả
    vehicle_name    TEXT        NOT NULL,         -- snapshot tên xe
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sales_customer ON sales (customer_id);
CREATE INDEX idx_sales_created  ON sales (created_at DESC);

-- ───────────────────────────── events ────────────────────────────
CREATE TABLE events (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title        TEXT        NOT NULL,
    description  TEXT,
    type         event_type  NOT NULL,
    starts_at    TIMESTAMPTZ,
    ends_at      TIMESTAMPTZ,
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_by   BIGINT      NOT NULL REFERENCES users(id),  -- chỉ manager
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE event_registrations (
    id              BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id        BIGINT      NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    customer_id     BIGINT      NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    spins_remaining INT         NOT NULL DEFAULT 1 CHECK (spins_remaining >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, customer_id)
);

-- ───────────────────────────── lucky wheel ───────────────────────
-- các ô phần thưởng của 1 sự kiện vòng quay
CREATE TABLE event_prizes (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id      BIGINT      NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    image_url     TEXT,
    weight        INT         NOT NULL CHECK (weight >= 0),  -- trọng số xác suất (p = weight / tổng weight)
    stock         INT,                                       -- số lượng còn (null = không giới hạn)
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_event_prizes_event ON event_prizes (event_id) WHERE is_active;

-- lịch sử mỗi lượt quay
CREATE TABLE event_spins (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id      BIGINT      NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    customer_id   BIGINT      NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    prize_id      BIGINT      REFERENCES event_prizes(id),   -- null nếu trúng ô "chúc may mắn lần sau"
    prize_name    TEXT        NOT NULL,                      -- snapshot tên phần thưởng
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_event_spins_event    ON event_spins (event_id);
CREATE INDEX idx_event_spins_customer ON event_spins (customer_id);

-- ───────────────────────────── activity logs ─────────────────────
CREATE TABLE activity_logs (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id    BIGINT      REFERENCES users(id),
    actor_name  TEXT,                             -- snapshot
    action      TEXT        NOT NULL,             -- 'sale.create','inventory.add','catalog.create','event.create','customer.update'...
    target_type TEXT,
    target_id   BIGINT,
    detail      JSONB,                            -- dữ liệu chi tiết để filter/hiển thị
    ip          INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_logs_action  ON activity_logs (action);
CREATE INDEX idx_logs_actor   ON activity_logs (actor_id);
CREATE INDEX idx_logs_created ON activity_logs (created_at DESC);

-- ───────────────────────────── settings ──────────────────────────
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (key, value) VALUES
  ('rank_limits', '{"svip": 3, "vip": 5}');

COMMIT;
