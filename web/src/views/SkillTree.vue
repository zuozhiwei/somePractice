<template>
  <div class="page">
    <div class="page-title-row">
      <div>
        <h1 class="page-title">技能树</h1>
        <p class="page-subtitle">{{ skills.length }} 个关卡 · {{ skills.filter(skill => skill.status === 'COMPLETED').length }} 个已通关</p>
      </div>
      <div class="legend">
        <span class="legend-item"><span class="dot locked"></span>未解锁</span>
        <span class="legend-item"><span class="dot learning"></span>学习中</span>
        <span class="legend-item"><span class="dot mastered"></span>已通关</span>
      </div>
    </div>

    <div class="section-card" style="margin-bottom:20px">
      <div class="section-body" style="padding:12px 24px">
        <el-tabs v-model="activeCategory">
          <el-tab-pane label="全部" name="all" />
          <el-tab-pane v-for="category in categories" :key="category" :label="category" :name="category" />
        </el-tabs>
      </div>
    </div>

    <div class="tree-wrapper" v-loading="loading" ref="treeContainer">
      <svg aria-hidden="true" class="connections-svg" :width="svgWidth" :height="svgHeight" v-if="svgWidth > 0">
        <defs>
          <marker id="arrowActive" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto">
            <polygon points="0 0, 6 3, 0 6" fill="#216354" />
          </marker>
          <marker id="arrowDimmed" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto">
            <polygon points="0 0, 6 3, 0 6" fill="#bfcbbf" />
          </marker>
        </defs>
        <g v-for="conn in connections" :key="conn.key">
          <path :d="conn.path"
                :stroke="conn.active ? '#216354' : '#bfcbbf'"
                :stroke-width="conn.active ? 2 : 1.5"
                fill="none"
                :stroke-dasharray="conn.active ? 'none' : '6 4'"
                :marker-end="conn.active ? 'url(#arrowActive)' : 'url(#arrowDimmed)'"
                stroke-linecap="round" />
        </g>
      </svg>

      <div class="tree-columns" ref="columnsEl">
        <div v-for="column in columns" :key="column.id" class="tree-column">
          <div class="column-label">
            <span class="col-num">{{ column.number || '·' }}</span>
            <span>{{ column.name }}</span>
          </div>
          <div v-for="skill in column.skills" :key="skill.id" class="skill-node-wrapper"
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
                   role="button" tabindex="0" @keydown.enter.prevent="goDetail(skill)" @keydown.space.prevent="goDetail(skill)"
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
      <el-empty v-if="!loading && !filteredSkills.length" :description="loadError ? '无法载入技能树' : '当前没有关卡'">
        <el-button v-if="loadError" @click="load">重试</el-button>
      </el-empty>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMapStore } from '../store/map'
import { mapsApi } from '../api/maps'
import { ElMessage } from 'element-plus'

const router = useRouter()
const skills = ref([])
const mapStore = useMapStore()
const loadError = ref(false)
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
  else nodeRefMap.delete(id)
}

function statusText(s) {
  return { COMPLETED: '已通关', LEARNING: '学习中', LOCKED: '未解锁' }[s] || s
}

const filteredSkills = computed(() => {
  if (activeCategory.value === 'all') return skills.value
  return skills.value.filter(s => s.category === activeCategory.value)
})

const categories = computed(() => [...new Set(skills.value.map(skill => skill.category).filter(Boolean))])
const columns = computed(() => {
  const chapters = new Map((mapStore.progress?.maps || []).map((chapter, i) => [chapter.id,
    { id: chapter.id, number: i + 1, name: chapter.name, skills: [] }
  ]))
  for (const skill of filteredSkills.value) {
    const id = skill.mapId ?? 'unassigned'
    if (!chapters.has(id)) chapters.set(id, { id, name: skill.mapId ? '章节 ' + id : '未分章', skills: [] })
    chapters.get(id).skills.push(skill)
  }
  return [...chapters.values()].filter(chapter => chapter.skills.length)
})

function computeConnections() {
  const treeEl = treeContainer.value
  if (!treeEl) return

  const visibleList = filteredSkills.value
  // Build set of visible skill IDs to filter out cross-tab connections
  const visibleIds = new Set(visibleList.map(s => s.id))
  const skillMap = new Map(skills.value.map(s => [s.id, s]))
  const treeRect = treeEl.getBoundingClientRect()
  const treeLeft = treeRect.left + treeEl.clientLeft - treeEl.scrollLeft
  const treeTop = treeRect.top + treeEl.clientTop - treeEl.scrollTop

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
  svgWidth.value = Math.max(columnsEl.value?.scrollWidth + (columnsEl.value?.offsetLeft || 0) || 0, el.clientWidth)
  svgHeight.value = Math.max(columnsEl.value?.offsetHeight + (columnsEl.value?.offsetTop || 0) || 0, el.clientHeight)
}

function getLockReason(skill) {
  if (!skill.prerequisiteIds?.length) return ''
  const skillMap = new Map(skills.value.map(s => [s.id, s]))
  const names = skill.prerequisiteIds
    .filter(pid => skillMap.get(pid)?.status !== 'COMPLETED')
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

async function updateDiagram() {
  await nextTick()
  computeConnections()
  updateSvgSize()
}

watch(columns, updateDiagram, { flush: 'post' })

async function load() {
  loading.value = true
  loadError.value = false
  try {
    const [treeRes] = await Promise.all([mapsApi.levelTree(), mapStore.fetchProgress()])
    skills.value = treeRes.data.data || []
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

let resizeObserver
onMounted(() => {
  resizeObserver = new ResizeObserver(updateDiagram)
  resizeObserver.observe(columnsEl.value)
  resizeObserver.observe(treeContainer.value)
  load()
})
onBeforeUnmount(() => resizeObserver?.disconnect())
</script>

<style scoped>
.page { max-width: 1600px; margin: 0 auto; }
.page-title-row { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0; }

.legend { display: flex; gap: 20px; font-size: 13px; color: var(--color-text-secondary); margin-top: 4px; }
.legend-item { display: flex; align-items: center; gap: 6px; }
.dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.dot.locked { background: #bfcbbf; }
.dot.learning { background: #216354; }
.dot.mastered { background: #3b825f; }

.tree-wrapper {
  position: relative; overflow: auto;
  background: var(--color-surface);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--color-border);
  background-image: radial-gradient(#d1dccf 1px, transparent 1px);
  background-size: 18px 18px;
  min-height: 330px;
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
  font-size: 12px; font-weight: 600; color: #667568;
  text-transform: uppercase; letter-spacing: 2px;
}

.col-num {
  width: 24px; height: 24px; border-radius: 50%;
  background: linear-gradient(135deg, #216354, #397c67);
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
.skill-node.locked { background: #f4f5ef; }
.skill-node.learning { border-color: #216354; }
.skill-node.mastered { border-color: #3b825f; background: #f0f6ed; }

.node-top { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.node-lv { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px; }
.node-lv.lv-1, .node-lv.lv-2 { background: #edf0e9; color: #58685d; }
.node-lv.lv-3 { background: #faf0d7; color: #895b19; }
.node-lv.lv-4, .node-lv.lv-5 { background: #fee2e2; color: #dc2626; }
.node-cat { font-size: 11px; color: #667568; }

.node-name { font-size: 14px; font-weight: 600; color: var(--color-text); margin-bottom: 8px; line-height: 1.4; }

.node-status { display: flex; align-items: center; gap: 5px; font-size: 12px; color: #667568; margin-bottom: 10px; }
.status-indicator { width: 6px; height: 6px; border-radius: 50%; }
.status-indicator.locked { background: #bfcbbf; }
.status-indicator.learning { background: #216354; }
.status-indicator.completed { background: #3b825f; }

.node-progress { display: flex; align-items: center; gap: 8px; }
.progress-bar { flex: 1; height: 4px; background: #edf0e9; border-radius: 2px; overflow: hidden; }
.progress-fill { height: 100%; background: #216354; border-radius: 2px; transition: width .3s; }
.progress-fill.complete { background: #3b825f; }
.progress-pct { font-size: 11px; color: #216354; font-weight: 600; }
.progress-done { color: #3b825f; font-size: 12px; }

.page-title-row { flex-wrap: wrap; gap: 12px; margin-bottom: 18px; }
.page-title-row .page-subtitle { margin-bottom: 0; }
.skill-node { overflow-wrap: anywhere; }
.skill-node:focus-visible { outline: 3px solid #a8854d; outline-offset: 2px; }
@media(max-width:600px) {
  .tree-wrapper { padding: 16px 12px; }
  .tree-column { min-width: 170px; }
}
</style>
