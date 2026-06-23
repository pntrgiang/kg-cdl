# Kanji Group — Car Dealer (FiveM)

Website quản lý & showroom cho doanh nghiệp car dealer trong FiveM (độc lập, không nối server game).
Tông màu trắng + tím, điểm nhấn vàng gold theo nhận diện **Kanji Group**.

- **Frontend:** Nuxt 3 + Tailwind + Pinia
- **Backend:** Go (chi + pgx) + PostgreSQL
- **Auth:** JWT access (30') + refresh token 90 ngày (thu hồi được)

Kiến trúc chi tiết: [`docs/PLAN.md`](docs/PLAN.md).

## Yêu cầu
- Go 1.23+ · Node 20+ / npm · Docker + Docker Compose

## Chạy local

```bash
cp .env.example .env          # 1. tạo env

make setup                    # 2. db + tạo bảng + seed (dev account + 881 xe GTA5)

make api                      # 3. backend → http://localhost:8080   (terminal 1)

make frontend-install         # 4. cài deps frontend (lần đầu)
make frontend                 # 5. Nuxt dev → http://localhost:3000  (terminal 2)

make vehicle-images           # 6. (tuỳ chọn) tải ~705 ảnh xe thật (~184MB) + cập nhật DB
make demo                     # 7. (tuỳ chọn) tạo dữ liệu demo: nhân viên, sự kiện…
make batch1                   # 8. (tuỳ chọn) đặt danh sách xe đang bán = đợt 1 (20/06/2026)
```

> **Xem xe:** trang chi tiết dùng ảnh render chính thức (đúng xe gốc 100%) có **phóng to / rê xem
> chi tiết** (`VehicleImageViewer`). Đã bỏ phương án 3D Sketchfab vì model cộng đồng thường là bản
> mod, không đảm bảo đúng xe.
>
> **Giới thiệu xe:** nội dung tiếng Việt **tự sinh từ dữ liệu thật** của xe (hãng/dòng/số chỗ),
> sạch bản quyền, sẵn cho cả 881 xe. Quản lý chỉnh sửa trực tiếp ở trang chi tiết xe;
> tạo lại bằng `make descriptions`. Mỗi xe có link "Xem thêm trên GTA Wiki" để ghi nguồn.
>
> **Ảnh xe:** dùng MỘT nguồn duy nhất [FiveM docs](https://docs.fivem.net/vehicles/)
> (render chính thức, nền trong suốt, `.webp`), tự host trong `frontend/public/vehicles/img/`.
> 872/881 xe có ảnh thật; 9 xe còn lại (mod/DLC) dùng placeholder SVG theo class.
> Lệnh seed tự dùng ảnh thật khi có.

Mở `http://localhost:3000`. Adminer (xem DB): `http://localhost:8081` (server `db`).

### Tài khoản

| Vai trò | Tài khoản | Mật khẩu | Tạo bởi |
|---------|-----------|----------|---------|
| Dev | `admin` | `admin123` | seed |
| Quản lý | `manager` | `manager123` | `make demo` |
| Nhân viên | `staff` | `staff123` | `make demo` |
| Khách hàng | tự đăng ký | — | trang Đăng ký khách hàng |

> Đổi mật khẩu dev và `JWT_SECRET` trước khi lên production.

## Tính năng

**Khách / công khai**
- Xem xe đang mở bán, sắp mở bán; giá gốc / % giảm / giá sau giảm; chi tiết xe + giới thiệu.
- Đăng ký / đăng nhập khách hàng. Tham gia sự kiện & quay vòng quay may mắn.
- Hạng khách (phổ thông/vip/svip) tự tính theo tổng chi tiêu, giới hạn svip≤3, vip≤5.

**Nhân viên (staff)**
- Bán xe (chọn xe trong kho + chọn/tạo khách hàng).
- Nhập kho từ danh mục 881 xe GTA5, hoặc tạo xe mod mới.
- Đặt giảm giá, đổi trạng thái xe; xem khách hàng; xem nhật ký (có filter).

**Quản lý (manager)** — toàn bộ quyền staff + tạo sự kiện + cập nhật khách hàng.

**Dev** — tạo nhân viên & đặt thứ hạng (staff/manager không tự đổi quyền nhau).

## Kiểm thử

```bash
bash backend/scripts/e2e_test.sh   # 13 kịch bản: RBAC, ranking, claim, vòng quay, refresh, log
```

## Cấu trúc

```
docs/        Tài liệu kế hoạch & kiến trúc
db/          Migration PostgreSQL + seed data xe GTA5
backend/     Go API (cmd/{api,migrate,seed}, internal/{auth,store,server,...})
frontend/    Nuxt 3 (pages, components, stores, composables, middleware)
img/         Logo & ảnh nhận diện
```

## Lộ trình
- [x] Giai đoạn 0 — khung dự án
- [x] Giai đoạn 1 — Auth + RBAC (access/refresh, dev/manager/staff/customer)
- [x] Giai đoạn 2 — Catalog 881 xe GTA5 + nhập kho + xe mod
- [x] Giai đoạn 3 — Bán xe + khách hàng + ranking tự động
- [x] Giai đoạn 4 — Sự kiện + vòng quay may mắn (random ở backend)
- [x] Giai đoạn 5 — Nhật ký + filter + UI hoàn thiện
