#!/usr/bin/env bash
# Tạo dữ liệu demo để xem ngay trên web. Idempotent ở mức "chạy 1 lần sau reset".
set -euo pipefail
API=http://localhost:8080/api
j() { python3 -c "import sys,json;d=json.load(sys.stdin);print(d$1)"; }

AT=$(curl -s -X POST $API/auth/login -d '{"username":"admin","password":"admin123"}' | j "['token']['access_token']")
H="Authorization: Bearer $AT"

# nhân viên demo
curl -s -X POST $API/admin/users -H "$H" -d '{"username":"manager","password":"manager123","display_name":"Quản Lý Demo","role":"manager"}' >/dev/null || true
curl -s -X POST $API/admin/users -H "$H" -d '{"username":"staff","password":"staff123","display_name":"Nhân Viên Demo","role":"staff"}' >/dev/null || true

# vài xe nổi bật vào kho
add_stock() { # name_search price qty status
  local cid
  cid=$(curl -s "$API/catalog?search=$1" -H "$H" | j "[0]['id']")
  curl -s -X POST $API/inventory -H "$H" -d "{\"catalog_id\":$cid,\"base_price\":$2,\"quantity\":$3,\"status\":\"$4\"}" | j "['id']"
}

I1=$(add_stock "adder"   2500000 3 on_sale)
I2=$(add_stock "zentorno" 1900000 2 on_sale)
I3=$(add_stock "t20"     2200000 2 on_sale)
I4=$(add_stock "osiris"  1950000 4 on_sale)
add_stock "italigtb" 1500000 5 on_sale >/dev/null
add_stock "krieger"  3200000 1 upcoming >/dev/null
add_stock "deveste"  3500000 2 upcoming >/dev/null

# giảm giá vài xe
curl -s -X POST $API/inventory/$I1/discount -H "$H" -d '{"percent":15}' >/dev/null
curl -s -X POST $API/inventory/$I3/discount -H "$H" -d '{"percent":25}' >/dev/null
curl -s -X POST $API/inventory/$I4/discount -H "$H" -d '{"percent":10}' >/dev/null

# sự kiện vòng quay (manager)
MAT=$(curl -s -X POST $API/auth/login -d '{"username":"manager","password":"manager123"}' | j "['token']['access_token']")
curl -s -X POST $API/events -H "Authorization: Bearer $MAT" -d '{
  "title":"Vòng quay may mắn mùa hè",
  "description":"Tham gia quay trúng xe sang và nhiều phần quà hấp dẫn từ Kanji Group!",
  "type":"lucky_wheel",
  "prizes":[
    {"name":"Xe Adder miễn phí","weight":1,"stock":1},
    {"name":"Voucher giảm 50%","weight":3,"stock":10},
    {"name":"Voucher giảm 10%","weight":10},
    {"name":"Chúc may mắn lần sau","weight":20}
  ]
}' >/dev/null

echo "✅ Demo data đã tạo (manager/manager123, staff/staff123)"
