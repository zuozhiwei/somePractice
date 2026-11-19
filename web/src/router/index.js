import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../store/auth'
import { authApi } from '../api/auth'

const routes = [
  { path: '/', redirect: '/learn' },
  { path: '/login', component: () => import('../views/Login.vue'), meta: { guest: true } },
  { path: '/register', component: () => import('../views/Register.vue'), meta: { guest: true } },
  { path: '/forgot-password', component: () => import('../views/ForgotPassword.vue'), meta: { guest: true } },
  { path: '/learn', component: () => import('../views/LearnBoard.vue') },
  // 三种展示形式已合并进 /learn，旧路由重定向
  { path: '/map', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'story' } }) },
  { path: '/world', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'world' } }) },
  { path: '/skill-tree', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'tree' } }) },
  { path: '/skill/:id', redirect: to => `/level/${to.params.id}` },
  { path: '/map/:mapId', component: () => import('../views/MapDetail.vue') },
  { path: '/level/:id', component: () => import('../views/LevelDetail.vue') },
  { path: '/profile', component: () => import('../views/Profile.vue') },
  { path: '/leaderboard', component: () => import('../views/Leaderboard.vue') },
  { path: '/mentorship', component: () => import('../views/Mentorship.vue') },
  { path: '/chat', component: () => import('../views/Chat.vue') },
  { path: '/admin', component: () => import('../views/AdminDashboard.vue'), meta: { staff: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  if (to.meta.guest) {
    if (authStore.token) {
      next('/learn')
    } else {
      next()
    }
    return
  }
  if (!authStore.token) {
    next('/login')
    return
  }
  if (to.meta.staff) {
    try {
      const token = authStore.token
      const res = await authApi.profile()
      if (authStore.token !== token) return next('/login')
      authStore.setAuth(token, res.data.data)
    } catch {
      return next(authStore.token ? '/learn' : '/login')
    }
    if (!authStore.isTeacher && !authStore.isSuperAdmin) return next('/learn')
  }
  next()
})

export default router
