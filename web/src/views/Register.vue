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
              <circle cx="44" cy="6" r="3" fill="#7c3aed"/>
              <circle cx="20" cy="18" r="2" fill="#7c3aed" opacity="0.7"/>
              <circle cx="8" cy="26" r="2" fill="#7c3aed" opacity="0.5"/>
            </svg>
          </div>
          <h1 class="brand-title">加入我们</h1>
          <p class="brand-subtitle">开启你的技能升级之旅</p>
          <div class="brand-features">
            <div class="feature-item">
              <span class="feature-icon">◆</span> 前端 · 后端 · 数据库 · 运维
            </div>
            <div class="feature-item">
              <span class="feature-icon">◆</span> 逐级解锁，循序渐进
            </div>
            <div class="feature-item">
              <span class="feature-icon">◆</span> AI 审核 + 导师把关
            </div>
            <div class="feature-item">
              <span class="feature-icon">◆</span> 排行榜见证成长
            </div>
          </div>
        </div>
        <div class="brand-footer">计算机协会 · 2024</div>
      </div>

      <!-- Right: register form -->
      <div class="form-panel">
        <div class="form-card">
          <div class="form-header">
            <h2>创建账号</h2>
            <p>填写信息加入技能树学习系统</p>
          </div>

          <div v-if="isInvited" class="invite-banner">
            🧙 <b>师父邀请注册</b>：你正通过师父的邀请链接注册，将<b>默认成为内部成员</b>，无需选择身份或填写邀请码，注册成功即自动拜入其门下。
          </div>
          <div v-else-if="legacyInvite" class="invite-banner">
            此邀请链接已过期，请联系师父重新生成安全邀请链接。
          </div>

          <el-form :model="form" :rules="rules" ref="formRef" class="register-form">
            <el-form-item prop="username">
              <el-input
                v-model="form.username"
                placeholder="用户名"
                size="large"
                :prefix-icon="UserIcon"
                class="custom-input"
              />
            </el-form-item>
            <el-form-item prop="name">
              <el-input
                v-model="form.name"
                placeholder="姓名"
                size="large"
                class="custom-input"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="密码（至少6位）"
                size="large"
                show-password
                :prefix-icon="LockIcon"
                class="custom-input"
              />
            </el-form-item>
            <el-form-item prop="email">
              <el-input
                v-model="form.email"
                placeholder="邮箱"
                size="large"
                :prefix-icon="MailIcon"
                class="custom-input"
              />
            </el-form-item>
            <el-form-item v-if="!isInvited" prop="role">
              <el-select
                v-model="form.role"
                placeholder="选择角色"
                size="large"
                class="custom-select"
                style="width:100%"
              >
                <el-option label="🎓 内部成员（需邀请码）" value="INTERNAL" />
                <el-option label="👤 游客" value="EXTERNAL" />
              </el-select>
            </el-form-item>

            <el-form-item v-if="!isInvited && form.role === 'INTERNAL'" prop="teamInviteCode">
              <el-input
                v-model="form.teamInviteCode"
                placeholder="邀请码（内部成员必填，需向管理员获取）"
                size="large"
                class="custom-input"
              />
            </el-form-item>

            <el-form-item>
              <el-button
                type="primary"
                size="large"
                @click="register"
                :loading="loading"
                class="register-btn"
              >
                <span v-if="!loading">注 册</span>
              </el-button>
            </el-form-item>
          </el-form>

          <p class="switch-link">
            已有账号？<router-link to="/login">去登录 →</router-link>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, h, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authApi } from '../api/auth'
import { useAuthStore } from '../store/auth'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const inviteToken = ref(route.query.invite ? String(route.query.invite) : null)
const legacyInvite = computed(() => !!route.query.mentor && !inviteToken.value)
// 邀请注册：自动成为内部成员，隐藏身份选择与邀请码，只填基本信息
const isInvited = computed(() => !!inviteToken.value)
const formRef = ref(null)
const loading = ref(false)

const UserIcon = { render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, style: 'width:18px;height:18px' }, [h('path', { d: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2' }), h('circle', { cx: 12, cy: 7, r: 4 })]) }
const LockIcon = { render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, style: 'width:18px;height:18px' }, [h('rect', { x: 3, y: 11, width: 18, height: 11, rx: 2 }), h('path', { d: 'M7 11V7a5 5 0 0 1 10 0v4' })]) }
const MailIcon = { render: () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': 2, style: 'width:18px;height:18px' }, [h('rect', { x: 2, y: 4, width: 20, height: 16, rx: 2 }), h('path', { d: 'm22 4-10 8L2 4' })]) }

const form = reactive({
  username: '', name: '', password: '', email: '', role: isInvited.value ? 'INTERNAL' : 'EXTERNAL', inviteToken: inviteToken.value, teamInviteCode: ''
})
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, min: 6, message: '密码至少6位', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
  teamInviteCode: [{ required: true, message: '请输入邀请码', trigger: 'blur' }]
}

function particleStyle(i) {
  return {
    left: `${Math.random() * 100}%`,
    animationDelay: `${Math.random() * 10}s`,
    animationDuration: `${8 + Math.random() * 12}s`,
    width: `${2 + Math.random() * 4}px`,
    height: `${2 + Math.random() * 4}px`
  }
}

async function register() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await authApi.register(form)
    authStore.setAuth(res.data.data.token, res.data.data.user)
    ElMessage.success('注册成功')
    router.push('/learn')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '注册失败')
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
  width: 960px; min-height: 600px;
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

.form-header { margin-bottom: 28px; }
.form-header h2 { color: #1e293b; font-size: 26px; font-weight: 700; margin-bottom: 6px; }
.form-header p { color: #94a3b8; font-size: 14px; }

.register-form { margin-bottom: 20px; }
.invite-banner {
  font-size: 13px; line-height: 1.7;
  color: #92400e; background: #fef3c7;
  border: 1px solid #fde68a; border-radius: 10px;
  padding: 10px 14px; margin-bottom: 16px;
}

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

:deep(.custom-select .el-select__wrapper) {
  background: #f8fafc !important;
  border: 1px solid #e2e8f0 !important;
  border-radius: 10px !important;
  box-shadow: none !important;
}
:deep(.custom-select .el-select__wrapper:hover) {
  border-color: #a5b4fc !important;
}
:deep(.custom-select .el-select__wrapper.is-focus) {
  border-color: #6366f1 !important;
  box-shadow: 0 0 0 3px rgba(99,102,241,.12) !important;
}
:deep(.custom-select .el-select__placeholder) {
  color: #94a3b8 !important;
}
:deep(.custom-select .el-select__selected-item) {
  color: #1e293b !important;
}

.register-btn {
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
.register-btn:hover {
  box-shadow: 0 8px 25px rgba(99,102,241,.4) !important;
  transform: translateY(-2px);
}

/* ===== Transitions ===== */
.slide-fade-enter-active { transition: all .3s ease; }
.slide-fade-leave-active { transition: all .2s ease; }
.slide-fade-enter-from { opacity: 0; transform: translateY(-10px); }
.slide-fade-leave-to { opacity: 0; transform: translateY(-10px); }

/* ===== Form Footer ===== */
.switch-link { text-align: center; color: #64748b; font-size: 14px; }
.switch-link a { color: #6366f1; text-decoration: none; font-weight: 500; transition: color .2s; }
.switch-link a:hover { color: #8b5cf6; }

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .auth-container { flex-direction: column; width: 90vw; min-height: auto; }
  .brand-panel { flex: 0 0 auto; padding: 28px 24px; }
  .brand-features { display: none; }
  .brand-title { font-size: 24px; }
  .form-panel { padding: 28px 24px; }
}
</style>
