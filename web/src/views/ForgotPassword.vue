<template>
  <AccountLayout title="找回密码" :description="step === 1 ? '验证码将发到账号绑定的邮箱。' : '输入邮件中的验证码并设置新密码。'">
    <el-form v-if="step === 1" label-position="top" @submit.prevent="sendCode">
      <el-form-item label="注册邮箱">
        <el-input v-model.trim="email" name="email" type="email" autocomplete="email" placeholder="请输入注册邮箱" size="large" required />
      </el-form-item>
      <el-button class="account-submit" type="primary" native-type="submit" size="large" :loading="sending">获取验证码</el-button>
    </el-form>
    <el-form v-else label-position="top" @submit.prevent="resetPwd">
      <el-form-item label="注册邮箱">
        <el-input v-model.trim="email" type="email" autocomplete="email" name="email" size="large" required />
      </el-form-item>
      <el-form-item label="验证码">
        <el-input v-model.trim="code" name="code" autocomplete="one-time-code" inputmode="numeric" maxlength="6" placeholder="6 位数字" size="large" required />
      </el-form-item>
      <el-form-item label="新密码">
        <el-input v-model="newPassword" name="newPassword" type="password" autocomplete="new-password" show-password placeholder="至少 6 位" size="large" />
      </el-form-item>
      <el-form-item label="确认密码">
        <el-input v-model="confirmPassword" name="confirmPassword" type="password" autocomplete="new-password" show-password placeholder="再次输入新密码" size="large" />
      </el-form-item>
      <el-button class="account-submit" type="primary" native-type="submit" size="large" :loading="resetting">重置密码</el-button>
    </el-form>
    <div class="account-links">
      <router-link to="/login">返回登录</router-link>
      <button v-if="step === 1" type="button" :disabled="sending" @click="step = 2">已有验证码</button>
      <button v-else type="button" :disabled="resetting" @click="step = 1">重新获取验证码</button>
    </div>
  </AccountLayout>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import AccountLayout from '../components/AccountLayout.vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const step = ref(1)
const email = ref('')
const code = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const sending = ref(false)
const resetting = ref(false)

async function sendCode() {
  if (sending.value) return
  if (!email.value) {
    ElMessage.warning('请输入邮箱')
    return
  }
  sending.value = true
  try {
    await authApi.forgotPassword(email.value)
    ElMessage.success('验证码已发送，请查收邮件')
    step.value = 2
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '发送失败')
  } finally { sending.value = false }
}

async function resetPwd() {
  if (resetting.value) return
  if (!email.value || !code.value || !newPassword.value) {
    ElMessage.warning('请填写完整')
    return
  }
  if (!/^\d{6}$/.test(code.value)) {
    ElMessage.warning('请输入 6 位数字验证码')
    return
  }
  if (newPassword.value.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    ElMessage.warning('两次密码不一致')
    return
  }
  resetting.value = true
  try {
    await authApi.resetPassword({ email: email.value, code: code.value, newPassword: newPassword.value })
    ElMessage.success('密码已重置，请登录')
    router.push('/login')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '重置失败')
  } finally { resetting.value = false }
}
</script>
