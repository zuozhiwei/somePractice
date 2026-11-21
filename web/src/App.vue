<template>
  <div v-if="authStore.token" class="app-layout">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <aside class="sidebar" :class="{ 'is-open': menuOpen }" @keydown.esc="closeMenu(true)">
      <header class="sidebar-header">
        <router-link class="sidebar-brand" to="/learn" @click="closeMenu()">
          <TrailMark />
          <span class="brand-text"><strong>课后工坊</strong><small>大学生项目练习空间</small></span>
        </router-link>
        <button ref="menuButton" type="button" class="mobile-toggle" aria-controls="side-content"
                :aria-expanded="menuOpen" @click="menuOpen = !menuOpen">
          <span>{{ menuOpen ? '收起' : '导航' }}</span>
          <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="menuOpen ? 'm6 6 12 12M18 6 6 18' : 'M4 6h16M4 12h16M4 18h16'" /></svg>
        </button>
      </header>

      <div id="side-content" class="side-content">
        <nav class="sidebar-nav" aria-label="主导航">
          <section v-for="group in navGroups" :key="group.title" class="nav-group" :aria-label="group.title">
            <h2>{{ group.title }}</h2>
            <router-link v-for="link in group.links" :key="link.path" :to="link.path"
                         class="nav-item" :class="{ active: currentPath === link.path }"
                         :aria-current="currentPath === link.path ? 'page' : undefined" @click="closeMenu()">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="link.icon"/></svg>
              <span>{{ link.name }}</span>
              <span v-if="link.path === '/chat' && chatStore.totalUnread > 0" class="nav-badge"
                    :aria-label="chatStore.totalUnread + ' 条未读消息'">{{ chatStore.totalUnread > 99 ? '99+' : chatStore.totalUnread }}</span>
            </router-link>
          </section>
        </nav>
        <footer class="sidebar-user">
          <router-link to="/profile" class="user-card" @click="closeMenu()">
            <span class="user-avatar">{{ userName.charAt(0).toUpperCase() }}</span>
            <span class="user-info"><strong>{{ userName }}</strong><small>{{ roleName }}</small></span>
          </router-link>
          <button type="button" class="sign-out" aria-label="退出登录" title="退出登录" @click="logout">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M10 4H5v16h5m5-12 4 4-4 4m-7-4h11"/></svg>
          </button>
        </footer>
      </div>
    </aside>
    <main id="main-content" tabindex="-1" class="main-content"><router-view /></main>
  </div>
  <div v-else-if="route.meta.public" class="public-layout">
    <a href="#main-content" class="skip-link">跳到主要内容</a>
    <WorkshopHeader />
    <main id="main-content" tabindex="-1"><router-view /></main>
  </div>
  <main v-else class="auth-only"><router-view /></main>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './store/auth'
import { useChatStore } from './store/chat'
import TrailMark from './components/TrailMark.vue'
import WorkshopHeader from './workshop/WorkshopHeader.vue'
import './styles/theme.css'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()
const menuOpen = ref(false)
const menuButton = ref(null)
const userName = computed(() => authStore.user?.name || authStore.user?.username || '用户')
const roleName = computed(() => ({ SUPERADMIN: '超级管理员', TEACHER: '教师', INTERNAL: '学员', EXTERNAL: '自主学习' }[authStore.user?.role] || ''))
const currentPath = computed(() => ['/map/', '/level/'].some(prefix => route.path.startsWith(prefix)) ? '/learn' : route.path)
const navGroups = computed(() => [
  { title: '学习空间', links: [
    { name: '校园树洞课程', path: '/learn', icon: 'm3 5 6-2 6 3 6-2v15l-6 2-6-3-6 2V5Zm6-2v15m6-12v15' }
  ] },
  { title: '协作', links: [
    { name: authStore.user?.role === 'EXTERNAL' ? '我的提交' : '学习伙伴', path: '/mentorship', icon: 'M15 8a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm-9 12v-2a6 6 0 0 1 12 0v2M18 5a3 3 0 0 1 0 6m1 3a4 4 0 0 1 3 4' },
    { name: '聊天', path: '/chat', icon: 'M4 4h16v12H9l-5 4V4Zm4 5h8m-8 3h4' },
    { name: '排行榜', path: '/leaderboard', icon: 'M7 4h10v4a5 5 0 0 1-10 0V4Zm0 2H4v2a4 4 0 0 0 4 4m9-6h3v2a4 4 0 0 1-4 4m-4 1v5m-4 2h8' }
  ] },
  { title: '工作台', links: [
    { name: '个人中心', path: '/profile', icon: 'M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0ZM4 21v-1a8 8 0 0 1 16 0v1' },
    ...(authStore.isTeacher || authStore.isSuperAdmin ? [
      { name: authStore.isSuperAdmin ? '系统管理' : '作业审核', path: '/admin', icon: 'M3 7h10m4 0h4M3 17h4m4 0h10M17 7a2 2 0 1 1-4 0 2 2 0 0 1 4 0ZM11 17a2 2 0 1 1-4 0 2 2 0 0 1 4 0Z' }
    ] : [])
  ] }
])

function closeMenu(returnFocus = false) {
  if (returnFocus && menuOpen.value) menuButton.value?.focus()
  menuOpen.value = false
}
function logout() {
  closeMenu()
  chatStore.disconnect()
  authStore.logout()
  router.push('/')
}
watch(() => route.fullPath, () => closeMenu())
watch(() => authStore.token, token => {
  if (token) {
    chatStore.connect()
  }
}, { immediate: true })
</script>
