<template>
  <header class="public-header">
    <RouterLink class="public-brand" to="/" aria-label="课后工坊首页">
      <TrailMark /><strong>课后工坊</strong>
    </RouterLink>
    <nav aria-label="访客导航">
      <RouterLink :to="{ path: '/', hash: '#project-route' }" class="route-link">项目路线</RouterLink>
      <RouterLink to="/login" class="login-link">登录</RouterLink>
      <RouterLink to="/register" class="workshop-button compact">开始学习 <ArrowRight aria-hidden="true" /></RouterLink>
    </nav>
  </header>
</template>

<script setup>
import { ArrowRight } from '@element-plus/icons-vue'
import TrailMark from '../components/TrailMark.vue'
</script>

<style scoped>
.public-header { width: min(100% - 64px, 1240px); height: 76px; margin-inline: auto; display: flex; align-items: center; justify-content: space-between; gap: 14px; border-bottom: 1px solid var(--color-border); }
.public-brand { display: inline-flex; align-items: center; gap: 10px; font-size: 19px; color: var(--color-text); text-decoration: none; white-space: nowrap; }
.public-header nav { display: flex; gap: clamp(14px, 2.4vw, 30px); align-items: center; }
.public-header nav > a:not(.workshop-button) { color: var(--color-text-secondary); font-size: 14px; font-weight: 650; text-decoration: none; white-space: nowrap; }
.public-header nav > a:hover, .public-header nav > a[aria-current='page'] { color: var(--color-primary); }
.public-header nav > a.workshop-button:hover { color: white; }
@media (max-width: 620px) {
  .public-header { width: calc(100% - 32px); height: 64px; }
  .public-brand { font-size: 16px; gap: 7px; }
  .public-brand .trail-mark { width: 31px; height: 31px; }
  .public-header nav { gap: 11px; }
  .public-header .workshop-button { font-size: 12px; padding-inline: 10px; }
  .public-header nav > a:not(.workshop-button) { font-size: 12px; }
}
@media (max-width: 355px) { .public-header .route-link { display: none; } }
</style>
