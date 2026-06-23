// Chuẩn hóa gta_vehicles_raw.json -> gta_vehicles.json (gọn, dùng để seed).
// Đồng thời sinh ảnh placeholder SVG theo class vào frontend/public/vehicles/class/.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '../..')

const raw = JSON.parse(fs.readFileSync(path.join(__dirname, 'gta_vehicles_raw.json'), 'utf8'))

// Loại các Type không phải sản phẩm bán cho khách.
const EXCLUDE_TYPES = new Set(['TRAILER', 'TRAIN'])

const pretty = (s) =>
  (s || '')
    .toLowerCase()
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())

const seen = new Set()
const out = []
for (const v of raw) {
  if (EXCLUDE_TYPES.has(v.Type)) continue
  const code = (v.Name || '').toLowerCase()
  if (!code || seen.has(code)) continue
  seen.add(code)
  out.push({
    model_code: code,
    name: (v.DisplayName?.English || v.Name || code).trim(),
    brand: (v.ManufacturerDisplayName?.English || v.Manufacturer || '').trim(),
    class: pretty(v.Class),
    class_id: v.Class,
    seats: v.Seats ?? null,
    default_price: Math.max(0, Math.round(v.MonetaryValue || v.Price || 0)),
  })
}

out.sort((a, b) => a.name.localeCompare(b.name))
fs.writeFileSync(path.join(__dirname, 'gta_vehicles.json'), JSON.stringify(out, null, 0))
console.log('vehicles written:', out.length)

// ── ảnh placeholder theo class (tự host) ─────────────────────────
const classes = [...new Set(out.map((v) => v.class_id))]
const dir = path.join(root, 'frontend/public/vehicles/class')
fs.mkdirSync(dir, { recursive: true })

const PURPLE = '#3b2a78'
const PURPLE_DARK = '#2a1d5c'
const GOLD = '#c6a15b'

for (const c of classes) {
  const label = pretty(c)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="480" height="300" viewBox="0 0 480 300">
  <defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="${PURPLE}"/><stop offset="1" stop-color="${PURPLE_DARK}"/>
  </linearGradient></defs>
  <rect width="480" height="300" fill="url(#g)"/>
  <g fill="none" stroke="${GOLD}" stroke-width="2" opacity="0.85">
    <path d="M70 190 q15-55 60-58 l120 0 q40 0 60 30 l40 6 q20 4 20 22 l0 8 -360 0 z"/>
    <circle cx="135" cy="200" r="26"/><circle cx="345" cy="200" r="26"/>
  </g>
  <text x="240" y="265" font-family="Georgia, serif" font-size="26" fill="#fff" text-anchor="middle" letter-spacing="2">${label}</text>
  <text x="240" y="45" font-family="Georgia, serif" font-size="20" fill="${GOLD}" text-anchor="middle" letter-spacing="4">KANJI GROUP</text>
</svg>`
  fs.writeFileSync(path.join(dir, `${c}.svg`), svg)
}
console.log('class images written:', classes.length)
