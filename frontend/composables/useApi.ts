// useApiBase: URL gốc của backend.
export function useApiBase(): string {
  return useRuntimeConfig().public.apiBase as string
}

// useApi: gọi API có gắn Authorization. Tự refresh access token 1 lần khi gặp 401.
export function useApi() {
  const base = useApiBase()
  const auth = useAuthStore()

  async function call<T>(path: string, opts: any = {}, retried = false): Promise<T> {
    const headers: Record<string, string> = { ...(opts.headers || {}) }
    if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`
    try {
      return await $fetch<T>(`${base}${path}`, { ...opts, headers })
    } catch (e: any) {
      if (e?.response?.status === 401 && !retried && auth.refreshToken) {
        const ok = await auth.refresh()
        if (ok) return call<T>(path, opts, true)
      }
      throw e
    }
  }

  return {
    get: <T>(path: string, opts: any = {}) => call<T>(path, { ...opts, method: 'GET' }),
    post: <T>(path: string, body?: any, opts: any = {}) => call<T>(path, { ...opts, method: 'POST', body }),
    put: <T>(path: string, body?: any, opts: any = {}) => call<T>(path, { ...opts, method: 'PUT', body }),
    patch: <T>(path: string, body?: any, opts: any = {}) => call<T>(path, { ...opts, method: 'PATCH', body }),
    del: <T>(path: string, opts: any = {}) => call<T>(path, { ...opts, method: 'DELETE' }),
  }
}

// Định dạng tiền tệ (đồng).
export function formatMoney(n: number): string {
  return new Intl.NumberFormat('vi-VN').format(Math.round(n)) + ' $'
}

// Căn cước khách hàng phải có dạng "LUX" (in hoa) + 5 chữ số (vd LUX12345).
// Chuẩn hóa in hoa trước khi kiểm tra để khách gõ thường vẫn hợp lệ.
export function isValidNationalID(s?: string | null): boolean {
  return /^LUX[0-9]{5}$/.test((s || '').trim().toUpperCase())
}
export const NATIONAL_ID_HINT = 'Định dạng: LUX + 5 chữ số (vd LUX12345)'

// Múi giờ hiển thị cố định (độc lập trình duyệt/máy chủ).
function appTimezone(): string {
  try {
    return (useRuntimeConfig().public.timezone as string) || 'Asia/Ho_Chi_Minh'
  } catch {
    return 'Asia/Ho_Chi_Minh'
  }
}

// Định dạng ngày dd/mm/yyyy theo múi giờ nghiệp vụ.
export function formatDate(s?: string | Date | null): string {
  if (!s) return ''
  return new Date(s).toLocaleDateString('vi-VN', { timeZone: appTimezone() })
}

// Định dạng ngày + giờ theo múi giờ nghiệp vụ.
export function formatDateTime(s?: string | Date | null): string {
  if (!s) return ''
  return new Date(s).toLocaleString('vi-VN', {
    timeZone: appTimezone(),
    day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

// Nhãn trạng thái xe dễ hiểu.
export function vehicleStatusLabel(status: string): string {
  return (
    {
      on_sale: 'Đang mở bán',
      upcoming: 'Sắp mở bán',
      hidden: 'Đang ẩn',
      sold_out: 'Hết hàng',
    } as Record<string, string>
  )[status] || status
}
