<template>
  <div id="app">
    <div class="app-layout" v-if="authStore.token">
      <!-- Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-brand" @click="$router.push('/learn')">
          <div class="brand-icon">
            <svg viewBox="0 0 48 48" fill="none">
              <rect x="4" y="24" width="7" height="18" rx="2" fill="currentColor" opacity="0.5"/>
              <rect x="15" y="16" width="7" height="26" rx="2" fill="currentColor" opacity="0.7"/>
              <rect x="26" y="8" width="7" height="34" rx="2" fill="currentColor" opacity="0.85"/>
              <rect x="37" y="4" width="7" height="38" rx="2" fill="currentColor"/>
            </svg>
          </div>
          <div class="brand-text">
            <span class="brand-name">Skill Tree</span>
            <span class="brand-sub">技能树学习系统</span>
          </div>
        </div>

        <nav class="sidebar-nav">
          <div class="nav-group">
            <div class="nav-group-label">学习</div>
            <router-link to="/learn" class="nav-item nav-learn" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
              <span>冒险之旅</span>
            </router-link>
          </div>
          <div class="nav-group">
            <div class="nav-group-label">社交</div>
            <router-link to="/leaderboard" class="nav-item" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9H4.5a2.5 2.5 0 0 1 0-5C7 4 6 9 6 9z"/><path d="M18 9h1.5a2.5 2.5 0 0 0 0-5C17 4 18 9 18 9z"/><path d="M4 22h16"/><path d="M10 22V8h4v14"/><rect x="6" y="12" width="4" height="10"/><rect x="14" y="10" width="4" height="12"/></svg>
              <span>排行榜</span>
            </router-link>
            <router-link to="/mentorship" class="nav-item" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              <span>{{ authStore.user?.role === 'EXTERNAL' ? '我的提交' : '师徒' }}</span>
            </router-link>
            <router-link to="/chat" class="nav-item" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              <span>聊天</span>
              <span v-if="chatStore.totalUnread > 0" class="nav-badge">{{ chatStore.totalUnread > 99 ? '99+' : chatStore.totalUnread }}</span>
            </router-link>
          </div>
          <div class="nav-group">
            <div class="nav-group-label">系统</div>
            <router-link to="/profile" class="nav-item" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              <span>个人中心</span>
            </router-link>
            <router-link v-if="authStore.isTeacher" to="/admin" class="nav-item" active-class="active">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
              <span>管理</span>
            </router-link>
          </div>
        </nav>

        <div class="sidebar-progress" v-if="mapStore.progress">
          <div class="sp-row">
            <span class="sp-label">冒险进度</span>
            <span class="sp-value">{{ mapStore.completedLevels }}/{{ mapStore.totalLevels }}</span>
          </div>
          <div class="sp-bar">
            <div class="sp-fill" :style="{ width: mapStore.overallPct + '%' }"></div>
          </div>
          <div class="sp-current">{{ mapStore.currentMap || '未开始冒险' }}</div>
        </div>

        <div class="sidebar-user">
          <el-popover placement="right-start" :width="200" trigger="click">
            <template #reference>
              <div class="user-card">
                <el-avatar :size="36" class="user-avatar">{{ authStore.user?.username?.charAt(0)?.toUpperCase() }}</el-avatar>
                <div class="user-info">
                  <div class="user-name">{{ authStore.user?.username }}</div>
                  <div class="user-role">{{ roleLabel(authStore.user?.role) }}</div>
                </div>
              </div>
            </template>
            <div class="popover-menu">
              <div class="popover-item" @click="$router.push('/profile')">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                个人中心
              </div>
              <div class="popover-item" @click="$router.push('/learn')">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
                冒险之旅
              </div>
              <div class="popover-item danger" @click="logout">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16,17 21,12 16,7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
                退出登录
              </div>
            </div>
          </el-popover>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="main-content">
        <router-view />
      </main>
    </div>

    <!-- No sidebar for auth pages -->
    <div v-else class="auth-only">
      <router-view />
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from './store/auth'
import { useChatStore } from './store/chat'
import { useMapStore } from './store/map'
import { watch, onMounted } from 'vue'

const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()
const mapStore = useMapStore()

function roleLabel(r) {
  return { TEACHER: '教师', INTERNAL: '内部成员', EXTERNAL: '游客' }[r] || r
}

function logout() {
  chatStore.disconnect()
  authStore.logout()
  router.push('/login')
}

watch(() => authStore.token, (token, old) => {
  if (token && !old) {
    chatStore.connect()
    chatStore.refreshUnread()
    mapStore.fetchProgress()
  }
})

onMounted(() => {
  if (authStore.token) {
    chatStore.connect()
    chatStore.refreshUnread()
    mapStore.fetchProgress()
  }
})
</script>

<style>
:root {
  --sidebar-width: 220px;
  --color-bg: #f0f2f5;
  --color-surface: #ffffff;
  --color-primary: #6366f1;
  --color-primary-light: #818cf8;
  --color-text: #1e293b;
  --color-text-secondary: #64748b;
  --color-border: #e2e8f0;
  --radius: 12px;
  --shadow-sm: 0 1px 3px rgba(0,0,0,.06);
  --shadow-md: 0 4px 12px rgba(0,0,0,.06);
  --shadow-lg: 0 8px 24px rgba(0,0,0,.08);
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  background: var(--color-bg);
  color: var(--color-text);
  -webkit-font-smoothing: antialiased;
}

#app { min-height: 100vh; }

/* ===== Layout ===== */
.app-layout { display: flex; min-height: 100vh; }
.auth-only { min-height: 100vh; }

/* ===== Sidebar ===== */
.sidebar {
  width: var(--sidebar-width);
  background: linear-gradient(180deg, #1e1b4b 0%, #172033 100%);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0; left: 0; bottom: 0;
  z-index: 100;
  overflow-y: auto;
}

.sidebar-brand {
  display: flex; align-items: center; gap: 10px;
  padding: 20px 18px;
  cursor: pointer;
  border-bottom: 1px solid rgba(255,255,255,.06);
}
.brand-icon { width: 36px; height: 36px; color: #818cf8; flex-shrink: 0; }
.brand-icon svg { width: 100%; height: 100%; }
.brand-text { display: flex; flex-direction: column; }
.brand-name { font-size: 16px; font-weight: 700; color: #fff; letter-spacing: 1px; }
.brand-sub { font-size: 10px; color: rgba(255,255,255,.35); letter-spacing: 1px; }

.sidebar-nav { flex: 1; padding: 8px 10px; overflow-y: auto; }

.nav-group { display: flex; flex-direction: column; gap: 2px; margin-bottom: 12px; }
.nav-group-label {
  font-size: 10px; letter-spacing: 2px;
  color: rgba(255,255,255,.32);
  padding: 10px 14px 4px;
  text-transform: uppercase;
}

.sidebar-progress {
  margin: 0 14px 8px;
  padding: 10px 12px;
  background: rgba(255,255,255,.05);
  border-radius: 10px;
}
.sp-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.sp-label { font-size: 11px; color: rgba(255,255,255,.45); }
.sp-value { font-size: 11px; font-weight: 700; color: #a5b4fc; }
.sp-bar { height: 5px; background: rgba(255,255,255,.12); border-radius: 3px; overflow: hidden; }
.sp-fill {
  height: 100%; border-radius: 3px;
  background: linear-gradient(90deg, #6366f1, #8b5cf6);
  transition: width .4s;
}
.sp-current {
  margin-top: 6px;
  font-size: 11px; color: rgba(255,255,255,.35);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.nav-item {
  position: relative;
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  color: rgba(255,255,255,.55);
  text-decoration: none;
  font-size: 14px;
  transition: all .2s;
}
.nav-item svg { width: 20px; height: 20px; flex-shrink: 0; }
.nav-item:hover { color: #fff; background: rgba(255,255,255,.06); }
.nav-item.active {
  color: #fff;
  background: linear-gradient(135deg, rgba(99,102,241,.3), rgba(139,92,246,.2));
  box-shadow: 0 2px 8px rgba(99,102,241,.15);
}
.nav-item.active::before {
  content: '';
  position: absolute; left: -10px; top: 22%; bottom: 22%;
  width: 3px; border-radius: 2px;
  background: linear-gradient(180deg, #818cf8, #a78bfa);
}
.nav-learn {
  background: linear-gradient(135deg, rgba(99,102,241,.22), rgba(139,92,246,.16));
  border: 1px solid rgba(129,140,248,.3);
  color: #e0e7ff;
}
.nav-learn.active {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  box-shadow: 0 4px 14px rgba(99,102,241,.35);
}
.nav-badge {
  margin-left: auto;
  min-width: 18px; height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: #ef4444;
  color: #fff;
  font-size: 11px; font-weight: 600;
  display: inline-flex; align-items: center; justify-content: center;
  line-height: 1;
}

.sidebar-user { padding: 14px; border-top: 1px solid rgba(255,255,255,.06); }
.user-card {
  display: flex; align-items: center; gap: 10px;
  padding: 6px; border-radius: 8px;
  cursor: pointer; transition: background .2s;
}
.user-card:hover { background: rgba(255,255,255,.05); }
.user-avatar { background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-weight: 600; }
.user-name { font-size: 13px; color: #fff; font-weight: 500; }
.user-role { font-size: 11px; color: rgba(255,255,255,.4); }

.popover-menu { padding: 4px 0; }
.popover-item {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 14px; font-size: 13px; color: #334155;
  cursor: pointer; border-radius: 6px; transition: background .15s;
}
.popover-item:hover { background: #f1f5f9; }
.popover-item.danger { color: #ef4444; }
.popover-item.danger:hover { background: #fef2f2; }

/* ===== Main Content ===== */
.main-content {
  flex: 1;
  margin-left: var(--sidebar-width);
  padding: 32px;
  min-height: 100vh;
}

/* ===== Shared Page Styles ===== */
.page-title {
  font-size: 24px; font-weight: 700; color: var(--color-text);
  margin-bottom: 4px;
}
.page-subtitle {
  font-size: 14px; color: var(--color-text-secondary);
  margin-bottom: 24px;
}

.card-grid { display: grid; gap: 20px; }
.card-grid.cols-2 { grid-template-columns: 1fr 1fr; }
.card-grid.cols-3 { grid-template-columns: repeat(3, 1fr); }

.stat-card {
  background: var(--color-surface);
  border-radius: var(--radius);
  padding: 24px;
  box-shadow: var(--shadow-sm);
  display: flex; align-items: center; gap: 16px;
  transition: box-shadow .2s;
}
.stat-card:hover { box-shadow: var(--shadow-md); }
.stat-icon {
  width: 48px; height: 48px; border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
  font-size: 22px; flex-shrink: 0;
}
.stat-value { font-size: 28px; font-weight: 700; color: var(--color-text); }
.stat-label { font-size: 13px; color: var(--color-text-secondary); margin-top: 2px; }

.section-card {
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}
.section-header {
  padding: 18px 24px;
  border-bottom: 1px solid var(--color-border);
  font-size: 15px; font-weight: 600;
  display: flex; align-items: center; justify-content: space-between;
}
.section-body { padding: 24px; }

/* Element Plus overrides */
.el-card {
  border-radius: var(--radius) !important;
  border: none !important;
  box-shadow: var(--shadow-sm) !important;
}
.el-table {
  border-radius: var(--radius);
  overflow: hidden;
}
.el-table th.el-table__cell {
  background: #f8fafc;
  color: var(--color-text-secondary);
  font-weight: 600;
  font-size: 13px;
}
.el-tabs__nav-wrap::after { height: 1px; }
.el-tabs__item { font-size: 14px; }
.el-button--primary {
  background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
  border: none !important;
}
.el-button--primary:hover {
  opacity: .9;
}

@media (max-width: 768px) {
  .sidebar { width: 60px; }
  .sidebar .brand-text, .sidebar .nav-item span, .sidebar .user-info { display: none; }
  .sidebar-brand { justify-content: center; padding: 16px 0; }
  .nav-item { justify-content: center; padding: 12px; }
  .main-content { margin-left: 60px; padding: 20px; }
  .card-grid.cols-2, .card-grid.cols-3 { grid-template-columns: 1fr; }
}
</style>
