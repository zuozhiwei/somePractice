<template>
  <section class="expedition" aria-labelledby="atlas-title">
    <header class="atlas-head">
      <div>
        <h1 id="atlas-title">江湖地图</h1>
        <p v-if="chapters.length">{{ chapters.length }} 个章节区域 <span>·</span> {{ cleared }} / {{ levelCount }} 关已通关</p>
      </div>
      <div class="atlas-actions">
        <button type="button" class="mission-link" @click="journalOpen = true"><el-icon><Notebook /></el-icon>探索手记 <small>{{ visited.size }}/{{ chapters.length }}</small></button>
        <button type="button" class="mission-link" @click="showMission(null)"><el-icon><Tickets /></el-icon>推荐任务</button>
      </div>
    </header>

    <div v-if="error || loading || !chapters.length" class="world-message" :role="error ? 'alert' : 'status'">
      <el-icon><Compass /></el-icon>
      <p>{{ error ? '世界地图加载失败' : loading ? '正在展开世界地图…' : '还没有发布学习章节。' }}</p>
      <button v-if="error" type="button" class="mission-link" @click="load">重新加载</button>
    </div>
    <template v-else>
      <ExpeditionScene ref="scene" :atlas="atlas" :places="places" :selected-id="selectedId" :restore-focus="restoreFocus"
                       @select="select" @enter="openPlace" @arrive="visitCamp" @mission="showMission" />
      <article v-if="selected" class="place-card" aria-live="polite">
        <div class="place-icon" :class="selected.status?.toLowerCase()">
          <el-icon><component :is="selected.level ? (selected.status === 'LOCKED' ? Lock : selected.status === 'COMPLETED' ? CircleCheck : Flag) : MapLocation" /></el-icon>
        </div>
        <div class="place-text">
          <p class="place-meta">第 {{ selected.chapterIndex + 1 }} 章 · {{ selected.level ? selected.chapter.name + ' / 第 ' + selected.number + ' 关' : '章节营地' }}</p>
          <h2>{{ selected.name }}</h2>
          <p v-if="selected.level && selected.status === 'LOCKED'" class="description">{{ unlockHint }}</p>
          <p v-else-if="!selected.level" class="description">{{ selected.guide.role }} · {{ selected.guide.name }}：「{{ selected.guide.greeting }}」</p>
          <p v-else-if="selected.description" class="description">{{ selected.description }}</p>
          <div class="place-stats">
            <span>{{ selected.statusText }}</span>
            <template v-if="selected.level">
              <span><el-icon><DocumentChecked /></el-icon>{{ selected.level.taskDone }} / {{ selected.level.taskTotal }} 项任务</span>
              <span><el-icon><Medal /></el-icon>{{ selected.level.xpReward }} XP</span>
            </template>
            <span v-else>{{ selected.chapter.levelsCompleted }} / {{ selected.chapter.totalLevels }} 关已通关</span>
          </div>
        </div>
        <button type="button" class="enter" :disabled="selected.level && selected.status === 'LOCKED'" @click="openPlace(selectedId)">
          {{ selected.level ? (selected.status === 'LOCKED' ? '关卡未解锁' : selected.status === 'COMPLETED' ? '回顾关卡' : '进入关卡') : '查看本章关卡' }}
          <el-icon><component :is="selected.level && selected.status === 'LOCKED' ? Lock : Right" /></el-icon>
        </button>
      </article>
    </template>
    <el-dialog v-model="journalOpen" title="探索手记" width="min(94vw, 520px)">
      <p class="journal-summary">已探访 {{ visited.size }} / {{ chapters.length }} 处营地</p>
      <div class="journal-list">
        <button v-for="(chapter,index) in chapters" :key="chapter.id" type="button" @click="journalTravel(chapter.id)">
          <span><strong>{{ chapter.name }}</strong><small>{{ visited.has(chapter.id) ? guideFor(index).role + ' · ' + guideFor(index).name : '尚未拜访引路人' }}</small></span>
          <span>{{ visited.has(chapter.id) ? '重访' : '前往' }} →</span>
        </button>
      </div>
    </el-dialog>
    <MissionDialog v-if="missionOpen" v-model="missionOpen" :chapter-id="missionChapter" @submitted="refresh" />
  </section>
</template>

<script setup>
import { computed, ref, shallowRef, defineAsyncComponent, onBeforeUnmount, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Tickets, Notebook, Compass, MapLocation, Lock, CircleCheck, Flag, Medal, DocumentChecked, Right } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { mapsApi } from '../api/maps'
import { useMapStore } from '../store/map'
import ExpeditionScene from '../world/ExpeditionScene.vue'
import { buildAtlas } from '../world/worldAtlas'
import { guideFor } from '../world/campGuides'

const MissionDialog = defineAsyncComponent(() => import('../world/MissionDialog.vue'))
const router = useRouter()
const route = useRoute()
const mapStore = useMapStore()
const loading = ref(true), error = ref(false), missionOpen = ref(false)
const chapters = shallowRef([]), atlas = shallowRef(null), selectedId = ref(null)
const restoreFocus = ref(false)
const scene = ref(null), journalOpen = ref(false), missionChapter = ref(null)
const visited = shallowRef(new Set()), pendingVisits = new Set()
const cleared = computed(() => chapters.value.reduce((sum, chapter) => sum + chapter.levelsCompleted, 0))
const levelCount = computed(() => chapters.value.reduce((sum, chapter) => sum + chapter.totalLevels, 0))
const levelStatus = { LOCKED: '未解锁', LEARNING: '可挑战', COMPLETED: '已通关' }
const chapterStatus = { LOCKED: '可探索', IN_PROGRESS: '学习中', COMPLETED: '已通关' }
const places = computed(() => {
  const result = new Map()
  chapters.value.forEach((chapter, chapterIndex) => {
    result.set('hub:' + chapter.id, { chapter, chapterIndex, guide:guideFor(chapterIndex), name: chapter.name, description: chapter.description, status: chapter.status, statusText: chapterStatus[chapter.status] })
    chapter.levels.forEach((level, i) => {
      result.set('level:' + level.id, {
        chapter, chapterIndex, level, number: i + 1, name: level.name,
        description: level.description, status: level.status, statusText: levelStatus[level.status]
      })
    })
  })
  return result
})
const selected = computed(() => places.value.get(selectedId.value))
const unlockHint = computed(() => {
  const names = selected.value?.level?.prerequisiteNames || []
  return names.length ? '完成前置关卡「' + names.join('」、「') + '」后解锁。' : '完成前置关卡后解锁。'
})

let generation = 0
async function load() {
  const current = ++generation
  loading.value = !atlas.value
  error.value = false
  try {
    const { data } = await mapsApi.atlas()
    if (current !== generation) return
    if (!Array.isArray(data.data?.chapters)) throw new Error('Missing chapters')
    chapters.value = data.data.chapters
    visited.value = new Set([...visited.value, ...(data.data.visitedChapterIds || [])].filter(id => chapters.value.some(chapter => chapter.id === id)))
    atlas.value = buildAtlas(chapters.value)
    if (!places.value.has(selectedId.value)) {
      const nodes = atlas.value.nodes.filter(node => node.kind === 'level')
      const first = nodes.find(node => places.value.get(node.id).status === 'LEARNING') || nodes[0] || atlas.value.nodes[0]
      restoreFocus.value = places.value.has(route.query.place)
      selectedId.value = restoreFocus.value ? route.query.place : first?.id ?? null
    }
  } catch {
    if (current === generation) error.value = true
  } finally {
    if (current === generation) loading.value = false
  }
}
function select(id) { selectedId.value = id }
function showMission(chapterId) {
  missionChapter.value = chapterId
  missionOpen.value = true
}
async function visitCamp(id) {
  if (visited.value.has(id) || pendingVisits.has(id)) return
  pendingVisits.add(id)
  try {
    await mapsApi.visit(id)
    visited.value = new Set([...visited.value,id])
  } catch { ElMessage.warning('探访记录保存失败，到营地再次拜访即可重试。') }
  finally { pendingVisits.delete(id) }
}
function journalTravel(id) {
  journalOpen.value = false
  scene.value?.visitCampById(id)
}
async function openPlace(id) {
  const place = places.value.get(id)
  if (!place || (place.level && place.status === 'LOCKED')) return
  await router.replace({ path:route.path, query:{ ...route.query, place:id }, hash:route.hash })
  if (!place.level) {
    router.push('/map/' + place.chapter.id)
  } else {
    router.push({ path: '/level/' + place.level.id, query: { n: place.level.orderIndex } })
  }
}
function refresh() {
  mapStore.refresh()
  load()
}
onMounted(load)
onBeforeUnmount(() => { generation++ })
</script>

<style scoped>
.expedition { display:grid; min-width:0; gap:14px; color:var(--color-text); }
.atlas-head { display:flex; align-items:center; justify-content:space-between; gap:12px; }
.atlas-head h1,.place-text h2 { margin:0; line-height:1.3; color:#1d3d33; }
.atlas-actions { display:flex; gap:6px; flex-wrap:wrap; }
.atlas-actions small { color:#68735c; }
.journal-summary { color:#596c60; margin-bottom:12px; }
.journal-list { display:grid; gap:6px; }
.journal-list button { display:flex; justify-content:space-between; align-items:center; gap:12px; padding:10px 12px; border-radius:8px; border:1px solid #d9dfd1; background:#fafaf1; color:#335542; text-align:left; }
.journal-list small { display:block; font-size:12px; color:#687769; margin-top:2px; }
.atlas-head h1 { font-size:clamp(22px,2.2vw,32px); }
.atlas-head p { margin:4px 0 0; font-size:13px; color:#596c60; }
.atlas-head p span { padding:0 4px; color:#8b987e; }
.mission-link,.enter { display:inline-flex; align-items:center; justify-content:center; gap:7px; background:#f8f7eb; border:1px solid #cdd6c4; border-radius:9px; color:#345642; font:inherit; font-size:13px; font-weight:650; cursor:pointer; padding:10px 14px; }
.mission-link:hover { background:#edf1e4; }
.world-message,.place-card { background:var(--color-surface); border:1px solid var(--color-border); border-radius:14px; }
.world-message { display:grid; justify-items:center; gap:12px; padding:48px 18px; color:#556f5e; }
.world-message .el-icon { font-size:25px; }
.place-card { display:flex; gap:15px; padding:17px 20px; align-items:center; }
.place-icon { display:grid; place-items:center; flex:0 0 46px; height:48px; border:1px solid #e1d4b3; border-radius:11px; font-size:22px; color:#8c6830; background:#f6efd8; }
.place-icon.locked { color:#63726a; background:#eef0e7; border-color:#d9dfd1; }
.place-icon.completed { color:#34664e; background:#e6f0e2; border-color:#cbdcc8; }
.place-text { flex:1; min-width:0; }
.place-meta,.place-stats,.description { font-size:12px; line-height:1.6; color:#5f7264; }
.place-meta { font-weight:650; }
.place-text h2 { margin:3px 0; font-size:clamp(17px,1.6vw,21px); overflow-wrap:anywhere; }
.description { font-size:13px; overflow-wrap:anywhere; }
.place-stats,.place-stats span { display:flex; align-items:center; gap:5px; }
.place-stats { margin-top:5px; gap:14px; flex-wrap:wrap; }
.enter { background:#245b46; color:#fff; border-color:transparent; white-space:nowrap; }
.enter:hover { background:#194733; }
.enter:disabled { color:#596c60; background:#e3e9df; cursor:default; }
@media(max-width:640px) {
  .atlas-head { align-items:flex-start; flex-wrap:wrap; }
  .atlas-actions .mission-link { padding:8px 9px; font-size:12px; }
  .place-card { flex-wrap:wrap; padding:13px; gap:10px; }
  .place-icon { flex-basis:38px; height:42px; font-size:19px; }
  .enter { width:100%; }
}
</style>
