<template>
  <div class="learn-page">
    <nav class="view-switcher" aria-label="学习视图">
      <RouterLink v-for="item in views" :key="item.key"
                  :to="{ path: '/learn', query: { ...route.query, view: item.key } }"
                  :aria-current-value="view === item.key ? 'page' : 'false'"
                  :class="{ selected: view === item.key }">
        {{ item.label }}
      </RouterLink>
    </nav>
    <component :is="currentComponent" :key="view" />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const views = [
  { key: 'world', label: '世界地图', component: defineAsyncComponent(() => import('./WorldMap.vue')) },
  { key: 'story', label: '章节目录', component: defineAsyncComponent(() => import('./MapBoard.vue')) },
  { key: 'tree', label: '技能树', component: defineAsyncComponent(() => import('./SkillTree.vue')) }
]
const valid = key => views.some(item => item.key === key)
function lastView() {
  try { return localStorage.getItem('learn-view') } catch { return null }
}
const view = computed(() => {
  if (valid(route.query.view)) return route.query.view
  const saved = lastView()
  return valid(saved) ? saved : 'world'
})
const currentComponent = computed(() => views.find(item => item.key === view.value).component)

watch(() => route.query.view, key => {
  if (route.path !== '/learn') return
  if (!valid(key)) {
    router.replace({ path: route.path, query: { ...route.query, view: view.value }, hash: route.hash })
    return
  }
  try { localStorage.setItem('learn-view', key) } catch { /* The URL still preserves the view. */ }
}, { immediate: true })
</script>

<style scoped>
.learn-page { min-width: 0; }
.view-switcher {
  display: flex;
  align-items: stretch;
  gap: clamp(18px, 3vw, 34px);
  margin-bottom: clamp(16px, 2vw, 26px);
  border-bottom: 1px solid var(--color-border, #d4dacf);
}
.view-switcher a {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: 0 2px;
  border-bottom: 3px solid transparent;
  color: var(--color-text-secondary, #55685e);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
}
.view-switcher a:hover,
.view-switcher a.selected { color: var(--color-primary, #196450); }
.view-switcher a.selected { border-bottom-color: currentColor; }
.view-switcher a:focus-visible { outline: 2px solid var(--color-primary, #196450); outline-offset: 2px; }
</style>
