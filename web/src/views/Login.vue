<template>
  <div class="auth-wrapper">
    <!-- Animated tech background -->
    <div class="bg-animation">
      <div class="grid-lines"></div>
      <div class="orb orb-1"></div>
      <div class="orb orb-2"></div>
      <div class="orb orb-3"></div>
      <div class="particles">
        <span v-for="i in 20" :key="i" class="particle" :style="particleStyle(i)"></span>
      </div>
    </div>

    <div class="auth-container">
      <!-- Left: brand panel -->
      <div class="brand-panel">
        <div class="brand-content">
          <div class="logo-icon">
            <svg viewBox="0 0 48 48" fill="none">
              <rect x="4" y="24" width="8" height="20" rx="2" fill="currentColor" opacity="0.4"/>
              <rect x="16" y="16" width="8" height="28" rx="2" fill="currentColor" opacity="0.6"/>
              <rect x="28" y="8" width="8" height="36" rx="2" fill="currentColor" opacity="0.8"/>
              <rect x="40" y="4" width="8" height="40" rx="2" fill="currentColor"/>
              <circle cx="44" cy="6" r="3" fill="#00d4ff"/>
              <circle cx="20" cy="18" r="2" fill="#00d4ff" opacity="0.7"/>
              <circle cx="8" cy="26" r="2" fill="#00d4ff" opacity="0.5"/>
            </svg>
          </div>
          <h1 class="brand-title">Skill Tree</h1>
          <p class="brand-subtitle">技能树学习与任务训练系统</p>
          <div class="brand-features">
            <div class="feature-item">
              <span class="feature-icon">◇</span> 15 个技能模块
            </div>
            <div class="feature-item">
              <span class="feature-icon">◇</span> 100+ 实战任务
            </div>
            <div class="feature-item">
              <span class="feature-icon">◇</span> AI 智能验收
            </div>
            <div class="feature-item">
              <span class="feature-icon">◇</span> 师徒协作成长
            </div>
          </div>
        </div>
        <div class="brand-footer">计算机协会 · 2024</div>
      </div>

      <!-- Right: login form -->
      <div class="form-panel">
        <div class="form-card">
          <div class="form-header">
            <h2>欢迎回来</h2>
            <p>登录你的账号继续学习之旅</p>
          </div>

          <el-form :model="form" :rules="rules" ref="formRef" class="login-form">
            <el-form-item prop="username">
              <el-input
                v-model="form.username"
                placeholder="用户名"
                size="large"
                :prefix-icon="UserIcon"
                class="custom-input"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="密码"
                size="large"
                show-password
                :prefix-icon="LockIcon"
                class="custom-input"
                @keyup.enter="login"
              />
            </el-form-item>
            <el-form-item>
              <el-button
                type="primary"
                size="large"
                @click="login"
                :loading="loading"
                class="login-btn"
              >
                <span v-if="!loading">登 录</span>
              </el-button>
            </el-form-item>
          </el-form>

          <div class="form-footer">
            <p class="switch-link">
              还没有账号？<router-link to="/register">立即注册 →</router-link>
            </p>
            <p class="switch-link" style="margin-top:6px">
              <router-link to="/forgot-password">忘记密码？</router-link>
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, h } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import { useAuthStore } from '../store/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref(null)
const loading = ref(false)

const UserIcon = { render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, style: 'width:18px;height:18px' }, [h('path', { d: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2' }), h('circle', { cx: 12, cy: 7, r: 4 })]) }
const LockIcon = { render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, style: 'width:18px;height:18px' }, [h('rect', { x: 3, y: 11, width: 18, height: 11, rx: 2 }), h('path', { d: 'M7 11V7a5 5 0 0 1 10 0v4' })]) }

const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

function particleStyle(i) {
  const left = Math.random() * 100
  const delay = Math.random() * 10
  const duration = 8 + Math.random() * 12
  const size = 2 + Math.random() * 4
  return {
    left: `${left}%`,
    animationDelay: `${delay}s`,
    animationDuration: `${duration}s`,
    width: `${size}px`,
    height: `${size}px`
  }
}

async function login() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await authApi.login(form)
    authStore.setAuth(res.data.data.token, res.data.data.user)
    ElMessage.success('登录成功')
    router.push('/learn')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-wrapper {
  position: relative;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  overflow: hidden;
  background: linear-gradient(135deg, #e8f0fe 0%, #f3e8ff 30%, #e0f2fe 60%, #f0f4ff 100%);
}

/* ===== Animated Tech Background ===== */
.bg-animation {
  position: absolute; inset: 0; pointer-events: none; z-index: 0;
}

.grid-lines {
  position: absolute; inset: 0;
  background-image:
    linear-gradient(rgba(99,102,241,.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(99,102,241,.06) 1px, transparent 1px);
  background-size: 60px 60px;
  animation: gridMove 20s linear infinite;
}

@keyframes gridMove {
  0% { transform: translate(0, 0); }
  100% { transform: translate(60px, 60px); }
}

.orb {
  position: absolute; border-radius: 50%;
  filter: blur(100px); opacity: 0.25;
}
.orb-1 { width: 500px; height: 500px; background: #6366f1; top: -150px; right: -150px; animation: orbFloat 8s ease-in-out infinite; }
.orb-2 { width: 350px; height: 350px; background: #8b5cf6; bottom: -100px; left: -100px; animation: orbFloat 10s ease-in-out infinite reverse; }
.orb-3 { width: 250px; height: 250px; background: #06b6d4; top: 45%; left: 35%; animation: orbFloat 12s ease-in-out infinite; }

@keyframes orbFloat {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(30px, -30px) scale(1.1); }
  66% { transform: translate(-20px, 20px) scale(0.9); }
}

.particles { position: absolute; inset: 0; }
.particle {
  position: absolute; bottom: -10px;
  background: #6366f1; border-radius: 50%;
  opacity: 0;
  animation: particleRise linear infinite;
}
@keyframes particleRise {
  0% { transform: translateY(0) scale(1); opacity: 0; }
  10% { opacity: 0.4; }
  90% { opacity: 0.1; }
  100% { transform: translateY(-100vh) scale(0); opacity: 0; }
}

/* ===== Container ===== */
.auth-container {
  position: relative; z-index: 1;
  display: flex;
  width: 960px; min-height: 560px;
  border-radius: 20px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(99,102,241,.12), 0 4px 24px rgba(0,0,0,.06);
}

/* ===== Left Brand Panel ===== */
.brand-panel {
  flex: 0 0 420px;
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 30%, #7c3aed 70%, #8b5cf6 100%);
  display: flex; flex-direction: column; justify-content: center; align-items: center;
  padding: 48px 36px;
  position: relative;
}
.brand-panel::before {
  content: '';
  position: absolute; top: 0; left: 0; right: 0; bottom: 0;
  background:
    radial-gradient(ellipse at 20% 20%, rgba(255,255,255,.15) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 80%, rgba(255,255,255,.08) 0%, transparent 50%);
}

.brand-content { position: relative; z-index: 1; text-align: center; }
.logo-icon { width: 72px; height: 72px; margin: 0 auto 20px; color: #fff; }
.logo-icon svg { width: 100%; height: 100%; }

.brand-title {
  font-size: 32px; font-weight: 800; color: #fff; letter-spacing: 2px;
  margin-bottom: 8px;
  text-shadow: 0 2px 8px rgba(0,0,0,.15);
}
.brand-subtitle { color: rgba(255,255,255,.8); font-size: 14px; margin-bottom: 36px; letter-spacing: 1px; }

.brand-features { text-align: left; display: inline-flex; flex-direction: column; gap: 14px; }
.feature-item { color: rgba(255,255,255,.85); font-size: 14px; display: flex; align-items: center; gap: 10px; }
.feature-icon { color: #fbbf24; font-size: 12px; }

.brand-footer { position: relative; z-index: 1; color: rgba(255,255,255,.45); font-size: 12px; margin-top: auto; letter-spacing: 2px; }

/* ===== Right Form Panel ===== */
.form-panel {
  flex: 1;
  background: #fff;
  display: flex; justify-content: center; align-items: center;
  padding: 48px 32px;
}
.form-card { width: 100%; max-width: 340px; }

.form-header { margin-bottom: 32px; }
.form-header h2 { color: #1e293b; font-size: 26px; font-weight: 700; margin-bottom: 6px; }
.form-header p { color: #94a3b8; font-size: 14px; }

.login-form { margin-bottom: 24px; }

:deep(.custom-input .el-input__wrapper) {
  background: #f8fafc !important;
  border: 1px solid #e2e8f0 !important;
  border-radius: 10px !important;
  box-shadow: none !important;
  transition: all .3s;
}
:deep(.custom-input .el-input__wrapper:hover) {
  border-color: #a5b4fc !important;
}
:deep(.custom-input .el-input__wrapper.is-focus) {
  border-color: #6366f1 !important;
  box-shadow: 0 0 0 3px rgba(99,102,241,.12) !important;
  background: #fff !important;
}
:deep(.custom-input .el-input__inner) {
  color: #1e293b !important;
}
:deep(.custom-input .el-input__inner::placeholder) {
  color: #94a3b8 !important;
}
:deep(.custom-input .el-input__prefix) {
  color: #94a3b8 !important;
}

.login-btn {
  width: 100%;
  height: 46px !important;
  border-radius: 10px !important;
  font-size: 16px !important;
  font-weight: 600 !important;
  letter-spacing: 4px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
  border: none !important;
  transition: all .3s;
}
.login-btn:hover {
  box-shadow: 0 8px 25px rgba(99,102,241,.4) !important;
  transform: translateY(-2px);
}

/* ===== Form Footer ===== */
.form-footer { text-align: center; }
.hint-text {
  display: block;
  color: #94a3b8;
  font-size: 12px;
  margin-bottom: 12px;
  padding: 6px 12px;
  background: #f1f5f9;
  border-radius: 6px;
  border: 1px dashed #cbd5e1;
}
.switch-link { color: #64748b; font-size: 14px; }
.switch-link a { color: #6366f1; text-decoration: none; font-weight: 500; transition: color .2s; }
.switch-link a:hover { color: #8b5cf6; }

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .auth-container { flex-direction: column; width: 90vw; min-height: auto; }
  .brand-panel { flex: 0 0 auto; padding: 32px 24px; }
  .brand-features { display: none; }
  .brand-title { font-size: 24px; }
  .form-panel { padding: 32px 24px; }
}
</style>
