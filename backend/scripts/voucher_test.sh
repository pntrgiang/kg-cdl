#!/usr/bin/env bash
# Test: voucher có số lượng giới hạn + 1 khách không dùng cùng voucher 2 lần.
set -euo pipefail
API=http://localhost:8080/api
j(){ python3 -c "import sys,json;d=json.load(sys.stdin);print(d$1)"; }
pass(){ echo "✅ $1"; }; fail(){ echo "❌ $1"; exit 1; }
psql(){ docker exec -i kg_cdl_db psql -U kg -d kg_cdl -tAc "$1"; }

MGR=$(curl -s -X POST $API/auth/login -d '{"username":"manager","password":"manager123"}' | j "['token']['access_token']")
HM="Authorization: Bearer $MGR"

echo "== 1. Tạo voucher số lượng = 1 =="
VID=$(curl -s -X POST $API/vouchers -H "$HM" -d '{"name":"VTEST 5% SL1","discount_percent":5,"max_amount":0,"quantity":1}' | j "['id']")
[ -n "$VID" ] && pass "voucher id=$VID quantity=1" || fail "voucher"

echo "== 2. Validate: tạo voucher thiếu số lượng -> 400 =="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/vouchers -H "$HM" -d '{"name":"x","discount_percent":5,"max_amount":0}')
[ "$CODE" = "400" ] && pass "thiếu quantity bị chặn (400)" || fail "validate=$CODE"

echo "== 3. Tạo 2 khách + phát voucher cho cả 2 =="
for i in A B; do
  curl -s -X POST $API/auth/customer/register -d "{\"username\":\"vt$i\",\"password\":\"vtest123\",\"national_id\":\"VT00$i\",\"full_name\":\"VKhach $i\"}" >/dev/null
done
CA=$(psql "SELECT id FROM customers WHERE national_id='VT00A'")
CB=$(psql "SELECT id FROM customers WHERE national_id='VT00B'")
psql "INSERT INTO customer_vouchers (customer_id, voucher_id, status) VALUES ($CA,$VID,'available'),($CB,$VID,'available')" >/dev/null
pass "phát voucher cho A,B"

echo "== 4. Nhập 3 xe để bán =="
CAT=$(curl -s "$API/catalog?search=adder" -H "$HM" | j "[0]['id']")
mkinv(){ curl -s -X POST $API/inventory -H "$HM" -d "{\"catalog_id\":$CAT,\"base_price\":100000,\"quantity\":1,\"status\":\"on_sale\"}" | j "['id']"; }
I1=$(mkinv); I2=$(mkinv); I3=$(mkinv)
pass "3 xe"

echo "== 5. A dùng voucher mua xe -> OK (used_count=1) =="
CVA=$(curl -s $API/customers/$CA/prizes -H "$HM" | python3 -c "import sys,json;print(json.load(sys.stdin)['vouchers'][0]['id'])")
F=$(curl -s -X POST $API/sales -H "$HM" -d "{\"inventory_id\":$I1,\"customer_id\":$CA,\"customer_voucher_id\":$CVA}" | j "['sale']['final_price']")
UC=$(psql "SELECT used_count FROM vouchers WHERE id=$VID")
[ "$UC" = "1" ] && pass "A dùng OK, used_count=1 (giá $F)" || fail "used_count=$UC"

echo "== 6. B dùng voucher -> hết số lượng (409) =="
CVB=$(curl -s $API/customers/$CB/prizes -H "$HM" | python3 -c "d=__import__('json').load(__import__('sys').stdin)['vouchers'];print(d[0]['id'] if d else 'NONE')")
if [ "$CVB" = "NONE" ]; then
  pass "voucher đã ẩn khỏi B (hết số lượng) — đúng"
else
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/sales -H "$HM" -d "{\"inventory_id\":$I2,\"customer_id\":$CB,\"customer_voucher_id\":$CVB}")
  [ "$CODE" = "409" ] && pass "B bị chặn (409 hết số lượng)" || fail "B code=$CODE"
fi

echo "== 7. A có voucher khác cùng loại, dùng lại -> chặn (đã dùng) =="
# tăng số lượng voucher lên 5, phát thêm 1 cái cho A
psql "UPDATE vouchers SET quantity=5 WHERE id=$VID" >/dev/null
psql "INSERT INTO customer_vouchers (customer_id, voucher_id, status) VALUES ($CA,$VID,'available')" >/dev/null
# danh sách khả dụng của A phải RỖNG (đã dùng voucher này rồi)
NVA=$(curl -s $API/customers/$CA/prizes -H "$HM" | python3 -c "import sys,json;print(len(json.load(sys.stdin)['vouchers']))")
[ "$NVA" = "0" ] && pass "A không còn thấy voucher đã dùng (dù còn số lượng & có bản phát mới)" || fail "A còn $NVA voucher"
# thử ép dùng bản mới -> chặn
CV2=$(psql "SELECT id FROM customer_vouchers WHERE customer_id=$CA AND voucher_id=$VID AND status='available' LIMIT 1")
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/sales -H "$HM" -d "{\"inventory_id\":$I3,\"customer_id\":$CA,\"customer_voucher_id\":$CV2}")
[ "$CODE" = "409" ] && pass "A dùng lại cùng voucher bị chặn (409)" || fail "A reuse code=$CODE"

echo "== dọn dữ liệu test =="
psql "DELETE FROM customer_vouchers WHERE voucher_id=$VID" >/dev/null
psql "DELETE FROM sales WHERE customer_id IN ($CA,$CB)" >/dev/null
psql "DELETE FROM inventory WHERE id IN ($I1,$I2,$I3)" >/dev/null
psql "DELETE FROM vouchers WHERE id=$VID" >/dev/null
psql "DELETE FROM refresh_tokens WHERE subject_type='customer' AND subject_id IN ($CA,$CB)" >/dev/null
psql "DELETE FROM customers WHERE id IN ($CA,$CB)" >/dev/null
echo ""; echo "🎉 VOUCHER QUANTITY + ONE-USE PASS"
