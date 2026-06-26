-- Mốc nhân viên xem lịch đặt gần nhất (cho badge thông báo kiểu Facebook).
-- Lịch tạo SAU mốc này = "mới" -> hiện badge; xem trang đặt lịch -> cập nhật mốc = now() -> badge về 0.
ALTER TABLE users ADD COLUMN IF NOT EXISTS bookings_seen_at timestamptz NOT NULL DEFAULT now();
