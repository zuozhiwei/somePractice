<template>
  <div class="page">
    <div class="page-title-row">
      <div>
        <h1 class="page-title">技能树</h1>
        <p class="page-subtitle">按依赖关系展开，逐级解锁技能模块</p>
      </div>
      <div class="legend">
        <span class="legend-item"><span class="dot locked"></span>未解锁</span>
        <span class="legend-item"><span class="dot learning"></span>学习中</span>
        <span class="legend-item"><span class="dot mastered"></span>已通关</span>
      </div>
    </div>

    <div class="section-card" style="margin-bottom:20px">
      <div class="section-body" style="padding:12px 24px">
        <el-tabs v-model="activeCategory" @tab-change="onTabChange">
          <el-tab-pane label="全部" name="all" />
          <el-tab-pane label="前端" name="前端" />
          <el-tab-pane label="后端" name="后端" />
          <el-tab-pane label="数据库" name="数据库" />
          <el-tab-pane label="运维" name="运维" />
          <el-tab-pane label="工具" name="工具" />
          <el-tab-pane label="综合" name="综合" />
          <el-tab-pane label="软技能" name="软技能" />
        </el-tabs>
      </div>
    </div>

    <div class="tree-wrapper" v-loading="loading" ref="treeContainer">
      <svg class="connections-svg" :width="svgWidth" :height="svgHeight" v-if="svgWidth > 0">
        <defs>
          <marker id="arrowActive" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto">
            <polygon points="0 0, 6 3, 0 6" fill="#6366f1" />
          </marker>
          <marker id="arrowDimmed" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto">
            <polygon points="0 0, 6 3, 0 6" fill="#cbd5e1" />
          </marker>
        </defs>
        <g v-for="conn in connections" :key="conn.key">
          <path :d="conn.path"
                :stroke="conn.active ? '#6366f1' : '#cbd5e1'"
                :stroke-width="conn.active ? 2 : 1.5"
                fill="none"
                :stroke-dasharray="conn.active ? 'none' : '6 4'"
                :marker-end="conn.active ? 'url(#arrowActive)' : 'url(#arrowDimmed)'"
                stroke-linecap="round" />
        </g>
      </svg>

      <div class="tree-columns" ref="columnsEl">
        <div v-for="(column, colIdx) in columns" :key="colIdx" class="tree-column">
          <div class="column-label">
            <span class="col-num">{{ colIdx + 1 }}</span>
            <span>{{ mapLabel(colIdx) }}</span>
          </div>
          <div v-for="skill in column" :key="skill.id" class="skill-node-wrapper"
               :ref="el => setNodeRef(skill.id, el)" :data-sid="skill.id">
            <el-tooltip :disabled="skill.status !== 'LOCKED' || !getLockReason(skill)"
                         :content="getLockReason(skill)"
                         placement="top"
                         effect="dark">
              <div class="skill-node"
                   :class="{
                     locked: skill.status === 'LOCKED',
                     learning: skill.status === 'LEARNING',
                     mastered: skill.status === 'COMPLETED'
                   }"
                   @click="goDetail(skill)"
              >
                <div class="node-top">
                  <span class="node-lv" :class="'lv-'+skill.tier">Lv.{{ skill.tier }}</span>
                  <span class="node-cat">{{ skill.category }}</span>
                </div>
                <div class="node-name">{{ skill.name }}</div>
                <div class="node-status">
                  <span class="status-indicator" :class="skill.status.toLowerCase()"></span>
                  {{ statusText(skill.status) }}
                </div>
                <div class="node-progress">
                  <div class="progress-bar">
                    <div class="progress-fill" :style="{ width: skill.progress + '%' }"
                         :class="{ complete: skill.progress === 100 }"></div>
                  </div>
                  <span class="progress-pct" v-if="skill.progress > 0 && skill.progress < 100">{{ skill.progress }}%</span>
                  <span class="progress-done" v-if="skill.progress === 100">✓</span>
                </div>
              </div>
            </el-tooltip>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { mapsApi } from '../api/maps'
import { ElMessage } from 'element-plus'

const router = useRouter()
const skills = ref([])
const mapNames = ref({})
const loading = ref(true)
const activeCategory = ref('all')
const treeContainer = ref(null)
const columnsEl = ref(null)
const svgWidth = ref(0)
const svgHeight = ref(0)
const connections = ref([])

// Use a reactive Map for node refs so Vue tracks changes
const nodeRefMap = new Map()
function setNodeRef(id, el) {
  if (el) nodeRefMap.set(id, el)
}

// Invalidate all cached refs when switching tabs
function invalidateRefs() {
  nodeRefMap.clear()
}

function statusText(s) {
  return { COMPLETED: '已通关', LEARNING: '学习中', LOCKED: '未解锁' }[s] || s
}

const filteredSkills = computed(() => {
  if (activeCategory.value === 'all') return skills.value
  return skills.value.filter(s => s.category === activeCategory.value)
})

// 42 关按地图分组：每 6 关一张地图（levelId 1-6 → 地图1 ... 37-42 → 地图7），避免线性链拆出 42 列
const columns = computed(() => {
  const list = filteredSkills.value
  const cols = []
  for (const s of list) {
    const mapIdx = Math.min(6, Math.floor((s.id - 1) / 6))
    if (!cols[mapIdx]) cols[mapIdx] = []
    cols[mapIdx].push(s)
  }
  for (const col of cols) if (col) col.sort((a, b) => a.id - b.id)
  return cols.filter(Boolean)
})

function mapLabel(colIdx) {
  const id = colIdx + 1
  return mapNames.value[id] || `第 ${id} 章`
}

function computeConnections() {
  const treeEl = treeContainer.value
  if (!treeEl) return

  const visibleList = filteredSkills.value
  // Build set of visible skill IDs to filter out cross-tab connections
  const visibleIds = new Set(visibleList.map(s => s.id))
  const skillMap = new Map(skills.value.map(s => [s.id, s]))
  const treeRect = treeEl.getBoundingClientRect()
  const treeLeft = treeRect.left
  const treeTop = treeRect.top

  const conns = []

  for (const skill of visibleList) {
    if (!skill.prerequisiteIds?.length) continue

    const targetEl = nodeRefMap.get(skill.id)
    if (!targetEl) continue
    const tRect = targetEl.getBoundingClientRect()
    const tcx = tRect.left - treeLeft + tRect.width / 2
    const tcy = tRect.top - treeTop + tRect.height / 2

    for (const pid of skill.prerequisiteIds) {
      // Only draw connection if prerequisite is ALSO visible in current tab/view
      if (!visibleIds.has(pid)) continue

      const sourceEl = nodeRefMap.get(pid)
      if (!sourceEl) continue
      const sRect = sourceEl.getBoundingClientRect()
      const scx = sRect.left - treeLeft + sRect.width / 2
      const scy = sRect.top - treeTop + sRect.height / 2

      // 同一列（水平中心接近）→ 上下结构：源底部出 → 目标顶部进
      // 不同列（源在左）→ 左右结构：源右缘出 → 目标左缘进
      const sameColumn = Math.abs(scx - tcx) < 40
      let ex, ey, ix, iy
      if (sameColumn) {
        ex = scx
        ey = sRect.bottom - treeTop
        ix = tcx
        iy = tRect.top - treeTop
      } else {
        ex = sRect.right - treeLeft
        ey = scy
        ix = tRect.left - treeLeft
        iy = tcy
      }
      // 贝塞尔控制点：沿出/入方向延伸一半距离，形成顺滑的竖/横向曲线
      const dx = ix - ex
      const dy = iy - ey
      const c1x = sameColumn ? ex : ex + dx * 0.5
      const c1y = sameColumn ? ey + dy * 0.5 : ey
      const c2x = sameColumn ? ix : ix - dx * 0.5
      const c2y = sameColumn ? iy - dy * 0.5 : iy
      const path = `M ${ex} ${ey} C ${c1x} ${c1y}, ${c2x} ${c2y}, ${ix} ${iy}`

      const prereq = skillMap.get(pid)
      conns.push({
        key: `${pid}->${skill.id}`,
        path,
        active: prereq?.status === 'COMPLETED'
      })
    }
  }

  connections.value = conns
}

function updateSvgSize() {
  const el = treeContainer.value
  if (!el) return
  svgWidth.value = Math.max(el.scrollWidth, el.clientWidth)
  svgHeight.value = Math.max(el.scrollHeight, el.clientHeight)
}

function getLockReason(skill) {
  if (!skill.prerequisiteIds?.length) return ''
  const skillMap = new Map(skills.value.map(s => [s.id, s]))
  const names = skill.prerequisiteIds
    .map(pid => skillMap.get(pid)?.name)
    .filter(Boolean)
  if (!names.length) return ''
  return '前置关卡未通关：' + names.join('、')
}

function goDetail(skill) {
  if (skill.status === 'LOCKED') {
    const reason = getLockReason(skill)
    if (reason) {
      ElMessage.warning(reason)
    } else {
      ElMessage.warning('该技能尚未解锁')
    }
    return
  }
  router.push(`/level/${skill.id}`)
}

function onTabChange() {
  invalidateRefs()
  nextTick(() => {
    nextTick(() => {
      computeConnections()
      updateSvgSize()
    })
  })
}

// Watch filtered skills to recompute
watch(filteredSkills, async () => {
  await nextTick()
  await nextTick()
  computeConnections()
  updateSvgSize()
})

onMounted(async () => {
  try {
    const [treeRes, mapsRes] = await Promise.all([
      mapsApi.levelTree(),
      mapsApi.list().catch(() => null)
    ])
    skills.value = treeRes.data.data || []
    const names = {}
    for (const mp of mapsRes?.data?.data || []) names[mp.id] = mp.name
    mapNames.value = names
    await nextTick()
    await nextTick()
    await nextTick()
    computeConnections()
    updateSvgSize()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page { max-width: 1600px; margin: 0 auto; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0; }

.legend { display: flex; gap: 20px; font-size: 13px; color: var(--color-text-secondary); margin-top: 4px; }
.legend-item { display: flex; align-items: center; gap: 6px; }
.dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.dot.locked { background: #cbd5e1; }
.dot.learning { background: #6366f1; }
.dot.mastered { background: #22c55e; }

.tree-wrapper {
  position: relative; overflow: auto;
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  min-height: 400px;
  padding: 28px 20px;
}

.connections-svg {
  position: absolute; top: 0; left: 0;
  pointer-events: none; z-index: 1;
  overflow: visible;
}

.tree-columns {
  display: flex; gap: 18px;
  min-width: max-content;
  position: relative; z-index: 2;
}

.tree-column {
  display: flex; flex-direction: column; gap: 14px;
  min-width: 190px; max-width: 210px;
}

.column-label {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 0; margin-bottom: 4px;
  font-size: 12px; font-weight: 600; color: #94a3b8;
  text-transform: uppercase; letter-spacing: 2px;
}

.col-num {
  width: 24px; height: 24px; border-radius: 50%;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 12px;
}

.skill-node-wrapper { position: relative; }

.skill-node {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 16px;
  cursor: pointer;
  transition: all .2s;
}
.skill-node:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
.skill-node.locked { opacity: 0.45; }
.skill-node.learning { border-color: #6366f1; }
.skill-node.mastered { border-color: #22c55e; background: #f0fdf4; }

.node-top { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.node-lv { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px; }
.node-lv.lv-1, .node-lv.lv-2 { background: #f1f5f9; color: #64748b; }
.node-lv.lv-3 { background: #fef3c7; color: #d97706; }
.node-lv.lv-4, .node-lv.lv-5 { background: #fee2e2; color: #dc2626; }
.node-cat { font-size: 11px; color: #94a3b8; }

.node-name { font-size: 14px; font-weight: 600; color: var(--color-text); margin-bottom: 8px; line-height: 1.4; }

.node-status { display: flex; align-items: center; gap: 5px; font-size: 12px; color: #94a3b8; margin-bottom: 10px; }
.status-indicator { width: 6px; height: 6px; border-radius: 50%; }
.status-indicator.locked { background: #cbd5e1; }
.status-indicator.learning { background: #6366f1; }
.status-indicator.mastered { background: #22c55e; }

.node-progress { display: flex; align-items: center; gap: 8px; }
.progress-bar { flex: 1; height: 4px; background: #f1f5f9; border-radius: 2px; overflow: hidden; }
.progress-fill { height: 100%; background: #6366f1; border-radius: 2px; transition: width .3s; }
.progress-fill.complete { background: #22c55e; }
.progress-pct { font-size: 11px; color: #6366f1; font-weight: 600; }
.progress-done { color: #22c55e; font-size: 12px; }
</style>
