# KG Car Dealer (FiveM) — Kế hoạch dự án

Website quản lý & showroom cho doanh nghiệp car dealer trong FiveM.
**Phạm vi:** website độc lập (không tích hợp server game).

---

## 1. Tech stack

| Lớp | Công nghệ | Lý do |
|-----|-----------|-------|
| Frontend | Nuxt 3 (Vue 3 + TypeScript), Pinia, Nuxt UI / Tailwind | SSR/SPA linh hoạt, dev nhanh |
| Backend | Go 1.22+, `chi` router, `pgx` + `sqlc`, `golang-migrate` | type-safe, dễ review |
| DB | PostgreSQL 16 | yêu cầu |
| Auth | JWT access (15') + refresh (90 ngày, lưu DB, thu hồi được) | UX tốt + an toàn |
| Mật khẩu | bcrypt | chuẩn |
| Hạ tầng dev | Docker Compose (Postgres + adminer) | chạy local nhanh |

### Cấu trúc thư mục (monorepo)
```
kg-cdl/
├─ docs/                 # tài liệu (file này)
├─ db/
│  ├─ migrations/        # golang-migrate
│  └─ seed/              # data xe GTA5
├─ backend/              # Go
│  ├─ cmd/api/           # entrypoint
│  ├─ internal/
│  │  ├─ auth/           # jwt, middleware, rbac
│  │  ├─ handler/        # http handlers
│  │  ├─ service/        # business logic (ranking, sale...)
│  │  ├─ repo/           # sqlc generated + queries
│  │  └─ config/
│  └─ scripts/           # tải data xe GTA5
└─ frontend/             # Nuxt 3
   ├─ pages/
   ├─ components/
   ├─ stores/
   └─ middleware/
```

---

## 2. Vai trò & phân quyền (RBAC)

| Role | Mô tả | Quyền nổi bật |
|------|-------|----------------|
| `dev` | Quản trị hệ thống | Set rank cho staff/manager, mọi quyền |
| `manager` | Quản lý dealer | Tạo sự kiện, sửa khách hàng, nhập kho, bán xe, xem log |
| `staff` | Nhân viên | Bán xe, nhập kho, xem khách hàng (chỉ xem), xem log |
| `customer` | Khách hàng | Xem xe (đang/sắp bán), giá, đăng ký sự kiện |

**Quy tắc quan trọng:** chỉ `dev` được đổi rank của staff/manager. Staff/manager **không** tự nâng/hạ cấp nhau → enforce ở backend (endpoint set-role chỉ cho `dev`).

Customer có rank riêng: `regular` / `vip` / `svip` (xem mục 4).

---

## 3. Cây giao diện (tabs)

**Khu vực khách hàng (public + customer login):**
- 🚗 Xe đang mở bán (giá, % giảm, giá gốc/giá sau giảm, ảnh, tên)
- 🔜 Xe sắp mở bán
- 🎁 Sự kiện khuyến mãi (vòng quay may mắn, đăng ký tham gia)
- Chi tiết xe (mô tả/giới thiệu, thông số, ảnh)

**Khu vực nhân viên (staff/manager, sau đăng nhập):**
- 💰 Bán xe (chọn xe trong kho + chọn/tạo khách hàng)
- 📦 Nhập kho (chọn từ catalog, hoặc tạo xe mod mới)
- 🎉 Tạo sự kiện *(chỉ manager)*
- 👥 Quản lý khách hàng (staff: xem; manager: sửa)
- 📜 Log hoạt động (có filter)

**Khu vực dev:**
- ⚙️ Quản lý nhân viên & set rank

---

## 4. Logic xếp hạng khách hàng

- Rank tính theo **tổng tiền đã mua** (`total_spent`).
- Giới hạn cấu hình được: `svip = 3`, `vip = 5` (lưu ở bảng `settings`).
- **Recompute tự động** sau mỗi giao dịch: sắp xếp toàn bộ khách theo `total_spent` giảm dần →
  - Top 3 → `svip`
  - 3 kế tiếp → `vip`
  - Còn lại → `regular`
- Xử lý hoà (tie): cùng `total_spent` thì người đạt mốc **sớm hơn** (theo `last_purchase_at` cũ hơn) được ưu tiên giữ hạng cao.
- Lưu lịch sử thay đổi rank để minh bạch (`customer_rank_history`).

---

## 5. Luồng nghiệp vụ chính

### Bán xe
1. Nhân viên chọn xe từ **kho** (inventory, còn tồn > 0).
2. Chọn khách hàng có sẵn **hoặc** tạo mới (tên, SĐT, số căn cước).
3. Áp giá (giá sau giảm nếu xe đang khuyến mãi).
4. Hệ thống: tạo `sale`, giảm `inventory.quantity`, cộng `customer.total_spent`, recompute rank, ghi `activity_log`.

### Nhập kho
1. Chọn mẫu xe từ `vehicle_catalog`.
2. Nếu là xe mod chưa có → form tạo mới (tên, hãng, class, ảnh, mô tả).
3. Nhập `base_price`, `quantity`, (tuỳ chọn) khuyến mãi.
4. Ghi `activity_log`.

### Tạo sự kiện *(manager)*
- Loại: `lucky_wheel` (vòng quay), `discount_campaign`.
- Khách đăng ký tham gia → `event_registrations` (mặc định 1 lượt quay).

### Vòng quay may mắn (lucky wheel)
- Manager tạo event + danh sách ô thưởng `event_prizes` (tên, ảnh, `weight`, `stock`).
- Xác suất mỗi ô = `weight / tổng weight` các ô đang active. Ô "chúc may mắn lần sau" = một prize không có `stock`/giá trị.
- Khách bấm quay: backend kiểm tra `spins_remaining > 0`, random theo weight (loại ô đã hết `stock`), ghi `event_spins`, giảm `spins_remaining` và `stock`, ghi `activity_log`.
- **Random ở backend** (không tin client) để tránh gian lận.

### Luồng tài khoản khách hàng (claim)
- **Nhân viên tạo khách** (lúc bán): nhập tên, SĐT, số căn cước — **không** đặt username/password. `created_by` = nhân viên.
- **Khách tự đăng ký** bằng số căn cước:
  - Nếu `national_id` **đã tồn tại** (do nhân viên tạo trước) → gắn username/password vào đúng bản ghi đó, set `claimed_at`, hiển thị lại thông tin (tên, SĐT, lịch sử mua) đã có.
  - Nếu **chưa tồn tại** → tạo bản ghi mới, các trường thông tin để trống cho khách tự điền.
- `national_id` là UNIQUE nên không bị trùng/đúp khách.

---

## 6. Schema DB

Xem `db/migrations/0001_init.sql`. Tóm tắt bảng:

- `users` — tài khoản nhân viên/dev (role: dev/manager/staff)
- `refresh_tokens` — refresh token thu hồi được
- `customers` — khách hàng (name, phone, national_id, total_spent, rank)
- `customer_rank_history` — lịch sử đổi rank
- `vehicle_catalog` — danh mục mẫu xe (GTA5 + mod), `is_mod` flag
- `inventory` — xe trong kho (link catalog, base_price, quantity)
- `discounts` — khuyến mãi theo inventory (percent, thời gian)
- `sales` — giao dịch bán xe
- `events` — sự kiện
- `event_registrations` — đăng ký tham gia (số lượt quay còn lại)
- `event_prizes` — ô thưởng vòng quay (weight, stock)
- `event_spins` — lịch sử mỗi lượt quay
- `activity_logs` — log hoạt động (filter được)
- `settings` — cấu hình (giới hạn rank...)

---

## 7. API endpoint (phác thảo)

```
# Auth
POST   /api/auth/login              {username, password}  -> access+refresh
POST   /api/auth/refresh            {refresh_token}
POST   /api/auth/logout             (thu hồi refresh)
POST   /api/auth/customer/register
POST   /api/auth/customer/login

# Public / customer
GET    /api/vehicles?status=on_sale|upcoming
GET    /api/vehicles/:id
GET    /api/events
POST   /api/events/:id/register     (customer)

# Staff/manager
POST   /api/sales                   (bán xe)
GET    /api/inventory
POST   /api/inventory               (nhập kho)
POST   /api/catalog                 (tạo xe mod mới)
GET    /api/customers
PUT    /api/customers/:id           (manager only)
POST   /api/customers               (tạo khi bán)
GET    /api/logs?actor=&action=&from=&to=
POST   /api/events                  (manager only)

# Dev
GET    /api/admin/users
PUT    /api/admin/users/:id/role    (dev only)
```

---

## 8. Đề xuất cải thiện thêm (cần bạn xác nhận)

1. **Audit log bất biến**: mọi hành động ghi kèm `actor_id`, `ip`, `timestamp` — đã có trong plan.
2. **Soft delete** cho customers/inventory thay vì xoá cứng (giữ lịch sử bán).
3. **Lịch sử giá**: lưu giá tại thời điểm bán trong `sales` (không tham chiếu sống) để báo cáo chính xác.
4. **Rate limit** cho login & customer register (chống brute-force).
5. **Phân trang + search** cho danh sách xe/khách/log.
6. **i18n**: giao diện tiếng Việt mặc định, có thể thêm EN sau.
7. **Vòng quay may mắn**: cần định nghĩa phần thưởng & xác suất — ta thiết kế bảng `event_prizes` khi làm tới.
8. **Dashboard cho manager**: doanh thu, top khách, xe bán chạy (giai đoạn 2).

---

## 9. Lộ trình triển khai

- **Giai đoạn 0:** Khung dự án + Docker Compose + migration đầu tiên.
- **Giai đoạn 1:** Auth (nhân viên + khách) + RBAC.
- **Giai đoạn 2:** Catalog + script tải data xe GTA5 (tải **ảnh về tự host** trong `frontend/public/vehicles/` hoặc thư mục static của backend; DB lưu đường dẫn nội bộ) + nhập kho.
- **Giai đoạn 3:** Bán xe + khách hàng + ranking tự động.
- **Giai đoạn 4:** Sự kiện + vòng quay + đăng ký.
- **Giai đoạn 5:** Log + filter + UI hoàn thiện.
