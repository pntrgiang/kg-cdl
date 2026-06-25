-- Mốc vô hiệu phiên: token phát TRƯỚC mốc này sẽ bị từ chối (đăng xuất ngay).
ALTER TABLE users ADD COLUMN IF NOT EXISTS tokens_valid_after timestamptz;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS tokens_valid_after timestamptz;
