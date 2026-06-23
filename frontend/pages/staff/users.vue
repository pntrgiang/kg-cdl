<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'dev' })
const api = useApi()
const auth = useAuthStore()

const { data: users, refresh } = await useAsyncData('users', () => api.get<any[]>('/api/admin/users'))

const form = reactive({ username: '', password: '', display_name: '', role: 'staff', national_id: '' })
const msg = ref(''); const okMsg = ref('')

// ── chọn dữ liệu từ khách hàng có sẵn ──
const showCustPick = ref(false)
const custSearch = ref('')
const custResults = ref<any[]>([])
const pickedCust = ref<any>(null)
let ct: any
watch(custSearch, (q) => {
  clearTimeout(ct)
  ct = setTimeout(async () => {
    if (q.trim().length < 2) { custResults.value = []; return }
    try { custResults.value = await api.get<any[]>(`/api/customers?search=${encodeURIComponent(q)}`) } catch { custResults.value = [] }
  }, 250)
})
// chọn 1 khách -> điền sẵn tên hiển thị + căn cước, gợi ý username = căn cước
function pickCustomer(c: any) {
  pickedCust.value = c
  form.display_name = c.full_name || ''
  form.national_id = c.national_id || ''
  if (!form.username.trim()) form.username = c.national_id || ''
  custResults.value = []
  custSearch.value = ''
}
function clearPicked() { pickedCust.value = null; form.national_id = '' }

async function create() {
  msg.value = ''; okMsg.value = ''
  try {
    await api.post('/api/admin/users', { ...form, national_id: form.national_id.trim() })
    okMsg.value = `Đã tạo ${form.username}.`
    form.username = form.password = form.display_name = form.national_id = ''
    form.role = 'staff'
    pickedCust.value = null
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Tạo nhân viên thất bại.' }
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

const roleLabel: Record<string, string> = { dev: 'Dev', manager: 'Quản lý', staff: 'Nhân viên' }
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">⚙️ Quản lý nhân viên (Dev)</h1>
    <p class="mb-4 text-sm text-slate-500">Chỉ Dev mới được tạo và đặt thứ hạng nhân viên.</p>
    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <div class="grid gap-6 lg:grid-cols-3">
      <div class="card p-4">
        <h2 class="mb-3 font-semibold">Tạo nhân viên</h2>

        <!-- chọn dữ liệu từ khách hàng có sẵn -->
        <div class="mb-3 rounded-lg border border-brand-100 bg-brand-50/40 p-3">
          <button class="flex w-full items-center justify-between text-sm font-medium text-brand-700" @click="showCustPick = !showCustPick">
            <span>👥 Lấy dữ liệu từ khách hàng có sẵn</span>
            <span class="text-xs">{{ showCustPick ? '▲' : '▼' }}</span>
          </button>
          <div v-if="showCustPick" class="mt-2">
            <input v-model="custSearch" class="input !py-1.5 text-sm" placeholder="Tìm khách theo tên, SĐT, căn cước…" />
            <div v-if="custResults.length" class="mt-1 max-h-44 space-y-1 overflow-auto rounded-lg border bg-white p-1">
              <button v-for="c in custResults" :key="c.id" type="button"
                class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-brand-50"
                @click="pickCustomer(c)">
                <strong>{{ c.full_name }}</strong>
                <span class="text-slate-400"> · {{ c.national_id }}<span v-if="c.phone"> · {{ c.phone }}</span></span>
              </button>
            </div>
            <p class="mt-1 text-xs text-slate-500">Chọn 1 khách để tự điền tên hiển thị + căn cước (và gợi ý tài khoản).</p>
          </div>
          <div v-if="pickedCust" class="mt-2 flex items-center justify-between rounded-md bg-brand-100 px-2 py-1 text-xs text-brand-800">
            <span>Đã lấy từ khách: <strong>{{ pickedCust.full_name }}</strong> ({{ pickedCust.national_id }})</span>
            <button class="text-slate-500 hover:text-red-600" @click="clearPicked">✕</button>
          </div>
        </div>

        <div class="space-y-3">
          <div><label class="label">Tài khoản</label><input v-model="form.username" class="input" /></div>
          <div><label class="label">Tên hiển thị</label><input v-model="form.display_name" class="input" /></div>
          <div>
            <label class="label">Số căn cước <span class="font-normal text-slate-400">(tuỳ chọn)</span></label>
            <input v-model="form.national_id" class="input uppercase placeholder:normal-case" placeholder="LUX12345"
              @input="form.national_id = form.national_id.toUpperCase()" />
          </div>
          <div><label class="label">Mật khẩu (≥6)</label><input v-model="form.password" type="password" class="input" /></div>
          <div>
            <label class="label">Thứ hạng</label>
            <select v-model="form.role" class="input">
              <option value="staff">Nhân viên</option>
              <option value="manager">Quản lý</option>
              <option value="dev">Dev</option>
            </select>
          </div>
          <button class="btn-primary w-full" @click="create">Tạo</button>
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
