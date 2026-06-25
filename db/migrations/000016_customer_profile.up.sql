-- Bổ sung giới tính & ngày sinh cho khách hàng.
ALTER TABLE customers
  ADD COLUMN IF NOT EXISTS gender text,
  ADD COLUMN IF NOT EXISTS birth_date date;
