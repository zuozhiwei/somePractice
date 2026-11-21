<template>
  <div class="course-bar">
    <span class="course-name"><Reading aria-hidden="true" />校园树洞</span>
    <div ref="switcher" class="view-switcher" @focusout="onFocusOut" @keydown.esc="onEscape">
      <button ref="trigger" type="button" class="view-trigger" aria-controls="course-views"
              :aria-expanded="open" @click="open = !open">
        <span>课程视图</span><strong>{{ currentView.label }}</strong><ArrowDown aria-hidden="true" />
      </button>
      <nav v-show="open" id="course-views" class="view-options" aria-label="课程视图">
        <RouterLink v-for="view in learningViews" :key="view.key" custom
                    :to="destination(view.key)" v-slot="{ href, navigate }">
          <a :href="href" :aria-label="view.label" :aria-description="view.description"
             :aria-current="view.key === active ? (route.name === 'learn' ? 'page' : 'location') : undefined"
             @click.exact="closeAndFocus" @click="navigate">
            <span><strong>{{ view.label }}</strong><small>{{ view.description }}</small></span>
            <Check v-if="view.key === active" aria-hidden="true" />
          </a>
        </RouterLink>
      </nav>
    </div>
  </div>
  <RouterView />
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowDown, Check, Reading } from '@element-plus/icons-vue'
import { learningView, learningViews } from './learningViews'

const route = useRoute()
const active = ref('story')
const currentView = computed(() => learningViews.find(view => view.key === active.value))
const open = ref(false)
const trigger = ref(null)
const switcher = ref(null)

watch(() => route.fullPath, () => {
  open.value = false
  if (route.name === 'learn') {
    active.value = learningView(route.query.view)
  } else if (route.params.id || route.params.mapId) {
    // Remember which view opened each chapter or lesson when revisiting browser history.
    const saved = window.history.state?.learningTab
    if (saved) active.value = learningView(saved)
    else window.history.replaceState({ ...window.history.state, learningTab: active.value }, '')
  }
}, { immediate: true })

function destination(view) {
  const { id, mapId } = route.params
  const place = id ? 'level:' + id : mapId ? 'hub:' + mapId : route.query.place
  const query = place ? { place } : {}
  if (view !== 'world') query.view = view
  return { name: 'learn', query }
}

function closeAndFocus() {
  open.value = false
  trigger.value?.focus({ preventScroll: true })
}

function onEscape(event) {
  if (!open.value) return
  event.preventDefault()
  event.stopPropagation()
  closeAndFocus()
}

function onFocusOut(event) {
  if (!switcher.value?.contains(event.relatedTarget)) open.value = false
}

function onOutsidePress(event) {
  if (open.value && !switcher.value?.contains(event.target)) open.value = false
}

onMounted(() => document.addEventListener('pointerdown', onOutsidePress))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutsidePress))
</script>

<style scoped>
.course-bar {
  display: flex; align-items: center; gap: 14px; margin-bottom: 14px;
  color: var(--color-text); font-size: 14px;
}
.course-name { display: inline-flex; align-items: center; gap: 7px; font-weight: 650; }
.course-name svg { width: 17px; height: 17px; color: var(--color-primary); }
.view-switcher { position: relative; }
.view-trigger {
  display: inline-flex; align-items: center; gap: 8px; min-height: 40px; padding: 0 11px;
  background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 10px;
  font: inherit; color: inherit; cursor: pointer;
}
.view-trigger span { font-size: 12px; color: var(--color-text-secondary); }
.view-trigger strong { font-weight: 650; }
.view-trigger svg { width: 13px; height: 13px; color: var(--color-primary); }
.view-trigger:hover, .view-trigger[aria-expanded="true"] { border-color: #8ca596; }
.view-trigger[aria-expanded="true"] svg { transform: rotate(180deg); }
.view-options {
  position: absolute; top: calc(100% + 5px); left: 0; z-index: 40;
  width: 195px; padding: 5px; border: 1px solid var(--color-border);
  border-radius: 12px; background: var(--color-surface); box-shadow: 0 8px 24px #263d2d22;
}
.view-options a {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  min-height: 48px; padding: 6px 10px; border-radius: 7px; text-decoration: none; color: inherit;
}
.view-options a:hover, .view-options a:focus-visible { background: #f1f3e9; }
.view-options a[aria-current] { background: #e7eee2; color: var(--color-primary); }
.view-options strong, .view-options small { display: block; }
.view-options strong { font-size: 14px; font-weight: 650; }
.view-options small { font-size: 12px; line-height: 1.5; color: var(--color-text-secondary); }
.view-options svg { flex: none; width: 16px; height: 16px; }
@media (max-width: 610px) {
  .course-bar { justify-content: space-between; gap: 8px; margin-bottom: 10px; }
  .view-trigger { min-height: 44px; }
  .view-options { right: 0; left: auto; max-width: calc(100vw - 28px); }
}
</style>
