<template>
  <AccountLayout title="找回密码">
    <nav class="recovery-methods" aria-label="选择找回方式">
      <button v-for="item in methods" :key="item.key" type="button" :aria-pressed="method === item.key"
              :disabled="busy" @click="choose(item.key)">{{ item.label }}</button>
    </nav>
    <p class="recovery-help" role="status">{{ instruction }}</p>
    <el-form label-position="top" @submit.prevent="submit">
      <el-form-item :label="manual ? '登录用户名' : '登录用户名或绑定邮箱'">
        <el-input v-model.trim="identifier" name="username" autocomplete="username" :disabled="busy"
                  :placeholder="manual ? '请输入登录用户名' : '请输入用户名或邮箱'" maxlength="254" size="large" required />
      </el-form-item>
      <template v-if="enterCode">
        <el-form-item :label="manual ? '管理员恢复码' : '邮箱验证码'">
          <el-input v-model.trim="code" name="code" autocomplete="one-time-code" :disabled="busy"
                    :inputmode="manual ? 'text' : 'numeric'" :maxlength="manual ? 19 : 6"
                    :placeholder="manual ? '请输入管理员给你的恢复码' : '请输入邮件中的 6 位数字'" size="large" required />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="newPassword" name="newPassword" type="password" autocomplete="new-password" show-password placeholder="至少 6 位" size="large" required />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="confirmation" name="confirmPassword" type="password" autocomplete="new-password" show-password size="large" required />
        </el-form-item>
      </template>
      <p v-if="error" class="recovery-error" role="alert">{{ error }}</p>
      <el-button class="account-submit" type="primary" native-type="submit" size="large" :loading="busy">
        {{ enterCode ? '保存新密码' : '获取邮箱验证码' }}
      </el-button>
    </el-form>
    <div class="account-links">
      <router-link to="/login">返回登录</router-link>
      <button v-if="!manual" type="button" :disabled="busy" @click="toggleEmailStep">{{ enterCode ? '重新获取验证码' : '已有邮箱验证码' }}</button>
    </div>
  </AccountLayout>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AccountLayout from '../components/AccountLayout.vue'
import { authApi } from '../api/auth'

const router = useRouter()
const methods = [{ key: 'email', label: '邮箱验证码' }, { key: 'admin', label: '管理员恢复码' }]
const method = ref('email')
const emailStep = ref(false)
const requestAccepted = ref(false)
const identifier = ref('')
const code = ref('')
const newPassword = ref('')
const confirmation = ref('')
const busy = ref(false)
const error = ref('')
const manual = computed(() => method.value === 'admin')
const enterCode = computed(() => manual.value || emailStep.value)
const instruction = computed(() => {
  if (manual.value) return '没有绑定邮箱或收不到邮件？请把登录用户名告知任课教师或网站管理员。管理员核实身份后，会私下交给你一个 15 分钟内有效的恢复码。'
  if (requestAccepted.value) return '若账号绑定了可用邮箱，请在收件箱或垃圾邮件中查找验证码，5 分钟内有效。仍收不到可选择「管理员恢复码」。'
  if (emailStep.value) return '请输入账号信息和邮件中的验证码，设置新密码。'
  return '验证码将发往账号绑定的邮箱。没有绑定邮箱或收不到邮件时，可选择「管理员恢复码」。'
})

function resetFields() {
  code.value = ''
  error.value = ''
  newPassword.value = ''
  confirmation.value = ''
  requestAccepted.value = false
}
function choose(key) {
  if (busy.value || method.value === key) return
  method.value = key
  emailStep.value = false
  resetFields()
}
function toggleEmailStep() {
  emailStep.value = !emailStep.value
  resetFields()
}
async function submit() {
  if (busy.value) return
  error.value = ''
  if (!identifier.value) { error.value = '请输入登录用户名或绑定邮箱'; return }
  if (enterCode.value) {
    if (manual.value ? !/^[A-Z2-7]{16}$/i.test(code.value.replaceAll('-', '')) : !/^\d{6}$/.test(code.value)) {
      error.value = manual.value ? '请输入完整的管理员恢复码' : '请输入 6 位数字验证码'
      return
    }
    const bytes = new TextEncoder().encode(newPassword.value).length
    if (newPassword.value.length < 6 || bytes > 72) { error.value = '密码至少 6 位，长度不能超过 72 字节'; return }
    if (confirmation.value !== newPassword.value) { error.value = '两次密码不一致'; return }
  }
  busy.value = true
  try {
    if (!enterCode.value) {
      await authApi.forgotPassword(identifier.value)
      emailStep.value = true
      requestAccepted.value = true
    } else {
      await authApi.resetPassword({ identifier: identifier.value, method: method.value, code: code.value, newPassword: newPassword.value })
      ElMessage.success('密码已更新，请用新密码登录')
      await router.replace('/login')
    }
  } catch (e) {
    error.value = e.response?.data?.message || '请求失败，请稍后重试'
  } finally { busy.value = false }
}
</script>

<style scoped>
.recovery-methods { display: flex; border-bottom: 1px solid var(--color-border); gap: 16px; }
.recovery-methods button { background: none; border: 0; border-bottom: 2px solid transparent; padding: 8px 0; color: var(--color-text-secondary); }
.recovery-methods button[aria-pressed="true"] { border-color: var(--color-primary); color: var(--color-primary); font-weight: 650; }
.recovery-help { font-size: 13px; color: var(--color-text-secondary); line-height: 1.65; margin: 14px 0 18px; }
.recovery-error { color: #973823; margin-bottom: 12px; font-size: 13px; }
</style>
