<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'dev' })
const api = useApi()
const auth = useAuthStore()

const { data: users, refresh } = await useAsyncData('users', () => api.get<any[]>('/api/admin/users'))

// Chỉ THĂNG CẤP khách hàng ĐÃ CÓ TÀI KHOẢN lên nhân viên (không tạo tay).
const selectedRole = ref('staff')
const msg = ref(''); const okMsg = ref('')

const custSearch = ref('')
const custResults = ref<any[]>([])
const pickedCust = ref<any>(null)
const searching = ref(false)
let ct: any
watch(custSearch, (q) => {
  clearTimeout(ct)
  ct = setTimeout(async () => {
    if (q.trim().length < 2) { custResults.value = []; return }
    searching.value = true
    try {
      const all = await api.get<any[]>(`/api/customers?search=${encodeURIComponent(q)}`)
      // chỉ giữ khách ĐÃ TẠO TÀI KHOẢN (có username/đã claim)
      custResults.value = (all || []).filter((c) => c.username || c.claimed_at)
    } catch { custResults.value = [] }
    finally { searching.value = false }
  }, 250)
})
function pickCustomer(c: any) {
  pickedCust.value = c
  custResults.value = []
  custSearch.value = ''
}
function clearPicked() { pickedCust.value = null }

const roleLabel: Record<string, string> = { dev: 'Dev', manager: 'Quản lý', staff: 'Nhân viên' }

async function create() {
  msg.value = ''; okMsg.value = ''
  if (!pickedCust.value) { msg.value = 'Hãy chọn một khách hàng đã có tài khoản.'; return }
  const c = pickedCust.value
  try {
    // username = tài khoản khách đã đăng ký; mật khẩu để trống -> backend dùng lại mật khẩu của khách
    await api.post('/api/admin/users', {
      username: c.username || c.national_id,
      password: '',
      display_name: c.full_name,
      role: selectedRole.value,
      national_id: c.national_id,
    })
    okMsg.value = `Đã đặt ${c.full_name} (${c.national_id}) làm ${roleLabel[selectedRole.value]}. Khách đăng nhập bằng đúng tài khoản/mật khẩu đã đăng ký.`
    pickedCust.value = null; custSearch.value = ''; custResults.value = []; selectedRole.value = 'staff'
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Thăng cấp thất bại.' }
}

async function changeRole(u: any, role: string) {
  msg.value = ''
  try { await api.put(`/api/admin/users/${u.id}/role`, { role }); await refresh() }
  catch (e: any) { msg.value = e?.data?.error || 'Đổi quyền thất bại.' }
}

async function removeUser(u: any) {
  msg.value = ''; okMsg.value = ''
  if (!confirm(`Xoá nhân viên "${u.display_name}" (${u.username})?\nNếu nhân viên đã có giao dịch/sự kiện, hệ thống sẽ vô hiệu hoá để giữ lịch sử.`)) return
  try {
    const res = await api.del<any>(`/api/admin/users/${u.id}`)
    okMsg.value = res.hard ? `Đã xoá ${u.username}.` : `Đã vô hiệu hoá ${u.username} (vì đã có lịch sử hoạt động).`
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Xoá thất bại.' }
}
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">⚙️ Quản lý nhân viên (Dev)</h1>
    <p class="mb-4 text-sm text-slate-500">Chỉ Dev mới được thăng cấp khách hàng thành nhân viên và đặt thứ hạng.</p>
    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <div class="grid gap-6 lg:grid-cols-3">
      <div class="card p-4">
        <h2 class="mb-1 font-semibold">Thăng cấp khách hàng → nhân viên</h2>
        <p class="mb-3 text-xs text-slate-500">Chỉ chọn được khách <strong>đã tạo tài khoản</strong>. Họ sẽ đăng nhập bằng đúng tài khoản/mật khẩu đã đăng ký.</p>

        <div class="space-y-3">
          <!-- chọn khách đã có tài khoản -->
          <div>
            <label class="label">Tìm khách hàng (đã có tài khoản)</label>
            <input v-model="custSearch" class="input" placeholder="Tìm theo tên, SĐT, căn cước…" />
            <p v-if="searching" class="mt-1 text-xs text-slate-400">Đang tìm…</p>
            <div v-else-if="custResults.length" class="mt-1 max-h-48 space-y-1 overflow-auto rounded-lg border bg-white p-1">
              <button v-for="c in custResults" :key="c.id" type="button"
                class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-brand-50"
                @click="pickCustomer(c)">
                <strong>{{ c.full_name }}</strong>
                <span class="text-slate-400"> · {{ c.national_id }}<span v-if="c.phone"> · {{ c.phone }}</span></span>
              </button>
            </div>
            <p v-else-if="custSearch.trim().length >= 2 && !pickedCust" class="mt-1 text-xs text-amber-600">Không tìm thấy khách đã tạo tài khoản phù hợp.</p>
          </div>

          <!-- khách đã chọn -->
          <div v-if="pickedCust" class="flex items-center justify-between rounded-lg bg-brand-50 px-3 py-2 text-sm text-brand-800 ring-1 ring-brand-100">
            <span>✓ <strong>{{ pickedCust.full_name }}</strong> ({{ pickedCust.national_id }})</span>
            <button class="text-slate-500 hover:text-red-600" @click="clearPicked">✕</button>
          </div>

          <div>
            <label class="label">Thứ hạng</label>
            <select v-model="selectedRole" class="input">
              <option value="staff">Nhân viên</option>
              <option value="manager">Quản lý</option>
              <option value="dev">Dev</option>
            </select>
          </div>
          <button class="btn-primary w-full" :disabled="!pickedCust" @click="create">
            {{ pickedCust ? 'Thăng cấp' : 'Chọn khách hàng trước' }}
          </button>
        </div>
      </div>

      <div class="card overflow-x-auto lg:col-span-2">
        <table class="w-full min-w-[520px] text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
            <tr><th class="p-3">Tài khoản</th><th class="p-3">Tên</th><th class="p-3">Thứ hạng</th><th class="p-3"></th></tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id" class="border-t">
              <td class="p-3 font-medium">{{ u.username }}</td>
              <td class="p-3">{{ u.display_name }}</td>
              <td class="p-3">
                <select
                  :value="u.role" class="rounded border px-2 py-1 text-xs"
                  :disabled="u.id === auth.user?.id"
                  @change="changeRole(u, ($event.target as HTMLSelectElement).value)"
                >
                  <option value="staff">Nhân viên</option>
                  <option value="manager">Quản lý</option>
                  <option value="dev">Dev</option>
                </select>
                <span v-if="u.id === auth.user?.id" class="ml-2 text-xs text-slate-400">(bạn)</span>
              </td>
              <td class="p-3 text-right">
                <button
                  v-if="u.id !== auth.user?.id"
                  class="text-xs font-medium text-red-600 hover:underline"
                  @click="removeUser(u)"
                >Xoá</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
