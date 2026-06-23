-- Index cho subquery "tổng đã nhập" và hoàn trả: sales theo inventory_id.
CREATE INDEX IF NOT EXISTS idx_sales_inventory ON sales(inventory_id);

-- Chống race: mỗi khách chỉ có TỐI ĐA 1 lịch đặt 'pending' cho cùng 1 xe
-- (hai request đặt lịch đồng thời không thể cùng lọt qua kiểm tra chống spam).
CREATE UNIQUE INDEX IF NOT EXISTS uq_booking_pending
    ON bookings(customer_id, inventory_id) WHERE status = 'pending';
