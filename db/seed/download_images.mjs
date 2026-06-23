// Tải ảnh xe từ MỘT nguồn duy nhất: FiveM docs (render chính thức, nền trong suốt).
//   https://docs.fivem.net/vehicles/<model_code>.webp
// Lưu tự host vào frontend/public/vehicles/img/<model_code>.webp.
// Sinh db/seed/update_images.sql để cập nhật image_url cho các xe tải thành công.
//
// Dùng: node db/seed/download_images.mjs            (toàn bộ catalog)
//       node db/seed/download_images.mjs surfer ...  (chỉ vài code)
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '../..')
const outDir = path.join(root, 'frontend/public/vehicles/img')
fs.mkdirSync(outDir, { recursive: true })

const BASE = 'https://docs.fivem.net/vehicles'
const EXT = 'webp'
const CONCURRENCY = 24

const all = JSON.parse(fs.readFileSync(path.join(__dirname, 'gta_vehicles.json'), 'utf8'))
const argCodes = process.argv.slice(2)
const codes = argCodes.length ? argCodes : all.map((v) => v.model_code)

let ok = 0, miss = 0, done = 0
const succeeded = []

async function fetchOne(code) {
  const dest = path.join(outDir, `${code}.${EXT}`)
  if (fs.existsSync(dest) && fs.statSync(dest).size > 0) { succeeded.push(code); ok++; return }
  try {
    const res = await fetch(`${BASE}/${code}.${EXT}`)
    if (res.ok) {
      const buf = Buffer.from(await res.arrayBuffer())
      if (buf.length > 0) {
        fs.writeFileSync(dest, buf)
        succeeded.push(code)
        ok++
        return
      }
    }
    miss++
  } catch {
    miss++
  } finally {
    done++
    if (done % 100 === 0) console.log(`  ...${done}/${codes.length} (ok=${ok} miss=${miss})`)
  }
}

async function run() {
  for (let i = 0; i < codes.length; i += CONCURRENCY) {
    await Promise.all(codes.slice(i, i + CONCURRENCY).map(fetchOne))
  }
  console.log(`Xong: tải ${ok} ảnh, thiếu ${miss} (không có trong nguồn).`)

  // Sinh SQL cập nhật image_url cho TẤT CẢ xe được xét:
  //   có ảnh -> /vehicles/img/<code>.webp ; thiếu -> /vehicles/class/<class_id>.svg
  const done2 = new Set(succeeded)
  const esc = (s) => `'${String(s).replace(/'/g, "''")}'`
  const rows = all
    .filter((v) => codes.includes(v.model_code))
    .map((v) => {
      const url = done2.has(v.model_code)
        ? `/vehicles/img/${v.model_code}.${EXT}`
        : `/vehicles/class/${v.class_id}.svg`
      return `(${esc(v.model_code)}, ${esc(url)})`
    })
  const sql =
    `-- Tự sinh bởi download_images.mjs (nguồn: FiveM docs).\n` +
    `UPDATE vehicle_catalog c SET image_url = v.url\n` +
    `FROM (VALUES\n  ${rows.join(',\n  ')}\n) AS v(code, url)\nWHERE c.model_code = v.code;\n`
  fs.writeFileSync(path.join(__dirname, 'update_images.sql'), sql)
  console.log(`Đã ghi update_images.sql (${rows.length} xe).`)
}
run()
