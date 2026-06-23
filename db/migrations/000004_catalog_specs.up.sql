-- Thông số kỹ thuật xe (số chỗ + điểm hiệu năng 0-100 chuẩn hóa từ data dump).
ALTER TABLE vehicle_catalog
  ADD COLUMN seats         INT,
  ADD COLUMN rate_speed    INT,
  ADD COLUMN rate_accel    INT,
  ADD COLUMN rate_braking  INT,
  ADD COLUMN rate_traction INT;
