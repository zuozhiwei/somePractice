<template>
  <div class="fp-wrapper">
    <div class="fp-card">
      <div class="fp-header">
        <h2>忘记密码</h2>
        <p>输入注册邮箱，获取验证码重置密码</p>
      </div>

      <!-- Step 1: Enter email & send code -->
      <div v-if="step === 1">
        <el-form label-position="top">
          <el-form-item label="注册邮箱">
            <el-input v-model="email" placeholder="请输入注册时填写的邮箱" size="large" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" style="width:100%" @click="sendCode" :loading="sending">
              {{ sending ? '发送中...' : '获取验证码' }}
            </el-button>
          </el-form-item>
        </el-form>
        <p class="fp-link">
          <router-link to="/login">← 返回登录</router-link>
        </p>
      </div>

      <!-- Step 2: Enter code & new password -->
      <div v-else>
        <el-form label-position="top">
          <el-form-item label="验证码">
            <el-input v-model="code" placeholder="6位数字验证码" size="large" maxlength="6" />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="newPassword" type="password" show-password
                      placeholder="至少6位" size="large" />
          </el-form-item>
          <el-form-item label="确认密码">
            <el-input v-model="confirmPassword" type="password" show-password
                      placeholder="再次输入新密码" size="large" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" style="width:100%" @click="resetPwd" :loading="resetting">
              {{ resetting ? '重置中...' : '重置密码' }}
            </el-button>
          </el-form-item>
        </el-form>
        <p class="fp-link">
          <a href="#" @click.prevent="step = 1">← 重新发送验证码</a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
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
  if (!code.value || !newPassword.value) {
    ElMessage.warning('请填写完整')
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

<style scoped>
.fp-wrapper {
  min-height: 100vh; display: flex; justify-content: center; align-items: center;
  background: linear-gradient(135deg, #e8f0fe 0%, #f3e8ff 30%, #e0f2fe 60%, #f0f4ff 100%);
}
.fp-card {
  width: 420px; padding: 44px 40px;
  background: #fff; border-radius: 20px;
  box-shadow: 0 20px 60px rgba(99,102,241,.1), 0 4px 16px rgba(0,0,0,.04);
}
.fp-header { margin-bottom: 28px; text-align: center; }
.fp-header h2 { color: #1e293b; font-size: 26px; font-weight: 700; margin-bottom: 8px; }
.fp-header p { color: #94a3b8; font-size: 14px; }
.fp-link { text-align: center; margin-top: 18px; }
.fp-link a { color: #6366f1; text-decoration: none; font-size: 14px; font-weight: 500; transition: color .2s; }
.fp-link a:hover { color: #8b5cf6; }

:deep(.el-input__wrapper) {
  background: #f8fafc !important;
  border: 1px solid #e2e8f0 !important;
  border-radius: 10px !important;
  box-shadow: none !important;
  transition: all .3s;
}
:deep(.el-input__wrapper:hover) {
  border-color: #a5b4fc !important;
}
:deep(.el-input__wrapper.is-focus) {
  border-color: #6366f1 !important;
  box-shadow: 0 0 0 3px rgba(99,102,241,.1) !important;
}
:deep(.el-input__inner) { color: #1e293b !important; }
:deep(.el-input__inner::placeholder) { color: #94a3b8 !important; }
:deep(.el-form-item__label) { color: #475569 !important; font-weight: 500; }

:deep(.el-button--primary) {
  background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
  border: none !important;
  border-radius: 10px !important;
  font-weight: 600;
  transition: all .3s;
}
:deep(.el-button--primary:hover) {
  box-shadow: 0 6px 20px rgba(99,102,241,.35) !important;
  transform: translateY(-1px);
}
</style>
