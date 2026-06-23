// Sinh nội dung giới thiệu tiếng Việt GỐC cho tất cả xe, dựa trên dữ kiện thật
// (tên, hãng, dòng xe/class, số chỗ). Không sao chép nội dung có bản quyền.
// - Ghi description trở lại gta_vehicles.json (để seed dùng được).
// - Sinh db/seed/update_descriptions.sql để cập nhật DB hiện tại.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const file = path.join(__dirname, 'gta_vehicles.json')
const list = JSON.parse(fs.readFileSync(file, 'utf8'))

// Cụm mô tả loại xe + các câu nhấn theo class.
const CLASS = {
  SUPER:         { type: 'siêu xe thuộc dòng Super', flavor: ['Sở hữu tốc độ tối đa và khả năng tăng tốc thuộc hàng đỉnh cao.', 'Thiết kế khí động học cùng động cơ mạnh mẽ mang lại trải nghiệm lái đẳng cấp.', 'Lựa chọn trong mơ cho những tay đua khao khát tốc độ.'] },
  SPORT:         { type: 'xe thể thao (Sports)', flavor: ['Cân bằng tốt giữa tốc độ và khả năng vào cua linh hoạt.', 'Phù hợp cho cả đua phố lẫn di chuyển hằng ngày.', 'Vận hành nhanh nhẹn, kiểu dáng năng động.'] },
  SPORT_CLASSIC: { type: 'xe thể thao cổ điển (Sports Classic)', flavor: ['Vẻ đẹp hoài cổ kết hợp cảm giác lái đầy chất chơi.', 'Mẫu xe sưu tầm được giới mê xe cổ săn lùng.', 'Dáng xe vượt thời gian, đậm chất di sản.'] },
  MUSCLE:        { type: 'xe cơ bắp (Muscle)', flavor: ['Động cơ gầm gừ, sức kéo mạnh đặc trưng xe Mỹ.', 'Phong cách hầm hố, uy lực trên mọi cung đường.', 'Lựa chọn của những ai yêu sức mạnh thuần túy.'] },
  SEDAN:         { type: 'xe sedan', flavor: ['Rộng rãi, êm ái, lý tưởng cho di chuyển hằng ngày.', 'Cân bằng giữa sự tiện nghi và thực dụng.', 'Mẫu xe gia đình đáng tin cậy.'] },
  SUV:           { type: 'xe gầm cao đa dụng (SUV)', flavor: ['Gầm cao, khoang rộng, phù hợp nhiều địa hình.', 'Vừa mạnh mẽ vừa tiện nghi cho cả gia đình.', 'Linh hoạt giữa phố thị và đường xấu.'] },
  COMPACT:       { type: 'xe đô thị cỡ nhỏ (Compact)', flavor: ['Nhỏ gọn, dễ luồn lách và đỗ trong thành phố.', 'Tiết kiệm, linh hoạt cho nhu cầu hằng ngày.', 'Gọn nhẹ nhưng vẫn đầy đủ tiện ích.'] },
  COUPE:         { type: 'xe coupe 2 cửa', flavor: ['Kiểu dáng thể thao, cá tính và lịch lãm.', 'Đường nét gọn gàng, thể hiện gu thẩm mỹ riêng.', 'Lựa chọn phong cách cho người trẻ.'] },
  OFF_ROAD:      { type: 'xe địa hình (Off-Road)', flavor: ['Chinh phục mọi địa hình gồ ghề, bùn lầy.', 'Khung gầm chắc chắn, lốp lớn bám đường tốt.', 'Bạn đồng hành cho những chuyến phiêu lưu.'] },
  MOTORCYCLE:    { type: 'mô tô', flavor: ['Nhanh nhẹn, len lỏi linh hoạt giữa dòng xe.', 'Cảm giác lái phấn khích, tốc độ ấn tượng.', 'Gọn gàng và cơ động bậc nhất.'] },
  VAN:           { type: 'xe van', flavor: ['Khoang chứa rộng, hữu dụng cho công việc.', 'Thực dụng, chở được nhiều người và hàng hóa.', 'Bền bỉ cho nhu cầu vận chuyển.'] },
  COMMERCIAL:    { type: 'xe thương mại cỡ lớn', flavor: ['Sức chở lớn, phục vụ vận tải hàng hóa.', 'Mạnh mẽ, bền bỉ cho công việc nặng.', 'Xương sống của hoạt động hậu cần.'] },
  INDUSTRIAL:    { type: 'xe công nghiệp', flavor: ['Cỗ máy chuyên dụng cho công trường.', 'Sức mạnh và độ bền cho công việc nặng nhọc.', 'Thiết kế thực dụng, vận hành ổn định.'] },
  SERVICE:       { type: 'xe dịch vụ', flavor: ['Phục vụ các nhu cầu vận hành đô thị.', 'Thực dụng và đáng tin cậy.', 'Quen thuộc trên mọi nẻo đường thành phố.'] },
  UTILITY:       { type: 'xe tiện ích', flavor: ['Đa năng cho nhiều mục đích công việc.', 'Chắc chắn và hữu dụng.', 'Hỗ trợ tốt cho công việc thường nhật.'] },
  EMERGENCY:     { type: 'xe khẩn cấp', flavor: ['Phục vụ nhiệm vụ cứu hộ, an ninh.', 'Trang bị chuyên dụng, phản ứng nhanh.', 'Luôn sẵn sàng trong tình huống khẩn cấp.'] },
  MILITARY:      { type: 'phương tiện quân sự', flavor: ['Bọc thép, sức mạnh áp đảo trên chiến trường.', 'Thiết kế cho nhiệm vụ hạng nặng.', 'Uy lực và độ bền vượt trội.'] },
  OPEN_WHEEL:    { type: 'xe đua bánh hở (Open Wheel)', flavor: ['Tốc độ thuần đường đua, bám đường cực tốt.', 'Cảm giác lái như tay đua chuyên nghiệp.', 'Đỉnh cao của hiệu năng tốc độ.'] },
  CYCLE:         { type: 'xe đạp', flavor: ['Thân thiện môi trường, rèn luyện sức khỏe.', 'Nhẹ nhàng, linh hoạt trong phố.', 'Lựa chọn xanh cho di chuyển ngắn.'] },
  BOAT:          { type: 'tàu thuyền', flavor: ['Lướt sóng mạnh mẽ trên mặt nước.', 'Trải nghiệm tự do giữa biển khơi.', 'Phương tiện lý tưởng cho hành trình trên nước.'] },
  PLANE:         { type: 'máy bay', flavor: ['Chinh phục bầu trời với tầm bay ấn tượng.', 'Đưa bạn vượt khoảng cách trong nháy mắt.', 'Tự do tung cánh trên không trung.'] },
  HELICOPTER:    { type: 'trực thăng', flavor: ['Cơ động cao, cất hạ cánh linh hoạt.', 'Tầm nhìn toàn cảnh từ trên cao.', 'Di chuyển nhanh, không ngại kẹt đường.'] },
}

const DEFAULT = { type: 'phương tiện', flavor: ['Một lựa chọn đáng cân nhắc trong bộ sưu tập của bạn.'] }

function describe(v) {
  const c = CLASS[v.class_id] || DEFAULT
  const idx = [...v.name].reduce((a, ch) => a + ch.charCodeAt(0), 0) % c.flavor.length
  const brandPart = v.brand ? ` đến từ hãng ${v.brand}` : ''
  const seatsPart = v.seats ? ` Xe có ${v.seats} chỗ ngồi.` : ''
  return `${v.name} là ${c.type}${brandPart} trong GTA V. ${c.flavor[idx]}${seatsPart}`
}

const esc = (s) => `'${String(s).replace(/'/g, "''")}'`
const rows = []
for (const v of list) {
  v.description = describe(v)
  rows.push(`(${esc(v.model_code)}, ${esc(v.description)})`)
}

fs.writeFileSync(file, JSON.stringify(list, null, 0))
const sql =
  `-- Tự sinh bởi descriptions.mjs: nội dung giới thiệu tiếng Việt gốc.\n` +
  `UPDATE vehicle_catalog c SET description = v.descr\n` +
  `FROM (VALUES\n  ${rows.join(',\n  ')}\n) AS v(code, descr)\nWHERE c.model_code = v.code;\n`
fs.writeFileSync(path.join(__dirname, 'update_descriptions.sql'), sql)
console.log(`Đã sinh giới thiệu cho ${list.length} xe + update_descriptions.sql`)
