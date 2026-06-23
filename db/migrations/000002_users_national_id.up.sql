-- Thêm số căn cước cho nhân viên (nullable, unique cho phép nhiều NULL).
ALTER TABLE users ADD COLUMN national_id TEXT UNIQUE;
