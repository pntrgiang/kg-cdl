import { defineStore } from 'pinia'

interface TokenPair {
  access_token: string
  refresh_token: string
  expires_at: string
}

interface User {
  id: number
  username: string
  display_name: string
  role: 'dev' | 'manager' | 'staff'
}

interface Customer {
  id: number
  username: string | null
  full_name: string
  phone: string
  national_id: string
  rank: 'regular' | 'vip' | 'svip'
  total_spent: number
}

const STORAGE_KEY = 'kg_auth'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    accessToken: '' as string,
    refreshToken: '' as string,
    kind: '' as '' | 'user' | 'customer',
    user: null as User | null,
    customer: null as Customer | null,
    ready: false,
  }),

  getters: {
    isAuthed: (s) => !!s.accessToken,
    isUser: (s) => s.kind === 'user',
    isCustomer: (s) => s.kind === 'customer',
    role: (s) => s.user?.role ?? '',
    isManager: (s) => s.user?.role === 'manager' || s.user?.role === 'dev',
    isDev: (s) => s.user?.role === 'dev',
    displayName: (s) => s.user?.display_name ?? s.customer?.full_name ?? '',
  },

  actions: {
    // Khôi phục token từ localStorage (gọi 1 lần khi app khởi động).
    hydrate() {
      if (this.ready || !import.meta.client) return
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        try {
          const d = JSON.parse(raw)
          this.accessToken = d.accessToken || ''
          this.refreshToken = d.refreshToken || ''
          this.kind = d.kind || ''
          this.user = d.user || null
          this.customer = d.customer || null
        } catch {}
      }
      this.ready = true
    },

    persist() {
      if (!import.meta.client) return
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          accessToken: this.accessToken,
          refreshToken: this.refreshToken,
          kind: this.kind,
          user: this.user,
          customer: this.customer,
        }),
      )
    },

    setSession(kind: 'user' | 'customer', token: TokenPair, profile: User | Customer) {
      this.accessToken = token.access_token
      this.refreshToken = token.refresh_token
      this.kind = kind
      if (kind === 'user') this.user = profile as User
      else this.customer = profile as Customer
      this.persist()
    },

    async loginStaff(username: string, password: string) {
      const api = useApiBase()
      const res = await $fetch<{ token: TokenPair; user: User }>(`${api}/api/auth/login`, {
        method: 'POST',
        body: { username, password },
      })
      this.setSession('user', res.token, res.user)
    },

    async loginCustomer(username: string, password: string) {
      const api = useApiBase()
      const res = await $fetch<{ token: TokenPair; customer: Customer }>(`${api}/api/auth/customer/login`, {
        method: 'POST',
        body: { username, password },
      })
      this.setSession('customer', res.token, res.customer)
    },

    async registerCustomer(body: { username: string; password: string; national_id: string; full_name?: string; phone?: string }) {
      const api = useApiBase()
      const res = await $fetch<{ token: TokenPair; customer: Customer }>(`${api}/api/auth/customer/register`, {
        method: 'POST',
        body,
      })
      this.setSession('customer', res.token, res.customer)
    },

    // Làm mới access token bằng refresh token. Trả về true nếu thành công.
    async refresh(): Promise<boolean> {
      if (!this.refreshToken) return false
      const api = useApiBase()
      try {
        const res = await $fetch<{ token: TokenPair }>(`${api}/api/auth/refresh`, {
          method: 'POST',
          body: { refresh_token: this.refreshToken },
        })
        this.accessToken = res.token.access_token
        this.refreshToken = res.token.refresh_token
        this.persist()
        return true
      } catch {
        this.logout()
        return false
      }
    },

    async changePassword(oldPassword: string, newPassword: string) {
      const api = useApi()
      await api.post('/api/auth/change-password', { old_password: oldPassword, new_password: newPassword })
    },

    async logout() {
      const api = useApiBase()
      if (this.refreshToken) {
        try {
          await $fetch(`${api}/api/auth/logout`, { method: 'POST', body: { refresh_token: this.refreshToken } })
        } catch {}
      }
      this.accessToken = ''
      this.refreshToken = ''
      this.kind = ''
      this.user = null
      this.customer = null
      if (import.meta.client) localStorage.removeItem(STORAGE_KEY)
    },
  },
})
