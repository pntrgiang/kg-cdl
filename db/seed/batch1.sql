-- Đợt mở bán 1 — từ 20/06/2026.
-- Đặt danh sách xe đang mở bán đúng theo img-temp/xe-dot-1.png.
BEGIN;

-- xóa kho & khuyến mãi demo cũ (chưa có giao dịch tham chiếu)
DELETE FROM discounts;
DELETE FROM inventory;

-- nhập 5 xe đợt 1 (catalog_id, giá, số lượng, mở bán 20/06/2026)
INSERT INTO inventory (catalog_id, base_price, quantity, status, on_sale_at, note, created_by)
SELECT c.id, v.price, v.qty, 'on_sale', TIMESTAMPTZ '2026-06-20 12:00:00+00', 'Đợt mở bán 1', 1
FROM (VALUES
  ('surfer',  200000, 30),
  ('cruiser',  15000, 30),
  ('faggio2',  45000, 20),
  ('blista2', 145000, 10),  -- Blista Compact (coupe HD Universe), không phải 'blista' (hatchback)
  ('buffalo', 250000, 10)
) AS v(code, price, qty)
JOIN vehicle_catalog c ON c.model_code = v.code;

COMMIT;
