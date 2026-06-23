-- Voucher có số lượng giới hạn (tổng số lượt được dùng), trừ dần sau mỗi lần sử dụng.
ALTER TABLE vouchers
    ADD COLUMN quantity   INT NOT NULL DEFAULT 0,   -- tổng số lượt được phép dùng (0 = không giới hạn)
    ADD COLUMN used_count INT NOT NULL DEFAULT 0;   -- số lượt đã dùng
