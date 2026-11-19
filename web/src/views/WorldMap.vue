<template>
  <section class="expedition" aria-label="学习冒险地图">
    <div v-if="loading || error" class="world-message" :role="error ? 'alert' : 'status'">
      <el-icon><Compass /></el-icon><p>{{ error ? '暂时无法读取学习进度' : '正在展开你的航海图…' }}</p>
      <button v-if="error" type="button" class="quiet" @click="load">重新加载</button>
    </div>
    <div v-else-if="!chapters.length" class="world-message"><el-icon><MapLocation /></el-icon><p>还没有发布学习章节。</p></div>
    <template v-else>
      <div class="atlas-card">
        <header class="atlas-head">
          <div class="chapter-title">
            <div class="chapter-meta"><span class="chapter-mark"></span>第 {{ chapterIndex + 1 }} 章<span class="meta-line"></span>{{ chapterStatus }}</div>
            <h2>{{ chapter.name }}</h2>
          </div>
          <div class="atlas-controls">
            <label class="select-wrap"><el-icon><MapLocation /></el-icon><span>切换章节</span>
              <select :value="chapter.id" aria-label="切换章节" @change="openChapter(Number($event.target.value))">
                <option v-for="(item,index) in chapters" :key="item.id" :value="item.id">{{ index + 1 }} · {{ item.name }}{{ item.status === 'LOCKED' ? '（未解锁）' : '' }}</option>
              </select><el-icon class="caret"><ArrowDown /></el-icon>
            </label>
            <button type="button" class="quiet" @click="missionOpen = true"><el-icon><Tickets /></el-icon>推荐委托</button>
          </div>
        </header>

        <div v-if="chapterBusy || chapterError" class="world-message chapter-message" :role="chapterError ? 'alert' : 'status'">
          <p>{{ chapterError ? '这个章节暂时无法打开' : '正在读取关卡…' }}</p>
          <button v-if="chapterError" class="quiet" @click="openChapter(chapter.id)">重试</button>
        </div>
        <template v-else>
          <ExpeditionScene :key="`${chapter.id}-${page}`" :stops="stops" :starting-id="startingId" :selected-id="selectedId"
                           :theme-index="chapterIndex" :seed="chapter.id * 37 + page" @select="select" @enter="enter" @mission="missionOpen = true" />
          <div class="atlas-foot">
            <div class="chapter-progress"><span>章节进度</span><span class="track"><i :style="{ width: percent + '%' }"></i></span><b>{{ chapter.levelsCompleted }} / {{ chapter.totalLevels }} 关</b></div>
            <nav v-if="pageCount > 1" class="page-controls" aria-label="切换地图路段">
              <button type="button" :disabled="page === 0" aria-label="上一段" @click="changePage(page-1)"><el-icon><ArrowLeft /></el-icon></button>
              <span>路段 {{ page+1 }} / {{ pageCount }}</span>
              <button type="button" :disabled="page + 1 === pageCount" aria-label="下一段" @click="changePage(page+1)"><el-icon><ArrowRight /></el-icon></button>
            </nav>
            <span class="map-hint">点小路或用方向键行走，靠近据点按 E 进入</span>
            <span class="touch-hint">轻点小路行走，点据点查看关卡</span>
          </div>
        </template>
      </div>

      <article v-if="!chapterBusy && !chapterError && selected" class="quest-card" aria-live="polite">
        <div class="quest-icon" :class="selected.status.toLowerCase()"><el-icon><component :is="selected.status === 'LOCKED' ? Lock : selected.status === 'COMPLETED' ? CircleCheck : Flag" /></el-icon></div>
        <div class="quest-main">
          <div class="quest-meta">第 {{ selected.number }} 关 <span></span> {{ STATUS[selected.status] }}</div>
          <h3>{{ selected.name }}</h3>
          <p v-if="selected.status === 'LOCKED'">{{ unlockHint }}</p>
          <p v-else-if="selected.description">{{ selected.description }}</p>
          <div class="quest-stats"><span><el-icon><DocumentChecked /></el-icon>{{ selected.taskDone }} / {{ selected.taskTotal }} 项任务</span><span><el-icon><Medal /></el-icon>{{ selected.xpReward }} XP</span></div>
        </div>
        <button class="quest-action" type="button" :disabled="selected.status === 'LOCKED'" @click="enter(selected)">
          {{ selected.status === 'LOCKED' ? '尚未解锁' : selected.status === 'COMPLETED' ? '回顾关卡' : '进入关卡' }}
          <el-icon><component :is="selected.status === 'LOCKED' ? Lock : Right" /></el-icon>
        </button>
      </article>
      <div v-else-if="!chapterBusy && !chapterError" class="world-message"><p>这一章还没有发布关卡。</p></div>
    </template>
    <MissionDialog v-if="missionOpen" v-model="missionOpen" @submitted="refreshAfterSubmission" />
  </section>
</template>

<script setup>
import { ref, computed, defineAsyncComponent, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { MapLocation, Tickets, ArrowDown, ArrowLeft, ArrowRight, Lock, Flag, CircleCheck, Medal, Right, Compass, DocumentChecked } from '@element-plus/icons-vue'
import { useMapStore } from '../store/map'
import { mapsApi } from '../api/maps'
import ExpeditionScene from '../world/ExpeditionScene.vue'
import { chapterStops, currentLevel, STOPS_PER_PAGE, STATUS } from '../world/expedition'

const MissionDialog = defineAsyncComponent(() => import('../world/MissionDialog.vue'))
const mapStore = useMapStore(), router = useRouter()
const loading = ref(true), error = ref(false), chapterBusy = ref(false), chapterError = ref(false), missionOpen = ref(false)
const progress = ref(null), chapterId = ref(null), detail = ref(null), selectedId = ref(null), startingId = ref(null), page = ref(0)
let chapterRequest = 0, progressRequest = 0
const chapters = computed(() => progress.value?.maps || [])
const chapterIndex = computed(() => chapters.value.findIndex(item => item.id === chapterId.value))
const chapter = computed(() => detail.value || chapters.value[chapterIndex.value])
const chapterStatus = computed(() => ({COMPLETED:'已通关', IN_PROGRESS:'探索中', LOCKED:'未解锁'})[chapter.value?.status])
const levels = computed(() => detail.value?.levels || [])
const pageCount = computed(() => Math.ceil(levels.value.length / STOPS_PER_PAGE))
const stops = computed(() => chapterStops(levels.value, page.value, chapter.value?.status))
const selected = computed(() => stops.value.find(stop => stop.id === selectedId.value))
const unlockHint = computed(() => {
  const names = selected.value?.prerequisiteNames || []
  return names.length ? `完成前置关卡「${names.join('」、「')}」后解锁。` : '完成前置关卡后，可以来这里挑战。'
})
const percent = computed(() => chapter.value?.totalLevels ? Math.min(100, 100 * chapter.value.levelsCompleted / chapter.value.totalLevels) : 0)

function select(stop) { selectedId.value = stop.id }
function enter(stop) {
  if (!stop || stop.status === 'LOCKED') return
  router.push({ path: `/level/${stop.id}`, query: { n: stop.orderIndex } })
}
function changePage(index, preferred) {
  page.value = index
  const choice = stops.value.find(stop => stop.id === preferred) || currentLevel(stops.value, chapter.value.status)
  selectedId.value = choice?.id ?? null
  startingId.value = selectedId.value
}
async function openChapter(id, preferred) {
  const request = ++chapterRequest
  chapterId.value = id
  chapterBusy.value = true
  chapterError.value = false
  detail.value = null
  try {
    const response = await mapsApi.detail(id)
    if (request !== chapterRequest) return
    detail.value = response.data.data
    const choice = levels.value.find(level => level.id === preferred) || currentLevel(levels.value, chapter.value.status)
    const index = Math.max(0, levels.value.indexOf(choice))
    changePage(Math.floor(index / STOPS_PER_PAGE), choice?.id)
  } catch {
    if (request === chapterRequest) chapterError.value = true
  } finally { if (request === chapterRequest) chapterBusy.value = false }
}
async function load() {
  const request = ++progressRequest
  ++chapterRequest
  loading.value = true
  error.value = false
  const previousChapter = chapterId.value, previousLevel = selectedId.value
  try {
    const result = await mapStore.fetchProgress(true)
    if (request !== progressRequest) return
    if (!Array.isArray(result?.maps)) throw new Error('Missing chapter list')
    progress.value = result
    const target = chapters.value.find(item => item.id === previousChapter) ||
      chapters.value.find(item => item.id === result.currentMapId && item.status !== 'LOCKED') ||
      chapters.value.find(item => item.status === 'IN_PROGRESS') || chapters.value[0]
    if (target) {
      chapterId.value = target.id
      loading.value = false
      await openChapter(target.id, previousLevel)
    }
  } catch { if (request === progressRequest) error.value = true }
  finally { if (request === progressRequest) loading.value = false }
}
function refreshAfterSubmission() { load() }
onMounted(load)
onBeforeUnmount(() => { chapterRequest++; progressRequest++ })
</script>

<style scoped>
.expedition { width:100%; min-width:0; display:grid; gap:16px; color:var(--color-text,#193a31); }
.atlas-card,.quest-card,.world-message { background:var(--color-surface,#fffef9); border:1px solid var(--color-border,#d8dfd2); border-radius:16px; }
.atlas-card { overflow:hidden; box-shadow:0 5px 22px #304c3720; }
.atlas-head,.atlas-foot { display:flex; align-items:center; justify-content:space-between; gap:16px; }
.atlas-head { padding:18px 22px 17px; }
.chapter-title { min-width:0; }
.chapter-meta,.quest-meta { display:flex; align-items:center; gap:8px; color:#627565; font-weight:650; font-size:11px; letter-spacing:.06em; }
.chapter-mark { width:7px;height:7px;transform:rotate(45deg);background:#b47e38; }
.meta-line { width:16px;height:1px;background:#aebca7; }
.chapter-title h2 { font-size:clamp(21px,2vw,31px); font-weight:750; color:#203f34; line-height:1.3; margin:5px 0 0; overflow-wrap:anywhere; }
.atlas-controls { display:flex; gap:8px; flex-shrink:0; }
.quiet,.select-wrap { display:inline-flex; align-items:center; justify-content:center; gap:7px; border:1px solid #d6ddcf; background:#f6f7ed; color:#375948; border-radius:8px; padding:8px 11px; font-size:12px; font-weight:650; cursor:pointer; }
.quiet:hover,.select-wrap:hover { background:#edf1e3; }
.select-wrap { position:relative; }
.select-wrap select { position:absolute; inset:0; width:100%; height:100%; opacity:0; cursor:pointer; }
.select-wrap:focus-within,.quest-action:focus-visible,.quiet:focus-visible { outline:2px solid #376952; outline-offset:2px; }
.atlas-controls .el-icon {font-size:15px;} .atlas-controls .caret {font-size:11px;}
.atlas-foot { padding:12px 20px; border-top:1px solid #bdccba; flex-wrap:wrap; }
.chapter-progress,.page-controls { display:flex; align-items:center; gap:10px; font-size:12px; color:#5d7063; }
.chapter-progress b {font-variant-numeric:tabular-nums;color:#375643;}
.track { width:clamp(56px,9vw,124px);height:5px;border-radius:4px;overflow:hidden;background:#dce3d5; }
.track i {display:block;height:100%;background:#49755b;}
.map-hint,.touch-hint {font-size:12px;color:#617365;}
.touch-hint {display:none;}
@media(pointer:coarse),(max-width:620px) { .map-hint {display:none;} .touch-hint {display:inline;} }
.page-controls button { display:grid;place-items:center;border:1px solid #d1d9ca; border-radius:5px;padding:5px; background:#fffef9; color:#365444;cursor:pointer; }
.page-controls button:disabled { opacity:.4;cursor:default; }
.quest-card { display:flex; align-items:center; gap:16px; padding:18px 22px; }
.quest-icon { display:grid; place-items:center; width:50px; height:54px; flex-shrink:0; font-size:24px; color:#93682a; border:1px solid #e5d8b5; background:#f7efd9; border-radius:13px; }
.quest-icon.locked {color:#61746b; background:#edf0e8; border-color:#d8dfd4;}
.quest-icon.completed {color:#346851;background:#e7efe0; border-color:#cdddc3;}
.quest-main {flex:1;min-width:0;}
.quest-meta { color:#7c6943; }
.quest-meta span {height:3px;width:3px;border-radius:50%;background:#ad9d78;}
.quest-main h3 { font-size:clamp(17px,1.6vw,22px); margin:4px 0; }
.quest-main p { font-size:13px;color:#586b5d;line-height:1.65;margin:3px 0; overflow-wrap:anywhere; }
.quest-stats,.quest-stats span { display:flex;align-items:center;gap:5px; }
.quest-stats {margin-top:7px;gap:16px;font-size:12px;color:#69785d;}
.quest-action { display:inline-flex;align-items:center;justify-content:center;gap:12px;border:0;border-radius:9px;background:#235c47;color:#fffdf1;font-weight:700;padding:12px 17px;cursor:pointer; }
.quest-action:hover {background:#164934;}.quest-action:disabled {background:#e4e9df;color:#637164;cursor:default;}
.world-message {display:grid;justify-items:center;gap:12px;padding:50px 16px;text-align:center;color:#506850;}
.world-message p {margin:0;} .world-message>.el-icon {font-size:26px;}
.chapter-message {border-radius:0;border:0;background:#e4ebde;}
@media(max-width:620px) { .atlas-head {align-items:flex-start;padding:14px;gap:10px;flex-wrap:wrap;} .atlas-foot {padding:11px 12px;gap:8px;} .atlas-controls{gap:5px;} .quiet,.select-wrap{font-size:12px;padding:8px;} .atlas-controls .caret {display:none;} .map-hint{display:none;} .quest-card{flex-wrap:wrap;gap:10px;padding:14px;} .quest-icon{width:40px;height:44px;font-size:20px;} .quest-action{width:100%;} }
</style>
