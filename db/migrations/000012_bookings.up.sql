-- Cờ cho phép khách đặt lịch xem/mua xe trên từng mục kho.
ALTER TABLE inventory ADD COLUMN booking_open boolean NOT NULL DEFAULT false;

-- Lịch đặt xem/mua xe của khách hàng.
CREATE TABLE bookings (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    inventory_id bigint NOT NULL REFERENCES inventory(id) ON DELETE CASCADE,
    customer_id  bigint NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    vehicle_name text   NOT NULL,                       -- ảnh chụp tên xe lúc đặt
    visit_date   date   NOT NULL,                       -- ngày khách muốn đến xem
    note         text,                                  -- ghi chú của khách (tuỳ chọn)
    status       text   NOT NULL DEFAULT 'pending',     -- pending | accepted | rejected
    handled_by   bigint REFERENCES users(id),           -- nhân viên/quản lý xử lý
    handled_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_bookings_status   ON bookings(status, created_at);
CREATE INDEX idx_bookings_customer ON bookings(customer_id);
