import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../store/auth'

const routes = [
  { path: '/', redirect: '/learn' },
  { path: '/login', component: () => import('../views/Login.vue'), meta: { guest: true } },
  { path: '/register', component: () => import('../views/Register.vue'), meta: { guest: true } },
  { path: '/forgot-password', component: () => import('../views/ForgotPassword.vue'), meta: { guest: true } },
  { path: '/learn', component: () => import('../views/LearnBoard.vue') },
  // 三种展示形式已合并进 /learn，旧路由重定向
  { path: '/map', redirect: '/learn' },
  { path: '/world', redirect: '/learn' },
  { path: '/skill-tree', redirect: '/learn' },
  { path: '/skill/:id', redirect: to => `/level/${to.params.id}` },
  { path: '/map/:mapId', component: () => import('../views/MapDetail.vue') },
  { path: '/level/:id', component: () => import('../views/LevelDetail.vue') },
  { path: '/profile', component: () => import('../views/Profile.vue') },
  { path: '/leaderboard', component: () => import('../views/Leaderboard.vue') },
  { path: '/mentorship', component: () => import('../views/Mentorship.vue') },
  { path: '/chat', component: () => import('../views/Chat.vue') },
  { path: '/admin', component: () => import('../views/AdminDashboard.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
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
  } else {
    next()
  }
})

export default router
