import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../store/auth'
import { authApi } from '../api/auth'

const routes = [
  { path: '/', component: () => import('../views/WorkshopHome.vue'), meta: { public: true } },
  { path: '/login', component: () => import('../views/Login.vue'), meta: { guest: true } },
  { path: '/register', component: () => import('../views/Register.vue'), meta: { guest: true } },
  { path: '/forgot-password', component: () => import('../views/ForgotPassword.vue'), meta: { guest: true } },
  {
    path: '/learn',
    component: () => import('../workshop/LearningLayout.vue'),
    children: [
      { path: '', name: 'learn', component: () => import('../views/LearnBoard.vue') },
      { path: '/map/:mapId', component: () => import('../views/MapDetail.vue') },
      { path: '/level/:id', component: () => import('../views/LevelDetail.vue') }
    ]
  },
  // 保留原教学链接与学习视图。
  { path: '/garden', redirect: '/learn?view=world' },
  { path: '/map', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'story' } }) },
  { path: '/world', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'world' } }) },
  { path: '/skill-tree', redirect: to => ({ path: '/learn', query: { ...to.query, view: 'tree' } }) },
  { path: '/skill/:id', redirect: to => `/level/${to.params.id}` },
  { path: '/profile', component: () => import('../views/Profile.vue') },
  { path: '/leaderboard', component: () => import('../views/Leaderboard.vue') },
  { path: '/mentorship', component: () => import('../views/Mentorship.vue') },
  { path: '/chat', component: () => import('../views/Chat.vue') },
  { path: '/admin', component: () => import('../views/AdminDashboard.vue'), meta: { staff: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    // Lesson headings are available after the lesson has loaded and Markdown has rendered.
    if (to.query.guide === '1' && to.hash && /^\/level\/\d+$/.test(to.path)) return
    if (savedPosition) return savedPosition
    if (to.hash === '#project-route' || to.hash === '#stages') return { el: to.hash, top: 20 }
    if (to.path !== from.path || to.query.view !== from.query.view) return { top: 0 }
  }
})

export function afterSignIn(destination) {
  if (typeof destination !== 'string' || !/^\/(?!\/)[^\\]*$/.test(destination)) return '/learn'
  const target = router.resolve(destination)
  return target.matched.length && !target.meta.guest ? target.fullPath : '/learn'
}

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  if (to.path === '/' && authStore.token) return next('/learn')
  if (to.meta.public) return next()
  if (to.meta.guest) {
    if (authStore.token) {
      next(afterSignIn(to.query.next))
    } else {
      next()
    }
    return
  }
  if (!authStore.token) {
    next({ path: '/login', query: { next: to.fullPath } })
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
