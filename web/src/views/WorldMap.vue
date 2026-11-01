<template>
  <main class="world-shell">
    <header class="game-header">
      <div><span class="overline">A TRAVELER'S TALE</span><h1>雾湾岛</h1><p>循着小路，看看今天会遇见谁。</p></div>
      <div class="travel-log"><span>✧ 旅人手记</span><strong>{{ discovered.length }} / {{ locations.length }} 处足迹</strong></div>
    </header>
    <div v-if="error" class="message" role="alert">暂时无法进入岛屿。<button @click="load">再试一次</button></div>
    <div v-else-if="loading" class="message" role="status">正在靠岸……</div>
    <div v-show="!loading && !error" class="game-stage" :aria-busy="loading">
      <canvas ref="canvas" class="game-canvas" tabindex="0" aria-label="雾湾岛。使用方向键沿道路行走，靠近地点后按 E 互动。" />
      <div class="instructions">沿小路行走 · 点击地点自动寻路 · E 互动</div>
      <div class="scene-tools"><button @click="engine?.toggleZoom()" aria-label="切换全岛和跟随视角">⌕ 视野</button><button @click="journalOpen = true">☷ 手记</button></div>
      <div v-if="nearby" class="place-name" role="status"><span>你来到了</span><strong>{{ nearby.name }}</strong></div>
      <div class="dpad" aria-label="移动控制">
        <button class="up" aria-label="向上走" @pointerdown.prevent="press(0,-1,$event)" @pointerup="release" @pointercancel="release" @lostpointercapture="release">▲</button>
        <button aria-label="向左走" @pointerdown.prevent="press(-1,0,$event)" @pointerup="release" @pointercancel="release" @lostpointercapture="release">◀</button>
        <button aria-label="向下走" @pointerdown.prevent="press(0,1,$event)" @pointerup="release" @pointercancel="release" @lostpointercapture="release">▼</button>
        <button aria-label="向右走" @pointerdown.prevent="press(1,0,$event)" @pointerup="release" @pointercancel="release" @lostpointercapture="release">▶</button>
      </div>
      <button class="interact" :disabled="!nearby" @click="engine?.interact()">✦<small>{{ nearby ? '敲门' : '互动' }}</small></button>
    </div>
    <p v-if="!loading && !error" class="help">WASD / 方向键沿路移动 · 点小路可自动行走 · 走近金色灯火，敲门拜访。</p>

    <el-dialog v-model="journalOpen" title="旅人手记" width="min(92vw, 540px)" @closed="focusGame">
      <p class="journal-note">在小岛上走一走。选一个目的地，就沿着小路出发。</p>
      <div class="destinations"><button v-for="place in locations" :key="place.id" @click="visit(place)"><span>⌂ {{ place.name }}</span><small>{{ discovered.includes(place.id) ? '曾经到访' : '还未拜访' }} →</small></button></div>
    </el-dialog>

    <el-dialog v-model="doorOpen" :title="selectedPlace?.name || '旅人的委托'" width="min(94vw, 840px)" top="5vh" :close-on-click-modal="false" @closed="focusGame">
      <div class="encounter"><div class="traveler-portrait" :style="{ backgroundImage: `url(${character})` }"></div><p>{{ selectedPlace?.greeting }}</p></div>
      <div v-if="missionLoading" class="dialog-state" role="status">正在取出你的委托……</div>
      <div v-else-if="missionError" class="dialog-state" role="alert">委托暂时未能送达。<button @click="loadMission">重新领取</button></div>
      <template v-else-if="mission?.task">
        <div class="mission-heading"><span class="seal">旅人专属委托</span><span class="status">{{ taskStatus }}</span></div>
        <h2 class="mission-title">{{ mission.task.title }}</h2>
        <p v-if="mission.state === 'WAITING_REVIEW'" class="waiting-note">你已提交当前可做的委托，正在等待审核。可以查看记录或补充成果，审核通过后再来领取新的委托。</p>
        <div class="mission-tabs" role="tablist" aria-label="委托内容">
          <button id="mission-task-tab" role="tab" :aria-selected="tab === 'task'" aria-controls="mission-task-panel" :class="{ active: tab === 'task' }" @click="tab = 'task'">委托内容</button>
          <button id="mission-guide-tab" role="tab" :aria-selected="tab === 'guide'" aria-controls="mission-guide-panel" :class="{ active: tab === 'guide' }" @click="tab = 'guide'">随信手册</button>
        </div>
        <div v-if="tab === 'task'" id="mission-task-panel" role="tabpanel" aria-labelledby="mission-task-tab" class="mission-body">
          <TutorialReader :content="mission.task.description || '请向老师了解本项任务的要求。'" />
          <div v-if="mission.task.acceptanceCriteria?.length" class="criteria"><h3>完成这份委托需要</h3><ul><li v-for="(item, index) in mission.task.acceptanceCriteria" :key="index">{{ item.criteria }}</li></ul></div>
          <p v-if="mission.task.aiFeedback" class="feedback">{{ mission.task.aiFeedback }}</p>
        </div>
        <div v-else id="mission-guide-panel" role="tabpanel" aria-labelledby="mission-guide-tab" class="mission-body">
          <details v-for="(tutorial, index) in mission.tutorials" :key="tutorial.id" :open="index === 0"><summary>{{ tutorial.title }}</summary><TutorialReader :content="tutorial.content" /></details>
          <p v-if="!mission.tutorials?.length">这份委托没有附带手册，遇到困难可以向老师求助。</p>
        </div>
        <div class="mission-actions"><button v-if="mission.task.submissionStatus !== 'NONE'" :disabled="opening" @click="openHistory">查看提交记录</button><button :disabled="missionLoading" @click="loadMission">刷新委托</button><button class="primary" :disabled="opening" @click="openSubmit">{{ opening ? '正在打开…' : mission.state === 'WAITING_REVIEW' ? '补充提交' : '提交成果' }}</button></div>
      </template>
      <div v-else class="dialog-state"><strong>{{ emptyTitle }}</strong><p>{{ emptyMessage }}</p><button @click="loadMission">再看看</button></div>
    </el-dialog>
    <component :is="submitComponent" v-if="submitComponent" ref="submitDialog" :task="submissionTask" @done="onSubmitted" />
    <component :is="historyComponent" v-if="historyComponent" ref="historyDialog" />
  </main>
</template>

<script setup>
import { ref, shallowRef, computed, watch, defineAsyncComponent, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { adventureApi } from '../api/adventure'
import { useAuthStore } from '../store/auth'
import { useMapStore } from '../store/map'
import { AdventureEngine } from '../world/adventure'
import { LOCATIONS } from '../world/island'
import island from '../world/assets/adventure-island.webp'
import character from '../world/assets/traveler.webp'

const TutorialReader = defineAsyncComponent(() => import('../components/TutorialReader.vue'))
const auth = useAuthStore(), mapStore = useMapStore(), canvas = ref(null), engine = shallowRef(null)
const loading = ref(true), error = ref(false), journalOpen = ref(false), doorOpen = ref(false), nearby = ref(null)
const selectedPlace = ref(null), mission = ref(null), missionLoading = ref(false), missionError = ref(false), tab = ref('task')
const submitComponent = shallowRef(null), historyComponent = shallowRef(null), submitDialog = ref(null), historyDialog = ref(null), submissionTask = ref(null), opening = ref(false)
const locations = LOCATIONS
const discoveryKey = `island-discoveries:${auth.user?.id || 'traveler'}`
function readDiscoveries() {
  try { const ids = JSON.parse(localStorage.getItem(discoveryKey) || '[]'); return Array.isArray(ids) ? ids.filter(id => locations.some(place => place.id === id)) : [] }
  catch { return [] }
}
const discovered = ref(readDiscoveries())
let requestId = 0, disposed = false, lastHint = 0
const taskStatus = computed(() => ({ NONE: '尚未提交', PENDING: '等待审核', REJECTED: '可以修改后重试', AI_REVIEWED: '等待最终审核', APPROVED: '已完成' }[mission.value?.task?.submissionStatus] || ''))
const emptyTitle = computed(() => mission.value?.state === 'COMPLETE' ? '所有委托都完成了！' : '暂时没有新的委托')
const emptyMessage = computed(() => ({ COMPLETE: '这段旅程的委托已经全部通过，继续在岛上探索吧。', EMPTY: '老师还在准备新的内容，稍后再来看看。', BLOCKED: '当前学习进度需要老师确认，请联系老师后再来领取。' }[mission.value?.state] || '稍后再来看看。'))
watch([journalOpen, doorOpen], ([journal, door]) => {
  engine.value?.setPaused(journal || door)
  if (!door) { requestId++; missionLoading.value = false }
}, { flush: 'sync' })
function press(x, y, event) { event.currentTarget.setPointerCapture(event.pointerId); engine.value?.setControl(x, y) }
function release() { engine.value?.setControl(0, 0) }
function focusGame() { if (!doorOpen.value && !journalOpen.value) canvas.value?.focus({ preventScroll: true }) }
function hint(message) { if (Date.now() - lastHint > 1500) { lastHint = Date.now(); ElMessage.info(message) } }
function visit(place) { journalOpen.value = false; engine.value?.travelTo(place, place) }
async function openPlace(place) {
  if (disposed || doorOpen.value) return
  selectedPlace.value = place
  if (!discovered.value.includes(place.id)) {
    discovered.value.push(place.id)
    try { localStorage.setItem(discoveryKey, JSON.stringify(discovered.value)) } catch { /* Storage can be unavailable in private mode. */ }
  }
  doorOpen.value = true; tab.value = 'task'
  await loadMission()
}
async function loadMission() {
  const current = ++requestId
  missionLoading.value = true; missionError.value = false; mission.value = null
  try {
    const response = await adventureApi.mission()
    if (current === requestId && !disposed) mission.value = response.data.data
  } catch { if (current === requestId && !disposed) missionError.value = true }
  finally { if (current === requestId && !disposed) missionLoading.value = false }
}
async function openSubmit() {
  if (!mission.value?.task || opening.value) return
  opening.value = true
  try {
    submissionTask.value = mission.value.task
    if (!submitComponent.value) submitComponent.value = (await import('../components/TaskSubmitDialog.vue')).default
    if (disposed || !doorOpen.value) return
    await nextTick(); submitDialog.value?.open()
  } catch { ElMessage.error('提交窗口加载失败，请重试。') } finally { opening.value = false }
}
async function openHistory() {
  if (!mission.value?.task || opening.value) return
  const task = mission.value.task; opening.value = true
  try {
    if (!historyComponent.value) historyComponent.value = (await import('../components/SubmissionHistoryDialog.vue')).default
    if (disposed || !doorOpen.value) return
    await nextTick(); historyDialog.value?.open(task)
  } catch { ElMessage.error('提交记录加载失败，请重试。') } finally { opening.value = false }
}
async function onSubmitted() { await loadMission(); mapStore.refresh() }
async function load() {
  engine.value?.stop(); loading.value = true; error.value = false
  try {
    await nextTick()
    if (disposed || !canvas.value) return
    const scene = new AdventureEngine(canvas.value, { background: island, character, onEnter: openPlace, onNearby: place => { nearby.value = place }, onHint: hint })
    engine.value = scene
    await scene.prepare()
    if (disposed) { scene.stop(); return }
    loading.value = false; await nextTick(); scene.start()
  } catch { if (!disposed) { error.value = true; loading.value = false } }
}
onMounted(load)
onBeforeUnmount(() => { disposed = true; requestId++; engine.value?.stop() })
</script>

<style scoped>
.world-shell{max-width:1250px;margin:auto;color:#263e37}.game-header{display:flex;justify-content:space-between;align-items:center;gap:20px;padding:5px 6px 18px}.overline{font-size:10px;letter-spacing:.2em;color:#a37b47;font-weight:800}.game-header h1{font-size:29px;margin:4px 0;color:#294c41;letter-spacing:.12em}.game-header p,.help{font-size:13px;color:#71887e;margin:0}.travel-log{display:grid;gap:6px;text-align:right}.travel-log span{font-size:11px;color:#9d825b;letter-spacing:.1em}.travel-log strong{font-size:13px;color:#5c7565}.game-stage{position:relative;overflow:hidden;border-radius:18px;border:4px solid #3e6654;background:#7cb395;box-shadow:0 15px 35px #203f332b}.game-canvas{width:100%;aspect-ratio:3/2;display:block;cursor:pointer;touch-action:manipulation}.game-canvas:focus{outline:none}.game-canvas:focus-visible{outline:3px solid #f5d38a;outline-offset:-3px}.instructions,.scene-tools button{padding:9px 12px;background:#173c35df;color:#fff5dd;border:1px solid #ffffff52;border-radius:9px;font-size:12px}.instructions{position:absolute;top:12px;left:12px;pointer-events:none}.scene-tools{position:absolute;top:12px;right:12px;display:flex;gap:6px}.scene-tools button{cursor:pointer}.place-name{position:absolute;top:60px;left:50%;transform:translateX(-50%);display:grid;text-align:center;gap:4px;color:#fff4d7;text-shadow:0 2px 4px #143e32;pointer-events:none}.place-name span{font-size:11px}.place-name strong{font-size:19px;letter-spacing:.1em}.help{margin:12px 5px}.dpad{position:absolute;bottom:15px;left:15px;display:grid;grid-template-columns:repeat(3,43px);gap:4px;touch-action:none}.dpad .up{grid-column:2}.dpad button,.interact{border:1px solid #fff8;background:#173c35df;color:#fff5dd;border-radius:12px;height:43px;touch-action:none;font-size:17px;cursor:pointer}.dpad button:active,.interact:active{background:#b88c42}.interact{position:absolute;bottom:20px;right:20px;width:65px;height:65px;border-radius:50%;font-size:25px}.interact:disabled{opacity:.55;cursor:default}.interact small{display:block;font-size:11px}.message{padding:70px;text-align:center}.message button,.dialog-state button{border:0;background:none;color:#38795a;text-decoration:underline;cursor:pointer}.journal-note{line-height:1.8;color:#718275}.destinations{display:grid;gap:8px}.destinations button{display:flex;align-items:center;gap:12px;width:100%;text-align:left;padding:15px;border:1px solid #e4dfd1;background:#fcfaf5;color:#455b46;border-radius:10px;cursor:pointer}.destinations button span{flex:1}.destinations small{color:#8d9686;font-size:12px}.encounter{display:flex;align-items:center;gap:18px;border:1px solid #e8e0cc;background:#fcfaf3;border-radius:12px;padding:12px 18px}.encounter p{font-size:14px;line-height:1.8;color:#6e705d;margin:0}.traveler-portrait{width:48px;height:56px;flex-shrink:0;background-size:144px 224px;background-repeat:no-repeat}.mission-heading{display:flex;justify-content:space-between;align-items:center;margin-top:22px;gap:12px}.seal{font-size:11px;color:#a1793e;letter-spacing:.12em;font-weight:700}.status{font-size:12px;color:#778674}.mission-title{font-size:22px;line-height:1.5;color:#344e3c;margin:8px 0 18px}.mission-tabs{display:flex;gap:10px;border-bottom:1px solid #e5e9df;padding-bottom:10px}.mission-tabs button{padding:8px 14px;border:0;border-radius:7px;background:none;color:#819078;cursor:pointer}.mission-tabs button.active{background:#edf2e6;color:#4a6747;font-weight:700}.mission-body{max-height:48vh;overflow:auto;padding:18px 3px;line-height:1.8}.mission-body summary{cursor:pointer;font-weight:700;padding:12px 0;color:#536b4f}.criteria{margin-top:20px;background:#f7f8f1;padding:14px 18px;border-radius:10px}.criteria h3{font-size:14px;margin:0}.criteria ul{padding-left:20px;font-size:13px}.feedback{white-space:pre-wrap;color:#7c725c;background:#faf5e9;padding:14px;border-radius:8px}.waiting-note{font-size:13px;line-height:1.8;color:#987334;background:#fff8e7;padding:12px;border-radius:8px}.mission-actions{display:flex;justify-content:flex-end;gap:8px;flex-wrap:wrap;padding-top:14px;border-top:1px solid #e5e9df}.mission-actions button{border:1px solid #d8e1d4;border-radius:8px;padding:10px 16px;background:#fff;color:#567150;cursor:pointer}.mission-actions button.primary{background:#436d50;color:#fff;border-color:#436d50}.mission-actions button:disabled{opacity:.5;cursor:wait}.dialog-state{padding:28px 6px;color:#7a856e;line-height:1.8}.world-shell button:focus-visible{outline:3px solid #c99f4f;outline-offset:2px}@media(max-width:650px){.game-header h1{font-size:23px}.game-header p,.instructions{display:none}.game-stage{border-width:3px}.game-canvas{aspect-ratio:4/5}.dpad{bottom:9px;left:9px;grid-template-columns:repeat(3,39px)}.dpad button{height:39px}.interact{width:55px;height:55px;bottom:12px;right:12px}.scene-tools{top:8px;right:8px}.scene-tools button{padding:7px}.encounter{padding:10px}.mission-title{font-size:19px}.mission-body{max-height:44vh}.place-name{top:50px}.travel-log strong{font-size:11px}}
</style>
