-- Tài khoản tạm (vd LUX00000): nhận xe bán bị thất lạc thông tin khách thật,
-- giữ số lượng kho luôn đúng. Tài khoản loại này KHÔNG được xếp hạng thành viên.
ALTER TABLE customers ADD COLUMN IF NOT EXISTS exclude_from_rank boolean NOT NULL DEFAULT false;

-- Đánh dấu LUX00000 là tài khoản tạm (nếu tồn tại).
UPDATE customers SET exclude_from_rank = true, rank = 'regular' WHERE national_id = 'LUX00000';
