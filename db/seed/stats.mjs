// Sinh điểm hiệu năng (0-100) từ data dump game (dữ kiện, không bản quyền).
// - Ghi seats + rate_* trở lại gta_vehicles.json (để seed dùng).
// - Sinh db/seed/update_stats.sql để cập nhật DB hiện tại.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const raw = JSON.parse(fs.readFileSync(path.join(__dirname, 'gta_vehicles_raw.json'), 'utf8'))
const list = JSON.parse(fs.readFileSync(path.join(__dirname, 'gta_vehicles.json'), 'utf8'))

// mốc tham chiếu (tinh chỉnh để xe thường có thanh hợp lý), giá trị vượt mốc = 100.
const REF = { speed: 60, accel: 0.45, braking: 1.1, traction: 2.9 }
const pct = (val, ref) => Math.max(0, Math.min(100, Math.round(((val || 0) / ref) * 100)))

const byCode = new Map(raw.map((x) => [x.Name.toLowerCase(), x]))

const esc = (s) => `'${String(s).replace(/'/g, "''")}'`
const rows = []
for (const v of list) {
  const r = byCode.get(v.model_code)
  v.rate_speed = r ? pct(r.MaxSpeed, REF.speed) : 0
  v.rate_accel = r ? pct(r.Acceleration, REF.accel) : 0
  v.rate_braking = r ? pct(r.MaxBraking, REF.braking) : 0
  v.rate_traction = r ? pct(r.MaxTraction, REF.traction) : 0
  v.seats = v.seats ?? (r ? r.Seats : null)
  rows.push(
    `(${esc(v.model_code)}, ${v.seats ?? 'NULL'}, ${v.rate_speed}, ${v.rate_accel}, ${v.rate_braking}, ${v.rate_traction})`,
  )
}

fs.writeFileSync(path.join(__dirname, 'gta_vehicles.json'), JSON.stringify(list, null, 0))
const sql =
  `-- Tự sinh bởi stats.mjs: thông số hiệu năng chuẩn hóa.\n` +
  `UPDATE vehicle_catalog c SET seats = v.seats, rate_speed = v.sp, rate_accel = v.ac,\n` +
  `  rate_braking = v.br, rate_traction = v.tr\n` +
  `FROM (VALUES\n  ${rows.join(',\n  ')}\n) AS v(code, seats, sp, ac, br, tr)\nWHERE c.model_code = v.code;\n`
fs.writeFileSync(path.join(__dirname, 'update_stats.sql'), sql)
console.log(`Đã sinh thông số cho ${list.length} xe + update_stats.sql`)
