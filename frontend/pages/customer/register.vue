<script setup lang="ts">
const auth = useAuthStore()
const api = useApi()
const form = reactive({ username: '', password: '', national_id: '', full_name: '', phone: '' })
const err = ref('')
const loading = ref(false)

// tra cứu thông tin theo căn cước khi rời focus
const lookupMsg = ref<{ text: string; cls: string } | null>(null)
const lookingUp = ref(false)
const lookupDone = ref(false) // đã tra cứu xong chưa (mở khoá 2 field tên/SĐT)

// đổi căn cước -> in hoa toàn bộ + đồng bộ Tài khoản = căn cước + khoá lại 2 field
function onNationalIDInput() {
  form.national_id = form.national_id.toUpperCase()
  form.username = form.national_id // tài khoản đăng nhập chính là số căn cước
  lookupDone.value = false
  lookupMsg.value = null
}

async function lookupNationalID() {
  lookupMsg.value = null
  const nid = form.national_id.trim()
  if (!nid) { lookupDone.value = false; return }
  lookingUp.value = true
  try {
    const res = await api.get<any>(`/api/auth/customer/lookup?national_id=${encodeURIComponent(nid)}`)
    if (!res.found) {
      lookupMsg.value = { text: 'Chưa có thông tin cho số căn cước này. Vui lòng nhập thông tin bên dưới.', cls: 'text-slate-500' }
    } else if (res.claimed) {
      lookupMsg.value = { text: 'Số căn cước này đã có tài khoản. Vui lòng đăng nhập.', cls: 'text-red-600' }
    } else {
      form.full_name = res.full_name || ''
      form.phone = res.phone || ''
      lookupMsg.value = { text: 'Đã tìm thấy thông tin và điền sẵn cho bạn. Bạn có thể chỉnh lại nếu cần.', cls: 'text-green-600' }
    }
  } catch {
    lookupMsg.value = { text: 'Không tra cứu được, bạn có thể tự nhập thông tin.', cls: 'text-slate-500' }
  } finally {
    lookingUp.value = false
    lookupDone.value = true // dù có hay không có thông tin đều mở khoá để khách tự sửa
  }
}

async function submit() {
  err.value = ''
  if (!isValidNationalID(form.national_id)) {
    err.value = 'Số căn cước không hợp lệ. ' + NATIONAL_ID_HINT
    return
  }
  loading.value = true
  try {
    await auth.registerCustomer({ ...form })
    navigateTo('/')
  } catch (e: any) {
    err.value = e?.data?.error || 'Đăng ký thất bại.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-sm">
    <div class="card p-6">
      <div class="mb-6 text-center">
        <img src="/logo.png" class="mx-auto h-14 w-14 object-contain" alt="Kanji Group" />
        <h1 class="mt-2 font-serif text-xl font-bold text-brand-900">Đăng ký khách hàng</h1>
        <p class="text-xs text-slate-500">
          Nếu số căn cước đã được nhân viên tạo trước, thông tin & lịch sử mua của bạn sẽ được giữ lại.
        </p>
      </div>
      <form class="space-y-3" @submit.prevent="submit">
        <div>
          <label class="label">Số căn cước *</label>
          <input v-model="form.national_id" class="input uppercase placeholder:normal-case" placeholder="LUX12345" @blur="lookupNationalID" @input="onNationalIDInput" />
          <p v-if="lookingUp" class="mt-1 text-xs text-slate-400">Đang tra cứu…</p>
          <p v-else-if="lookupMsg" class="mt-1 text-xs" :class="lookupMsg.cls">{{ lookupMsg.text }}</p>
          <p v-else class="mt-1 text-xs text-slate-400">{{ NATIONAL_ID_HINT }}</p>
          <p class="mt-1 text-xs text-slate-400">
            💡 Cách lấy số căn cước: nhấn phím <kbd class="rounded border border-slate-300 bg-slate-100 px-1 font-mono text-[11px] text-slate-600">Esc</kbd> trong game, số căn cước sẽ hiển thị ở <strong>góc trên bên trái</strong> màn hình.
          </p>
        </div>
        <div>
          <label class="label">Tài khoản đăng nhập</label>
          <input
            :value="form.national_id || '—'"
            class="input cursor-not-allowed bg-slate-100 uppercase text-slate-500"
            readonly disabled autocomplete="username"
          />
          <p class="mt-1 text-xs text-slate-400">Tài khoản đăng nhập chính là số căn cước của bạn, không thể thay đổi.</p>
        </div>
        <div>
          <label class="label">Mật khẩu * (≥6 ký tự)</label>
          <input v-model="form.password" type="password" class="input" autocomplete="new-password" />
        </div>
        <div>
          <label class="label">Họ tên</label>
          <input
            v-model="form.full_name"
            class="input disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-400"
            :disabled="!lookupDone || lookingUp"
            :placeholder="lookupDone ? 'Họ tên' : 'Nhập số căn cước trước để kiểm tra'"
          />
        </div>
        <div>
          <label class="label">Số điện thoại</label>
          <input
            v-model="form.phone"
            class="input disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-400"
            :disabled="!lookupDone || lookingUp"
            :placeholder="lookupDone ? 'Số điện thoại' : 'Nhập số căn cước trước để kiểm tra'"
          />
        </div>
        <p v-if="err" class="text-sm text-red-600">{{ err }}</p>
        <button class="btn-primary w-full" :disabled="loading">{{ loading ? 'Đang xử lý…' : 'Đăng ký' }}</button>
      </form>
      <div class="mt-4 text-center text-sm text-slate-500">
        Đã có tài khoản? <NuxtLink to="/login" class="text-brand-600 hover:underline">Đăng nhập</NuxtLink>
      </div>
    </div>
  </div>
</template>
