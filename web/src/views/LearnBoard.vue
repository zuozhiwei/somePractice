<template>
  <div class="learn-page">
    <!-- 视图切换器：同一份冒险，三种打开方式 -->
    <div class="view-switcher">
      <div class="vs-inner">
        <button v-for="v in views" :key="v.key" class="vs-btn" :class="{ active: view === v.key }" @click="switchView(v.key)">
          <span class="vs-icon">{{ v.icon }}</span>
          <span class="vs-name">{{ v.name }}</span>
        </button>
      </div>
      <span class="vs-hint">同一份冒险，三种打开方式 —— 想怎么学就怎么切</span>
    </div>

    <div class="view-body">
      <component :is="viewComponent" :key="view" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, defineAsyncComponent } from 'vue'

// 地图和技能树依赖不同的绘制资源，按需加载以减少首次进入的流量。
const MapBoard = defineAsyncComponent(() => import('./MapBoard.vue'))
const WorldMap = defineAsyncComponent(() => import('./WorldMap.vue'))
const SkillTree = defineAsyncComponent(() => import('./SkillTree.vue'))

const views = [
  { key: 'story', icon: '📖', name: '剧情' },
  { key: 'world', icon: '🗺️', name: '世界' },
  { key: 'tree', icon: '🌳', name: '技能树' }
]

// 记住用户上次选择的展示形式
const savedView = localStorage.getItem('learn-view')
const view = ref(views.some(item => item.key === savedView) ? savedView : 'world')

const viewComponent = computed(() => ({ story: MapBoard, world: WorldMap, tree: SkillTree }[view.value]))

function switchView(k) {
  view.value = k
  localStorage.setItem('learn-view', k)
}
</script>

<style scoped>
.learn-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 64px);
}

/* ===== 视图切换器 ===== */
.view-switcher {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}
.vs-inner {
  display: flex;
  gap: 6px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 5px;
  box-shadow: var(--shadow-sm);
}
.vs-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  border: none;
  background: none;
  padding: 9px 20px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: #64748b;
  transition: all .2s;
}
.vs-btn:hover { color: #1e293b; background: #f1f5f9; }
.vs-btn.active {
  color: #fff;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  box-shadow: 0 3px 10px rgba(99, 102, 241, .3);
}
.vs-icon { font-size: 16px; }
.vs-hint { font-size: 12px; color: #94a3b8; }

/* ===== 视图区 ===== */
.view-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

@media (max-width: 768px) {
  .learn-page { height: calc(100vh - 40px); }
  .view-switcher { flex-wrap: wrap; gap: 10px; }
  .vs-btn { padding: 8px 14px; font-size: 13px; }
  .vs-hint { display: none; }
}
</style>
