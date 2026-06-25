// Sitemap động: các trang tĩnh công khai + từng trang chi tiết xe (đang bán & sắp mở bán).
// Lấy danh sách xe từ API; nếu API lỗi vẫn trả về sitemap với các trang tĩnh.
export default defineEventHandler(async (event) => {
  const cfg = useRuntimeConfig()
  const site = (cfg.public.siteUrl as string) || 'https://kg-cdl.ddns.net'
  const api = (cfg.public.apiBase as string) || site

  const urls: { loc: string; changefreq: string; priority: string }[] = [
    { loc: `${site}/`, changefreq: 'daily', priority: '1.0' },
    { loc: `${site}/upcoming`, changefreq: 'daily', priority: '0.8' },
    { loc: `${site}/events`, changefreq: 'weekly', priority: '0.6' },
  ]

  try {
    const [onsale, upcoming] = await Promise.all([
      $fetch<any[]>(`${api}/api/vehicles?status=on_sale`).catch(() => [] as any[]),
      $fetch<any[]>(`${api}/api/vehicles?status=upcoming`).catch(() => [] as any[]),
    ])
    const seen = new Set<number>()
    for (const v of [...(onsale || []), ...(upcoming || [])]) {
      if (v?.id && !seen.has(v.id)) {
        seen.add(v.id)
        urls.push({ loc: `${site}/vehicles/${v.id}`, changefreq: 'weekly', priority: '0.7' })
      }
    }
  } catch {
    // bỏ qua: vẫn trả sitemap với trang tĩnh
  }

  const body =
    `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n` +
    urls
      .map(
        (u) =>
          `  <url><loc>${u.loc}</loc><changefreq>${u.changefreq}</changefreq><priority>${u.priority}</priority></url>`,
      )
      .join('\n') +
    `\n</urlset>\n`

  setHeader(event, 'content-type', 'application/xml; charset=utf-8')
  setHeader(event, 'cache-control', 'public, max-age=3600')
  return body
})
