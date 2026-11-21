<template>
  <section class="expedition" aria-labelledby="atlas-title">
    <header class="atlas-head">
      <div>
        <h1 id="atlas-title">项目世界地图</h1>
        <p>选一座岛探索，在据点做任务，把通过的作品收进成果册。</p>
      </div>
      <dl v-if="atlas" class="stats">
        <div><dt>已通关</dt><dd>{{ cleared }}<small> / {{ levelCount }}</small></dd></div>
        <div><dt>累计经验</dt><dd>{{ totalXp }}<small> XP</small></dd></div>
      </dl>
    </header>

    <div v-if="!atlas" class="world-message" :role="error ? 'alert' : 'status'">
      <el-icon><Compass /></el-icon>
      <p>{{ error ? '世界地图加载失败' : loading ? '正在展开世界地图…' : '还没有发布学习章节。' }}</p>
      <button v-if="error" type="button" class="mission-link" @click="load">重新加载</button>
    </div>
    <template v-else>
      <p v-if="error" role="alert" class="sync-error">进度更新失败，正在显示上次的地图。<button @click="load">重试</button></p>
      <aside v-if="reward" class="reward" role="status">
        <el-icon><Medal /></el-icon>
        <div><strong>{{ reward.title }}</strong><span>{{ reward.detail }}</span></div>
        <button type="button" aria-label="收起成果提醒" @click="reward = null">×</button>
      </aside>
      <div class="atlas-layout">
        <ExpeditionScene ref="scene" :atlas="atlas" :places="places" :visited="visited" :selected-id="selectedId" :restore-focus="restoreFocus"
                         @select="select" @enter="openPlace" @arrive="visitCamp" @mission="showMission" />
        <article v-if="selected" class="place-card" aria-live="polite">
          <div class="place-chapter"><el-icon><Location /></el-icon> 第 {{ selected.chapter.chapterNumber ?? selected.chapterIndex + 1 }} 章 · {{ selected.chapter.name }}</div>
          <div class="place-status">
            <el-icon><component :is="selected.level ? (selected.status === 'LOCKED' ? Lock : selected.status === 'COMPLETED' ? CircleCheck : Flag) : Compass" /></el-icon>
            {{ selected.level ? selected.statusText : visited.has(selected.chapter.id) ? '已探访营地' : '章节营地' }}
          </div>
          <h2>{{ selected.level ? selected.name : selected.chapter.description || selected.name }}</h2>
          <p class="description">{{ selected.level ? (selected.status === 'LOCKED' ? unlockHint : selected.description) : selected.guide.greeting }}</p>
          <div class="place-progress">
            <div><span>{{ selected.level ? '已通过任务' : '已通关据点' }}</span><b>{{ placeDone }} / {{ placeTotal }}</b></div>
            <progress :value="placeDone" :max="placeTotal || 1" :aria-label="selected.level ? '本关任务进度' : '本章进度'" />
          </div>
          <div class="actions">
            <template v-if="selected.level">
              <button v-if="selected.status !== 'LOCKED'" type="button" class="workshop-button warm" @click="openPlace(selectedId)">
                {{ selected.status === 'COMPLETED' ? '回顾本关成果' : placeDone ? '继续本关任务' : '进入本关任务' }} <el-icon><Right /></el-icon>
              </button>
              <button v-else type="button" class="workshop-button warm" @click="openPlace('hub:' + selected.chapter.id)">查看先修路线 <el-icon><Right /></el-icon></button>
              <button v-if="nextLevel && nextLevel.id !== selected.level.id" class="panel-link" @click="locateNext">定位可挑战关卡</button>
              <button v-else-if="selected.status !== 'COMPLETED'" class="panel-link" @click="showMission(selected.chapter.id)">查看本章推荐任务</button>
            </template>
            <template v-else>
              <button type="button" class="workshop-button warm" @click="openPlace(selectedId)">查看本章关卡 <el-icon><Right /></el-icon></button>
              <button v-if="selected.chapter.currentLevelId" class="panel-link" @click="showMission(selected.chapter.id)">打开本章推荐任务</button>
              <button v-else-if="nextLevel" class="panel-link" @click="locateNext">定位可挑战关卡</button>
            </template>
          </div>
        </article>
      </div>
      <ExpeditionJournal :chapters="chapters" :achievements="achievements" :visited="visited" @locate="locateChapter" @mission="showMission(null)" />
    </template>
    <MissionDialog v-if="missionOpen" v-model="missionOpen" :chapter-id="missionChapter" @submitted="refresh" />
  </section>
</template>

<script setup>
import { computed, ref, shallowRef, defineAsyncComponent, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { CircleCheck, Compass, Flag, Location, Lock, Medal, Right } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import 'element-plus/es/components/message/style/css'
import { mapsApi } from '../api/maps'
import { useMapStore } from '../store/map'
import { useAuthStore } from '../store/auth'
import ExpeditionScene from '../world/ExpeditionScene.vue'
import ExpeditionJournal from '../world/ExpeditionJournal.vue'
import { buildAtlas } from '../world/worldAtlas'
import { guideFor } from '../world/campGuides'

const MissionDialog = defineAsyncComponent(() => import('../world/MissionDialog.vue'))
const router = useRouter()
const route = useRoute()
const mapStore = useMapStore()
const auth = useAuthStore()
const loading = ref(true), error = ref(false), missionOpen = ref(false)
const chapters = shallowRef([]), atlas = shallowRef(null), selectedId = ref(null)
const restoreFocus = ref(false)
const scene = ref(null), missionChapter = ref(null)
const visited = shallowRef(new Set()), pendingVisits = new Set()
const totalXp = ref(0), achievements = shallowRef([]), reward = ref(null)
const cleared = computed(() => chapters.value.reduce((sum, chapter) => sum + chapter.levelsCompleted, 0))
const levelCount = computed(() => chapters.value.reduce((sum, chapter) => sum + chapter.totalLevels, 0))
const nextLevel = computed(() => chapters.value.flatMap(chapter => chapter.levels).find(level => level.status === 'LEARNING'))
const levelStatus = { LOCKED: '未解锁', LEARNING: '可挑战', COMPLETED: '已通关' }
const places = computed(() => {
  const result = new Map()
  chapters.value.forEach((chapter, chapterIndex) => {
    result.set('hub:' + chapter.id, { chapter, chapterIndex, guide: guideFor(chapterIndex), name: chapter.name, status: chapter.status })
    for (const level of chapter.levels) {
      result.set('level:' + level.id, { chapter, chapterIndex, level, name: level.name, description: level.description, status: level.status, statusText: levelStatus[level.status] })
    }
  })
  return result
})
const selected = computed(() => places.value.get(selectedId.value))
const placeDone = computed(() => selected.value?.level?.taskDone ?? selected.value?.chapter.levelsCompleted ?? 0)
const placeTotal = computed(() => selected.value?.level?.taskTotal ?? selected.value?.chapter.totalLevels ?? 0)
const unlockHint = computed(() => {
  const names = selected.value?.level?.prerequisiteNames || []
  return names.length ? '完成「' + names.join('」、「') + '」后可以进入。' : '完成前置关卡后可以进入。'
})

let generation = 0, inFlight = false, topology = '', atlasETag = '', refreshTimer, seen
function updateRewards(data) {
  achievements.value = data.achievements || []
  totalXp.value = data.totalXp ?? 0
  const userId = auth.user?.id
  if (!userId) return
  const key = 'world-map-rewards:' + userId
  if (!seen) {
    try { seen = JSON.parse(localStorage.getItem(key)) } catch { /* In-memory tracking still works. */ }
  }
  const completeChapters = chapters.value.filter(chapter => chapter.status === 'COMPLETED')
  if (Array.isArray(seen?.taskIds) && Array.isArray(seen?.chapterIds)) {
    const passed = new Set(seen.taskIds)
    const gained = achievements.value.filter(item => !passed.has(item.taskId))
    const stamps = completeChapters.filter(chapter => !seen.chapterIds.includes(chapter.id))
    if (gained.length || stamps.length) {
      const first = gained[0]
      const gain = totalXp.value - seen.xp
      const title = stamps.length ? '点亮了「' + stamps[0].name + '」章节印章' : '「' + first.title + '」已通过'
      const detail = gained.length ? gained.length + ' 项任务成果已收录' : '可以回顾本章的交付记录'
      reward.value = { title, detail: gain > 0 ? detail + ' · 累计经验增加 ' + gain + ' XP' : detail }
    }
  }
  seen = { taskIds: achievements.value.map(item => item.taskId), chapterIds: completeChapters.map(item => item.id), xp: totalXp.value }
  try { localStorage.setItem(key, JSON.stringify(seen)) } catch { /* The server keeps the achievement records. */ }
}
async function load(checkVersion = false) {
  const conditional = checkVersion === true
  if (!auth.token || (conditional && inFlight)) return
  inFlight = true
  const current = ++generation
  try {
    const { data, headers, status } = await mapsApi.atlas(conditional ? atlasETag : undefined)
    if (current !== generation) return
    if (status === 304) {
      error.value = false
      return
    }
    if (!Array.isArray(data.data?.chapters)) throw new Error('Missing chapters')
    chapters.value = data.data.chapters
    const graphKey = JSON.stringify(chapters.value.map(chapter => [chapter.id, chapter.levels.map(level => level.id)]))
    if (topology !== graphKey) {
      atlas.value = chapters.value.length ? buildAtlas(chapters.value) : null
      topology = graphKey
    }
    const known = new Set(chapters.value.map(chapter => chapter.id))
    visited.value = new Set([...visited.value, ...(data.data.visitedChapterIds || [])].filter(id => known.has(id)))
    if (!places.value.has(selectedId.value)) {
      restoreFocus.value = places.value.has(route.query.place)
      const first = nextLevel.value || chapters.value[0]?.levels[0]
      selectedId.value = restoreFocus.value ? route.query.place : first ? 'level:' + first.id : atlas.value?.nodes[0]?.id
    }
    updateRewards(data.data)
    atlasETag = headers.etag || ''
    error.value = false
  } catch {
    if (current === generation) error.value = true
  } finally {
    if (current === generation) {
      inFlight = false
      loading.value = false
    }
  }
}
function select(id) { selectedId.value = id }
function showMission(chapterId) { missionChapter.value = chapterId; missionOpen.value = true }
async function visitCamp(id) {
  if (visited.value.has(id) || pendingVisits.has(id)) return
  const token = auth.token
  pendingVisits.add(id)
  try {
    await mapsApi.visit(id)
    if (auth.token !== token) return
    visited.value = new Set([...visited.value, id])
    const chapter = chapters.value.find(item => item.id === id)
    if (chapter) ElMessage.success('已记录首次探访 · ' + chapter.name)
  } catch {
    if (auth.token === token) ElMessage.warning('探访记录保存失败，到营地再次拜访即可重试。')
  } finally {
    if (auth.token === token) pendingVisits.delete(id)
  }
}
function locateChapter(id) {
  scene.value?.visitCampById(id)
  scene.value?.$el.scrollIntoView({ block: 'nearest' })
}
function locateNext() {
  if (nextLevel.value) scene.value?.focusLevel(nextLevel.value.id)
}
async function openPlace(id) {
  const place = places.value.get(id)
  if (!place || (place.level && place.status === 'LOCKED')) return
  await router.replace({ path: route.path, query: { ...route.query, view: 'world', place: id }, hash: route.hash })
  router.push(place.level ? '/level/' + place.level.id : '/map/' + place.chapter.id)
}
function refresh() { mapStore.refresh(); load() }
function refreshVisible() { if (!document.hidden) load(true) }
watch(() => auth.token, token => {
  generation++
  inFlight = false
  topology = ''
  atlasETag = ''
  seen = undefined
  chapters.value = []
  atlas.value = null
  selectedId.value = null
  visited.value = new Set()
  pendingVisits.clear()
  totalXp.value = 0
  achievements.value = []
  reward.value = null
  missionOpen.value = false
  error.value = false
  loading.value = Boolean(token)
  if (token) load()
}, { flush: 'sync' })
onMounted(() => {
  load()
  document.addEventListener('visibilitychange', refreshVisible)
  refreshTimer = window.setInterval(refreshVisible, 30000)
})
onBeforeUnmount(() => {
  generation++
  window.clearInterval(refreshTimer)
  document.removeEventListener('visibilitychange', refreshVisible)
})
</script>

<style scoped>
.expedition { display:grid;min-width:0;gap:16px;color:var(--color-text);max-width:1480px;margin-inline:auto; }
.atlas-head { display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap; }
.atlas-head h1 { font-size:clamp(24px,2.7vw,37px);line-height:1.2;letter-spacing:-.045em; }
.atlas-head p { margin-top:6px;font-size:14px;line-height:1.6;color:#536858; }
.stats { display:flex;gap:20px;padding:9px 15px;background:#efeee2;border-radius:12px; }
.stats dt { color:#526454;font-size:11px; }
.stats dd { color:#294c38;font-size:19px;font-weight:750;font-variant-numeric:tabular-nums; }
.stats small { font-size:12px;font-weight:550; }
.atlas-layout { display:grid;grid-template-columns:minmax(0, 2.55fr) minmax(220px, .9fr);gap:12px;align-items:stretch; }
.place-card { display:flex;flex-direction:column;padding:clamp(16px,1.7vw,25px);border-radius:16px;background:#204638;color:#fff9e6; }
.place-chapter { display:flex;gap:6px;align-items:center;font-size:12px;color:#dce6d1; }
.place-status { display:flex;align-items:center;gap:6px;align-self:flex-start;border:1px solid #a1a77a;border-radius:6px;padding:3px 7px;margin-top:19px;color:#f2dfa4;font-size:11px; }
.place-card h2 { margin:8px 0;font-size:clamp(20px,2vw,29px);line-height:1.3;letter-spacing:-.04em;overflow-wrap:anywhere; }
.description { line-height:1.7;font-size:14px;color:#e7efdf;overflow-wrap:anywhere; }
.place-progress { margin-top:auto;padding-top:21px; }
.place-progress div { display:flex;align-items:center;justify-content:space-between;gap:10px;font-size:12px;color:#e5ead8; }
.place-progress progress { display:block;width:100%;height:6px;margin-top:7px;border:0;border-radius:4px;background:#476757;overflow:hidden;appearance:none; }
.place-progress progress::-webkit-progress-bar {background:#476757;}
.place-progress progress::-webkit-progress-value {background:#f1cf79;}
.place-progress progress::-moz-progress-bar {background:#f1cf79;}
.actions { display:grid;gap:5px;margin-top:16px; }
.panel-link { background:none;color:#e3eed9;border:0;padding:8px;min-height:34px;font-size:13px; }
.panel-link:hover {text-decoration:underline;text-underline-offset:3px;}
.world-message,.reward,.sync-error { border:1px solid #d6decf;border-radius:12px;background:#fffdf5;padding:14px;color:#41604d; }
.world-message { display:grid;justify-items:center;gap:12px;padding:32px 16px; }
.reward { display:flex;align-items:center;gap:12px;background:#fbefc7;border-color:#ddc67a; }
.reward > .el-icon { font-size:28px;color:#8c6521; }
.reward strong,.reward span {display:block;}
.reward span {font-size:12px;margin-top:3px;}
.reward button { margin-left:auto;align-self:start;border:0;background:none;color:#5b613d;font-size:21px;padding-inline:5px; }
.mission-link,.sync-error button { border:1px solid #bdcdb7;color:#31533d;background:#f2f5e7;padding:7px 12px;border-radius:6px; }
@media(max-width:1050px) { .atlas-layout {grid-template-columns:minmax(0,2fr) minmax(200px,.9fr);} }
@media(max-width:900px) { .atlas-layout {grid-template-columns:1fr;} .place-card {display:grid;grid-template-columns:minmax(0,1fr) minmax(155px,.65fr);gap:4px 16px;} .place-status {display:none;} .place-card h2,.description,.place-chapter {grid-column:1;} .place-progress {grid-column:2;grid-row:1/3;padding:0;} .actions {grid-column:2;grid-row:3/5;margin:0;align-self:start;} }
@media(max-width:550px) { .expedition {gap:12px;} .atlas-head {gap:9px;} .stats {padding:4px 10px;gap:14px;} .stats div {display:flex;align-items:baseline;gap:4px;} .stats dd {font-size:16px;} .place-card {display:flex;gap:0;} .place-progress {padding-top:12px;} .actions {display:flex;align-items:center;flex-wrap:wrap;align-self:stretch;margin-top:12px;} .actions .workshop-button {min-height:40px;} }
</style>
