-- Sự kiện quay số "chỉ định": quản lý chọn sẵn người tham gia (không cần khách đăng ký).
-- invite_only=true -> chỉ những người được seed vào event_registrations mới có trong vòng quay,
-- khách không tự đăng ký được và sự kiện ẩn khỏi danh sách công khai.
ALTER TABLE events ADD COLUMN IF NOT EXISTS invite_only boolean NOT NULL DEFAULT false;
