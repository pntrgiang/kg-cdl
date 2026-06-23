#!/usr/bin/env bash
# Test luồng sự kiện quay số trúng thưởng + voucher.
set -euo pipefail
API=http://localhost:8080/api
j() { python3 -c "import sys,json;d=json.load(sys.stdin);print(d$1)"; }
pass(){ echo "✅ $1"; }; fail(){ echo "❌ $1"; exit 1; }

MGR=$(curl -s -X POST $API/auth/login -d '{"username":"manager","password":"manager123"}' | j "['token']['access_token']")
HM="Authorization: Bearer $MGR"

echo "== 1. Tạo 4 khách có tài khoản (đủ điều kiện) =="
for i in 1 2 3 4; do
  curl -s -X POST $API/auth/customer/register -d "{\"username\":\"draw$i\",\"password\":\"draw1234\",\"national_id\":\"DRAW00$i\",\"full_name\":\"Khach Quay $i\"}" >/dev/null
done
pass "4 khách đăng ký"

echo "== 2. Quản lý tạo voucher 10% tối đa 10000 =="
VID=$(curl -s -X POST $API/vouchers -H "$HM" -d '{"name":"Voucher 10% tối đa 10.000$","discount_percent":10,"max_amount":10000}' | j "['id']")
[ -n "$VID" ] && pass "voucher id=$VID" || fail "voucher"

echo "== 3. Tạo sự kiện quay số (hạn 26/06, thưởng voucher, 2 người trúng) =="
EV=$(curl -s -X POST $API/events/draw -H "$HM" -d "{\"title\":\"Khuyến mãi tháng 6\",\"description\":\"Quay số trúng voucher\",\"register_deadline\":\"2026-06-26T23:59:59+07:00\",\"prize_type\":\"voucher\",\"voucher_id\":$VID,\"winners_count\":2}")
EID=$(echo "$EV" | j "['id']")
ELIG=$(echo "$EV" | j "['eligible_count']")
echo "  event id=$EID, đủ điều kiện=$ELIG"
[ "$ELIG" -ge 4 ] && pass "tạo sự kiện + đếm đủ điều kiện" || fail "eligible=$ELIG"

echo "== 4. Quay số (nháp) =="
W=$(curl -s -X POST $API/events/$EID/draw -H "$HM")
NW=$(echo "$W" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))")
echo "$W" | python3 -c "import sys,json;[print('   trúng:',x['customer_name'],'-',x['status']) for x in json.load(sys.stdin)]"
[ "$NW" = "2" ] && pass "quay ra 2 người trúng (pending)" || fail "winners=$NW"

echo "== 5. Quay lần 2 phải bị chặn (đã drawn) =="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/events/$EID/draw -H "$HM")
[ "$CODE" = "409" ] && pass "quay lại bị chặn (409)" || fail "redraw=$CODE"

echo "== 6. Xác nhận → công bố + phát voucher =="
curl -s -X POST $API/events/$EID/confirm -H "$HM" | python3 -c "import sys,json;[print('   xác nhận:',x['customer_name'],'-',x['status']) for x in json.load(sys.stdin)]"
DS=$(curl -s $API/events/$EID -H "$HM" | j "['draw_status']")
[ "$DS" = "published" ] && pass "đã công bố (published)" || fail "draw_status=$DS"

echo "== 7. Người trúng có voucher khả dụng =="
# lấy 1 customer_id người trúng
WIN_CID=$(curl -s $API/events/$EID -H "$HM" | python3 -c "import sys,json;print(json.load(sys.stdin)['winners'][0]['customer_id'])")
PRIZES=$(curl -s $API/customers/$WIN_CID/prizes -H "$HM")
NV=$(echo "$PRIZES" | python3 -c "import sys,json;print(len(json.load(sys.stdin)['vouchers']))")
[ "$NV" -ge 1 ] && pass "khách trúng có $NV voucher" || fail "vouchers=$NV"

echo "== 8. Bán xe cho người trúng, dùng voucher =="
CVID=$(echo "$PRIZES" | python3 -c "import sys,json;print(json.load(sys.stdin)['vouchers'][0]['id'])")
# nhập 1 xe giá 200000 để bán
CAT=$(curl -s "$API/catalog?search=adder" -H "$HM" | j "[0]['id']")
INV=$(curl -s -X POST $API/inventory -H "$HM" -d "{\"catalog_id\":$CAT,\"base_price\":200000,\"quantity\":1,\"status\":\"on_sale\"}" | j "['id']")
SALE=$(curl -s -X POST $API/sales -H "$HM" -d "{\"inventory_id\":$INV,\"customer_id\":$WIN_CID,\"customer_voucher_id\":$CVID}")
FINAL=$(echo "$SALE" | j "['sale']['final_price']")
# 200000 * 10% = 20000 nhưng tối đa 10000 -> final = 190000
echo "   giá cuối: $FINAL (mong đợi 190000 do giảm 10% bị chặn ở 10.000)"
[ "$(python3 -c "print(int(float('$FINAL'))==190000)")" = "True" ] && pass "voucher áp đúng (giảm tối đa 10.000)" || fail "final=$FINAL"

echo "== 9. Voucher đã dùng, không còn khả dụng =="
NV2=$(curl -s $API/customers/$WIN_CID/prizes -H "$HM" | python3 -c "import sys,json;print(len(json.load(sys.stdin)['vouchers']))")
[ "$NV2" = "0" ] && pass "voucher chuyển sang đã dùng" || fail "còn $NV2 voucher"

echo "== 10. Nhật ký có log quay số =="
NL=$(curl -s "$API/logs?action=draw.run" -H "$HM" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))")
[ "$NL" -ge 1 ] && pass "log draw.run: $NL" || fail "logs=$NL"

echo ""; echo "🎉 LUỒNG QUAY SỐ + VOUCHER PASS"
