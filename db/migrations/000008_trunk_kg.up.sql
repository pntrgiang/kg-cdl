-- Cốp xe (kg). Đặt 10 cho toàn bộ xe hiện có; xe mới bắt buộc nhập (validate ở app).
ALTER TABLE vehicle_catalog ADD COLUMN trunk_kg INT NOT NULL DEFAULT 10;
