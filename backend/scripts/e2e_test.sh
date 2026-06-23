#!/usr/bin/env bash
# Smoke test toàn bộ luồng nghiệp vụ qua API.
set -euo pipefail
API=http://localhost:8080/api
j() { python3 -c "import sys,json;d=json.load(sys.stdin);print(d$1)"; }

pass() { echo "✅ $1"; }
fail() { echo "❌ $1"; exit 1; }

echo "== 1. Đăng nhập dev =="
DEV=$(curl -s -X POST $API/auth/login -d '{"username":"admin","password":"admin123"}')
DEV_AT=$(echo "$DEV" | j "['token']['access_token']")
[ -n "$DEV_AT" ] && pass "dev login" || fail "dev login"

echo "== 2. Dev tạo manager + staff =="
curl -s -X POST $API/admin/users -H "Authorization: Bearer $DEV_AT" \
  -d '{"username":"manager1","password":"secret1","display_name":"Quản Lý 1","role":"manager"}' >/dev/null
curl -s -X POST $API/admin/users -H "Authorization: Bearer $DEV_AT" \
  -d '{"username":"staff1","password":"secret1","display_name":"Nhân Viên 1","role":"staff"}' >/dev/null
pass "tạo manager + staff"

MGR=$(curl -s -X POST $API/auth/login -d '{"username":"manager1","password":"secret1"}')
MGR_AT=$(echo "$MGR" | j "['token']['access_token']")
STF=$(curl -s -X POST $API/auth/login -d '{"username":"staff1","password":"secret1"}')
STF_AT=$(echo "$STF" | j "['token']['access_token']")
[ -n "$MGR_AT" ] && [ -n "$STF_AT" ] && pass "manager + staff login" || fail "login mgr/staff"

echo "== 3. Staff KHÔNG được tạo user (RBAC) =="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/admin/users -H "Authorization: Bearer $STF_AT" \
  -d '{"username":"x","password":"secret1","display_name":"x","role":"staff"}')
[ "$CODE" = "403" ] && pass "staff bị chặn tạo user (403)" || fail "RBAC staff: $CODE"

echo "== 4. Nhập kho (chọn catalog có sẵn) =="
CAT_ID=$(curl -s "$API/catalog?search=adder" -H "Authorization: Bearer $STF_AT" | j "[0]['id']")
INV=$(curl -s -X POST $API/inventory -H "Authorization: Bearer $STF_AT" \
  -d "{\"catalog_id\":$CAT_ID,\"base_price\":1000000,\"quantity\":2,\"status\":\"on_sale\"}")
INV_ID=$(echo "$INV" | j "['id']")
[ -n "$INV_ID" ] && pass "nhập kho id=$INV_ID" || fail "nhập kho"

echo "== 5. Đặt giảm giá 20% =="
DISC=$(curl -s -X POST $API/inventory/$INV_ID/discount -H "Authorization: Bearer $STF_AT" -d '{"percent":20}')
FINAL=$(echo "$DISC" | j "['final_price']")
[ "$(python3 -c "print(int(float('$FINAL'))==800000)")" = "True" ] && pass "giảm giá: final=$FINAL (đúng 800000)" || fail "discount final=$FINAL"

echo "== 6. Tạo 5 khách + bán xe để test ranking =="
declare -a CUST_IDS
SPENDS=(5000000 4000000 3000000 2000000 1000000)
for i in 0 1 2 3 4; do
  C=$(curl -s -X POST $API/customers -H "Authorization: Bearer $STF_AT" \
    -d "{\"full_name\":\"Khach $i\",\"phone\":\"090000000$i\",\"national_id\":\"CCCD00$i\"}")
  CUST_IDS[$i]=$(echo "$C" | j "['id']")
done
pass "tạo 5 khách"

echo "== 7. Nhập thêm kho giá cao để bán cho từng khách =="
for i in 0 1 2 3 4; do
  IV=$(curl -s -X POST $API/inventory -H "Authorization: Bearer $STF_AT" \
    -d "{\"catalog_id\":$CAT_ID,\"base_price\":${SPENDS[$i]},\"quantity\":1,\"status\":\"on_sale\"}")
  IVID=$(echo "$IV" | j "['id']")
  curl -s -X POST $API/sales -H "Authorization: Bearer $STF_AT" \
    -d "{\"inventory_id\":$IVID,\"customer_id\":${CUST_IDS[$i]}}" >/dev/null
done
pass "bán 5 xe"

echo "== 8. Kiểm tra ranking (svip<=3, vip<=5) =="
RANKS=$(curl -s "$API/customers" -H "Authorization: Bearer $STF_AT")
echo "$RANKS" | python3 -c "
import sys,json
d=json.load(sys.stdin)
top=[(c['full_name'],c['rank'],c['total_spent']) for c in d if c['total_spent']>0]
top.sort(key=lambda x:-x[2])
for n,r,s in top: print(f'  {n}: {r} ({s})')
svip=[c for c in d if c['rank']=='svip']
assert len(svip)==3, f'svip count {len(svip)}'
print('  => svip=3 đúng')
"
pass "ranking đúng"

echo "== 9. Bán hết kho -> hết hàng (409) =="
# INV_ID còn 2 xe, bán 2 lần rồi lần 3 phải 409
curl -s -X POST $API/sales -H "Authorization: Bearer $STF_AT" -d "{\"inventory_id\":$INV_ID,\"customer_id\":${CUST_IDS[0]}}" >/dev/null
curl -s -X POST $API/sales -H "Authorization: Bearer $STF_AT" -d "{\"inventory_id\":$INV_ID,\"customer_id\":${CUST_IDS[0]}}" >/dev/null
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/sales -H "Authorization: Bearer $STF_AT" -d "{\"inventory_id\":$INV_ID,\"customer_id\":${CUST_IDS[0]}}")
[ "$CODE" = "409" ] && pass "hết hàng trả 409" || fail "out of stock code=$CODE"

echo "== 10. Staff KHÔNG tạo được sự kiện, Manager thì được =="
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/events -H "Authorization: Bearer $STF_AT" \
  -d '{"title":"x","type":"lucky_wheel","prizes":[{"name":"A","weight":1}]}')
[ "$CODE" = "403" ] && pass "staff bị chặn tạo event (403)" || fail "event RBAC staff=$CODE"
EV=$(curl -s -X POST $API/events -H "Authorization: Bearer $MGR_AT" \
  -d '{"title":"Vòng quay Tết","type":"lucky_wheel","prizes":[{"name":"Xe Adder","weight":1,"stock":1},{"name":"Chúc may mắn","weight":9}]}')
EV_ID=$(echo "$EV" | j "['id']")
[ -n "$EV_ID" ] && pass "manager tạo event id=$EV_ID" || fail "manager event"

echo "== 11. Khách đăng ký (claim căn cước đã có) + quay số =="
# CCCD000 đã được staff tạo ở bước 6 -> đăng ký phải claim và giữ tên 'Khach 0'
REG=$(curl -s -X POST $API/auth/customer/register -d '{"username":"khach0","password":"secret1","national_id":"CCCD000"}')
C_AT=$(echo "$REG" | j "['token']['access_token']")
C_NAME=$(echo "$REG" | j "['customer']['full_name']")
[ "$C_NAME" = "Khach 0" ] && pass "claim giữ thông tin cũ (full_name=$C_NAME)" || fail "claim name=$C_NAME"

curl -s -X POST $API/events/$EV_ID/register -H "Authorization: Bearer $C_AT" >/dev/null
SPIN=$(curl -s -X POST $API/events/$EV_ID/spin -H "Authorization: Bearer $C_AT")
PRIZE=$(echo "$SPIN" | j "['prize_name']")
[ -n "$PRIZE" ] && pass "quay trúng: $PRIZE" || fail "spin"
# quay lần 2 phải hết lượt (409)
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST $API/events/$EV_ID/spin -H "Authorization: Bearer $C_AT")
[ "$CODE" = "409" ] && pass "hết lượt quay trả 409" || fail "spin again=$CODE"

echo "== 12. Token refresh =="
RT=$(echo "$DEV" | j "['token']['refresh_token']")
NEW=$(curl -s -X POST $API/auth/refresh -d "{\"refresh_token\":\"$RT\"}")
NEW_AT=$(echo "$NEW" | j "['token']['access_token']")
[ -n "$NEW_AT" ] && pass "refresh token OK" || fail "refresh"

echo "== 13. Log có filter =="
LOGS=$(curl -s "$API/logs?action=sale.create" -H "Authorization: Bearer $MGR_AT")
N=$(echo "$LOGS" | python3 -c "import sys,json;print(len(json.load(sys.stdin)))")
[ "$N" -ge 5 ] && pass "log filter sale.create: $N dòng" || fail "logs=$N"

echo ""
echo "🎉 TẤT CẢ TEST PASS"
